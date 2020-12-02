package release

import (
	"bytes"

	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"

	"yunion.io/x/log"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
)

func GetReleaseResources(
	cli *helm.Client, rel *release.Release,
	indexer *client.CacheFactory, cluster api.ICluster,
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
		man := model.GetK8sModelManagerByKind(gvk.Kind)
		if man == nil {
			log.Warningf("not fond %s manager", gvk.Kind)
			return nil
		}
		keyword := man.Keyword()
		unstructObj := info.Object.(*unstructured.Unstructured)
		newObj := man.GetK8sResourceInfo().Object.DeepCopyObject()
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructObj.Object, newObj); err != nil {
			return err
		}
		namespace := info.Namespace
		metaObj := newObj.(metav1.Object)
		modelObj, err := model.NewK8SModelObjectByName(man, clusterMan, namespace, metaObj.GetName())
		if err != nil {
			return err
		}
		obj, err := model.GetObject(modelObj)
		if err != nil {
			return err
		}
		if list, ok := ret[keyword]; ok {
			list = append(list, obj)
		} else {
			list = []interface{}{obj}
			ret[keyword] = list
		}
		return nil
	})
	return ret, nil
}
