package models

import (
	"context"
	"time"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

const (
	K8S_SYNC_STATUS_QUEUED  = "queued"
	K8S_SYNC_STATUS_QUEUING = "queuing"
	K8S_SYNC_STATUS_SYNCING = "syncing"
	K8S_SYNC_STATUS_IDLE    = "idle"
	K8S_SYNC_STATUS_ERROR   = "error"
)

type SSyncableK8sBaseResourceManager struct{}

type SSyncableK8sBaseResource struct {
	SyncStatus    string    `width:"20" charset:"ascii" default:"idle" list:"domain"`
	SyncMessage   string    `charset:"utf8" list:"domain"`
	LastSync      time.Time `list:"domain"`
	LastSyncEndAt time.Time `list:"domain"`
}

func (self *SSyncableK8sBaseResource) CanSync() bool {
	if self.SyncStatus == K8S_SYNC_STATUS_QUEUED || self.SyncStatus == K8S_SYNC_STATUS_SYNCING {
		if self.LastSync.IsZero() || time.Now().Sub(self.LastSync) > 30*time.Minute {
			return true
		} else {
			return false
		}
	}
	return true
}

func (m *SSyncableK8sBaseResourceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.SyncableK8sBaseResourceListInput) (*sqlchemy.SQuery, error) {
	if len(query.SyncStatus) > 0 {
		q = q.In("sync_status", query.SyncStatus)
	}
	return q, nil
}

type IK8sSyncModel interface {
	db.IModel

	SetSyncStatus(status string)
	SetLastSync(time.Time)
	SetLastSyncEndAt(time.Time)
	SetSyncMessage(msg string)
}

func (self *SSyncableK8sBaseResource) SetSyncStatus(status string) {
	self.SyncStatus = status
}

func (self *SSyncableK8sBaseResource) SetLastSync(syncTime time.Time) {
	self.LastSync = syncTime
}

func (self *SSyncableK8sBaseResource) SetLastSyncEndAt(endAt time.Time) {
	self.LastSyncEndAt = endAt
}

func (self *SSyncableK8sBaseResource) SetSyncMessage(msg string) {
	self.SyncMessage = msg
}

func (self *SSyncableK8sBaseResource) SaveSyncMessage(obj IK8sSyncModel, msg string) {
	db.Update(obj, func() error {
		obj.SetSyncMessage(msg)
		return nil
	})
}

func (_ *SSyncableK8sBaseResource) MarkSyncing(obj IK8sSyncModel, userCred mcclient.TokenCredential) error {
	_, err := db.Update(obj, func() error {
		obj.SetSyncStatus(K8S_SYNC_STATUS_SYNCING)
		obj.SetLastSync(timeutils.UtcNow())
		obj.SetLastSyncEndAt(time.Time{})
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to MarkSyncing")
	}
	return nil
}

func (_ *SSyncableK8sBaseResource) MarkEndSync(ctx context.Context, userCred mcclient.TokenCredential, obj IK8sSyncModel) error {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	_, err := db.Update(obj, func() error {
		obj.SetSyncStatus(K8S_SYNC_STATUS_IDLE)
		obj.SetLastSyncEndAt(timeutils.UtcNow())
		obj.SetSyncMessage("")
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "MarkEndSync")
	}
	return nil
}

func (_ *SSyncableK8sBaseResource) MarkErrorSync(ctx context.Context, obj IK8sSyncModel, err error) error {
	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	_, updateErr := db.Update(obj, func() error {
		obj.SetSyncStatus(K8S_SYNC_STATUS_ERROR)
		obj.SetSyncMessage(err.Error())
		return nil
	})
	if err != nil {
		return errors.Wrap(updateErr, "MarkErrorSync")
	}
	return nil
}
