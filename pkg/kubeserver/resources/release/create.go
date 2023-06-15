package release

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	//"k8s.io/helm/pkg/helm"
	"helm.sh/helm/v3/pkg/release"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
	//helmtypes "yunion.io/x/kubecomps/pkg/kubeserver/types/helm"
)

func generateName(nameTemplate string) (string, error) {
	t, err := template.New("name-template").Funcs(sprig.TxtFuncMap()).Parse(nameTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	err = t.Execute(&b, nil)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func GenerateName(nameTemplate string) (string, error) {
	return generateName(nameTemplate)
}

/*type CreateUpdateReleaseRequest struct {
	ChartName   string   `json:"chart_name"`
	Namespace   string   `json:"namespace"`
	ReleaseName string   `json:"release_name"`
	Version     string   `json:"version"`
	ReUseValues bool     `json:"reuse_values"`
	ResetValues bool     `json:"reset_values"`
	DryRun      bool     `json:"dry_run"`
	Values      string   `json:"values"`
	Sets        []string `json:"sets"`
	Timeout     int64    `json:"timeout"`
}

func NewCreateUpdateReleaseReq(params jsonutils.JSONObject) (*CreateUpdateReleaseRequest, error) {
	var req CreateUpdateReleaseRequest
	err := params.Unmarshal(&req)
	if err != nil {
		return nil, err
	}
	if req.Timeout == 0 {
		req.Timeout = 1500 // set default 15 mins timeout
	}
	return &req, nil
}

func (c *CreateUpdateReleaseRequest) Vals() ([]byte, error) {
	return MergeBytesValues([]byte(c.Values), c.Sets)
}

type valueFiles []string

func (v valueFiles) String() string {
	return fmt.Sprintf("%s", v)
}

func (v valueFiles) Type() string {
	return "valueFiles"
}

func (v *valueFiles) Set(value string) error {
	for _, fp := range strings.Split(value, ",") {
		*v = append(*v, fp)
	}
	return nil
}

func mergeValues(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}
		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			dest[k] = v
			continue
		}
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = nextMap
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValues(destMap, nextMap)
	}
	return dest
}

func MergeValues(values, stringValues []string) ([]byte, error) {
	return MergeValuesF([]string{}, values, stringValues)
}

func MergeBytesValues(vbytes []byte, values []string) ([]byte, error) {
	base := map[string]interface{}{}
	currentMap := map[string]interface{}{}
	if err := yaml.Unmarshal(vbytes, &currentMap); err != nil {
		return []byte{}, fmt.Errorf("Failed to parse: %s, error: %v", string(vbytes), err)
	}
	base = mergeValues(base, currentMap)

	for _, value := range values {
		if err := strvals.ParseInto(value, base); err != nil {
			return []byte{}, fmt.Errorf("failed parsing set data: %s", err)
		}
	}
	return yaml.Marshal(base)
}

func MergeValuesF(valueFiles valueFiles, values, stringValues []string) ([]byte, error) {
	base := map[string]interface{}{}

	// parse values files
	for _, filePath := range valueFiles {
		currentMap := map[string]interface{}{}

		var bbytes []byte
		var err error
		bbytes, err = ioutil.ReadFile(filePath)
		if err != nil {
			return []byte{}, err
		}

		if err := yaml.Unmarshal(bbytes, &currentMap); err != nil {
			return []byte{}, fmt.Errorf("Failed to parse %s: %s", filePath, err)
		}
		// Merge with the previous map
		base = mergeValues(base, currentMap)
	}

	// parse set values
	for _, value := range values {
		if err := strvals.ParseInto(value, base); err != nil {
			return []byte{}, fmt.Errorf("failed parsing set data: %s", err)
		}
	}

	// parse set string values
	for _, value := range stringValues {
		if err := strvals.ParseIntoString(value, base); err != nil {
			return []byte{}, fmt.Errorf("failed parsing set string: %s", err)
		}
	}

	return yaml.Marshal(base)
}*/

func (man *SReleaseManager) ValidateCreateData(req *common.Request) error {
	input := &api.ReleaseCreateInput{}
	if err := req.DataUnmarshal(input); err != nil {
		return err
	}
	if input.Namespace == "" {
		input.Namespace = req.GetDefaultNamespace()
	}
	if input.ReleaseName == "" {
		name, err := generateName("")
		if err != nil {
			return err
		}
		input.ReleaseName = name
	}
	segs := strings.Split(input.ChartName, "/")
	if len(segs) != 2 {
		return httperrors.NewInputParameterError("Illegal chart name: %q", input.ChartName)
	}
	input.Repo = segs[0]
	input.ChartName = segs[1]
	req.Data.Update(jsonutils.Marshal(input))
	return nil
}

func (man *SReleaseManager) Create(req *common.Request) (interface{}, error) {
	input := &api.ReleaseCreateInput{}
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	cli, err := req.GetHelmClient(input.Namespace)
	if err != nil {
		return nil, err
	}
	return ReleaseCreate(cli.Release(), input)
}

/*func validateInfraCreate(cli *helm.Client, chartPkg *helmtypes.ChartPackage) error {
	releases, err := ListReleases(cli.Release(), api.ReleaseListQuery{All: true})
	if err != nil {
		return err
	}
	if releases == nil {
		return nil
	}
	for _, rls := range releases {
		if rls.ChartInfo.Metadata.Name == chartPkg.Metadata.Name {
			return httperrors.NewBadRequestError("Release %s already created by chart %s", rls.Name, rls.ChartInfo.Metadata.Name)
		}
	}
	return nil
}*/

func ReleaseCreate(cli helm.IRelease, opt *api.ReleaseCreateInput) (*release.Release, error) {
	log.Infof("Deploying chart=%q, release name=%q", opt.ChartName, opt.ReleaseName)

	return cli.Create(opt)
}
