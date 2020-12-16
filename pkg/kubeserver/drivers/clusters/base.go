package clusters

import (
	"context"

	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
)

type SBaseDriver struct {
	modeType            api.ModeType
	providerType        api.ProviderType
	clusterResourceType api.ClusterResourceType
}

func newBaseDriver(mt api.ModeType, pt api.ProviderType, ct api.ClusterResourceType) *SBaseDriver {
	return &SBaseDriver{
		modeType:            mt,
		providerType:        pt,
		clusterResourceType: ct,
	}
}

func (d *SBaseDriver) GetMode() api.ModeType {
	return d.modeType
}

func (d *SBaseDriver) GetProvider() api.ProviderType {
	return d.providerType
}

func (d *SBaseDriver) GetResourceType() api.ClusterResourceType {
	return d.clusterResourceType
}

func (d *SBaseDriver) GetMachineDriver(mT api.MachineResourceType) models.IMachineDriver {
	drv := models.GetMachineDriver(d.GetProvider(), mT)
	return drv
}

func (d *SBaseDriver) ValidateCreateData(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ClusterCreateInput) error {
	return nil
}

func (d *SBaseDriver) ValidateDeleteCondition() error {
	return nil
}

func (d *SBaseDriver) NeedGenerateCertificate() bool {
	return false
}

func (d *SBaseDriver) NeedCreateMachines() bool {
	return true
}

func (d *SBaseDriver) CreateClusterResource(man *models.SClusterManager, data *api.ClusterCreateInput) error {
	// do nothing
	return nil
}

func (d *SBaseDriver) GetAddonsManifest(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) (string, error) {
	return "", nil
}

func (d *SBaseDriver) ValidateCreateMachines(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *models.SCluster,
	_ *api.ClusterMachineCommonInfo,
	data []*api.CreateMachineData,
) ([]*api.CreateMachineData, []*api.CreateMachineData, error) {
	var needControlplane bool
	var err error
	var clusterId string
	if cluster != nil {
		clusterId = cluster.GetId()
		needControlplane, err = cluster.NeedControlplane()
	}
	if err != nil {
		return nil, nil, errors.Wrapf(err, "check cluster need controlplane")
	}
	controls, nodes := drivers.GetControlplaneMachineDatas(clusterId, data)
	if needControlplane {
		if len(controls) == 0 {
			return nil, nil, httperrors.NewInputParameterError("controlplane node must created")
		}
	}
	return controls, nodes, nil
}

func (d *SBaseDriver) CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, data []*api.CreateMachineData) ([]manager.IMachine, error) {
	return nil, nil
}

func (d *SBaseDriver) StartSyncStatus(cluster *models.SCluster, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterSyncstatusTask", cluster, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (d *SBaseDriver) GetUsableInstances(s *mcclient.ClientSession) ([]api.UsableInstance, error) {
	return nil, nil
}

func (d *SBaseDriver) RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machines []manager.IMachine, task taskman.ITask) error {
	return nil
}

func (d *SBaseDriver) RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machines []manager.IMachine, task taskman.ITask) error {
	return nil
}

func (d *SBaseDriver) ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machines []manager.IMachine) error {
	return nil
}

func (d *SBaseDriver) GetClusterUsers(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUser, error) {
	return nil, nil
}

func (d *SBaseDriver) GetClusterUserGroups(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUserGroup, error) {
	return nil, nil
}
