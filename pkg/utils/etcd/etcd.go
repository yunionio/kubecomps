package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/pkg/transport"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	// EtcdListenClientPort defines the port etcd listen on for client traffic
	EtcdListenClientPort = 2379

	// EtcdListenPeerPort defines the port etcd listen on for peer traffic
	EtcdListenPeerPort = 2380
)

// Client provides connection parameters for an etcd cluster
type Client struct {
	Endpoints []string
	TLS       *tls.Config
	caFile    string
	certFile  string
	keyFile   string
}

// New creates a new EtcdCluster client
func New(endpoints []string, ca, cert, key string) (*Client, error) {
	client := Client{Endpoints: endpoints}

	var err error
	caFile, err := createTempFile(ca)
	if err != nil {
		return nil, err
	}
	defer ifErrRemove(err, caFile)
	certFile, err := createTempFile(cert)
	if err != nil {
		return nil, err
	}
	defer ifErrRemove(err, certFile)
	keyFile, err := createTempFile(key)
	if err != nil {
		return nil, err
	}
	defer ifErrRemove(err, keyFile)

	if ca != "" || cert != "" || key != "" {
		tlsInfo := transport.TLSInfo{
			CertFile:      certFile,
			KeyFile:       keyFile,
			TrustedCAFile: caFile,
		}
		tlsConfig, err := tlsInfo.ClientConfig()
		if err != nil {
			return nil, err
		}
		client.TLS = tlsConfig
		client.caFile = caFile
		client.certFile = certFile
		client.keyFile = keyFile
	}

	return &client, nil
}

// Sync synchronizes client's endpoints with the known endpoints from the etcd membership.
func (c *Client) Sync() error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: 20 * time.Second,
		TLS:         c.TLS,
	})
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = cli.Sync(ctx)
	cancel()
	if err != nil {
		return err
	}
	log.V(1).Infof("etcd endpoints read from etcd: %s", strings.Join(cli.Endpoints(), ","))

	c.Endpoints = cli.Endpoints()
	return nil
}

// Member struct defines an etcd member; it is used for avoiding to spread go.etcd.io/etcd dependency
// across kubeadm codebase
type Member struct {
	Name    string
	PeerURL string
}

// GetMemberID returns the member ID of the given peer URL
func (c Client) GetMemberID(peerURL string) (uint64, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: 30 * time.Second,
		TLS:         c.TLS,
	})
	if err != nil {
		return 0, err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.MemberList(ctx)
	cancel()
	if err != nil {
		return 0, err
	}

	for _, member := range resp.Members {
		if member.GetPeerURLs()[0] == peerURL {
			return member.GetID(), nil
		}
	}
	return 0, nil
}

// RemoveMember notifies an etcd cluster to remove an existing member
func (c Client) RemoveMember(id uint64) ([]Member, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: 30 * time.Second,
		TLS:         c.TLS,
	})
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	// Remove an existing member from the cluster
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.MemberRemove(ctx, id)
	cancel()
	if err != nil {
		return nil, err
	}

	// Returns the updated list of etcd members
	ret := []Member{}
	for _, m := range resp.Members {
		ret = append(ret, Member{Name: m.Name, PeerURL: m.PeerURLs[0]})
	}

	return ret, nil
}

// AddMember notifies an existing etcd cluster that a new member is joining
func (c *Client) AddMember(name string, peerAddrs string) ([]Member, error) {
	// Parse the peer address, required to add the client URL later to the list
	// of endpoints for this client. Parsing as a first operation to make sure that
	// if this fails no member addition is performed on the etcd cluster.
	parsedPeerAddrs, err := url.Parse(peerAddrs)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing peer address %s", peerAddrs)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: 20 * time.Second,
		TLS:         c.TLS,
	})
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	// Adds a new member to the cluster
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.MemberAdd(ctx, []string{peerAddrs})
	cancel()
	if err != nil {
		return nil, err
	}

	// Returns the updated list of etcd members
	ret := []Member{}
	for _, m := range resp.Members {
		// fixes the entry for the joining member (that doesn't have a name set in the initialCluster returned by etcd)
		if m.Name == "" {
			ret = append(ret, Member{Name: name, PeerURL: m.PeerURLs[0]})
		} else {
			ret = append(ret, Member{Name: m.Name, PeerURL: m.PeerURLs[0]})
		}
	}

	// Add the new member client address to the list of endpoints
	c.Endpoints = append(c.Endpoints, GetClientURLByIP(parsedPeerAddrs.Hostname()))

	return ret, nil
}

// GetVersion returns the etcd version of the cluster.
// An error is returned if the version of all endpoints do not match
func (c Client) GetVersion() (string, error) {
	var clusterVersion string

	versions, err := c.GetClusterVersions()
	if err != nil {
		return "", err
	}
	for _, v := range versions {
		if clusterVersion != "" && clusterVersion != v {
			return "", errors.Errorf("etcd cluster contains endpoints with mismatched versions: %v", versions)
		}
		clusterVersion = v
	}
	if clusterVersion == "" {
		return "", errors.Error("could not determine cluster etcd version")
	}
	return clusterVersion, nil
}

// GetClusterVersions returns a map of the endpoints and their associated versions
func (c Client) GetClusterVersions() (map[string]string, error) {
	versions := make(map[string]string)
	statuses, err := c.GetClusterStatus()
	if err != nil {
		return versions, err
	}

	for ep, status := range statuses {
		versions[ep] = status.Version
	}
	return versions, nil
}

// ClusterAvailable returns true if the cluster status indicates the cluster is available.
func (c Client) ClusterAvailable() (bool, error) {
	_, err := c.GetClusterStatus()
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetClusterStatus returns nil for status Up or error for status Down
func (c Client) GetClusterStatus() (map[string]*clientv3.StatusResponse, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         c.TLS,
	})
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	clusterStatus := make(map[string]*clientv3.StatusResponse)
	for _, ep := range c.Endpoints {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := cli.Status(ctx, ep)
		cancel()
		if err != nil {
			return nil, err
		}
		clusterStatus[ep] = resp
	}
	return clusterStatus, nil
}

// WaitForClusterAvailable returns true if all endpoints in the cluster are available after retry attempts, an error is returned otherwise
func (c Client) WaitForClusterAvailable(retries int, retryInterval time.Duration) (bool, error) {
	for i := 0; i < retries; i++ {
		if i > 0 {
			fmt.Printf("[util/etcd] Waiting %v until next retry\n", retryInterval)
			time.Sleep(retryInterval)
		}
		log.Infof("attempting to see if all cluster endpoints (%s) are available %d/%d", c.Endpoints, i+1, retries)
		resp, err := c.ClusterAvailable()
		if err != nil {
			switch err {
			case context.DeadlineExceeded:
				fmt.Println("[util/etcd] Attempt timed out")
			default:
				fmt.Printf("[util/etcd] Attempt failed with error: %v\n", err)
			}
			continue
		}
		return resp, nil
	}
	return false, errors.Error("timeout waiting for etcd cluster to be available")
}

func (c Client) Cleanup() {
	os.Remove(c.caFile)
	os.Remove(c.keyFile)
	os.Remove(c.certFile)
}

// GetPeerURL creates an HTTPS URL that uses the configured advertise
// address and peer port for the API controller
func GetPeerURL(ip string) string {
	return "https://" + net.JoinHostPort(ip, strconv.Itoa(EtcdListenPeerPort))
}

// GetClientURLByIP creates an HTTPS URL based on an IP address
// and the client listening port.
func GetClientURLByIP(ip string) string {
	return "https://" + net.JoinHostPort(ip, strconv.Itoa(EtcdListenClientPort))
}

func createTempFile(contents string) (string, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	defer ifErrRemove(err, f.Name())
	if err = f.Close(); err != nil {
		return "", err
	}
	err = ioutil.WriteFile(f.Name(), []byte(contents), 0644)
	if err != nil {
		return "", err
	}
	return f.Name(), nil
}

func ifErrRemove(err error, path string) {
	if err != nil {
		if err := os.Remove(path); err != nil {
			log.Warningf("Error removing file '%s': %v", path, err)
		}
	}
}
