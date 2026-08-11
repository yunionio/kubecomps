package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

func collectClusterK8sManagers(man ISyncableManager, out *[]db.IHistoryDataManager) {
	for _, sub := range man.GetSubManagers() {
		collectClusterK8sManagers(sub, out)
	}
	if hM, ok := man.(db.IHistoryDataManager); ok {
		*out = append(*out, hM)
	}
}

// CleanK8sHistoryData hard-deletes soft-deleted rows of in-cluster K8s resources
// (ClusterManager sub-tree). Cluster/Machine/Fed* are not included.
func CleanK8sHistoryData(ctx context.Context, timeBefore time.Time) {
	managers := make([]db.IHistoryDataManager, 0)
	for _, man := range GetClusterManager().GetSubManagers() {
		collectClusterK8sManagers(man, &managers)
	}
	for _, hM := range managers {
		keyword := ""
		if km, ok := hM.(db.IModelManager); ok {
			keyword = km.Keyword()
		}
		start := time.Now()
		cnt, err := hM.HistoryDataClean(ctx, timeBefore)
		if err != nil {
			log.Errorf("clean %s history data error: %v", keyword, err)
			continue
		}
		if cnt > 0 {
			log.Infof("clean %d %s history data cost %s", cnt, keyword, time.Since(start).Round(time.Second))
		}
	}
}

// AutoCleanK8sHistoryData is the cron entry that purges soft-deleted K8s resources
// older than options.K8sHistoryDataKeepDays.
func AutoCleanK8sHistoryData(ctx context.Context, userCred mcclient.TokenCredential, startRun bool) {
	keepDays := options.Options.K8sHistoryDataKeepDays
	if keepDays <= 0 {
		keepDays = 30
	}
	timeBefore := time.Now().AddDate(0, 0, -keepDays)
	log.Infof("AutoCleanK8sHistoryData start, keep_days=%d, before=%s", keepDays, timeBefore.Format(time.RFC3339))
	CleanK8sHistoryData(ctx, timeBefore)
}

// PerformHistoryDataClean is the clusters class action that cleans soft-deleted
// in-cluster K8s resources older than the given keep days.
func (m *SClusterManager) PerformHistoryDataClean(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	keepDays := options.Options.K8sHistoryDataKeepDays
	if keepDays <= 0 {
		keepDays = 30
	}
	if !gotypes.IsNil(data) && data.Contains("day") {
		day, _ := data.Int("day")
		keepDays = int(day)
	}
	timeBefore := time.Now().AddDate(0, 0, -keepDays)
	log.Infof("PerformHistoryDataClean keep_days=%d, before=%s", keepDays, timeBefore.Format(time.RFC3339))
	go CleanK8sHistoryData(ctx, timeBefore)
	return nil, nil
}
