package kubespray

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/utils/ansibler"
	"yunion.io/x/kubecomps/pkg/utils/ansibler/execute"
	"yunion.io/x/kubecomps/pkg/utils/ansibler/stdoutcallback"
	"yunion.io/x/kubecomps/pkg/utils/ansibler/stdoutcallback/results"
)

const (
	DefaultAnsiblePath = "/opt/yunion/ansible"
)

var (
	DefaultPlaybookConfigPath = "ansible.cfg"
)

func newDefaultAnsiblePath(ksv, fp string) string {
	return filepath.Join(DefaultAnsiblePath, ksv, fp)
}

type AnsibleRunner struct {
	playbookPath    string
	action          string
	inventory       *KubesprayInventory
	inventoryFile   string
	extraVarsFile   string
	limitHosts      []string
	playbook        *ansibler.AnsiblePlaybookCmd
	resultCollector *bytes.Buffer
}

func NewAnsibleRunner(playbookPath string, kubeVersion string, hosts ...*KubesprayInventoryHost) (*AnsibleRunner, error) {
	if _, err := os.Stat(playbookPath); err != nil {
		return nil, errors.Wrapf(err, "check playbook path %q", playbookPath)
	}

	runner := &AnsibleRunner{
		playbookPath:    playbookPath,
		inventory:       NewKubesprayInventory(kubeVersion, hosts...),
		resultCollector: new(bytes.Buffer),
	}

	return runner, nil
}

func (r *AnsibleRunner) SetAction(action string) *AnsibleRunner {
	r.action = action
	return r
}

func (r *AnsibleRunner) initInventoryFile() error {
	if r.inventoryFile != "" {
		return errors.Errorf("alreay init inventory file %q", r.inventoryFile)
	}

	tf, err := ioutil.TempFile(os.TempDir(), r.action)
	if err != nil {
		return errors.Wrap(err, "create temporary file")
	}
	defer tf.Close()

	content, err := r.inventory.ToString()
	if err != nil {
		return errors.Wrap(err, "construct inventory content")
	}

	if _, err := tf.WriteString(content); err != nil {
		return errors.Wrap(err, "write content to inventory file")
	}
	r.inventoryFile = tf.Name()

	return nil
}

func getVarsJSONString(vars interface{}) string {
	obj := jsonutils.Marshal(vars)
	return obj.PrettyString()
}

func (r *AnsibleRunner) SetExtraVars(vars interface{}) error {
	tf, err := ioutil.TempFile(os.TempDir(), "*.json")
	if err != nil {
		return errors.Error("create temporary file for extra vars")
	}
	defer tf.Close()

	if _, err := tf.WriteString(getVarsJSONString(vars)); err != nil {
		return errors.Wrapf(err, "write vars %v to file %s", vars, tf.Name())
	}
	r.extraVarsFile = tf.Name()
	return nil
}

func (r *AnsibleRunner) AddLimitHosts(checkHost bool, hosts ...string) error {
	if checkHost {
		for _, host := range hosts {
			if !r.inventory.IsIncludeHost(host) {
				return errors.Errorf("Inventory not include host %s", host)
			}
		}
	}
	r.limitHosts = append(r.limitHosts, hosts...)
	return nil
}

func (r *AnsibleRunner) Clear() error {
	var errs []error

	for _, host := range r.inventory.Hosts {
		if err := host.Clear(); err != nil {
			errs = append(errs, errors.Wrapf(err, "clear host %s", host.Hostname))
		}
	}

	if err := os.Remove(r.inventoryFile); err != nil {
		if os.IsNotExist(err) {
			errs = append(errs, errors.Wrapf(err, "clear inventory file %q", r.inventoryFile))
		}
	}

	return errors.NewAggregate(errs)
}

func (r *AnsibleRunner) Execute(command string, args []string, prefix string) error {
	log.Infof("Execute [%s]: %s %v", prefix, command, args)
	defaultExec := &execute.DefaultExecute{
		Write:       r.playbook.Writer,
		ResultsFunc: r.ResultsFunc(),
	}
	return defaultExec.Execute(command, args, prefix)
}

func (r *AnsibleRunner) ResultsFunc() stdoutcallback.StdoutCallbackResultsFunc {
	return func(prefix string, r io.Reader, w io.Writer) error {
		if r == nil {
			return errors.Errorf("AnsibleRunner for %s results: reader is nil", prefix)
		}

		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			txt := scanner.Text()
			log.Infof("%s %s %s", prefix, results.PrefixTokenSeparator, txt)
			fmt.Fprintf(w, "%s %s %s\n", prefix, results.PrefixTokenSeparator, txt)
		}

		return nil
	}
}

func (r *AnsibleRunner) Run(debug bool) error {
	/*
	 * connOpts := &ansibler.AnsiblePlaybookConnectionOptions{
	 *     User:       user,
	 *     PrivateKey: privateKey,
	 * }
	 */
	if err := r.initInventoryFile(); err != nil {
		return errors.Wrap(err, "init inventory file")
	}

	playbookOpts := &ansibler.AnsiblePlaybookOptions{
		Inventory:     r.inventoryFile,
		ExtraVarsFile: r.extraVarsFile,
		Debug:         debug,
	}

	if len(r.limitHosts) != 0 {
		limitOpt := strings.Join(r.limitHosts, ",")
		playbookOpts.Limit = limitOpt
	}

	escOpt := &ansibler.AnsiblePlaybookPrivilegeEscalationOptions{
		Become:     true,
		BecomeUser: "root",
	}

	playbook := &ansibler.AnsiblePlaybookCmd{
		Playbook: r.playbookPath,
		// ConnectionOptions: connOpts,
		Options:                    playbookOpts,
		PrivilegeEscalationOptions: escOpt,
		ExecPrefix:                 fmt.Sprintf("Kubeserver ansible for %s", r.action),
		Exec:                       r,
		Writer:                     r.resultCollector,
	}

	r.playbook = playbook

	if err := playbook.Run(); err != nil {
		return err
	}
	return nil
}

func (r *AnsibleRunner) GetOutput() string {
	return r.resultCollector.String()
}
