package resources

import (
	"k8s.io/apimachinery/pkg/runtime"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

type KindManager interface {
	Keyword() string
	KeywordPlural() string
	GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error)
}

var KindManagerMap SKindManagerMap

type SKindManagerMap map[string]KindManager

func init() {
	KindManagerMap = make(map[string]KindManager)
}

func (m SKindManagerMap) Register(kind string, man KindManager) {
	m[kind] = man
}

func (m SKindManagerMap) Get(obj runtime.Object) KindManager {
	man, ok := m[obj.GetObjectKind().GroupVersionKind().Kind]
	if !ok {
		return nil
	}
	return man
}
