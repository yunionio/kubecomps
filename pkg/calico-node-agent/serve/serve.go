package serve

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	capi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	cclient "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	coptions "github.com/projectcalico/libcalico-go/lib/options"
	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/calico-node-agent/client"
	"yunion.io/x/kubecomps/pkg/calico-node-agent/config"
	"yunion.io/x/kubecomps/pkg/calico-node-agent/types"
)

func Run(configFile string) {
	// Go's RNG is not seeded by default.  Do that now.
	rand.Seed(time.Now().UTC().UnixNano())

	nodeCfg, err := config.GetNodeConfigByFile(configFile)
	if err != nil {
		log.Fatalf("Get node config error: %v", err)
	}

	calicoCli, err := client.NewClient(client.DefaultConfigPath)
	if err != nil {
		log.Fatalf("New calico client error: %v", err)
	}

	srv := newServer(nodeCfg, calicoCli)
	if err := srv.start(context.Background()); err != nil {
		log.Fatalf("Start serve error: %v", err)
	}
}

type server struct {
	nodeConfig *types.NodeConfig
	calicoCli  cclient.Interface
}

func newServer(nodeCfg *types.NodeConfig, calicoCli cclient.Interface) *server {
	s := &server{
		nodeConfig: nodeCfg,
		calicoCli:  calicoCli,
	}
	return s
}

func (s *server) start(ctx context.Context) error {
	for {
		if err := s.syncIPPools(ctx); err != nil {
			log.Errorf("syncIPPools error: %v", err)
		}
		time.Sleep(30 * time.Minute)
	}
}

func getIPPoolName(nodeName string, pool *types.NodeIPPool) string {
	name := fmt.Sprintf("%s-%s", nodeName, pool.CIDR)
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "/", "-")
	return name
}

func getIPPoolNodeSelector(nodeName string) string {
	return fmt.Sprintf(`%s == "%s"`, kapiv1.LabelHostname, nodeName)
}

func getIPPoolLabels() map[string]string {
	return map[string]string{
		types.LabelManaged: types.LabelManagedValueAgent,
	}
}

func (s *server) getIPPoolMeta(pool *types.NodeIPPool) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{
		Name:   getIPPoolName(s.nodeConfig.NodeName, pool),
		Labels: getIPPoolLabels(),
	}
	return meta
}

func (s *server) transToIPPool(pool *types.NodeIPPool) (*capi.IPPool, error) {
	_, ipNet, err := pool.GetIPAndNet()
	if err != nil {
		return nil, errors.Wrap(err, "get pool ip and net")
	}
	maskLen, _ := ipNet.Mask.Size()
	ipPool := &capi.IPPool{
		ObjectMeta: s.getIPPoolMeta(pool),
		Spec: capi.IPPoolSpec{
			CIDR:         pool.CIDR,
			NodeSelector: getIPPoolNodeSelector(s.nodeConfig.NodeName),
			// TODO: find out how to set blockSize reasonable
			BlockSize: maskLen,
		},
	}
	return ipPool, nil
}

func (s *server) createIPPool(ctx context.Context, pool *types.NodeIPPool) (*capi.IPPool, error) {
	ipPool, err := s.transToIPPool(pool)
	if err != nil {
		return nil, errors.Wrap(err, "transToIPPool")
	}
	return s.calicoCli.IPPools().Create(ctx, ipPool, coptions.SetOptions{})
}

func (s *server) deleteIPPool(ctx context.Context, pool *capi.IPPool) error {
	_, err := s.calicoCli.IPPools().Delete(ctx, pool.Name, options.DeleteOptions{})
	return err
}

func (s *server) getIPPoolNodeSelector() string {
	return getIPPoolNodeSelector(s.nodeConfig.NodeName)
}

func (s *server) listIPPools(ctx context.Context) ([]capi.IPPool, error) {
	pools, err := s.calicoCli.IPPools().List(ctx, coptions.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "get ippools from calico datastore")
	}
	selfPools := make([]capi.IPPool, 0)
	nodeSelector := s.getIPPoolNodeSelector()
	for _, pool := range pools.Items {
		if pool.Spec.NodeSelector != nodeSelector {
			continue
		}
		managedVal, ok := pool.Labels[types.LabelManaged]
		if !ok {
			continue
		}
		if managedVal != types.LabelManagedValueAgent {
			continue
		}
		selfPools = append(selfPools, pool)
	}
	return selfPools, nil
}

func (s *server) isContainIPPool(remotePool *capi.IPPool) bool {
	for _, lp := range s.nodeConfig.IPPools {
		if lp.CIDR == remotePool.Spec.CIDR {
			return true
		}
	}
	return false
}

func (s *server) isRemoteExistLocalPool(remotePools []*capi.IPPool, localPool *types.NodeIPPool) bool {
	for _, rp := range remotePools {
		if rp.Spec.CIDR == localPool.CIDR {
			return true
		}
	}
	return false
}

func (s *server) syncIPPools(ctx context.Context) error {
	remotePools, err := s.listIPPools(ctx)
	if err != nil {
		return errors.Wrap(err, "list calico remote IPPools")
	}

	tmpPools := make([]*capi.IPPool, 0)
	// delete remote IPPool not in local configured
	for idx := range remotePools {
		rp := &remotePools[idx]
		if s.isContainIPPool(rp) {
			tmpPools = append(tmpPools, rp)
			continue
		}
		// delete remote pool
		if err := s.deleteIPPool(ctx, rp); err != nil {
			return errors.Wrapf(err, "delete remote IPPool %s")
		}
	}

	// create local pool to remote
	for idx := range s.nodeConfig.IPPools {
		lp := &s.nodeConfig.IPPools[idx]
		if s.isRemoteExistLocalPool(tmpPools, lp) {
			continue
		}
		rp, err := s.createIPPool(ctx, lp)
		if err != nil {
			return errors.Wrapf(err, "create local idx:%d pool to remote", idx)
		}
		tmpPools = append(tmpPools, rp)
	}

	// TODO: record tmpPools and start watch logical
	return nil
}
