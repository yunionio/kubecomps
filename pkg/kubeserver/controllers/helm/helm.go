package helm

import (
	"time"

	"helm.sh/helm/v3/pkg/repo"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

/*var OfficialRepos = []helm.Repo{
	{
		Name:   "stable",
		Url:    "https://kubernetes-charts.storage.googleapis.com",
		Source: "https://github.com/kubernetes/charts/tree/master/stable",
	},
	{
		Name:   "incubator",
		Url:    "https://kubernetes-charts-incubator.storage.googleapis.com",
		Source: "https://github.com/kubernetes/charts/tree/master/incubator",
	},
}*/

func Start() {
	var err error
	repos, err := models.RepoManager.ListRepos()
	if err != nil {
		log.Fatalf("List all repos error: %v", err)
	}
	entries := make([]*repo.Entry, 0)
	for _, obj := range repos {
		entries = append(entries, obj.ToEntry())
	}
	err = helm.SetupRepoBackendManager(entries)
	if err != nil {
		log.Fatalf("Setup RepoController error: %v", err)
	}
	startRepoRefresh(helm.RepoBackendManager)
}

func startRepoRefresh(man *helm.RepoCacheBackend) {
	tick := time.Tick(time.Duration(options.Options.RepoRefreshDuration) * time.Minute)
	for {
		select {
		case <-tick:
			doRepoRefresh(man)
		}
	}
}

func doRepoRefresh(man *helm.RepoCacheBackend) {
	repos, err := models.RepoManager.ListRepos()
	if err != nil {
		log.Errorf("List all repos error: %v", err)
		return
	}
	rs := []*repo.Entry{}
	for _, r := range repos {
		rs = append(rs, r.ToEntry())
	}
	if err := man.Update(rs...); err != nil {
		log.Errorf("Update all repos error: %v", err)
	}
	log.Debugf("Finish refresh all repos")
}
