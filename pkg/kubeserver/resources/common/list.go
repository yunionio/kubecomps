package common

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/dataselect"
)

type BaseList struct {
	*dataselect.ListMeta
	Cluster api.ICluster
}

func NewBaseList(cluster api.ICluster) *BaseList {
	return &BaseList{
		ListMeta: dataselect.NewListMeta(),
		Cluster:  cluster,
	}
}

func (l *BaseList) GetCluster() api.ICluster {
	return l.Cluster
}
