package machines

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	ocapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

type sBaseDriver struct{}

func newBaseDriver() *sBaseDriver {
	return &sBaseDriver{}
}

func (d *sBaseDriver) ValidateCreateData(name string) error {
	man := models.MachineManager
	err := man.ValidateName(name)
	if err != nil {
		return err
	}
	q := man.Query()
	q = man.FilterByName(q, name)
	if q.Count() != 0 {
		return httperrors.NewDuplicateNameError("name", name)
	}
	return nil
}

func (d *sBaseDriver) RequestPrepareMachine(ctx context.Context, userCred mcclient.TokenCredential, machine *models.SMachine, task taskman.ITask) error {
	/*cluster, err := machine.GetCluster()
	if err != nil {
		return errors.Wrap(err, "GetCluster")
	}*/
	createInput, err := machine.GetCreateInput(ctx, userCred)
	if err != nil {
		log.Errorf("Get create input error: %v", err)
	}
	input := &api.MachinePrepareInput{
		FirstNode: machine.IsFirstNode(),
		Role:      machine.GetRole(),
		Config:    createInput.Config,
	}
	/*input, err = cluster.FillMachinePrepareInput(input)
	if err != nil {
		return errors.Wrap(err, "FillMachinePrepareInput")
	}*/
	return machine.StartPrepareTask(ctx, task.GetUserCred(), jsonutils.Marshal(input).(*jsonutils.JSONDict), task.GetTaskId())
}

func (d *sBaseDriver) PrepareResource(session *mcclient.ClientSession, machine *models.SMachine, data *api.MachinePrepareInput) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *sBaseDriver) PostDelete(ctx context.Context, userCred mcclient.TokenCredential, m *models.SMachine, t taskman.ITask) error {
	t.SetStageComplete(ctx, nil)
	return nil
}

func (d *sBaseDriver) TerminateResource(session *mcclient.ClientSession, machine *models.SMachine) error {
	return nil
}

func (d *sBaseDriver) GetPrivateIP(session *mcclient.ClientSession, id string) (string, error) {
	return "", errors.Errorf("not impl")
}

func (d *sBaseDriver) UseClusterAPI() bool {
	return false
}

func (d *sBaseDriver) ValidateDeleteCondition(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machine *models.SMachine) error {
	return nil
}

func (d *sBaseDriver) ListNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine) ([]*ocapi.NetworkAddressDetails, error) {
	return nil, errors.Errorf("Machine %s not support list network address", m.GetName())
}

func (d *sBaseDriver) AttachNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine, opt *api.MachineAttachNetworkAddressInput) error {
	return errors.Errorf("Machine %s not support attach network address", m.GetName())
}

func (d *sBaseDriver) SyncNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine) error {
	return errors.Errorf("Machine %s not support sync network address", m.GetName())
}

func (d *sBaseDriver) GetInfoFromCloud(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine) (*api.CloudMachineInfo, error) {
	return nil, nil
}
