package release

import (
	"helm.sh/helm/v3/pkg/release"

	"yunion.io/x/log"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/dataselect"
)

var emptyList = &ReleaseList{
	BaseList: common.NewBaseList(nil),
	Releases: make([]*api.Release, 0),
}

type ReleaseList struct {
	*common.BaseList
	Releases []*api.Release
}

func ToRelease(release *release.Release, cluster api.ICluster) *api.Release {
	return &api.Release{
		Release:     release,
		ClusterMeta: api.NewClusterMeta(cluster),
		Status:      release.Info.Status.String(),
	}
}

func (man *SReleaseManager) List(req *common.Request) (common.ListResource, error) {
	q := api.ReleaseListQuery{}
	err := req.Query.Unmarshal(&q)
	if err != nil {
		return nil, err
	}
	q.Namespace = req.GetNamespaceQuery().ToRequestParam()
	cli, err := req.GetHelmClient(q.Namespace)
	if err != nil {
		return nil, err
	}
	return man.GetReleaseList(cli, req.GetCluster(), q, req.ToQuery())
}

func (man *SReleaseManager) GetReleaseList(helmclient *helm.Client, cluster api.ICluster, q api.ReleaseListQuery, dsQuery *dataselect.DataSelectQuery) (*ReleaseList, error) {
	list, err := ListReleases(helmclient.Release(), q)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return emptyList, nil
	}
	releaseList, err := ToReleaseList(list, cluster, dsQuery)
	return releaseList, err
}

func (l *ReleaseList) Append(obj interface{}) {
	l.Releases = append(l.Releases, ToRelease(obj.(*release.Release), l.GetCluster()))
}

func (l *ReleaseList) GetResponseData() interface{} {
	return l.Releases
}

func ToReleaseList(releases []*release.Release, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*ReleaseList, error) {
	list := &ReleaseList{
		BaseList: common.NewBaseList(cluster),
		Releases: make([]*api.Release, 0),
	}
	err := dataselect.ToResourceList(
		list,
		releases,
		dataselect.NewHelmReleaseDataCell,
		dsQuery,
	)
	return list, err
}

func ListReleases(helmclient helm.IRelease, q api.ReleaseListQuery) ([]*release.Release, error) {
	list := helmclient.List()
	list.All = q.All
	list.AllNamespaces = q.AllNamespace

	/*	stats := q.statusCodes()
		ops := []helm.ReleaseListOption{
			helm.ReleaseListSort(int32(rls.ListSort_LAST_RELEASED)),
			helm.ReleaseListOrder(int32(rls.ListSort_DESC)),
			helm.ReleaseListStatuses(stats),
		}
	*/if len(q.Filter) != 0 {
		log.Debugf("Apply filters: %v", q.Filter)
		list.Filter = q.Filter
	}
	// TODO: support namespace filter
	if len(q.Namespace) != 0 {
	}

	resp, err := list.Run()
	if err != nil {
		log.Errorf("Can't retrieve the list of releases: %v", err)
		return nil, err
	}
	return filterReleaseList(resp, q.Namespace), err
}

func filterReleaseList(rels []*release.Release, namespace string) []*release.Release {
	if len(namespace) == 0 {
		return rels
	}
	ret := make([]*release.Release, 0)
	for _, rls := range rels {
		if rls.Namespace == namespace {
			ret = append(ret, rls)
		}
	}
	return ret
}
