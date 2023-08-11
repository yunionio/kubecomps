package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/client/cmd"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/logclient"
)

func init() {
	taskman.RegisterTask(ClusterApplyAddonsTask{})
}

type ClusterApplyAddonsTask struct {
	taskman.STask
}

func (t *ClusterApplyAddonsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	t.SetStage("OnApplyAddons", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		conf, err := cluster.GetAddonsConfig()
		if err != nil {
			return nil, errors.Wrap(err, "get addons manifest config")
		}
		return nil, ApplyAddons(cluster, conf)
	})
}

func (t *ClusterApplyAddonsTask) OnApplyAddons(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterApplyAddons, nil, t.UserCred, true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterApplyAddonsTask) OnApplyAddonsFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.OnError(ctx, cluster, data)
}

func ApplyAddons(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) error {
	kubeconfig, err := cluster.GetKubeconfig()
	if err != nil {
		return err
	}

	drv := cluster.GetDriver()

	// apply yaml manifests
	cli, err := cmd.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		return err
	}
	manifest, err := drv.GetAddonsManifest(cluster, conf)
	if err != nil {
		return err
	}
	if len(manifest) != 0 {
		if err := cli.Apply(manifest); err != nil {
			return errors.Wrap(err, "apply manifest")
		}
	}
	// apply helm charts
	charts, err := drv.GetAddonsHelmCharts(cluster, conf)
	if err != nil {
		return errors.Wrap(err, "GetAddonsHelmCharts")
	}
	for _, chart := range charts {
		if chart == nil {
			continue
		}
		if err := chart.Validate(); err != nil {
			return errors.Wrap(err, "chart is invalid")
		}
		man := models.NewHelmComponentManager(chart.Namespace, chart.ReleaseName, chart.EmbedChartName)
		helmCli, err := man.NewHelmClient(cluster, chart.Namespace)
		if err != nil {
			return errors.Wrap(err, "NewHelmClient")
		}
		rls, _ := helmCli.Release().Get().Run(chart.ReleaseName)
		if rls != nil {
			if err := man.UpdateHelmResource(cluster, chart.Values); err != nil {
				return errors.Wrapf(err, "Update helm addon %s", chart.EmbedChartName)
			}
		} else {
			if err := man.CreateHelmResource(cluster, chart.Values); err != nil {
				return errors.Wrapf(err, "Install helm addon %s", chart.EmbedChartName)
			}
		}
	}
	return nil
}

func (t *ClusterApplyAddonsTask) OnError(ctx context.Context, obj *models.SCluster, err jsonutils.JSONObject) {
	t.SetStageFailed(ctx, err)
	logclient.LogWithStartable(t, obj, logclient.ActionClusterApplyAddons, err, t.UserCred, false)
}
