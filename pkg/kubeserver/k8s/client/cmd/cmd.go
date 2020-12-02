package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	tcmd "k8s.io/client-go/tools/clientcmd"

	"yunion.io/x/log"
)

const (
	retryIntervalKubectlApply = 10 * time.Second
	timeoutKubectlApply       = 15 * time.Minute
)

type Client struct {
	kubeconfigFile  string
	configOverrides tcmd.ConfigOverrides
	closeFn         func() error
}

func NewClientFromKubeconfig(kubeconfig string) (*Client, error) {
	f, err := createTempFile(kubeconfig)
	if err != nil {
		return nil, err
	}
	defer ifErrRemove(err, f)
	c := &Client{
		kubeconfigFile:  f,
		configOverrides: tcmd.ConfigOverrides{},
	}
	c.closeFn = c.removeKubeconfigFile
	return c, nil
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

func (c *Client) removeKubeconfigFile() error {
	return os.Remove(c.kubeconfigFile)
}

func (c *Client) kubectlManifestCmd(commandName, manifest string, args ...string) error {
	cmd := exec.Command("kubectl", c.buildKubectlArgs(commandName)...)
	cmd.Stdin = strings.NewReader(manifest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("couldn't kubectl apply, output: %s, error: %v", string(output), err)
	}
	return nil
}

func (c *Client) buildKubectlArgs(commandName string, nargs ...string) []string {
	args := []string{commandName}
	args = append(args, nargs...)
	if c.kubeconfigFile != "" {
		args = append(args, "--kubeconfig", c.kubeconfigFile)
	}
	if c.configOverrides.Context.Cluster != "" {
		args = append(args, "--cluster", c.configOverrides.Context.Cluster)
	}
	if c.configOverrides.Context.Namespace != "" {
		args = append(args, "--namespace", c.configOverrides.Context.Namespace)
	}
	if c.configOverrides.Context.AuthInfo != "" {
		args = append(args, "--user", c.configOverrides.Context.AuthInfo)
	}
	return append(args, "-f", "-")
}

func (c *Client) Apply(manifest string) error {
	return c.waitForKubectlApply(manifest)
}

func (c *Client) Delete(manifest string) error {
	return c.waitForKubectlDelete(manifest)
}

func (c *Client) kubectlDelete(manifest string) error {
	return c.kubectlManifestCmd("delete", manifest, "--ignore-not-found=true")
}

func (c *Client) kubectlApply(manifest string) error {
	return c.kubectlManifestCmd("apply", manifest)
}

func (c *Client) waitForKubectlApply(manifest string) error {
	err := wait.PollImmediate(retryIntervalKubectlApply, timeoutKubectlApply, func() (bool, error) {
		log.Infof("Waiting for kubectl apply...")
		err := c.kubectlApply(manifest)
		if err != nil {
			if strings.Contains(err.Error(), "refused") {
				// Connection was refused, probably because the API server is not ready yet.
				log.Infof("Waiting for kubectl apply... server not yet available: %v", err)
				return false, nil
			}
			if strings.Contains(err.Error(), "unable to recognize") {
				log.Infof("Waiting for kubectl apply... api not yet available: %v", err)
				return false, nil
			}
			log.Warningf("Waiting for kubectl apply... unknown error %v", err)
			return false, err
		}

		return true, nil
	})
	return err
}

func (c *Client) waitForKubectlDelete(manifest string) error {
	err := wait.PollImmediate(retryIntervalKubectlApply, timeoutKubectlApply, func() (bool, error) {
		log.Infof("Waiting for kubectl delete...")
		err := c.kubectlDelete(manifest)
		if err != nil {
			if strings.Contains(err.Error(), "refused") {
				// Connection was refused, probably because the API server is not ready yet.
				log.Infof("Waiting for kubectl delete... server not yet available: %v", err)
				return false, nil
			}
			if strings.Contains(err.Error(), "unable to recognize") {
				log.Infof("Waiting for kubectl delete... api not yet available: %v", err)
				return false, nil
			}
			log.Warningf("Waiting for kubectl delete... unknown error %v", err)
			return false, err
		}

		return true, nil
	})
	return err
}
