package release

import (
	"helm.sh/helm/v3/pkg/action"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
)

func (man *SReleaseManager) AllowPerformAction(req *common.Request, id string) bool {
	return man.AllowUpdateItem(req, id)
}

func (man *SReleaseManager) AllowPerformRollback(req *common.Request, id string) bool {
	return man.AllowPerformAction(req, id)
}

func (man *SReleaseManager) PerformRollback(req *common.Request, id string) (interface{}, error) {
	cli, err := req.GetHelmClient(req.GetDefaultNamespace())
	if err != nil {
		return nil, err
	}
	input := new(api.ReleaseRollbackInput)
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	if err := man.DoReleaseRollback(cli.Release().Rollback(), id, input); err != nil {
		return nil, err
	}
	return GetReleaseDetailFromRequest(req, id)
}

func (man *SReleaseManager) DoReleaseRollback(
	cli *action.Rollback,
	name string,
	input *api.ReleaseRollbackInput,
) error {
	cli.Version = input.Revision
	cli.Recreate = input.Recreate
	cli.Force = input.Force
	if err := cli.Run(name); err != nil {
		return err
	}
	return nil
}
