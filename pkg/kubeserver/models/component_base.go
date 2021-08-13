package models

import (
	"bytes"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	v1 "k8s.io/api/core/v1"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/embed"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/client/cmd"
	"yunion.io/x/kubecomps/pkg/kubeserver/templates/components"
)

type K8SComponentManager struct{}

func (m K8SComponentManager) EnsureNamespace(cluster *SCluster, namespace string) error {
	return EnsureNamespace(cluster, namespace)
}

func (m K8SComponentManager) NewKubectl(cluster *SCluster) (*cmd.Client, error) {
	kubeconfig, err := cluster.GetKubeconfig()
	if err != nil {
		return nil, errors.Wrap(err, "get kubeconfig")
	}
	cli, err := cmd.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "new kubectl client")
	}
	return cli, nil
}

func (m K8SComponentManager) KubectlApply(cluster *SCluster, manifest string) error {
	cli, err := m.NewKubectl(cluster)
	if err != nil {
		return err
	}
	if err := cli.Apply(manifest); err != nil {
		return errors.Wrap(err, "apply manifest")
	}
	return nil
}

func (m K8SComponentManager) KubectlDelete(cluster *SCluster, manifest string) error {
	cli, err := m.NewKubectl(cluster)
	if err != nil {
		return err
	}
	if err := cli.Delete(manifest); err != nil {
		return errors.Wrap(err, "delete manifest")
	}
	return nil
}

type HelmComponentManager struct {
	K8SComponentManager
	releaseName    string
	namespace      string
	embedChartName string
}

func NewHelmComponentManager(namespace string, releaseName string, embedChartName string) *HelmComponentManager {
	m := new(HelmComponentManager)
	m.releaseName = releaseName
	m.namespace = namespace
	m.embedChartName = embedChartName
	return m
}

func (m HelmComponentManager) NewHelmClient(cluster *SCluster, namespace string) (*helm.Client, error) {
	return NewHelmClient(cluster, namespace)
}

func LoadEmbedChart(chartName string) (*chart.Chart, error) {
	gzData := embed.Get(chartName)
	if gzData == nil {
		return nil, fmt.Errorf("not found embed chart %s", chartName)
	}
	gzReader := bytes.NewReader(gzData)
	return loader.LoadArchive(gzReader)
}

func (m HelmComponentManager) HelmInstall(
	cluster *SCluster,
	namespace string,
	chartName string,
	releaseName string,
	vals map[string]interface{}) (*release.Release, error) {
	cli, err := m.NewHelmClient(cluster, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "new helm client")
	}
	eChart, err := LoadEmbedChart(chartName)
	if err != nil {
		return nil, errors.Wrapf(err, "load chart %s", chartName)
	}
	install := cli.Release().Install()
	install.Namespace = namespace
	install.ReleaseName = releaseName
	install.Atomic = true
	install.Replace = true
	return install.Run(eChart, vals)
}

func (m HelmComponentManager) HelmUninstall(cluster *SCluster, namespace string, releaseName string) error {
	cli, err := m.NewHelmClient(cluster, namespace)
	if err != nil {
		return errors.Wrap(err, "new helm client")
	}
	uninstall := cli.Release().UnInstall()
	_, err = uninstall.Run(releaseName)
	return err
}

func (m HelmComponentManager) HelmUpdate(
	cluster *SCluster,
	namespace string,
	chartName string,
	releaseName string,
	vals map[string]interface{},
) (*release.Release, error) {
	cli, err := m.NewHelmClient(cluster, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "new helm client")
	}
	eChart, err := LoadEmbedChart(chartName)
	if err != nil {
		return nil, errors.Wrapf(err, "load chart %s", chartName)
	}
	upgrade := cli.Release().Upgrade()
	upgrade.Namespace = namespace
	// upgrade.Force = true
	return upgrade.Run(releaseName, eChart, vals)
}

func (m HelmComponentManager) CreateHelmResource(
	cluster *SCluster,
	vals map[string]interface{},
) error {
	if err := m.EnsureNamespace(cluster, m.namespace); err != nil {
		return errors.Wrapf(err, "%s ensure namespace %q", m.releaseName, m.namespace)
	}
	if _, err := m.HelmInstall(cluster, m.namespace, m.embedChartName, m.releaseName, vals); err != nil {
		return errors.Wrapf(err, "create helm %s release", m.releaseName)
	}
	return nil
}

func (m HelmComponentManager) DeleteHelmResource(cluster *SCluster) error {
	if err := m.HelmUninstall(cluster, m.namespace, m.releaseName); err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "release: not found") {
			return nil
		}
		return err
	}
	return nil
}

func (m HelmComponentManager) UpdateHelmResource(cluster *SCluster, vals map[string]interface{}) error {
	_, err := m.HelmUpdate(cluster, m.namespace, m.embedChartName, m.releaseName, vals)
	return err
}

func getSystemComponentCommonConfig(controllerPrefer bool) components.CommonConfig {
	conf := components.CommonConfig{}
	// inject tolerations
	conf.Tolerations = append(conf.Tolerations,
		v1.Toleration{
			Key:    "node-role.kubernetes.io/master",
			Effect: v1.TaintEffectNoSchedule,
		},
		v1.Toleration{
			Key:    "node-role.kubernetes.io/controlplane",
			Effect: v1.TaintEffectNoSchedule,
		},
	)

	controllerNodeTerm := v1.NodeSelectorTerm{
		MatchExpressions: []v1.NodeSelectorRequirement{
			{
				Key:      "onecloud.yunion.io/controller",
				Operator: v1.NodeSelectorOpIn,
				Values:   []string{"enable"},
			},
		},
	}

	controllerNodeSelector := &v1.NodeSelector{
		NodeSelectorTerms: []v1.NodeSelectorTerm{controllerNodeTerm},
	}

	requiredAffinity := &v1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: controllerNodeSelector,
	}

	preferredAffinity := &v1.NodeAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
			{
				Weight:     1,
				Preference: controllerNodeTerm,
			},
		},
	}

	// inject affinity
	if controllerPrefer {
		conf.Affinity = &v1.Affinity{
			NodeAffinity: preferredAffinity,
		}
	} else {
		conf.Affinity = &v1.Affinity{
			NodeAffinity: requiredAffinity,
		}
	}
	return conf
}
