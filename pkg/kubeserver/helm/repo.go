package helm

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	ErrRepoAlreadyExists = errors.Error("Repository already exists")
	ErrNoRepositories    = errors.Error("no repositories found. You must add one before updating")
)

type RepoClient struct {
	dataDir string
	*RepoConfig
}

type RepoConfig struct {
	RepositoryConfig string
	RepositoryCache  string
	RegistryConfig   string
	PluginDirectory  string
}

func NewRepoConfig(dataDir string) *RepoConfig {
	// TODO: fix helm path, now is init by InitEnv
	return &RepoConfig{
		//RepositoryConfig: path.Join(dataDir, "repositories.yaml"),
		//RepositoryCache:  path.Join(dataDir, "repository"),
		RepositoryConfig: helmpath.ConfigPath("repositories.yaml"),
		RepositoryCache:  helmpath.CachePath("repository"),
		RegistryConfig:   helmpath.ConfigPath("registry.json"),
		PluginDirectory:  helmpath.DataPath("plugins"),
	}
}

func (c *RepoConfig) GetSetting() *cli.EnvSettings {
	s := cli.New()
	s.RepositoryConfig = c.RepositoryConfig
	s.RepositoryCache = c.RepositoryCache
	s.PluginsDirectory = c.PluginDirectory
	s.RegistryConfig = c.RegistryConfig
	return s
}

func NewRepoClient(dataDir string) (*RepoClient, error) {
	if len(dataDir) == 0 {
		return nil, errors.Error("Helm state store path must specified")
	}
	if _, err := os.Stat(dataDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return nil, errors.Wrapf(err, "Make direcotry %q", dataDir)
			}
		} else {
			return nil, errors.Wrapf(err, "Get helm state directory %s", dataDir)
		}
	}
	repoConfig := NewRepoConfig(dataDir)
	return &RepoClient{
		dataDir:    dataDir,
		RepoConfig: repoConfig,
	}, nil
}

func (c RepoClient) Add(entry *repo.Entry) error {
	// Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(c.RepositoryConfig), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return errors.Wrapf(err, "mkdir %s", filepath.Dir(c.RepositoryConfig))
	}
	repoFile := c.RepositoryConfig

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer fileLock.Unlock()
	}
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	if f.Has(entry.Name) {
		return errors.Wrapf(ErrRepoAlreadyExists, "repository name (%s) already exists, please specify a different name", entry.Name)
	}

	r, err := repo.NewChartRepository(entry, getter.All(c.GetSetting()))
	if err != nil {
		return err
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be readched", entry.URL)
	}

	f.Update(entry)

	if err := f.WriteFile(c.RepositoryConfig, 0644); err != nil {
		return err
	}
	log.Infof("%q has been added to your repositories", entry.Name)
	return nil
}

func isNotExist(err error) bool {
	return os.IsNotExist(errors.Cause(err))
}

func (c RepoClient) Update(name string) error {
	repoFile := c.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if isNotExist(err) || len(f.Repositories) == 0 {
		return ErrNoRepositories
	}
	var repos []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(c.GetSetting()))
		if err != nil {
			return err
		}
		if r.Config.Name == name {
			repos = append(repos, r)
		}
	}

	return c.update(repos)
}

func (c RepoClient) update(repos []*repo.ChartRepository) error {
	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if ret, err := re.DownloadIndexFile(); err != nil {
				log.Errorf("...Unable to get an update from %q chart repository (%s): %s", re.Config.Name, re.Config.URL, err)
			} else {
				log.Infof("...Successfully got an update from the %q chart repository: %q", re.Config.Name, ret)
			}
		}(re)
		wg.Wait()
	}
	return nil
}

func (c RepoClient) GetEntry(name string) (*repo.Entry, error) {
	repoFile := c.RepositoryConfig
	r, err := repo.LoadFile(repoFile)
	if err != nil {
		return nil, err
	}
	for _, entry := range r.Repositories {
		if entry.Name == name {
			return entry, nil
		}
	}
	return nil, errors.Errorf("Not found repo %s", name)
}

func (c RepoClient) Remove(name string) error {
	repoFile := c.RepositoryConfig
	r, err := repo.LoadFile(repoFile)
	if isNotExist(err) || len(r.Repositories) == 0 {
		//return errors.New("no repositories configured")
		return nil
	}

	if !r.Remove(name) {
		return errors.Errorf("no repo named %q found", name)
	}
	if err := r.WriteFile(repoFile, 0644); err != nil {
		return err
	}
	if err := removeRepoCache(c.RepositoryCache, name); err != nil {
		return err
	}
	return nil
}

func removeRepoCache(root, name string) error {
	idx := filepath.Join(root, helmpath.CacheIndexFile(name))
	if _, err := os.Stat(idx); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "can't remove index file %s", idx)
	}
	return os.Remove(idx)
}
