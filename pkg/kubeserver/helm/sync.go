package helm

import (
	"helm.sh/helm/v3/pkg/repo"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

var RepoBackendManager *RepoCacheBackend

type RepoCacheBackend struct {
	client *RepoClient
}

func SetupRepoBackendManager(repos []*repo.Entry) error {
	var err error
	RepoBackendManager, err = newRepoCacheBackend()
	if err != nil {
		return err
	}
	return RepoBackendManager.Update(repos...)
}

func newRepoCacheBackend() (*RepoCacheBackend, error) {
	client, err := NewRepoClient(options.Options.HelmDataDir)
	if err != nil {
		return nil, err
	}
	return &RepoCacheBackend{
		client: client,
	}, nil
}

func (r *RepoCacheBackend) Update(repos ...*repo.Entry) error {
	for _, repo := range repos {
		if oldRepo, _ := r.client.GetEntry(repo.Name); oldRepo != nil {
			continue
		}
		if err := r.client.Add(repo); err != nil {
			log.Errorf("Add repo %s error: %v", repo.Name, err)
		}
	}
	return nil
}
