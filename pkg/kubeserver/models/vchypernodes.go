package models

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	vcHyperNodeManager *SVCHyperNodeManager
	_                  IClusterModel = new(SVCHyperNode)
)

func init() {
	GetVCHyperNodeManager()
}

type SVCHyperNodeManager struct {
	SClusterResourceBaseManager
}

type SVCHyperNode struct {
	SClusterResourceBase
}

func GetVCHyperNodeManager() *SVCHyperNodeManager {
	if vcHyperNodeManager == nil {
		vcHyperNodeManager = NewK8sModelManager(func() ISyncableManager {
			return &SVCHyperNodeManager{
				SClusterResourceBaseManager: NewClusterResourceBaseManager(
					new(SVCHyperNode),
					"vchypernodes_tbl",
					"vchypernode",
					"vchypernodes",
					api.ResourceNameVolcanoHyperNode,
					"topology.volcano.sh",
					"v1alpha1",
					api.KindNameVolcanoHyperNode,
					&unstructured.Unstructured{},
				),
			}
		}).(*SVCHyperNodeManager)
	}
	return vcHyperNodeManager
}

func (m *SVCHyperNodeManager) ListRemoteObjects(cli *client.ClusterManager) ([]interface{}, error) {
	return listVolcanoRemoteObjects(cli, api.KindNameVolcanoHyperNode, m.SClusterResourceBaseManager.ListRemoteObjects)
}

func (m *SVCHyperNodeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.VCHyperNodeCreateInput) (*api.VCHyperNodeCreateInput, error) {
	cInput, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.ClusterResourceCreateInput = *cInput
	if input.Name == "" {
		return nil, httperrors.NewNotEmptyError("name is empty")
	}
	if err := validateHyperNodeSpec(input); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SVCHyperNodeManager) NewRemoteObjectForCreate(_ IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.VCHyperNodeCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal hypernode create input")
	}
	return newHyperNodeObject(input)
}

func (obj *SVCHyperNode) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detailBase := obj.SClusterResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.ClusterResourceDetail)
	u, err := asUnstructured(k8sObj)
	if err != nil {
		return api.VCHyperNodeDetail{ClusterResourceDetail: detailBase, Status: obj.Status}
	}
	return hyperNodeDetailFromUnstructured(detailBase, u, cli, isList)
}

func (obj *SVCHyperNode) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if u, ok := extObj.(*unstructured.Unstructured); ok {
		obj.Status = hyperNodeStatus(nestedInt64(u, "status", "nodeCount"))
		return nil
	}
	return obj.SClusterResourceBase.SetStatusByRemoteObject(ctx, userCred, extObj)
}

func (obj *SVCHyperNode) GetDetailsRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	raw, err := obj.SClusterResourceBase.GetDetailsRawdata(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return volcanoRawdata(raw, api.VolcanoHyperNodeKind), nil
}

func (obj *SVCHyperNode) UpdateRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	updated, err := replaceVolcanoObject(obj, api.ResourceNameVolcanoHyperNode, "", data, api.VolcanoHyperNodeKind, api.VolcanoHyperNodeAPIVersion)
	if err != nil {
		return nil, err
	}
	return volcanoRawdata(K8SObjectToJSONObject(updated), api.VolcanoHyperNodeKind), nil
}

func validateHyperNodeSpec(input *api.VCHyperNodeCreateInput) error {
	if input.Tier < 1 {
		return httperrors.NewInputParameterError("tier must be >= 1")
	}
	if input.TierName == "" {
		return httperrors.NewNotEmptyError("tierName is empty")
	}
	if len(input.Members) == 0 {
		return httperrors.NewInputParameterError("members is empty")
	}
	for i := range input.Members {
		member := input.Members[i]
		switch member.Type {
		case "Node":
			if member.LabelKey == "" || member.LabelValue == "" {
				return httperrors.NewInputParameterError("node member requires labelKey and labelValue")
			}
		case "HyperNode":
			if member.ExactMatch == "" {
				return httperrors.NewInputParameterError("hypernode member requires exactMatch")
			}
		default:
			return httperrors.NewInputParameterError("member type must be Node or HyperNode")
		}
	}
	return nil
}

func newHyperNodeObject(input *api.VCHyperNodeCreateInput) (*unstructured.Unstructured, error) {
	members := make([]interface{}, 0, len(input.Members))
	for i := range input.Members {
		member := input.Members[i]
		selector := map[string]interface{}{}
		if member.Type == "Node" {
			selector["labelMatch"] = map[string]interface{}{
				"matchLabels": map[string]interface{}{
					member.LabelKey: member.LabelValue,
				},
			}
		} else {
			selector["exactMatch"] = map[string]interface{}{
				"name": member.ExactMatch,
			}
		}
		members = append(members, map[string]interface{}{
			"type":     member.Type,
			"selector": selector,
		})
	}
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(api.VolcanoHyperNodeAPIVersion)
	u.SetKind(api.VolcanoHyperNodeKind)
	u.SetName(input.Name)
	if err := unstructured.SetNestedField(u.Object, int64(input.Tier), "spec", "tier"); err != nil {
		return nil, err
	}
	if err := unstructured.SetNestedField(u.Object, input.TierName, "spec", "tierName"); err != nil {
		return nil, err
	}
	if err := unstructured.SetNestedSlice(u.Object, members, "spec", "members"); err != nil {
		return nil, err
	}
	return u, nil
}

func hyperNodeStatus(nodeCount int64) string {
	if nodeCount > 0 {
		return "Ready"
	}
	return "Pending"
}

func hyperNodeDetailFromUnstructured(base api.ClusterResourceDetail, u *unstructured.Unstructured, cli *client.ClusterManager, isList bool) api.VCHyperNodeDetail {
	nodeCount := nestedInt64(u, "status", "nodeCount")
	detail := api.VCHyperNodeDetail{
		ClusterResourceDetail: base,
		Tier:                  nestedInt64(u, "spec", "tier"),
		TierName:              nestedString(u, "spec", "tierName"),
		NodeCount:             nodeCount,
		Status:                hyperNodeStatus(nodeCount),
	}
	if isList || u == nil {
		return detail
	}
	selectors, hyperNodes := hyperNodeMemberRefs(u)
	detail.HyperNodes = hyperNodes
	detail.Nodes = matchNodesByLabelSelectors(cli, selectors)
	return detail
}

func hyperNodeMemberRefs(u *unstructured.Unstructured) ([]map[string]string, []string) {
	raw, _, _ := unstructured.NestedSlice(u.Object, "spec", "members")
	var selectors []map[string]string
	var hyperNodes []string
	seenHyperNode := map[string]struct{}{}
	for _, item := range raw {
		member, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		memberType, _, _ := unstructured.NestedString(member, "type")
		switch memberType {
		case "Node":
			matchLabels, found, _ := unstructured.NestedStringMap(member, "selector", "labelMatch", "matchLabels")
			if found && len(matchLabels) > 0 {
				selectors = append(selectors, matchLabels)
			}
		case "HyperNode":
			name, found, _ := unstructured.NestedString(member, "selector", "exactMatch", "name")
			if !found || name == "" {
				continue
			}
			if _, ok := seenHyperNode[name]; ok {
				continue
			}
			seenHyperNode[name] = struct{}{}
			hyperNodes = append(hyperNodes, name)
		}
	}
	return selectors, hyperNodes
}

func matchNodesByLabelSelectors(cli *client.ClusterManager, selectors []map[string]string) []string {
	if cli == nil || len(selectors) == 0 {
		return nil
	}
	indexer := cli.GetIndexer()
	if indexer == nil {
		return nil
	}
	nodes, err := indexer.NodeLister().List(labels.Everything())
	if err != nil {
		log.Errorf("list nodes for hypernode members: %v", err)
		return nil
	}
	var names []string
	seen := map[string]struct{}{}
	for _, node := range nodes {
		if node == nil {
			continue
		}
		nodeLabels := labels.Set(node.Labels)
		for _, selector := range selectors {
			if !labels.SelectorFromSet(selector).Matches(nodeLabels) {
				continue
			}
			if _, ok := seen[node.Name]; ok {
				break
			}
			seen[node.Name] = struct{}{}
			names = append(names, node.Name)
			break
		}
	}
	return names
}
