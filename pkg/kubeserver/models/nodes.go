package models

import (
	"context"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	nodeManager *SNodeManager
	_           IPodOwnerModel = new(SNode)
)

func init() {
	GetNodeManager()
}

func GetNodeManager() *SNodeManager {
	if nodeManager == nil {
		nodeManager = NewK8sModelManager(func() ISyncableManager {
			return &SNodeManager{
				SClusterResourceBaseManager: NewClusterResourceBaseManager(
					SNode{},
					"nodes_tbl",
					"k8s_node",
					"k8s_nodes",
					api.ResourceNameNode,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameNode,
					new(v1.Node),
				),
			}
		}).(*SNodeManager)
	}
	return nodeManager
}

type SNodeManager struct {
	SClusterResourceBaseManager
}

type SNode struct {
	SClusterResourceBase

	// v1.NodeSystemInfo
	NodeInfo jsonutils.JSONObject `list:"user"`
	// v1.NodeAddress
	Address jsonutils.JSONObject `list:"user"`

	// CpuCapacity is specified node CPU capacity in milicores
	CpuCapacity int64 `list:"user"`

	// MemoryCapacity is specified node memory capacity in bytes
	MemoryCapacity int64 `list:"user"`

	// PodCapacity is maximum number of pods that can be allocated on given node
	PodCapacity int64 `list:"user"`
}

func (m *SNodeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.NodeCreateInput) (*api.NodeCreateInput, error) {
	return nil, httperrors.NewBadRequestError("Not support node create")
}

func (m *SNodeManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NodeListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.ClusterResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SNodeManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []interface{} {
	return m.SClusterResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
}

func (node *SNode) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	allPods, err := GetPodManager().GetAllRawPods(cli)
	if err != nil {
		return nil, errors.Wrapf(err, "Get node %s pods", node.GetName())
	}
	rNode := rawObj.(*v1.Node)
	ret := make([]*v1.Pod, 0)
	for _, p := range allPods {
		if p.Spec.NodeName == rNode.Name && p.Status.Phase != v1.PodSucceeded && p.Status.Phase != v1.PodFailed {
			ret = append(ret, p)
		}
	}
	return ret, nil
}

func (obj *SNode) getNodeAllocatedResources(node *v1.Node, pods []*v1.Pod) (api.NodeAllocatedResources, error) {
	reqs, limits := map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}

	for _, pod := range pods {
		podReqs, podLimits, err := PodRequestsAndLimits(pod)
		if err != nil {
			return api.NodeAllocatedResources{}, err
		}
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = podReqValue.DeepCopy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = podLimitValue.DeepCopy()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}

	cpuRequests, cpuLimits, memoryRequests, memoryLimits := reqs[v1.ResourceCPU],
		limits[v1.ResourceCPU], reqs[v1.ResourceMemory], limits[v1.ResourceMemory]

	var cpuRequestsFraction, cpuLimitsFraction float64 = 0, 0
	if capacity := float64(node.Status.Capacity.Cpu().MilliValue()); capacity > 0 {
		cpuRequestsFraction = float64(cpuRequests.MilliValue()) / capacity * 100
		cpuLimitsFraction = float64(cpuLimits.MilliValue()) / capacity * 100
	}

	var memoryRequestsFraction, memoryLimitsFraction float64 = 0, 0
	if capacity := float64(node.Status.Capacity.Memory().MilliValue()); capacity > 0 {
		memoryRequestsFraction = float64(memoryRequests.MilliValue()) / capacity * 100
		memoryLimitsFraction = float64(memoryLimits.MilliValue()) / capacity * 100
	}

	var podFraction float64 = 0
	var podCapacity int64 = node.Status.Capacity.Pods().Value()
	if podCapacity > 0 {
		podFraction = float64(len(pods)) / float64(podCapacity) * 100
	}

	return api.NodeAllocatedResources{
		CPURequests:            cpuRequests.MilliValue(),
		CPURequestsFraction:    cpuRequestsFraction,
		CPULimits:              cpuLimits.MilliValue(),
		CPULimitsFraction:      cpuLimitsFraction,
		CPUCapacity:            node.Status.Capacity.Cpu().MilliValue(),
		MemoryRequests:         memoryRequests.Value(),
		MemoryRequestsFraction: memoryRequestsFraction,
		MemoryLimits:           memoryLimits.Value(),
		MemoryLimitsFraction:   memoryLimitsFraction,
		MemoryCapacity:         node.Status.Capacity.Memory().Value(),
		AllocatedPods:          len(pods),
		PodCapacity:            podCapacity,
		PodFraction:            podFraction,
	}, nil
}

func (node *SNode) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	rNode := k8sObj.(*v1.Node)
	out := api.NodeDetailV2{
		ClusterResourceDetail: node.SClusterResourceBase.GetDetails(cli, base, k8sObj, isList).(api.ClusterResourceDetail),
		Ready:                 node.getNodeConditionStatus(rNode, v1.NodeReady) == v1.ConditionTrue,
		Address:               rNode.Status.Addresses,
		NodeInfo:              rNode.Status.NodeInfo,
		Taints:                rNode.Spec.Taints,
		Unschedulable:         rNode.Spec.Unschedulable,
	}
	pods, err := node.GetRawPods(cli, rNode)
	if err != nil {
		log.Errorf("get pods: %v", err)
		return out
	}
	allocatedResources, err := node.getNodeAllocatedResources(rNode, pods)
	if err != nil {
		log.Errorf("get node allocatedResources: %v", err)
		return out
	}
	out.AllocatedResources = allocatedResources
	if isList {
		return out
	}
	out.Phase = rNode.Status.Phase
	out.PodCIDR = rNode.Spec.PodCIDR
	out.ProviderID = rNode.Spec.ProviderID
	out.ContainerImages = node.getContainerImages(*rNode)
	// TODO: fill others details
	return out
}

// getContainerImages returns container image strings from the given node.
func (obj *SNode) getContainerImages(node v1.Node) []string {
	var containerImages []string
	for _, image := range node.Status.Images {
		for _, name := range image.Names {
			containerImages = append(containerImages, name)
		}
	}
	return containerImages
}

func (node *SNode) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.NodeDetailV2, error) {
	return api.NodeDetailV2{}, nil
}

// NewFromRemoteObject create local db SNode model by remote k8s node object
func (m *SNodeManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	return m.SClusterResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
}

func (node *SNode) getNodeConditionStatus(k8sNode *v1.Node, conditionType v1.NodeConditionType) v1.ConditionStatus {
	for _, condition := range k8sNode.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return v1.ConditionUnknown
}

func (node *SNode) getStatusFromRemote(k8sNode *v1.Node) string {
	readyCondStatus := node.getNodeConditionStatus(k8sNode, v1.NodeReady)
	if readyCondStatus == v1.ConditionTrue {
		return api.NodeStatusReady
	}
	return api.NodeStatusNotReady
}

// UpdateFromRemoteObject update local db SNode model by remote k8s node object
func (node *SNode) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := node.SClusterResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	k8sNode := extObj.(*v1.Node)
	node.Address = jsonutils.Marshal(k8sNode.Status.Addresses)
	node.NodeInfo = jsonutils.Marshal(k8sNode.Status.NodeInfo)
	node.updateCapacity(k8sNode)
	return nil
}

func (node *SNode) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	k8sNode := extObj.(*v1.Node)
	node.Status = node.getStatusFromRemote(k8sNode)
	return nil
}

func (node *SNode) updateCapacity(k8sNode *v1.Node) {
	capacity := k8sNode.Status.Capacity
	// cpu status
	node.CpuCapacity = capacity.Cpu().MilliValue()
	// memory status
	node.MemoryCapacity = capacity.Memory().MilliValue()
	// pod status
	node.PodCapacity = capacity.Pods().Value()
}

func (m *SNodeManager) GetNodesByClusters(clusterIds []string) ([]SNode, error) {
	nodes := make([]SNode, 0)
	if err := GetResourcesByClusters(m, clusterIds, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (m *SNodeManager) getNodePods(node *SNode, pods []SPod) []SPod {
	ret := make([]SPod, 0)
	for _, p := range pods {
		if p.NodeId == node.GetId() {
			ret = append(ret, p)
		}
	}
	return ret
}

func (m *SNodeManager) Usage(clusters []sClusterUsage) (*api.NodeUsage, error) {
	clusterIds := make([]string, len(clusters))
	for i := range clusters {
		clusterIds[i] = clusters[i].Id
	}
	pods, err := GetPodManager().GetPodsByClusters(clusterIds)
	if err != nil {
		return nil, err
	}
	nodes, err := m.GetNodesByClusters(clusterIds)
	if err != nil {
		return nil, err
	}
	fillUsage := func(pods []SPod, nu *api.NodeUsage) *api.NodeUsage {
		for _, p := range pods {
			nu.Memory.Request += p.MemoryRequests
			nu.Memory.Limit += p.MemoryLimits
			nu.Cpu.Request += p.CpuRequests
			nu.Cpu.Limit += p.CpuLimits
		}
		return nu
	}
	eachUsages := make([]*api.NodeUsage, 0)
	for _, node := range nodes {
		nPods := m.getNodePods(&node, pods)
		nu := api.NewNodeUsage()
		nu.Count = 1
		if node.Status == api.NodeStatusReady {
			nu.ReadyCount = 1
		} else {
			nu.NotReadyCount = 1
		}
		nu.Memory = &api.MemoryUsage{
			Capacity: node.MemoryCapacity,
		}
		nu.Cpu = &api.CpuUsage{
			Capacity: node.CpuCapacity,
		}
		nu.Pod = &api.PodUsage{
			Count:    int64(len(nPods)),
			Capacity: node.PodCapacity,
		}
		eachUsages = append(eachUsages, fillUsage(nPods, nu))
	}

	totalUsage := api.NewNodeUsage()
	for _, each := range eachUsages {
		totalUsage.Add(each)
	}
	return totalUsage, nil
}

func (node *SNode) AllowPerformCordon(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, node, "cordon")
}

func (node *SNode) PerformCordon(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, node.SetNodeScheduleToggle(true)
}

func (node *SNode) AllowPerformUncordon(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, node, "uncordon")
}

func (node *SNode) PerformUncordon(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, node.SetNodeScheduleToggle(false)
}

func (node *SNode) GetRawNode() (*v1.Node, error) {
	obj, err := GetK8sObject(node)
	if err != nil {
		return nil, err
	}
	return obj.(*v1.Node), nil
}

func (obj *SNode) SetNodeScheduleToggle(unschedule bool) error {
	ccli, err := obj.GetClusterClient()
	if err != nil {
		return errors.Wrap(err, "get cluster client")
	}
	cli := ccli.GetHandler()
	node, err := obj.GetRawNode()
	if err != nil {
		return errors.Wrap(err, "get remote k8s node")
	}
	nodeObj := node.DeepCopy()
	nodeObj.Spec.Unschedulable = unschedule
	for i, taint := range nodeObj.Spec.Taints {
		if taint.Key == "node-role.kubernetes.io/master" {
			if !unschedule {
				taint.Effect = v1.TaintEffectPreferNoSchedule
			} else {
				taint.Effect = v1.TaintEffectNoSchedule
			}
			nodeObj.Spec.Taints[i] = taint
		}
	}
	if _, err := cli.UpdateV2(api.ResourceNameNode, nodeObj); err != nil {
		return err
	}
	return nil
}

func (obj *SNode) getNodeConditions(node v1.Node) []*api.Condition {
	var conditions []*api.Condition
	for _, condition := range node.Status.Conditions {
		conditions = append(conditions, &api.Condition{
			Type:               string(condition.Type),
			Status:             condition.Status,
			LastProbeTime:      condition.LastHeartbeatTime,
			LastTransitionTime: condition.LastTransitionTime,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	return SortConditions(conditions)
}
