package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type ISyncableManager interface {
	IClusterModelManager
	// GetSubManagers return sub resource manager
	GetSubManagers() []ISyncableManager
	// PurgeAllByCluster invoke when cluster deleted
	PurgeAllByCluster(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error
}

type SSyncableManager struct {
	subManagers []ISyncableManager
}

func newSyncableManager() *SSyncableManager {
	return &SSyncableManager{
		subManagers: make([]ISyncableManager, 0),
	}
}

func (m *SSyncableManager) AddSubManager(mans ...ISyncableManager) *SSyncableManager {
	m.subManagers = append(m.subManagers, mans...)
	return m
}

func (m *SSyncableManager) GetSubManagers() []ISyncableManager {
	return m.subManagers
}
