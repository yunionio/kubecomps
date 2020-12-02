package common

import (
	"context"
	"encoding/json"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
)

func JsonDecode(data jsonutils.JSONObject, obj interface{}) error {
	dataStr, err := data.GetString()
	if err != nil {
		return err
	}
	err = json.NewDecoder(strings.NewReader(dataStr)).Decode(obj)
	return err
}

func GetK8sObjectCreateMetaByRequest(req *Request) (*metav1.ObjectMeta, error) {
	objMeta, err := GetK8sObjectCreateMeta(req.Data)
	if err != nil {
		return nil, err
	}
	ns := req.GetDefaultNamespace()
	objMeta.Namespace = ns
	return objMeta, nil
}

func GetK8sObjectCreateMeta(data jsonutils.JSONObject) (*metav1.ObjectMeta, error) {
	name, err := data.GetString("name")
	if err != nil {
		return nil, httperrors.NewInputParameterError("name not provided")
	}

	labels := make(map[string]string)
	annotations := make(map[string]string)

	data.Unmarshal(&labels, "labels")
	data.Unmarshal(&annotations, "annotations")
	return &metav1.ObjectMeta{
		Name:        name,
		Labels:      labels,
		Annotations: annotations,
	}, nil
}

func GenerateName(base string) string {
	return api.GenerateName(base)
}

func ToConfigMap(configMap *v1.ConfigMap, cluster api.ICluster) api.ConfigMap {
	return api.ConfigMap{
		ObjectMeta: api.NewObjectMeta(configMap.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(configMap.TypeMeta),
	}
}

func ToConfigMaps(cfgs []*v1.ConfigMap, cluster api.ICluster) []api.ConfigMap {
	ret := make([]api.ConfigMap, 0)
	for _, c := range cfgs {
		ret = append(ret, ToConfigMap(c, cluster))
	}
	return ret
}

func ToSecret(secret *v1.Secret, cluster api.ICluster) *api.Secret {
	return &api.Secret{
		ObjectMeta: api.NewObjectMeta(secret.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(secret.TypeMeta),
		Type:       secret.Type,
	}
}

func ToSecrets(ss []*v1.Secret, cluster api.ICluster) []api.Secret {
	ret := make([]api.Secret, 0)
	for _, s := range ss {
		ret = append(ret, *ToSecret(s, cluster))
	}
	return ret
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

func GetPodTemplate(req *Request, wrapperKey string) (*v1.PodTemplateSpec, error) {
	if wrapperKey == "" {
		wrapperKey = "template"
	}
	ret := &v1.PodTemplateSpec{}
	if err := req.Data.Unmarshal(ret, wrapperKey); err != nil {
		return nil, httperrors.NewInputParameterError("invalid pod template")
	}
	return ret, nil
}

func AddObjectMetaDefaultLabel(meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	return AddObjectMetaRunLabel(meta)
}

func AddObjectMetaRunLabel(meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	if len(meta.Labels) == 0 {
		meta.Labels["run"] = meta.GetName()
	}
	return meta
}

func GetSelectorByObjectMeta(meta *metav1.ObjectMeta) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: meta.GetLabels(),
	}
}

func GetK8sObjectCreateMetaWithLabel(req *Request) (*metav1.ObjectMeta, *metav1.LabelSelector, error) {
	objMeta, err := GetK8sObjectCreateMetaByRequest(req)
	if err != nil {
		return nil, nil, err
	}
	return AddObjectMetaDefaultLabel(objMeta), GetSelectorByObjectMeta(objMeta), nil
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
		selector = GetSelectorByObjectMeta(objMeta).MatchLabels
	}
	svc := &v1.Service{
		ObjectMeta: *objMeta,
		Spec: v1.ServiceSpec{
			Selector: selector,
			Type:     v1.ServiceType(svcType),
			Ports:    GetServicePortsByMapping(opt.PortMappings),
		},
	}
	if opt.LoadBalancerNetwork != "" {
		svc.Annotations = map[string]string{
			YUNION_LB_NETWORK_ANNOTATION: opt.LoadBalancerNetwork,
		}
	}
	return svc
}

func CreateService(req *Request, svc *v1.Service) (*v1.Service, error) {
	return req.GetK8sClient().CoreV1().Services(svc.GetNamespace()).Create(context.Background(), svc, metav1.CreateOptions{})
}

func CreateServiceByOption(req *Request, objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) (*v1.Service, error) {
	svc := GetServiceFromOption(objMeta, opt)
	if svc == nil {
		return nil, nil
	}
	return CreateService(req, svc)
}

func CreateServiceIfNotExist(req *Request, objMeta *metav1.ObjectMeta, opt *api.ServiceCreateOption) (*v1.Service, error) {
	svc, err := req.GetIndexer().ServiceLister().Services(objMeta.GetNamespace()).Get(objMeta.GetName())
	if err != nil {
		if errors.IsNotFound(err) {
			return CreateServiceByOption(req, objMeta, opt)
		}
		return nil, err
	}
	return svc, nil
}
