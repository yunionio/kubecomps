package client

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type EIPHelper struct {
	*ResourceHelper
}

func NewEIPHelper(s *mcclient.ClientSession) *EIPHelper {
	return &EIPHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Elasticips),
	}
}
