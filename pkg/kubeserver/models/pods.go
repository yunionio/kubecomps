package models

import (
	"context"
	"encoding/base64"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"math"
	"strconv"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	res "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/getters"
)

var (
	podManager *SPodManager
	_          IClusterModel = new(SPod)
)

func init() {
	GetPodManager()
}

func GetPodManager() *SPodManager {
	if podManager == nil {
		podManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SPodManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SPod{},
					"pods_tbl",
					"pod",
					"pods",
					api.ResourceNamePod,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNamePod,
					new(v1.Pod),
				),
			}
		}).(*SPodManager)
	}
	return podManager
}

type SPodManager struct {
	SNamespaceResourceBaseManager
}

type SPod struct {
	SNamespaceResourceBase
	NodeId string `width:"36" charset:"ascii" nullable:"false"`

	// CpuRequests is number of allocated milicores
	CpuRequests int64 `list:"user"`
	// CpuLimits is defined cpu limit
	CpuLimits int64 `list:"user"`

	// MemoryRequests
	MemoryRequests int64 `list:"user"`
	// MemoryLimits
	MemoryLimits int64
}

func (m *SPodManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support pod create")
}

func (m *SPodManager) GetGCQuery() *sqlchemy.SQuery {
	q := m.SNamespaceResourceBaseManager.GetGCQuery()
	nodeIds := GetNodeManager().Query("id").SubQuery()
	q = q.NotIn("node_id", nodeIds)
	return q
}

func (m *SPodManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.PodListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.NamespaceResourceListInput)
	return q, err
}

func (p *SPod) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.PodDetailV2, error) {
	return api.PodDetailV2{}, nil
}

func (p *SPodManager) getPodStatus(pod *v1.Pod) api.PodStatus {
	var states []v1.ContainerState
	for _, containerStatus := range pod.Status.ContainerStatuses {
		states = append(states, containerStatus.State)
	}
	return api.PodStatus{
		PodStatusV2:     *getters.GetPodStatus(pod),
		PodPhase:        pod.Status.Phase,
		ContainerStates: states,
	}
}

func (p *SPodManager) getPodConditions(pod *v1.Pod) []api.Condition {
	var conditions []api.Condition
	for _, condition := range pod.Status.Conditions {
		conditions = append(conditions, api.Condition{
			Type:               string(condition.Type),
			Status:             condition.Status,
			LastProbeTime:      condition.LastProbeTime,
			LastTransitionTime: condition.LastTransitionTime,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	return conditions
}

func (p SPodManager) getRestartCount(pod *v1.Pod) int32 {
	var restartCount int32 = 0
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restartCount += containerStatus.RestartCount
	}
	return restartCount
}

// extractContainerResourceValue extracts the value of a resource in an already known container.
func extractContainerResourceValue(fs *v1.ResourceFieldSelector, container *v1.Container) (string,
	error) {
	divisor := res.Quantity{}
	if divisor.Cmp(fs.Divisor) == 0 {
		divisor = res.MustParse("1")
	} else {
		divisor = fs.Divisor
	}

	switch fs.Resource {
	case "limits.cpu":
		return strconv.FormatInt(int64(math.Ceil(float64(container.Resources.Limits.
			Cpu().MilliValue())/float64(divisor.MilliValue()))), 10), nil
	case "limits.memory":
		return strconv.FormatInt(int64(math.Ceil(float64(container.Resources.Limits.
			Memory().Value())/float64(divisor.Value()))), 10), nil
	case "requests.cpu":
		return strconv.FormatInt(int64(math.Ceil(float64(container.Resources.Requests.
			Cpu().MilliValue())/float64(divisor.MilliValue()))), 10), nil
	case "requests.memory":
		return strconv.FormatInt(int64(math.Ceil(float64(container.Resources.Requests.
			Memory().Value())/float64(divisor.Value()))), 10), nil
	}

	return "", fmt.Errorf("Unsupported container resource : %v", fs.Resource)
}

// evalValueFrom evaluates environment value from given source. For more details check:
// https://github.com/kubernetes/kubernetes/blob/d82e51edc5f02bff39661203c9b503d054c3493b/pkg/kubectl/describe.go#L1056
func evalValueFrom(src *v1.EnvVarSource, container *v1.Container, pod *v1.Pod,
	configMaps []*v1.ConfigMap, secrets []*v1.Secret) string {
	switch {
	case src.ConfigMapKeyRef != nil:
		name := src.ConfigMapKeyRef.LocalObjectReference.Name
		for _, configMap := range configMaps {
			if configMap.ObjectMeta.Name == name {
				return configMap.Data[src.ConfigMapKeyRef.Key]
			}
		}
	case src.SecretKeyRef != nil:
		name := src.SecretKeyRef.LocalObjectReference.Name
		for _, secret := range secrets {
			if secret.ObjectMeta.Name == name {
				return base64.StdEncoding.EncodeToString([]byte(
					secret.Data[src.SecretKeyRef.Key]))
			}
		}
	case src.ResourceFieldRef != nil:
		valueFrom, err := extractContainerResourceValue(src.ResourceFieldRef, container)
		if err != nil {
			valueFrom = ""
		}
		resource := src.ResourceFieldRef.Resource
		if valueFrom == "0" && (resource == "limits.cpu" || resource == "limits.memory") {
			valueFrom = "node allocatable"
		}
		return valueFrom
	case src.FieldRef != nil:
		gv, err := schema.ParseGroupVersion(src.FieldRef.APIVersion)
		if err != nil {
			log.V(2).Warningf("%v", err)
			return ""
		}
		gvk := gv.WithKind("Pod")
		internalFieldPath, _, err := runtime.NewScheme().ConvertFieldLabel(gvk, src.FieldRef.FieldPath, "")
		if err != nil {
			log.V(2).Warningf("%v", err)
			return ""
		}
		valueFrom, err := ExtractFieldPathAsString(pod, internalFieldPath)
		if err != nil {
			log.V(2).Warningf("%v", err)
			return ""
		}
		return valueFrom
	}
	return ""
}

// FormatMap formats map[string]string to a string.
func FormatMap(m map[string]string) (fmtStr string) {
	for key, value := range m {
		fmtStr += fmt.Sprintf("%v=%q\n", key, value)
	}
	fmtStr = strings.TrimSuffix(fmtStr, "\n")

	return
}

// ExtractFieldPathAsString extracts the field from the given object
// and returns it as a string.  The object must be a pointer to an
// API type.
func ExtractFieldPathAsString(obj interface{}, fieldPath string) (string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", nil
	}

	switch fieldPath {
	case "metadata.annotations":
		return FormatMap(accessor.GetAnnotations()), nil
	case "metadata.labels":
		return FormatMap(accessor.GetLabels()), nil
	case "metadata.name":
		return accessor.GetName(), nil
	case "metadata.namespace":
		return accessor.GetNamespace(), nil
	}

	return "", fmt.Errorf("unsupported fieldPath: %v", fieldPath)
}

func extractContainerInfo(containerList []v1.Container, pod *v1.Pod, configMaps []*v1.ConfigMap, secrets []*v1.Secret) []api.Container {
	containers := make([]api.Container, 0)
	for _, container := range containerList {
		vars := make([]api.EnvVar, 0)
		for _, envVar := range container.Env {
			variable := api.EnvVar{
				Name:      envVar.Name,
				Value:     envVar.Value,
				ValueFrom: envVar.ValueFrom,
			}
			if variable.ValueFrom != nil {
				variable.Value = evalValueFrom(variable.ValueFrom, &container, pod,
					configMaps, secrets)
			}
			vars = append(vars, variable)
		}
		vars = append(vars, evalEnvFrom(container, configMaps, secrets)...)

		containers = append(containers, api.Container{
			Name:     container.Name,
			Image:    container.Image,
			Env:      vars,
			Commands: container.Command,
			Args:     container.Args,
		})
	}
	return containers
}

func evalEnvFrom(container v1.Container, configMaps []*v1.ConfigMap, secrets []*v1.Secret) []api.EnvVar {
	vars := make([]api.EnvVar, 0)
	for _, envFromVar := range container.EnvFrom {
		switch {
		case envFromVar.ConfigMapRef != nil:
			name := envFromVar.ConfigMapRef.LocalObjectReference.Name
			for _, configMap := range configMaps {
				if configMap.ObjectMeta.Name == name {
					for key, value := range configMap.Data {
						valueFrom := &v1.EnvVarSource{
							ConfigMapKeyRef: &v1.ConfigMapKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: name,
								},
								Key: key,
							},
						}
						variable := api.EnvVar{
							Name:      envFromVar.Prefix + key,
							Value:     value,
							ValueFrom: valueFrom,
						}
						vars = append(vars, variable)
					}
					break
				}
			}
		case envFromVar.SecretRef != nil:
			name := envFromVar.SecretRef.LocalObjectReference.Name
			for _, secret := range secrets {
				if secret.ObjectMeta.Name == name {
					for key, value := range secret.Data {
						valueFrom := &v1.EnvVarSource{
							SecretKeyRef: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: name,
								},
								Key: key,
							},
						}
						variable := api.EnvVar{
							Name:      envFromVar.Prefix + key,
							Value:     base64.StdEncoding.EncodeToString(value),
							ValueFrom: valueFrom,
						}
						vars = append(vars, variable)
					}
					break
				}
			}
		}
	}
	return vars
}

func (p *SPod) getConditions(pod *v1.Pod) []*api.Condition {
	var conds []*api.Condition
	for _, cond := range pod.Status.Conditions {
		conds = append(conds, &api.Condition{
			Type:               string(cond.Type),
			Status:             cond.Status,
			LastProbeTime:      cond.LastProbeTime,
			LastTransitionTime: cond.LastTransitionTime,
			Reason:             cond.Reason,
			Message:            cond.Message,
		})
	}
	return SortConditions(conds)
}

func (p *SPod) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	pod := k8sObj.(*v1.Pod)
	warnings, _ := GetEventManager().GetWarningEventsByPods(cli, []*v1.Pod{pod})
	out := api.PodDetailV2{
		NamespaceResourceDetail: p.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Warnings:                warnings,
		PodStatus:               GetPodManager().getPodStatus(pod),
		RestartCount:            GetPodManager().getRestartCount(pod),
		PodIP:                   pod.Status.PodIP,
		QOSClass:                string(pod.Status.QOSClass),
		ContainerImages:         GetContainerImages(&pod.Spec),
		InitContainerImages:     GetInitContainerImages(&pod.Spec),
	}
	configmaps, err := GetConfigMapManager().GetRawConfigMaps(cli, pod.GetNamespace())
	if err != nil {
		log.Errorf("Get configmaps error: %v", err)
		return out
	}
	secrets, err := GetSecretManager().GetRawSecrets(cli, pod.GetNamespace())
	if err != nil {
		log.Errorf("Get secrets error: %v", err)
		return out
	}
	out.Containers = extractContainerInfo(pod.Spec.Containers, pod, configmaps, secrets)
	out.InitContainers = extractContainerInfo(pod.Spec.InitContainers, pod, configmaps, secrets)
	if isList {
		return out
	}
	out.Conditions = p.getConditions(pod)
	// TODO: fill secrets, pvcs...
	return out
}

func (m *SPodManager) GetAllRawPods(cluster *client.ClusterManager) ([]*v1.Pod, error) {
	return m.GetRawPods(cluster, v1.NamespaceAll)
}

func (m *SPodManager) GetRawPodsByObjectNamespace(cli *client.ClusterManager, obj runtime.Object) ([]*v1.Pod, error) {
	return m.GetRawPods(cli, obj.(metav1.Object).GetNamespace())
}

func (m *SPodManager) GetRawPods(cluster *client.ClusterManager, ns string) ([]*v1.Pod, error) {
	return m.GetRawPodsBySelector(cluster, ns, labels.Everything())
}

func (m *SPodManager) GetRawPodsBySelector(cluster *client.ClusterManager, ns string, selector labels.Selector) ([]*v1.Pod, error) {
	indexer := cluster.GetHandler().GetIndexer()

	list, err := indexer.PodLister().ByNamespace(ns).List(selector)
	if err != nil {
		return nil, err
	}
	res := make([]*v1.Pod, len(list))
	for idx, l := range list {
		newObj := &v1.Pod{}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.(*unstructured.Unstructured).Object, newObj); err != nil {
			return nil, err
		}
		res[idx] = newObj
	}
	return res, nil
}

func (m *SPodManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	model, err := m.SNamespaceResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	var kPod v1.Pod
	err = runtime.DefaultUnstructuredConverter.
		FromUnstructured(obj.(*unstructured.Unstructured).Object, &kPod)
	if err != nil {
		return nil, err
	}
	podObj := model.(*SPod)
	nodeName := kPod.Spec.NodeName
	if nodeName != "" {
		nodeObj, err := GetNodeManager().GetByName(userCred, podObj.ClusterId, nodeName)
		if err != nil {
			return nil, errors.Wrapf(err, "fetch pod's node by name: %s", nodeName)
		}
		podObj.NodeId = nodeObj.GetId()
	}
	return podObj, nil
}

// PodRequestsAndLimits returns a dictionary of all defined resources summed up for all
// containers of the pod.
func PodRequestsAndLimits(pod *v1.Pod) (reqs map[v1.ResourceName]resource.Quantity, limits map[v1.ResourceName]resource.Quantity, err error) {
	reqs, limits = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if value, ok := reqs[name]; !ok {
				reqs[name] = quantity.DeepCopy()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = quantity.DeepCopy()
			} else {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		for name, quantity := range container.Resources.Requests {
			value, ok := reqs[name]
			if !ok {
				reqs[name] = quantity.DeepCopy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				reqs[name] = quantity.DeepCopy()
			}
		}
		for name, quantity := range container.Resources.Limits {
			value, ok := limits[name]
			if !ok {
				limits[name] = quantity.DeepCopy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				limits[name] = quantity.DeepCopy()
			}
		}
	}
	return
}

func (p *SPod) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := p.SNamespaceResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	var k8sPod v1.Pod
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(extObj.(*unstructured.Unstructured).Object, &k8sPod)
	if err != nil {
		return err
	}

	reqs, limits, err := PodRequestsAndLimits(&k8sPod)
	if err != nil {
		return errors.Wrap(err, "get pod resource requests and limits")
	}
	cpuRequests, cpuLimits, memoryRequests, memoryLimits := reqs[v1.ResourceCPU],
		limits[v1.ResourceCPU], reqs[v1.ResourceMemory], limits[v1.ResourceMemory]
	p.CpuRequests = cpuRequests.MilliValue()
	p.CpuLimits = cpuLimits.MilliValue()
	p.MemoryRequests = memoryRequests.MilliValue()
	p.MemoryLimits = memoryLimits.MilliValue()
	return nil
}

func (p *SPod) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	var k8sPod v1.Pod
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(extObj.(*unstructured.Unstructured).Object, &k8sPod)
	if err != nil {
		return err
	}
	status := getters.GetPodStatus(&k8sPod)
	if status.Status != p.Status {
		p.Status = status.Status
	}
	return nil
}

func (m *SPodManager) GetPodsByClusters(clusterIds []string) ([]SPod, error) {
	pods := make([]SPod, 0)
	if err := GetResourcesByClusters(m, clusterIds, &pods); err != nil {
		return nil, err
	}
	return pods, nil
}

func (p *SPod) IsOwnedBy(ownerModel IClusterModel) (bool, error) {
	return IsPodOwner(ownerModel.(IPodOwnerModel), p)
}
