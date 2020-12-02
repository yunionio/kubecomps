package ssh

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/util/wait"
)

// RemoteSSHBashScript executes command on remote machine
func RemoteSSHBashScript(host string, port int, username string, passwd string, privateKey, content string) (string, error) {
	cli, err := ssh.NewClient(host, port, username, passwd, privateKey)
	if err != nil {
		return "", err
	}
	content = base64.StdEncoding.EncodeToString([]byte(content))
	tmpFile := fmt.Sprintf("/tmp/script-%s", seclib.RandomPassword(8))
	writeScript := fmt.Sprintf("echo '%s' | base64 -d > %s", content, tmpFile)
	execScript := fmt.Sprintf("sudo bash %s", tmpFile)
	// rmScript := fmt.Sprintf("rm %s", tmpFile)
	ret, err := cli.RawRun(writeScript, execScript) //, rmScript)
	if err != nil {
		return "", err
	}
	return strings.Join(ret, "\n"), nil
}

func RemoteSSHCommand(host string, port int, username string, passwd string, privateKey, cmd string) (string, error) {
	cli, err := ssh.NewClient(host, port, username, passwd, privateKey)
	if err != nil {
		return "", err
	}
	ret, err := cli.RawRun(cmd)
	if err != nil {
		return "", err
	}
	return strings.Join(ret, "\n"), nil
}

func CheckRemotePortOpen(host string, port int) error {
	log.Infof("CheckRemotePortOpen for remote: %s:%d", host, port)
	err := procutils.NewCommand("nc", "-z", host, fmt.Sprintf("%d", port)).Run()
	return err
}

func WaitRemotePortOpen(host string, port int, interval, timeout time.Duration) error {
	return wait.Poll(interval, timeout, func() (bool, error) {
		if err := CheckRemotePortOpen(host, port); err != nil {
			return false, nil
		}
		return true, nil
	})
}
