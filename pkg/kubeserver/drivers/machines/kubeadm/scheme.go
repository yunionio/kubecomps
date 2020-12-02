package kubeadm

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeproxyconfigv1alpha1 "k8s.io/kube-proxy/config/v1alpha1"
	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"

	"yunion.io/x/pkg/errors"
)

// GetCodecs returns a type that can be used to deserialize most kubeadm
// configuration types.
func GetCodecs() serializer.CodecFactory {
	sb := &scheme.Builder{GroupVersion: kubeadmv1beta1.SchemeGroupVersion}

	sb.Register(&kubeadmv1beta1.JoinConfiguration{}, &kubeadmv1beta1.InitConfiguration{}, &kubeadmv1beta1.ClusterConfiguration{})
	kubeadmScheme, err := sb.Build()
	if err != nil {
		panic(err)
	}
	return serializer.NewCodecFactory(kubeadmScheme)
}

func GetKubeProxyCodecs() serializer.CodecFactory {
	sb := &scheme.Builder{GroupVersion: kubeproxyconfigv1alpha1.SchemeGroupVersion}

	sb.Register(&kubeproxyconfigv1alpha1.KubeProxyConfiguration{})
	kubeproxyScheme, err := sb.Build()
	if err != nil {
		panic(err)
	}
	return serializer.NewCodecFactory(kubeproxyScheme)
}

// ConfigurationToYAML converts a kubeadm configuration type to its YAML
// representation.
func ConfigurationToYAML(obj runtime.Object) (string, error) {
	initcfg, err := util.MarshalToYamlForCodecs(obj, kubeadmv1beta1.SchemeGroupVersion, GetCodecs())
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal init configuration")
	}
	return string(initcfg), nil
}

func KubeProxyConfigurationToYAML(obj runtime.Object) (string, error) {
	cfg, err := util.MarshalToYamlForCodecs(obj, kubeproxyconfigv1alpha1.SchemeGroupVersion, GetKubeProxyCodecs())
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal kube proxy configuration")
	}
	return string(cfg), nil
}
