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
	cli, err := cmd.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		return err
	}
	manifest, err := cluster.GetDriver().GetAddonsManifest(cluster, conf)
	if err != nil {
		return err
	}
	if len(manifest) == 0 {
		return nil
	}
	return cli.Apply(manifest)
}

func (t *ClusterApplyAddonsTask) OnError(ctx context.Context, obj *models.SCluster, err jsonutils.JSONObject) {
	t.SetStageFailed(ctx, err)
	logclient.LogWithStartable(t, obj, logclient.ActionClusterApplyAddons, err, t.UserCred, false)
}
