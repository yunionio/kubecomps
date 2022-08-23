package models

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/api/legacyscheme"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
	k8sutil "yunion.io/x/kubecomps/pkg/kubeserver/k8s/util"
	"yunion.io/x/kubecomps/pkg/utils/k8serrors"
)

func RunBatchTask(
	ctx context.Context,
	items []db.IStandaloneModel,
	userCred mcclient.TokenCredential,
	data jsonutils.JSONObject,
	taskName, parentTaskId string,
) error {
	params := data.(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewParallelTask(ctx, taskName, items, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return fmt.Errorf("%s newTask error %s", taskName, err)
	}
	task.ScheduleRun(nil)
	return nil
}

func (m *SClusterManager) GetSystemClusterKubeconfig(apiServer string, cfg *rest.Config) (string, error) {
	cli, err := kubernetes.NewForConfig(cfg)
	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		// return "", errors.Errorf("Not found NAMESPACE in env")
		ns = NamespaceOneCloud
	}
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		return "", errors.Errorf("Not found HOSTNAME in env")
	}
	selfPod, err := cli.CoreV1().Pods(ns).Get(context.Background(), hostname, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "get pod %s/%s", ns, hostname)
	}
	svcAccount := selfPod.Spec.ServiceAccountName
	if err != nil {
		return "", errors.Wrap(err, "new kubernetes client")
	}
	token := cfg.BearerToken
	caData, err := ioutil.ReadFile(cfg.TLSClientConfig.CAFile)
	if err != nil {
		return "", errors.Wrapf(err, "read ca file %s", cfg.TLSClientConfig.CAFile)
	}

	tmplInput := map[string]string{
		"ClusterName": SystemClusterName,
		"Server":      apiServer,
		"Cert":        base64.StdEncoding.EncodeToString(caData),
		"User":        svcAccount,
		"Token":       token,
	}

	tmpl := `apiVersion: v1
kind: Config
clusters:
- name: "{{.ClusterName}}"
  cluster:
    server: "{{.Server}}"
    certificate-authority-data: "{{.Cert}}"
users:
- name: "{{.User}}"
  user:
    token: "{{.Token}}"
contexts:
- name: "{{.ClusterName}}"
  context:
    user: "{{.User}}"
    cluster: "{{.ClusterName}}"
current-context: "{{.ClusterName}}"
`
	outBuf := &bytes.Buffer{}
	if err := template.Must(template.New("tokenTemplate").Parse(tmpl)).Execute(outBuf, tmplInput); err != nil {
		return "", errors.Wrap(err, "generate kubeconfig")
	}
	return outBuf.String(), nil
}

func NewCheckIdOrNameError(res, resName string, err error) error {
	if errors.Cause(err) == sql.ErrNoRows {
		return httperrors.NewNotFoundError(fmt.Sprintf("resource %s/%s not found: %v", res, resName, err))
	}
	if errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
		return httperrors.NewDuplicateResourceError(fmt.Sprintf("resource %s/%s duplicate: %v", res, resName, err))
	}
	return httperrors.NewGeneralError(err)
}

func NewHelmClient(cluster *SCluster, namespace string) (*helm.Client, error) {
	clusterMan, err := client.GetManagerByCluster(cluster)
	if err != nil {
		return nil, err
	}
	kubeconfigPath, err := clusterMan.GetKubeConfigPath()
	if err != nil {
		return nil, err
	}
	return helm.NewClient(kubeconfigPath, namespace, true)
}

func EnsureNamespace(cluster *SCluster, namespace string) error {
	k8sMan, err := client.GetManagerByCluster(cluster)
	if err != nil {
		return errors.Wrap(err, "get cluster k8s manager")
	}
	lister := k8sMan.GetIndexer().NamespaceLister()
	cli, err := cluster.GetK8sClient()
	if err != nil {
		return errors.Wrap(err, "get cluster k8s client")
	}
	return k8sutil.EnsureNamespace(lister, cli, namespace)
}

func GetReleaseResources(
	cli *helm.Client, rel *release.Release,
	clusterMan model.ICluster,
) (map[string][]interface{}, error) {
	cfg := cli.GetConfig()
	ress, err := cfg.KubeClient.Build(bytes.NewBufferString(rel.Manifest), true)
	if err != nil {
		return nil, err
	}
	ret := make(map[string][]interface{})
	ress.Visit(func(info *resource.Info, err error) error {
		gvk := info.Object.GetObjectKind().GroupVersionKind()
		man := GetOriginK8sModelManager(gvk.Kind)
		if man == nil {
			log.Warningf("not fond %s manager", gvk.Kind)
			return nil
		}

		obj := info.Object.(*unstructured.Unstructured)
		objGVK := obj.GroupVersionKind()
		keyword := man.Keyword()
		handler := clusterMan.GetHandler()
		dCli, err := handler.Dynamic(objGVK.GroupKind(), objGVK.Version)
		if err != nil {
			log.Warningf("get %#v dynamic client error: %v", objGVK, err)
			return nil
		}
		getObj, err := dCli.Namespace(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
		// getObj, err := handler.DynamicGet(objGVK, obj.GetNamespace(), obj.GetName())
		if err != nil {
			log.Warningf("get resource %#v error: %v", objGVK, err)
			return nil
		}

		var jsonObj interface{}
		if k8sMan, ok := man.(model.IK8sModelManager); ok {
			modelObj, err := model.NewK8SModelObject(k8sMan, clusterMan, getObj)
			if err != nil {
				log.Errorf("%s NewK8sModelObject error: %v", keyword, err)
				return errors.Wrapf(err, "%s NewK8SModelObject", keyword)
			}
			jsonObj, err = model.GetObject(modelObj)
			if err != nil {
				return errors.Wrapf(err, "get %s object", modelObj.Keyword())
			}
		} else if dbMan, ok := man.(IClusterModelManager); ok {
			newObj := dbMan.GetK8sResourceInfo().Object.DeepCopyObject()
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(getObj.Object, newObj); err != nil {
				return errors.Wrapf(err, "convert from unstructured %v", getObj)
			}
			cluster := clusterMan.GetClusterObject().(*SCluster)
			// TODO: avoid use context.Background() and admin usercred
			dbObj, err := dbMan.NewFromRemoteObject(context.Background(), GetAdminCred(), cluster, newObj)
			if err != nil {
				return errors.Wrapf(err, "NewFromRemoteObject %v", getObj)
			}
			nsObj, isNsObj := dbObj.(INamespaceModel)
			if isNsObj {
				dbObj, err = FetchClusterResourceByName(dbMan, GetAdminCred(), cluster.GetId(), nsObj.GetNamespaceId(), nsObj.GetName())
			} else {
				dbObj, err = FetchClusterResourceByName(dbMan, GetAdminCred(), cluster.GetId(), "", nsObj.GetName())
			}
			if err != nil {
				return errors.Wrapf(err, "Fetch DB object %v", getObj)
			}
			ret := dbMan.FetchCustomizeColumns(context.Background(), GetAdminCred(), jsonutils.NewDict(), []interface{}{dbObj}, nil, true)
			jsonDictObj := jsonutils.Marshal(dbObj).(*jsonutils.JSONDict)
			jsonDictObj.Update(jsonutils.Marshal(ret[0]))
			jsonObj = jsonDictObj
		} else {
			log.Warningf("Invalid manager %s: %v", man.Keyword(), man)
			return nil
		}

		if list, ok := ret[keyword]; ok {
			list = append(list, jsonObj)
			ret[keyword] = list
		} else {
			list = []interface{}{jsonObj}
			ret[keyword] = list
		}
		return nil
	})
	return ret, nil
}

func GetChartRawFiles(chObj *chart.Chart) []*chart.File {
	files := make([]*chart.File, len(chObj.Raw))
	for idx, rf := range chObj.Raw {
		files[idx] = &chart.File{
			Name: filepath.Join(chObj.Name(), rf.Name),
			Data: rf.Data,
		}
	}
	return files
}

func GetK8SObjectTypeMeta(kObj runtime.Object) metav1.TypeMeta {
	v := reflect.ValueOf(kObj)
	if uObj, ok := kObj.(*unstructured.Unstructured); ok {
		return metav1.TypeMeta{
			APIVersion: uObj.GetAPIVersion(),
			Kind:       uObj.GetKind(),
		}
	} else {
		f := reflect.Indirect(v).FieldByName("TypeMeta")
		if !f.IsValid() {
			panic(fmt.Sprintf("get invalid object meta %#v", kObj))
		}
		return f.Interface().(metav1.TypeMeta)
	}
}

func K8SObjectToJSONObject(obj runtime.Object) jsonutils.JSONObject {
	ov := reflect.ValueOf(obj)
	return ValueToJSONDict(ov)
}

func isJSONObject(input interface{}) (jsonutils.JSONObject, bool) {
	val := reflect.ValueOf(input)
	obj, ok := val.Interface().(jsonutils.JSONObject)
	if !ok {
		return nil, false
	}
	return obj, true
}

func ValueToJSONObject(out reflect.Value) jsonutils.JSONObject {
	if gotypes.IsNil(out.Interface()) {
		return nil
	}

	if obj, ok := isJSONObject(out); ok {
		return obj
	}
	jsonBytes, err := json.Marshal(out.Interface())
	if err != nil {
		panic(fmt.Sprintf("marshal json: %v", err))
	}
	jObj, err := jsonutils.Parse(jsonBytes)
	if err != nil {
		panic(fmt.Sprintf("jsonutils.Parse bytes: %s, error %v", jsonBytes, err))
	}
	return jObj
}

func ValueToJSONDict(out reflect.Value) *jsonutils.JSONDict {
	jsonObj := ValueToJSONObject(out)
	if jsonObj == nil {
		return nil
	}
	return jsonObj.(*jsonutils.JSONDict)
}

func GetServicePortsByMapping(ps []api.PortMapping) []v1.ServicePort {
	ports := []v1.ServicePort{}
	for _, p := range ps {
		ports = append(ports, p.ToServicePort())
	}
	return ports
}

func GetServiceFromOption(objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) *v1.Service {
	if opt == nil {
		return nil
	}
	svcType := opt.Type
	if svcType == "" {
		svcType = string(v1.ServiceTypeClusterIP)
	}
	if opt.IsExternal {
		svcType = string(v1.ServiceTypeLoadBalancer)
	}
	selector := opt.Selector
	if len(selector) == 0 {
		selector = api.GetSelectorByObjectMeta(objMeta).MatchLabels
	}
	svc := &v1.Service{
		ObjectMeta: *objMeta,
		Spec: v1.ServiceSpec{
			Selector: selector,
			Type:     v1.ServiceType(svcType),
			Ports:    GetServicePortsByMapping(opt.PortMappings),
		},
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	if opt.LoadBalancerNetwork != "" {
		svc.Annotations[api.YUNION_LB_NETWORK_ANNOTATION] = opt.LoadBalancerNetwork
	}
	if opt.LoadBalancerCluster != "" {
		svc.Annotations[api.YUNION_LB_CLUSTER_ANNOTATION] = opt.LoadBalancerCluster
	}
	return svc
}

func ValidateAppCreateService(userCred mcclient.TokenCredential, nsInput api.NamespaceResourceCreateInput, opt *api.ServiceCreateOption, objMeta *metav1.ObjectMeta) error {
	if opt == nil {
		return nil
	}
	m := GetServiceManager()
	svc := GetServiceFromOption(objMeta, opt)
	clusterId := nsInput.ClusterId
	namespaceId := nsInput.NamespaceId
	dbObj, err := m.GetByName(userCred, clusterId, namespaceId, svc.GetName())
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return err
		}
	}
	if dbObj != nil {
		return httperrors.NewDuplicateNameError(dbObj.GetName(), dbObj.GetId())
	}
	if err := GetServiceManager().ValidateService(svc); err != nil {
		return errors.Wrap(err, "validate service")
	}
	return nil
}

func ValidatePodTemplate(userCred mcclient.TokenCredential, clusterId string, namespaceId string, template *v1.PodTemplateSpec) error {
	vols := template.Spec.Volumes
	validatePVC := func(vol v1.Volume) (v1.Volume, error) {
		pvc := vol.PersistentVolumeClaim
		pvcObj, err := GetPVCManager().GetByIdOrName(userCred, clusterId, namespaceId, pvc.ClaimName)
		if err != nil {
			return v1.Volume{}, errors.Wrapf(err, "get pvc by claimName %s", pvc.ClaimName)
		}
		vol.PersistentVolumeClaim.ClaimName = pvcObj.GetName()
		return vol, nil
	}
	for idx, vol := range vols {
		if vol.PersistentVolumeClaim == nil {
			continue
		}
		vol, err := validatePVC(vol)
		if err != nil {
			return errors.Wrap(err, "validate pvc")
		}
		vols[idx] = vol
	}
	return nil
}

func CreateServiceIfNotExist(cli *client.ClusterManager, objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) (*v1.Service, error) {
	svc, err := cli.GetHandler().GetIndexer().ServiceLister().Services(objMeta.GetNamespace()).Get(objMeta.GetName())
	if err != nil {
		if kerrors.IsNotFound(err) {
			return CreateServiceByOption(cli, objMeta, opt)
		}
		return nil, err
	}
	return svc, nil
}

func CreateServiceByOption(cli *client.ClusterManager, objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) (*v1.Service, error) {
	svc := GetServiceFromOption(objMeta, opt)
	if svc == nil {
		return nil, nil
	}
	return CreateService(cli, svc)
}

func CreateService(cliMan *client.ClusterManager, svc *v1.Service) (*v1.Service, error) {
	cli := cliMan.GetClientset()
	return cli.CoreV1().Services(svc.GetNamespace()).Create(context.Background(), svc, metav1.CreateOptions{})
}

// GetContainerImages returns container image strings from the given pod spec.
func GetContainerImages(podTemplate *v1.PodSpec) []api.ContainerImage {
	containerImages := []api.ContainerImage{}
	for _, container := range podTemplate.Containers {
		containerImages = append(containerImages, api.ContainerImage{
			Name:  container.Name,
			Image: container.Image,
		})
	}
	return containerImages
}

// GetInitContainerImages returns init container image strings from the given pod spec.
func GetInitContainerImages(podTemplate *v1.PodSpec) []api.ContainerImage {
	initContainerImages := []api.ContainerImage{}
	for _, initContainer := range podTemplate.InitContainers {
		initContainerImages = append(initContainerImages, api.ContainerImage{
			Name:  initContainer.Name,
			Image: initContainer.Image})
	}
	return initContainerImages
}

type condtionSorter []*api.Condition

func (s condtionSorter) Len() int {
	return len(s)
}

func (s condtionSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s condtionSorter) Less(i, j int) bool {
	c1 := s[i]
	c2 := s[j]
	return c1.LastTransitionTime.Before(&c2.LastTransitionTime)
}

func SortConditions(conds []*api.Condition) []*api.Condition {
	sort.Sort(condtionSorter(conds))
	return conds
}

// FilterPodsByControllerResource returns a subset of pods controlled by given deployment.
func FilterDeploymentPodsByOwnerReference(deployment *apps.Deployment, allRS []*apps.ReplicaSet,
	allPods []*v1.Pod) []*v1.Pod {
	var matchingPods []*v1.Pod
	for _, rs := range allRS {
		if metav1.IsControlledBy(rs, deployment) {
			matchingPods = append(matchingPods, FilterPodsByControllerRef(rs, allPods)...)
		}
	}

	return matchingPods
}

// FilterPodsByControllerRef returns a subset of pods controlled by given controller resource, excluding deployments.
func FilterPodsByControllerRef(owner metav1.Object, allPods []*v1.Pod) []*v1.Pod {
	var matchingPods []*v1.Pod
	for _, pod := range allPods {
		if metav1.IsControlledBy(pod, owner) {
			matchingPods = append(matchingPods, pod)
		}
	}
	return matchingPods
}

func GetRawPodsByController(cli *client.ClusterManager, obj metav1.Object) ([]*v1.Pod, error) {
	pods, err := GetPodManager().GetRawPods(cli, obj.GetNamespace())
	if err != nil {
		return nil, err
	}
	return FilterPodsByControllerRef(obj, pods), nil
}

// getPodInfo returns aggregate information about a group of pods.
func getPodInfo(current int32, desired *int32, pods []*v1.Pod) api.PodInfo {
	result := api.PodInfo{
		Current:  current,
		Desired:  desired,
		Warnings: make([]api.Event, 0),
	}

	for _, pod := range pods {
		switch pod.Status.Phase {
		case v1.PodRunning:
			result.Running++
		case v1.PodPending:
			result.Pending++
		case v1.PodFailed:
			result.Failed++
		case v1.PodSucceeded:
			result.Succeeded++
		}
	}

	return result
}

func GetPodInfo(current int32, desired *int32, pods []*v1.Pod) (*api.PodInfo, error) {
	podInfo := getPodInfo(current, desired, pods)
	// TODO: fill warnEvents
	// warnEvents, err := EventManager.GetWarningEventsByPods(obj.GetCluster(), pods)
	return &podInfo, nil
}

// GetInternalEndpoint returns internal endpoint name for the given service properties, e.g.,
// "my-service.namespace 80/TCP" or "my-service 53/TCP,53/UDP".
func GetInternalEndpoint(serviceName, namespace string, ports []v1.ServicePort) api.Endpoint {
	name := serviceName

	if namespace != v1.NamespaceDefault && len(namespace) > 0 && len(serviceName) > 0 {
		bufferName := bytes.NewBufferString(name)
		bufferName.WriteString(".")
		bufferName.WriteString(namespace)
		name = bufferName.String()
	}

	return api.Endpoint{
		Host:  name,
		Ports: GetServicePorts(ports),
	}
}

// Returns external endpoint name for the given service properties.
func getExternalEndpoint(ingress v1.LoadBalancerIngress, ports []v1.ServicePort) api.Endpoint {
	var host string
	if ingress.Hostname != "" {
		host = ingress.Hostname
	} else {
		host = ingress.IP
	}
	return api.Endpoint{
		Host:  host,
		Ports: GetServicePorts(ports),
	}
}

// GetServicePorts returns human readable name for the given service ports list.
func GetServicePorts(apiPorts []v1.ServicePort) []api.ServicePort {
	var ports []api.ServicePort
	for _, port := range apiPorts {
		ports = append(ports, api.ServicePort{port.Port, port.Protocol, port.NodePort})
	}
	return ports
}

// GetExternalEndpoints returns endpoints that are externally reachable for a service.
func GetExternalEndpoints(service *v1.Service) []api.Endpoint {
	var externalEndpoints []api.Endpoint
	if service.Spec.Type == v1.ServiceTypeLoadBalancer {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			externalEndpoints = append(externalEndpoints, getExternalEndpoint(ingress, service.Spec.Ports))
		}
	}

	for _, ip := range service.Spec.ExternalIPs {
		externalEndpoints = append(externalEndpoints, api.Endpoint{
			Host:  ip,
			Ports: GetServicePorts(service.Spec.Ports),
		})
	}

	return externalEndpoints
}

func getPodResourceVolumes(pod *v1.Pod, predicateF func(v1.Volume) bool) []v1.Volume {
	var cfgs []v1.Volume
	vols := pod.Spec.Volumes
	for _, vol := range vols {
		if predicateF(vol) {
			cfgs = append(cfgs, vol)
		}
	}
	return cfgs
}

func GetPodSecretVolumes(pod *v1.Pod) []v1.Volume {
	return getPodResourceVolumes(pod, func(vol v1.Volume) bool {
		return vol.VolumeSource.Secret != nil
	})
}

func GetPodConfigMapVolumes(pod *v1.Pod) []v1.Volume {
	return getPodResourceVolumes(pod, func(vol v1.Volume) bool {
		return vol.VolumeSource.ConfigMap != nil
	})
}

func GetConfigMapsForPod(pod *v1.Pod, cfgs []*v1.ConfigMap) []*v1.ConfigMap {
	if len(cfgs) == 0 {
		return nil
	}
	ret := make([]*v1.ConfigMap, 0)
	uniqM := make(map[string]bool, 0)
	for _, cfg := range cfgs {
		for _, vol := range GetPodConfigMapVolumes(pod) {
			if vol.ConfigMap.Name == cfg.GetName() {
				if _, ok := uniqM[cfg.GetName()]; !ok {
					uniqM[cfg.GetName()] = true
					ret = append(ret, cfg)
				}
			}
		}
	}
	return ret
}

func GetSecretsForPod(pod *v1.Pod, ss []*v1.Secret) []*v1.Secret {
	if len(ss) == 0 {
		return nil
	}
	ret := make([]*v1.Secret, 0)
	uniqM := make(map[string]bool, 0)
	for _, s := range ss {
		for _, vol := range GetPodSecretVolumes(pod) {
			if vol.Secret.SecretName == s.GetName() {
				if _, ok := uniqM[s.GetName()]; !ok {
					uniqM[s.GetName()] = true
					ret = append(ret, s)
				}
			}
		}
	}
	return ret
}

func ValidateCreateK8sObject(versionObj runtime.Object, internalObj interface{}, validateFunc func(internalObj interface{}) field.ErrorList) error {
	legacyscheme.Scheme.Default(versionObj)
	if err := legacyscheme.Scheme.Convert(versionObj, internalObj, nil); err != nil {
		return k8serrors.NewGeneralError(err)
	}
	if err := validateFunc(internalObj).ToAggregate(); err != nil {
		return httperrors.NewInputParameterError("%s", err)
	}
	return nil
}

func ValidateUpdateK8sObject(ovObj, nvObj runtime.Object, oObj, nObj interface{}, validateFunc func(newObj, oldObj interface{}) field.ErrorList) error {
	if err := legacyscheme.Scheme.Convert(ovObj, oObj, nil); err != nil {
		return k8serrors.NewGeneralError(err)
	}
	if err := legacyscheme.Scheme.Convert(nvObj, nObj, nil); err != nil {
		return k8serrors.NewGeneralError(err)
	}
	if err := validateFunc(nObj, oObj).ToAggregate(); err != nil {
		return httperrors.NewInputParameterError("%s", err)
	}
	return nil
}

// Methods below are taken from kubernetes repo:
// https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/deployment/util/deployment_util.go

// FindNewReplicaSet returns the new RS this given deployment targets (the one with the same pod template).
func FindNewReplicaSet(deployment *apps.Deployment, rsList []*apps.ReplicaSet) (*apps.ReplicaSet, error) {
	newRSTemplate := GetNewReplicaSetTemplate(deployment)
	for i := range rsList {
		if EqualIgnoreHash(rsList[i].Spec.Template, newRSTemplate) {
			// This is the new ReplicaSet.
			return rsList[i], nil
		}
	}
	// new ReplicaSet does not exist.
	return nil, nil
}

// GetNewReplicaSetTemplate returns the desired PodTemplateSpec for the new ReplicaSet corresponding to the given ReplicaSet.
// Callers of this helper need to set the DefaultDeploymentUniqueLabelKey k/v pair.
func GetNewReplicaSetTemplate(deployment *apps.Deployment) v1.PodTemplateSpec {
	// newRS will have the same template as in deployment spec.
	return v1.PodTemplateSpec{
		ObjectMeta: deployment.Spec.Template.ObjectMeta,
		Spec:       deployment.Spec.Template.Spec,
	}
}

// EqualIgnoreHash returns true if two given podTemplateSpec are equal, ignoring the diff in value of Labels[pod-template-hash]
// We ignore pod-template-hash because the hash result would be different upon podTemplateSpec API changes
// (e.g. the addition of a new field will cause the hash code to change)
// Note that we assume input podTemplateSpecs contain non-empty labels
func EqualIgnoreHash(template1, template2 v1.PodTemplateSpec) bool {
	// First, compare template.Labels (ignoring hash)
	labels1, labels2 := template1.Labels, template2.Labels
	if len(labels1) > len(labels2) {
		labels1, labels2 = labels2, labels1
	}
	// We make sure len(labels2) >= len(labels1)
	for k, v := range labels2 {
		if labels1[k] != v && k != apps.DefaultDeploymentUniqueLabelKey {
			return false
		}
	}
	// Then, compare the templates without comparing their labels
	template1.Labels, template2.Labels = nil, nil
	return equality.Semantic.DeepEqual(template1, template2)
}

func UpdatePodTemplate(temp *v1.PodTemplateSpec, input api.PodTemplateUpdateInput) error {
	if len(input.RestartPolicy) != 0 {
		temp.Spec.RestartPolicy = input.RestartPolicy
	}
	if len(input.DNSPolicy) != 0 {
		temp.Spec.DNSPolicy = input.DNSPolicy
	}
	cf := func(container *v1.Container, cs []api.ContainerUpdateInput) error {
		if len(cs) == 0 {
			return nil
		}
		for _, c := range cs {
			if container.Name == c.Name {
				container.Image = c.Image
				return nil
			}
		}
		return httperrors.NewNotFoundError("Not found container %s in input", container.Name)
	}
	for i, c := range temp.Spec.InitContainers {
		if err := cf(&c, input.InitContainers); err != nil {
			return err
		}
		temp.Spec.InitContainers[i] = c
	}
	for i, c := range temp.Spec.Containers {
		if err := cf(&c, input.Containers); err != nil {
			return err
		}
		temp.Spec.Containers[i] = c
	}
	return nil
}

// objs is: []*objs, e.g.: []*v1.Pod{}
// targets is: the pointer of []*v, e.g.: &[]*api.Pod{}
func ConvertRawToAPIObjects(
	man model.IK8sModelManager,
	cluster model.ICluster,
	objs interface{},
	targets interface{}) error {
	objsVal := reflect.ValueOf(objs)
	// get targets slice value
	targetsValue := reflect.Indirect(reflect.ValueOf(targets))
	for i := 0; i < objsVal.Len(); i++ {
		objVal := objsVal.Index(i)
		// get targetType *v, the pointer of targetType
		targetPtrType := targetsValue.Type().Elem()
		// get targetType v
		targetType := targetPtrType.Elem()
		// target is the *v instance
		target := reflect.New(targetType).Interface()
		if err := ConvertRawToAPIObject(man, cluster, objVal.Interface().(runtime.Object), target); err != nil {
			return err
		}
		newTargets := reflect.Append(targetsValue, reflect.ValueOf(target))
		targetsValue.Set(newTargets)
	}
	return nil
}

func ConvertRawToAPIObject(
	man model.IK8sModelManager,
	cluster model.ICluster,
	obj runtime.Object, target interface{}) error {
	mObj, err := model.NewK8SModelObject(man, cluster, obj)
	if err != nil {
		return err
	}
	mv := reflect.ValueOf(mObj)
	funcVal, err := model.FindFunc(mv, model.DMethodGetAPIObject)
	if err != nil {
		return err
	}
	ret := funcVal.Call(nil)
	if len(ret) != 2 {
		return fmt.Errorf("invalidate %s %s return value number", man.Keyword(), model.DMethodGetAPIObject)
	}
	if err := model.ValueToError(ret[1]); err != nil {
		return err
	}
	targetVal := reflect.ValueOf(target)
	targetVal.Elem().Set(ret[0].Elem())
	return nil
}
