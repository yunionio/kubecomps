package kubespray

import (
	"os"
	"strings"

	"yunion.io/x/pkg/errors"
)

type KubesprayRunner interface {
	AddLimitHosts(checkHost bool, hosts ...string) error
	Run(debug bool) error
	// GetOutput get ansible playbook output
	GetOutput() string
}

type DefaultKubesprayExecutor interface {
	Cluster(vars *KubesprayRunVars, hosts ...*KubesprayInventoryHost) KubesprayRunner
	Scale(vars *KubesprayRunVars, allHosts []*KubesprayInventoryHost, addedHosts ...*KubesprayInventoryHost) KubesprayRunner
	RemoveNode(vars *KubesprayRunVars, allHosts []*KubesprayInventoryHost, removeHosts ...*KubesprayInventoryHost) KubesprayRunner
}

type defaultKubesprayExecutor struct {
	err    error
	runner KubesprayRunner
}

func NewDefaultKubesprayExecutor() DefaultKubesprayExecutor {
	return new(defaultKubesprayExecutor)
}

func (f *defaultKubesprayExecutor) setRunner(
	sf func(*KubesprayRunVars, ...*KubesprayInventoryHost) (KubesprayRunner, error),
	vars *KubesprayRunVars,
	hosts ...*KubesprayInventoryHost,
) *defaultKubesprayExecutor {
	r, err := sf(vars, hosts...)
	f.err = err
	f.runner = r
	return f
}

func (f *defaultKubesprayExecutor) Cluster(vars *KubesprayRunVars, hosts ...*KubesprayInventoryHost) KubesprayRunner {
	return f.setRunner(NewDefaultKubesprayClusterRunner, vars, hosts...)
}

func (f *defaultKubesprayExecutor) Scale(vars *KubesprayRunVars, allHosts []*KubesprayInventoryHost, addedHosts ...*KubesprayInventoryHost) KubesprayRunner {
	return f.setRunner(
		func(vars *KubesprayRunVars, allHosts ...*KubesprayInventoryHost) (KubesprayRunner, error) {
			return NewDefaultKubesprayScaleRunner(vars, allHosts, addedHosts...)
		},
		vars,
		allHosts...)
}

func (f *defaultKubesprayExecutor) RemoveNode(
	vars *KubesprayRunVars,
	allHosts []*KubesprayInventoryHost,
	removeHosts ...*KubesprayInventoryHost,
) KubesprayRunner {
	return f.setRunner(
		func(vars *KubesprayRunVars, allHosts ...*KubesprayInventoryHost) (KubesprayRunner, error) {
			return NewDefaultKubesprayRemoveNodeRunner(vars, allHosts, removeHosts...)
		},
		vars,
		allHosts...,
	)
}

func (f *defaultKubesprayExecutor) Run(debug bool) error {
	if f.err != nil {
		return f.err
	}
	return f.runner.Run(debug)
}

func (f *defaultKubesprayExecutor) AddLimitHosts(checkHost bool, hosts ...string) error {
	if f.err != nil {
		return f.err
	}
	return f.runner.AddLimitHosts(checkHost, hosts...)
}

func (f *defaultKubesprayExecutor) GetOutput() string {
	return f.runner.GetOutput()
}

type kubesprayRunner struct {
	*AnsibleRunner
	hosts []*KubesprayInventoryHost

	playbookDir string
	// primary master node ansible vars
}

type KubesprayRunVars struct {
	KubesprayVars
}

func NewDefaultKubesprayClusterRunner(vars *KubesprayRunVars, hosts ...*KubesprayInventoryHost) (KubesprayRunner, error) {
	return newDefaultKubesprayRunner(DefaultKubesprayClusterYML, vars, hosts...)
}

func NewDefaultKubesprayUpgradeRunner(vars *KubesprayRunVars, hosts ...*KubesprayInventoryHost) (KubesprayRunner, error) {
	return newDefaultKubesprayRunner(DefaultKubesprayUpgradeClusterYML, vars, hosts...)
}

func checkTargetHosts(targetHosts []*KubesprayInventoryHost) (bool, error) {
	if len(targetHosts) == 0 {
		return false, errors.Error("empty added host")
	}

	isMasterAdded := false
	prevRole := targetHosts[0].Roles

	for _, ah := range targetHosts {
		curRole := ah.Roles
		if !prevRole.Equal(curRole) {
			return false, errors.Errorf("Added host role not same, %v != %v", prevRole.List(), curRole.List())
		}
	}
	if prevRole.Has(string(KubesprayNodeRoleMaster)) {
		isMasterAdded = true
	}

	return isMasterAdded, nil
}

func NewDefaultKubesprayScaleRunner(
	vars *KubesprayRunVars,
	allHosts []*KubesprayInventoryHost,
	addedHosts ...*KubesprayInventoryHost,
) (KubesprayRunner, error) {
	var runner KubesprayRunner
	var err error

	isMasterAdded, err := checkTargetHosts(addedHosts)
	if err != nil {
		return nil, errors.Wrap(err, "check target hosts")
	}

	if isMasterAdded {
		vars.IgnoreAssertErrors = "yes"
		vars.EtcdRetries = 20
		runner, err = newDefaultKubesprayRunner(DefaultKubesprayClusterYML, vars, allHosts...)
		if err != nil {
			return nil, err
		}
		if err := runner.AddLimitHosts(false, KubesprayNodeRoleEtcd, KubesprayNodeRoleMaster); err != nil {
			return nil, errors.Errorf("add limit group %s,%s", KubesprayNodeRoleEtcd, KubesprayNodeRoleMaster)
		}
	} else {
		runner, err = newDefaultKubesprayRunner(DefaultKubesprayScaleYML, vars, allHosts...)
		if err != nil {
			return nil, err
		}

		for _, aH := range addedHosts {
			if err := runner.AddLimitHosts(true, aH.Hostname); err != nil {
				return nil, errors.Wrap(err, "add limit host")
			}
		}
	}

	return runner, nil
}

func NewDefaultKubesprayRemoveNodeRunner(
	vars *KubesprayRunVars,
	allHosts []*KubesprayInventoryHost,
	hosts ...*KubesprayInventoryHost,
) (KubesprayRunner, error) {
	/*
	 * isMasterAdded, err := checkTargetHosts(addedHosts)
	 * if err != nil {
	 *     return nil, errors.Wrap(err, "check target hosts")
	 * }
	 */
	names := make([]string, len(hosts))
	for idx := range hosts {
		names[idx] = hosts[idx].Hostname
	}

	nodesVal := strings.Join(names, ",")
	vars.Node = nodesVal
	vars.DeleteNodesConfirmation = "yes"

	return newDefaultKubesprayRunner(DefaultKubesprayRemoveNodeYML, vars, allHosts...)
}

func newDefaultKubesprayRunner(clusterYMM string, vars *KubesprayRunVars, hosts ...*KubesprayInventoryHost) (KubesprayRunner, error) {
	os.Setenv("ANSIBLE_CONFIG", newDefaultAnsiblePath(vars.KubesprayVersion, DefaultPlaybookConfigPath))
	playbookPath := newDefaultAnsiblePath(vars.KubesprayVersion, clusterYMM)
	return NewRunner(playbookPath, vars, hosts...)
}

func NewRunner(playbookPath string, vars *KubesprayRunVars, hosts ...*KubesprayInventoryHost) (KubesprayRunner, error) {
	if err := vars.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate variables")
	}

	runner, err := NewAnsibleRunner(playbookPath, hosts...)
	if err != nil {
		return nil, errors.Wrap(err, "new ansible runner")
	}

	if err := runner.SetExtraVars(vars); err != nil {
		return nil, errors.Wrap(err, "set extra vars")
	}

	pr := &kubesprayRunner{
		AnsibleRunner: runner,
		hosts:         hosts,
	}

	return pr, nil
}

func (r *kubesprayRunner) Run(debug bool) error {
	// defer r.runner.Clear()
	return r.AnsibleRunner.Run(debug)
}
