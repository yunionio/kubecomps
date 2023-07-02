package models

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	limitRangeManager *SLimitRangeManager
	_                 IClusterModel = new(SLimitRange)
)

func init() {
	GetLimitRangeManager()
}

func GetLimitRangeManager() *SLimitRangeManager {
	if limitRangeManager == nil {
		limitRangeManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SLimitRangeManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SLimitRange{},
					"limitranges_tbl",
					"limitrange",
					"limitranges",
					api.ResourceNameLimitRange,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameLimitRange,
					new(v1.LimitRange),
				),
			}
		}).(*SLimitRangeManager)
	}
	return limitRangeManager
}

// +onecloud:swagger-gen-model-singular=limitrange
// +onecloud:swagger-gen-model-plural=limitranges
type SLimitRangeManager struct {
	SNamespaceResourceBaseManager
}

type SLimitRange struct {
	SNamespaceResourceBase
}

func (m *SLimitRangeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support replicasets create")
}

type limitRangesMap map[v1.LimitType]rangeMap

type rangeMap map[v1.ResourceName]*api.LimitRangeItem

func (rMap rangeMap) getRange(resource v1.ResourceName) *api.LimitRangeItem {
	r, ok := rMap[resource]
	if !ok {
		rMap[resource] = &api.LimitRangeItem{}
		return rMap[resource]
	}
	return r
}

func (obj *SLimitRange) toLimitRangesMap(lr *v1.LimitRange) limitRangesMap {
	rawLimitRanges := lr.Spec.Limits

	limitRanges := make(limitRangesMap, len(rawLimitRanges))

	for _, rawLimitRange := range rawLimitRanges {

		rangeMap := make(rangeMap)

		for resource, min := range rawLimitRange.Min {
			rangeMap.getRange(resource).Min = min.String()
		}

		for resource, max := range rawLimitRange.Max {
			rangeMap.getRange(resource).Max = max.String()
		}

		for resource, df := range rawLimitRange.Default {
			rangeMap.getRange(resource).Default = df.String()
		}

		for resource, dfR := range rawLimitRange.DefaultRequest {
			rangeMap.getRange(resource).DefaultRequest = dfR.String()
		}

		for resource, mLR := range rawLimitRange.MaxLimitRequestRatio {
			rangeMap.getRange(resource).MaxLimitRequestRatio = mLR.String()
		}

		limitRanges[rawLimitRange.Type] = rangeMap
	}

	return limitRanges
}

func (obj *SLimitRange) ToRangeItem(lr *v1.LimitRange) []*api.LimitRangeItem {
	limitRangeMap := obj.toLimitRangesMap(lr)
	limitRangeList := make([]*api.LimitRangeItem, 0)
	for limitType, rangeMap := range limitRangeMap {
		for resourceName, limit := range rangeMap {
			limit.ResourceName = resourceName.String()
			limit.ResourceType = string(limitType)
			limitRangeList = append(limitRangeList, limit)
		}
	}
	return limitRangeList
}

func (obj *SLimitRange) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	lr := k8sObj.(*v1.LimitRange)
	detail := api.LimitRangeDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Limits:                  obj.ToRangeItem(lr),
	}
	return detail
}
