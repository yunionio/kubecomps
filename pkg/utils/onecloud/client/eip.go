package client

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type EIPHelper struct {
	*ResourceHelper
}

func NewEIPHelper(s *mcclient.ClientSession) *EIPHelper {
	return &EIPHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Elasticips),
	}
}
