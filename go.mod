module yunion.io/x/kubecomps

go 1.15

require (
	github.com/projectcalico/libcalico-go v1.7.2-0.20201110235728-977c570b2f4b
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	yunion.io/x/jsonutils v0.0.0-20201110084044-3e4e1cb49769
	yunion.io/x/log v0.0.0-20200313080802-57a4ce5966b3
	yunion.io/x/onecloud v0.0.0-20200917023357-9047c8de39ad
	yunion.io/x/pkg v0.0.0-20201028134817-3ed15ee169bc
)

replace (
	github.com/docker/docker => github.com/docker/docker v0.0.0-20190731150326-928381b2215c
	github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0
	github.com/ugorji/go => github.com/ugorji/go v1.1.2
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.0.0
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => github.com/openshift/kubernetes-apimachinery v0.0.0-20191211181342-5a804e65bdc1
	k8s.io/apiserver => k8s.io/apiserver v0.17.0
	k8s.io/cli-runtime => github.com/openshift/kubernetes-cli-runtime v0.0.0-20191211181810-5b89652d688e
	k8s.io/client-go => github.com/openshift/kubernetes-client-go v0.0.0-20191211181558-5dcabadb2b45
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.0
	k8s.io/code-generator => k8s.io/code-generator v0.17.0
	k8s.io/component-base => k8s.io/component-base v0.17.0
	k8s.io/cri-api => k8s.io/cri-api v0.17.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.0
	k8s.io/kubectl => github.com/openshift/kubernetes-kubectl v0.0.0-20200114121535-5e67185ab42c
	k8s.io/kubelet => k8s.io/kubelet v0.17.0

	k8s.io/kubernetes => github.com/openshift/kubernetes v1.17.0-alpha.0.0.20191216151305-079984b0a154
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.0
	k8s.io/metrics => k8s.io/metrics v0.17.0
	k8s.io/node-api => k8s.io/node-api v0.17.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.0
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.0
	k8s.io/sample-controller => k8s.io/sample-controller v0.17.0
)
