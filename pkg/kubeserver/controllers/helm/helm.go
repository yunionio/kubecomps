package helm

import (
	"context"

	"helm.sh/helm/v3/pkg/repo"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
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

func Start(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	log.Infof("start sync helm repos")
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
	doRepoRefresh(man)
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
	for _, r := range repos {
		if err := r.DoSync(); err != nil {
			log.Errorf("Sync repo %s error: %v", r.GetName(), err)
		} else {
			log.Infof("Repo %s sync finished", r.GetName())
		}
	}
	if err := man.Update(rs...); err != nil {
		log.Errorf("Update all repos error: %v", err)
	}
	log.Infof("Finish refresh all repos")
}
