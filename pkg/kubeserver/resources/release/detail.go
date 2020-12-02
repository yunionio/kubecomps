package release

import (
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/chart"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
)

func (man *SReleaseManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetReleaseDetailFromRequest(req, id)
}

func GetReleaseDetailFromRequest(req *common.Request, id string) (*api.ReleaseDetail, error) {
	namespace := req.GetDefaultNamespace()
	cli, err := req.GetHelmClient(namespace)
	if err != nil {
		return nil, err
	}

	detail, err := GetReleaseDetail(cli, req.GetCluster(), req.ClusterManager, req.GetIndexer(), namespace, id)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func GetReleaseDetail(
	helmclient *helm.Client,
	cluster api.ICluster,
	clusterMan model.ICluster,
	indexer *client.CacheFactory,
	namespace, releaseName string,
) (*api.ReleaseDetail, error) {
	log.Infof("Get helm release: %q", releaseName)

	rls, err := helmclient.Release().ReleaseContent(releaseName, -1)
	if err != nil {
		return nil, err
	}

	res, err := GetReleaseResources(helmclient, rls, indexer, cluster, clusterMan)
	if err != nil {
		return nil, errors.Wrapf(err, "Get release resources: %v", releaseName)
	}

	return &api.ReleaseDetail{
		Release:   *ToRelease(rls, cluster),
		Resources: res,
		Files:     chart.GetChartRawFiles(rls.Chart),
	}, nil
}
