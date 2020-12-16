package manager

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type ICluster interface {
	GetName() string
	GetId() string
	RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error
	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
	//SetKubeconfig(kubeconfig string) error
	GetAPIServer() (string, error)
	GetKubeconfig() (string, error)
	GetStatus() string
	GetK8sResourceManager(kindName string) IK8sResourceManager
	SyncInfoFromCloud(ctx context.Context, s *mcclient.ClientSession) error
}

// bidirect sync callback
type IK8sResourceManager interface {
	db.IModelManager

	OnRemoteObjectCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster ICluster, resMan IK8sResourceManager, obj runtime.Object)
	OnRemoteObjectUpdate(ctx context.Context, userCred mcclient.TokenCredential, cluster ICluster, resMan IK8sResourceManager, oldObj, newObj runtime.Object)
	OnRemoteObjectDelete(ctx context.Context, userCred mcclient.TokenCredential, cluster ICluster, resMan IK8sResourceManager, obj runtime.Object)
}

type IClusterManager interface {
	IsClusterExists(userCred mcclient.TokenCredential, id string) (ICluster, bool, error)
	FetchClusterByIdOrName(userCred mcclient.TokenCredential, id string) (ICluster, error)
	CreateCluster(ctx context.Context, userCred mcclient.TokenCredential, data api.ClusterCreateInput) (ICluster, error)
	//GetNonSystemClusters() ([]ICluster, error)
	GetRunningClusters() ([]ICluster, error)
}

type IMachine interface {
	GetId() string
	GetName() string
	IsFirstNode() bool
	GetResourceId() string
	IsControlplane() bool
	IsRunning() bool
	GetPrivateIP() (string, error)
	GetEIP() (string, error)
	RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error
	GetStatus() string
	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
	SetPrivateIP(address string) error
	GetRole() string
	SyncInfoFromCloud(ctx context.Context, s *mcclient.ClientSession) error
}

type IMachineManager interface {
	FetchMachineByIdOrName(userCred mcclient.TokenCredential, id string) (IMachine, error)
	GetMachines(clusterId string) ([]IMachine, error)
	IsMachineExists(userCred mcclient.TokenCredential, id string) (IMachine, bool, error)
	CreateMachine(ctx context.Context, userCred mcclient.TokenCredential, data *api.CreateMachineData) (IMachine, error)
}

var (
	clusterManager IClusterManager
	machineManager IMachineManager
)

func RegisterClusterManager(man IClusterManager) {
	if clusterManager != nil {
		log.Fatalf("ClusterManager already registered")
	}
	clusterManager = man
}

func RegisterMachineManager(man IMachineManager) {
	if machineManager != nil {
		log.Fatalf("MachineManager already registered")
	}
	machineManager = man
}

func ClusterManager() IClusterManager {
	return clusterManager
}

func MachineManager() IMachineManager {
	return machineManager
}
