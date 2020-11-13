package client

import (
	"os"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	DefaultConfigPath = "/etc/calico/calicoctl.cfg"
)

// NewClient creates a new CalicoClient using connection information in the specified
// filename (if it exists), dropping back to environment variables for any
// parameter not loaded from file.
func NewClient(cf string) (client.Interface, error) {
	cfg, err := LoadClientConfig(cf)
	if err != nil {
		return nil, err
	}
	log.Infof("Loaded client config: %#v", cfg.Spec)

	c, err := client.New(*cfg)
	if err != nil {
		return nil, err
	}

	return c, err
}

// LoadClientConfig loads the client config from file if the file exists,
// otherwise will load from environment variables.
func LoadClientConfig(cf string) (*apiconfig.CalicoAPIConfig, error) {
	if _, err := os.Stat(cf); err != nil {
		if cf != DefaultConfigPath {
			log.Errorf("Error reading config file: %s\n", cf)
			return nil, errors.Wrapf(err, "reading config file %s", cf)
		}
		log.Warningf("Config file: %s cannot be read, reading config from environment", cf)
		cf = ""
	}

	return apiconfig.LoadClientConfig(cf)
}
