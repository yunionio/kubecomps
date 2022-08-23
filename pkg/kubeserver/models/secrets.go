package models

import (
	"context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	// "yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
)

var (
	secretManager *SSecretManager
	_             IPodOwnerModel = new(SSecret)
)

func GetSecretManager() *SSecretManager {
	if secretManager == nil {
		secretManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SSecretManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SSecret{},
					"secrets_tbl",
					"secret",
					"secrets",
					api.ResourceNameSecret,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameSecret,
					new(v1.Secret),
				),
				driverManager: drivers.NewDriverManager(""),
			}
		}).(*SSecretManager)
	}
	return secretManager
}

func init() {
	GetSecretManager()
}

type ISecretDriver interface {
	ValidateCreateData(input *api.SecretCreateInput) error
	ToData(input *api.SecretCreateInput) (map[string]string, error)
}

type SSecretManager struct {
	SNamespaceResourceBaseManager
	driverManager *drivers.DriverManager
}

type SSecret struct {
	SNamespaceResourceBase
	Type string `width:"64" charset:"ascii" nullable:"false" list:"user"`
}

func (m SSecretManager) GetDriver(typ v1.SecretType) (ISecretDriver, error) {
	drv, err := m.driverManager.Get(string(typ))
	if err != nil {
		if errors.Cause(err) == drivers.ErrDriverNotFound {
			return nil, httperrors.NewNotFoundError("secret get %s driver", typ)
		}
		return nil, err
	}
	return drv.(ISecretDriver), nil
}

func (m *SSecretManager) RegisterDriver(typ v1.SecretType, driver ISecretDriver) {
	if err := m.driverManager.Register(driver, string(typ)); err != nil {
		panic(errors.Wrapf(err, "secret register driver %s", typ))
	}
}

func (m *SSecretManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.SecretCreateInput) (*api.SecretCreateInput, error) {
	if _, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput); err != nil {
		return input, err
	}
	if input.Type == "" {
		return nil, httperrors.NewNotEmptyError("type is empty")
	}
	drv, err := m.GetDriver(input.Type)
	if err != nil {
		return nil, err
	}
	return input, drv.ValidateCreateData(input)
}

func (obj *SSecret) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	rawPods, err := GetPodManager().GetRawPodsByObjectNamespace(cli, rawObj)
	if err != nil {
		return nil, err
	}
	secName := obj.GetName()
	mountPods := make([]*v1.Pod, 0)
	markMap := make(map[string]bool, 0)
	for _, pod := range rawPods {
		cfgs := GetPodSecretVolumes(pod)
		for _, cfg := range cfgs {
			if cfg.Secret.SecretName == secName {
				if _, ok := markMap[pod.GetName()]; !ok {
					mountPods = append(mountPods, pod)
					markMap[pod.GetName()] = true
				}
			}
		}
	}
	return mountPods, err
}

func (obj *SSecret) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	rs := k8sObj.(*v1.Secret)
	detail := api.SecretDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Type:                    rs.Type,
	}
	if isList {
		return detail
	}
	detail.Data = rs.Data
	return detail
}

func (m *SSecretManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, body jsonutils.JSONObject) (interface{}, error) {
	input := new(api.SecretCreateInput)
	body.Unmarshal(input)
	if input.Type == "" {
		return nil, httperrors.NewNotEmptyError("type is empty")
	}
	drv, err := m.GetDriver(input.Type)
	if err != nil {
		return nil, err
	}
	data, err := drv.ToData(input)
	if err != nil {
		return nil, err
	}
	dataBytes := make(map[string][]byte)
	for k, v := range data {
		dataBytes[k] = []byte(v)
	}
	objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, err
	}
	return &v1.Secret{
		ObjectMeta: objMeta,
		Type:       input.Type,
		Data:       dataBytes,
	}, nil
}

func (m *SSecretManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.SecretListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.NamespaceResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNamespaceResourceBaseManager.ListItemFilter")
	}
	if input.Type != "" {
		q = q.Equals("type", input.Type)
	}
	return q, nil
}

/*func (m *SSecretManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	input := new(api.SecretListInput)
	if err := query.Unmarshal(input); err != nil {
		return nil, err
	}
	filters := db.NewCustomizeListFilters()
	ff := func(obj jsonutils.JSONObject) (bool, error) {
		secType, _ := obj.GetString("type")
		if input.Type != "" {
			log.Errorf("===try obj %s, type %q", obj, input.Type)
			if secType == input.Type {
				return true, nil
			} else {
				return false, nil
			}
		}
		return true, nil
	}
	filters.Append(ff)
	return filters, nil
}*/

func (m SSecretManager) GetRawSecrets(cluster *client.ClusterManager, ns string) ([]*v1.Secret, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.SecretLister().Secrets(ns).List(labels.Everything())
}

func (s *SSecret) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := s.SNamespaceResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	var rawSec v1.Secret
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(extObj.(*unstructured.Unstructured).Object, &rawSec)
	if err != nil {
		return err
	}
	s.Type = string(rawSec.Type)
	return nil
}
