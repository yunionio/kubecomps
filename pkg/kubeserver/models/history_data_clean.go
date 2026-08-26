package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

// IHistoryDataManager mirrors yunion.io/x/onecloud/pkg/cloudcommon/db.IHistoryDataManager.
// Defined locally because kubecomps release/3.11 vendor (20240305) does not include it yet.
type IHistoryDataManager interface {
	HistoryDataClean(ctx context.Context, timeBefor time.Time) (int, error)
}

func collectClusterK8sManagers(man ISyncableManager, out *[]IHistoryDataManager) {
	for _, sub := range man.GetSubManagers() {
		collectClusterK8sManagers(sub, out)
	}
	if hM, ok := man.(IHistoryDataManager); ok {
		*out = append(*out, hM)
	}
}

// HistoryDataClean hard-deletes soft-deleted rows older than timeBefor.
// Logic aligned with onecloud SStandaloneAnonResourceBaseManager.HistoryDataClean
// (release/3.11 pkg/cloudcommon/db/standalone_anon.go).
func (manager *SClusterResourceBaseManager) HistoryDataClean(ctx context.Context, timeBefor time.Time) (int, error) {
	q := manager.RawQuery("id").IsTrue("deleted").LE("deleted_at", timeBefor)
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return 0, nil
		}
		return 0, errors.Wrap(err, "Query")
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return 0, errors.Wrap(err, "rows.Scan")
		}
		ids = append(ids, id)
	}
	var purge = func(ids []string) error {
		vars := []interface{}{}
		placeholders := make([]string, len(ids))
		for i := range placeholders {
			placeholders[i] = "?"
			vars = append(vars, ids[i])
		}
		placeholder := strings.Join(placeholders, ",")
		sql := fmt.Sprintf(
			"delete from %s where id in (%s)",
			manager.TableSpec().Name(), placeholder,
		)
		lockman.LockRawObject(ctx, manager.Keyword(), "purge")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "purge")

		_, err = sqlchemy.GetDB().Exec(
			sql, vars...,
		)
		if err != nil {
			return errors.Wrapf(err, strings.ReplaceAll(sql, "?", "%s"), vars...)
		}
		return nil
	}

	var splitByLen = func(data []string, splitLen int) [][]string {
		var result [][]string
		for i := 0; i < len(data); i += splitLen {
			end := i + splitLen
			if end > len(data) {
				end = len(data)
			}
			result = append(result, data[i:end])
		}
		return result
	}
	idsArr := splitByLen(ids, 100)
	for i := range idsArr {
		err = purge(idsArr[i])
		if err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

// CleanK8sHistoryData hard-deletes soft-deleted rows of in-cluster K8s resources
// (ClusterManager sub-tree). Cluster/Machine/Fed* are not included.
func CleanK8sHistoryData(ctx context.Context, timeBefore time.Time) {
	managers := make([]IHistoryDataManager, 0)
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
