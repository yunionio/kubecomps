package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
)

type IMachineDriver interface {
	ValidateCreateData(s *mcclient.ClientSession, input *api.CreateMachineData) error

	GetProvider() api.ProviderType
	GetResourceType() api.MachineResourceType
	GetPrivateIP(session *mcclient.ClientSession, resourceId string) (string, error)
	UseClusterAPI() bool

	PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, m *SMachine, data *jsonutils.JSONDict) error

	RequestPrepareMachine(ctx context.Context, userCred mcclient.TokenCredential, m *SMachine, task taskman.ITask) error
	PrepareResource(session *mcclient.ClientSession, m *SMachine, data *api.MachinePrepareInput) (jsonutils.JSONObject, error)

	ValidateDeleteCondition(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, m *SMachine) error
	TerminateResource(session *mcclient.ClientSession, m *SMachine) error

	// NetworkAddress related interface
	ListNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *SMachine) ([]*computeapi.NetworkAddressDetails, error)
	AttachNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *SMachine, opt *api.MachineAttachNetworkAddressInput) error
	SyncNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *SMachine) error
}

var machineDrivers *drivers.DriverManager

func init() {
	machineDrivers = drivers.NewDriverManager("")
}

func RegisterMachineDriver(driver IMachineDriver) {
	resType := driver.GetResourceType()
	provider := driver.GetProvider()
	err := machineDrivers.Register(driver, string(provider), string(resType))
	if err != nil {
		log.Fatalf("machine driver provider %s, resource type %s driver register error: %v", provider, resType, err)
	}
}

func GetMachineDriver(provider api.ProviderType, resType api.MachineResourceType) IMachineDriver {
	drv, err := machineDrivers.Get(string(provider), string(resType))
	if err != nil {
		log.Fatalf("Get machine driver provider: %s, resource type: %s error: %v", provider, resType, err)
	}
	return drv.(IMachineDriver)
}
