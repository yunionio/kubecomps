package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/logclient"
)

func init() {
	taskman.RegisterTask(ClusterSyncTask{})
}

type ClusterSyncTask struct {
	taskman.STask
}

func (t *ClusterSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	t.SetStage("OnSyncComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		input := new(api.ClusterSyncInput)
		params := t.GetParams()
		if err := params.Unmarshal(input); err != nil {
			return nil, errors.Wrap(err, "unmarshal sync input")
		}
		if input.Force {
			if err := client.GetClustersManager().UpdateClient(cluster, true); err != nil {
				return nil, errors.Wrap(err, "update cluster to client manager")
			}
			if err := cluster.SetStatus(ctx, t.GetUserCred(), api.ClusterStatusRunning, "by syncing"); err != nil {
				return nil, errors.Wrap(err, "change cluster status to running")
			}
		} else {
			if err := client.GetClustersManager().AddClient(cluster); err != nil {
				if errors.Cause(err) != client.ErrClusterAlreadyAdded {
					return nil, errors.Wrap(err, "add cluster to client manager")
				}
			}
		}
		// do sync
		if err := cluster.SyncK8sMachinesConfig(ctx, t.UserCred); err != nil {
			log.Warningf("cluster %s sync machines config: %s", cluster.GetName(), err)
		}
		if err := cluster.SyncCallSyncTask(ctx, t.UserCred); err != nil {
			return nil, errors.Wrap(err, "SyncCallSyncTask")
		}
		return nil, nil
	})
}

func (t *ClusterSyncTask) OnSyncComplete(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterSync, nil, t.UserCred, true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterSyncTask) OnSyncCompleteFailed(ctx context.Context, cluster *models.SCluster, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterSync, reason, t.UserCred, false)
}
