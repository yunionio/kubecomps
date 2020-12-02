package client

/*import (
	"fmt"
	"io"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/api/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"
	//"k8s.io/helm/pkg/kube"

	"yunion.io/x/log"
	"yunion.io/x/kubecomps/pkg/kubeserver/types/api"
)

type GenericClient struct {
	*kube.Client
}

func NewGetter(kubeconfig string) (*GenericConfig, error) {
	return NewGenericConfig(kubeconfig)
}

func NewGeneric(kubeconfig string) (*GenericClient, error) {
	getter, err := NewGetter(kubeconfig)
	if err != nil {
		return nil, err
	}
	cli := &GenericClient{
		Client: kube.New(getter),
	}
	return cli, nil
}

func (c *GenericClient) Get(namespace string, reader io.Reader) (map[string][]runtime.Object, error) {
	// Since we don't know what order the objects come in, let's group them by the types, so
	// that when we print them, they come out looking good (headers apply to subgroups, etc.).
	objs := make(map[string][]runtime.Object)
	infos, err := c.BuildUnstructured(namespace, reader)
	if err != nil {
		return nil, err
	}

	var objPods = make(map[string][]v1.Pod)

	missing := []string{}
	err = processResource(infos, func(info *resource.Info) error {
		log.Debugf("Doing get for %s: %q", info.Mapping.GroupVersionKind.Kind, info.Name)
		if err := info.Get(); err != nil {
			log.Warningf("Failed get for resource %q: %s", info.Name, err)
			missing = append(missing, fmt.Sprintf("%v\t\t%s", info.Mapping.Resource, info.Name))
			return nil
		}

		// Use APIVersion/Kind as grouping mechanism. I'm not sure if you can have multiple
		// versions per cluster, but this certainly won't hurt anything, so let's be safe
		gvk := info.ResourceMapping().GroupVersionKind
		//vk := gvk.Version + "/" + gvk.Kind
		//objs[vk] = append(objs[vk], asVersioned(info))
		kind := strings.ToLower(gvk.Kind)
		objs[kind] = append(objs[kind], asVersioned(info))

		//Get the relation pods
		objPods, err = c.getSelectRelationPod(info, objPods)
		if err != nil {
			log.Warningf("get the relation pod is failed, err: %v", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	//here, we will add the objPods to the objs
	for key, podItems := range objPods {
		for i := range podItems {
			objs[key] = append(objs[key], &podItems[i])
		}
	}

	return objs, nil
}

func processResource(infos kube.Result, fn kube.ResourceActorFunc) error {
	if len(infos) == 0 {
		return kube.ErrNoObjectsVisited
	}

	for _, info := range infos {
		if err := fn(info); err != nil {
			return err
		}
	}
	return nil
}

func asVersioned(info *resource.Info) runtime.Object {
	converter := runtime.ObjectConvertor(scheme.Scheme)
	groupVersioner := runtime.GroupVersioner(schema.GroupVersions(scheme.Scheme.PrioritizedVersionsAllGroups()))
	if info.Mapping != nil {
		groupVersioner = info.Mapping.GroupVersionKind.GroupVersion()
	}

	if obj, err := converter.ConvertToVersion(info.Object, groupVersioner); err == nil {
		return obj
	}
	return info.Object
}

//get a kubernetes resources' relation pods
// kubernetes resource used select labels to relate pods
func (c *GenericClient) getSelectRelationPod(info *resource.Info, objPods map[string][]v1.Pod) (map[string][]v1.Pod, error) {
	if info == nil {
		return objPods, nil
	}

	log.Debugf("get relation pod of object: %s/%s/%s", info.Namespace, info.Mapping.GroupVersionKind.Kind, info.Name)

	versioned := asVersioned(info)
	selector, ok := getSelectorFromObject(versioned)
	if !ok {
		return objPods, nil
	}

	client, _ := c.KubernetesClientSet()

	pods, err := client.CoreV1().Pods(info.Namespace).List(metav1.ListOptions{
		FieldSelector: fields.Everything().String(),
		LabelSelector: labels.Set(selector).AsSelector().String(),
	})
	if err != nil {
		return objPods, err
	}

	for _, pod := range pods.Items {
		if pod.APIVersion == "" {
			pod.APIVersion = "v1"
		}

		if pod.Kind == "" {
			pod.Kind = api.ResourceKindPod
		}
		//vk := pod.GroupVersionKind().Version + "/" + pod.GroupVersionKind().Kind

		if !isFoundPod(objPods[pod.Kind], pod) {
			objPods[pod.Kind] = append(objPods[pod.Kind], pod)
		}
	}
	return objPods, nil
}

func getSelectorFromObject(obj runtime.Object) (map[string]string, bool) {
	switch typed := obj.(type) {

	case *v1.ReplicationController:
		return typed.Spec.Selector, true

	case *extv1beta1.ReplicaSet:
		return typed.Spec.Selector.MatchLabels, true
	case *appsv1.ReplicaSet:
		return typed.Spec.Selector.MatchLabels, true

	case *extv1beta1.Deployment:
		return typed.Spec.Selector.MatchLabels, true
	case *appsv1beta1.Deployment:
		return typed.Spec.Selector.MatchLabels, true
	case *appsv1beta2.Deployment:
		return typed.Spec.Selector.MatchLabels, true
	case *appsv1.Deployment:
		return typed.Spec.Selector.MatchLabels, true

	case *extv1beta1.DaemonSet:
		return typed.Spec.Selector.MatchLabels, true
	case *appsv1beta2.DaemonSet:
		return typed.Spec.Selector.MatchLabels, true
	case *appsv1.DaemonSet:
		return typed.Spec.Selector.MatchLabels, true

	case *batch.Job:
		return typed.Spec.Selector.MatchLabels, true

	case *appsv1beta1.StatefulSet:
		return typed.Spec.Selector.MatchLabels, true
	case *appsv1beta2.StatefulSet:
		return typed.Spec.Selector.MatchLabels, true
	case *appsv1.StatefulSet:
		return typed.Spec.Selector.MatchLabels, true

	default:
		return nil, false
	}
}

func isFoundPod(podItem []v1.Pod, pod v1.Pod) bool {
	for _, value := range podItem {
		if (value.Namespace == pod.Namespace) && (value.Name == pod.Name) {
			return true
		}
	}
	return false
}*/
