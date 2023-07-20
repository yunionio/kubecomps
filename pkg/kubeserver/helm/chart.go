package helm

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"helm.sh/helm/v3/cmd/helm/search"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

// SearchMaxScore suggests that any score higher than this is not considered a match
const SearchMaxScore = 25

type ChartClient struct {
	dataDir string
	*RepoConfig
}

func NewChartClient(dataDir string) *ChartClient {
	return &ChartClient{
		dataDir:    dataDir,
		RepoConfig: NewRepoConfig(dataDir),
	}
}

func (c ChartClient) setupSearchedVersion(devel bool) string {
	if devel {
		return ">0.0.0-0"
	}
	return ">0.0.0"
}

func (c ChartClient) SearchRepo(query api.ChartListInput, version string) ([]*api.ChartResult, error) {
	index, err := c.buildIndex(query.Repo)
	if err != nil {
		return nil, err
	}

	allVersion := query.AllVersion
	name := query.Name

	if version == "" {
		version = c.setupSearchedVersion(false)
	}

	var res []*search.Result
	if len(name) == 0 {
		res = index.All()
	} else {
		res, err = index.Search(name, SearchMaxScore, true)
		if err != nil {
			return nil, err
		}
	}

	search.SortScore(res)
	res = c.applyFilter(query, res)
	data, err := c.applyConstraint(version, allVersion, res)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c ChartClient) getSearchResultRepo(r *search.Result) string {
	parts := strings.Split(r.Name, "/")
	return parts[0]
}

func (c ChartClient) applyFilter(q api.ChartListInput, res []*search.Result) []*search.Result {
	filterF := func(r *search.Result) bool {
		if q.Repo != "" {
			if c.getSearchResultRepo(r) != q.Repo {
				return false
			}
		}
		if q.Name != "" {
			if r.Chart.Name != q.Name {
				return false
			}
		} else {
			if q.Keyword != "" {
				if !strings.Contains(r.Chart.Name, q.Keyword) && !utils.IsInStringArray(q.Keyword, r.Chart.Keywords) {
					return false
				}
			}
		}
		return true
	}
	ret := make([]*search.Result, 0)
	for _, r := range res {
		if filterF(r) {
			ret = append(ret, r)
		}
	}
	return ret
}

func (c ChartClient) applyConstraint(version string, allVersion bool, results []*search.Result) ([]*api.ChartResult, error) {
	res := make([]*api.ChartResult, 0)
	for _, chart := range results {
		res = append(res, &api.ChartResult{
			ChartVersion: chart.Chart,
			Metadata:     chart.Chart.Metadata,
			Repo:         c.getSearchResultRepo(chart),
		})
	}

	if len(version) == 0 {
		return res, nil
	}

	constraint, err := semver.NewConstraint(version)
	if err != nil {
		return res, errors.Wrapf(err, "an invalid version/constraint format %s", version)
	}

	data := res[:0]
	foundNames := map[string]bool{}
	for _, r := range res {
		if _, found := foundNames[r.Name]; found {
			continue
		}
		v, err := semver.NewVersion(r.ChartVersion.Version)
		if err != nil || constraint.Check(v) {
			data = append(data, r)
			if !allVersion {
				foundNames[r.Name] = true
			}
		}
	}

	return data, nil
}

func (c ChartClient) buildIndex(filterRepo string) (*search.Index, error) {
	// Load the repositories.yaml
	setting := c.RepoConfig.GetSetting()
	repoFile := setting.RepositoryConfig
	rf, err := repo.LoadFile(repoFile)
	if isNotExist(err) || len(rf.Repositories) == 0 {
		return nil, errors.Error("no repositories configured")
	}
	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
		f := filepath.Join(setting.RepositoryCache, helmpath.CacheIndexFile(n))
		if filterRepo != "" && n != filterRepo {
			continue
		}
		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			log.Errorf("Repo %q is corrupt or missing, Try `helm repo update`: %v", n, err)
			continue
		}

		i.AddRepo(n, ind, true)
	}
	return i, nil
}

func (c ChartClient) Show(repoName, chartName, version string) (*chart.Chart, error) {
	repoCli, err := NewRepoClient(c.dataDir)
	if err != nil {
		return nil, errors.Wrapf(err, "NewRepoClient from data dir %q", c.dataDir)
	}
	repo, err := repoCli.GetEntry(repoName)
	if err != nil {
		return nil, errors.Wrapf(err, "Get repo Entry %q", repoName)
	}
	return c.LocateChart(repoName, chartName, version, repo)
}

func (c ChartClient) getChartPath(chartName, version string, r *repo.Entry) (string, *repo.ChartVersion, error) {
	setting := c.GetSetting()
	idxFilePath := filepath.Join(setting.RepositoryCache, helmpath.CacheIndexFile(r.Name))
	idxFile, err := repo.LoadIndexFile(idxFilePath)
	if err != nil {
		return "", nil, errors.Wrapf(err, "LoadIndexFile: %q", idxFilePath)
	}
	cv, err := idxFile.Get(chartName, version)
	if err != nil {
		return "", nil, errors.Wrapf(err, "Get %s:%s from idxFilePath %q", chartName, version, idxFilePath)
	}
	chartPath := filepath.Join(filepath.Dir(idxFilePath), fmt.Sprintf("%s-%s.tgz", chartName, cv.Version))
	return chartPath, cv, nil
}

func (c ChartClient) LocateChartPath(repoName, chartName, version string, repo *repo.Entry) (string, error) {
	chPath, chartVersion, err := c.getChartPath(chartName, version, repo)
	if err != nil {
		return "", errors.Wrap(err, "getChartPath")
	}

	if digestNum, err := provenance.DigestFile(chPath); err == nil && digestNum == chartVersion.Digest {
		log.Infof("chart %s already exists, use it directly", chPath)
		return chPath, nil
	} else {
		log.Infof("download chart %s:%s of repo %v", chartName, version, repo)
	}

	pathOpt := c.NewChartPathOptions(version, false, repo)

	chartName = strings.Join([]string{repoName, chartName}, "/")

	cp, err := pathOpt.LocateChart(chartName, c.GetSetting())
	if err != nil {
		return "", errors.Wrapf(err, "LocateChart %q", chartName)
	}
	return cp, nil
}

func (c ChartClient) LocateChart(repoName, chartName, version string, repo *repo.Entry) (*chart.Chart, error) {
	chPath, err := c.LocateChartPath(repoName, chartName, version, repo)
	if err != nil {
		return nil, errors.Wrap(err, "LocateChartPath")
	}
	return loader.Load(chPath)
}

func (c ChartClient) NewChartPathOptions(
	version string,
	verify bool,
	repo *repo.Entry,
) *action.ChartPathOptions {
	return &action.ChartPathOptions{
		CaFile:   repo.CAFile,
		CertFile: repo.CertFile,
		KeyFile:  repo.KeyFile,
		Password: repo.Password,
		//RepoURL:  repo.URL,
		Username: repo.Username,
		Version:  version,
		Verify:   verify,
	}
}

var readmeFileNames = []string{"readme.md", "readme.txt", "readme"}

func FindChartReadme(ch *chart.Chart) *chart.File {
	for _, f := range ch.Files {
		for _, n := range readmeFileNames {
			if strings.EqualFold(f.Name, n) {
				return f
			}
		}
	}
	return nil
}
