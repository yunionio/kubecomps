package releaseapp

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/release"
)

type CreateReleaseAppRequest struct {
	*release.CreateUpdateReleaseRequest
}

func NewCreateReleaseAppRequest(data jsonutils.JSONObject) (*CreateReleaseAppRequest, error) {
	createOpt, err := release.NewCreateUpdateReleaseReq(data)
	if err != nil {
		return nil, err
	}
	return &CreateReleaseAppRequest{
		CreateUpdateReleaseRequest: createOpt,
	}, nil
}

func (r *CreateReleaseAppRequest) ToData() *release.CreateUpdateReleaseRequest {
	return r.CreateUpdateReleaseRequest
}

func (r *CreateReleaseAppRequest) IsSetsEmpty() bool {
	return len(r.Sets) == 0
}

func (app *SReleaseAppManager) ValidateCreateData(req *common.Request) error {
	data := req.Data
	ns, _ := data.GetString("namespace")
	if ns == "" {
		data.Set("namespace", jsonutils.NewString(req.GetDefaultNamespace()))
	}
	name, _ := data.GetString("release_name")
	if name == "" {
		name = app.hooker.GetReleaseName()
		data.Set("release_name", jsonutils.NewString(name))
	}
	chartName, _ := data.GetString("chart_name")
	if chartName == "" {
		chartName = app.hooker.GetChartName()
		data.Set("chart_name", jsonutils.NewString(chartName))
	}
	return nil
}

func (man *SReleaseAppManager) Create(req *common.Request) (interface{}, error) {
	createOpt, err := NewCreateReleaseAppRequest(req.Data)
	if err != nil {
		return nil, err
	}
	if createOpt.IsSetsEmpty() {
		createOpt.Sets = man.hooker.GetConfigSets().ToSets()
	}
	cli, err := req.GetHelmClient()
	if err != nil {
		return nil, err
	}
	return release.ReleaseCreate(cli.Release(), createOpt.ToData())
}
