module yunion.io/x/kubecomps

go 1.12

require (
	github.com/Azure/go-autorest/autorest v0.11.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/ceph/go-ceph v0.0.0-20181217221554-e32f9f0f2e94
	github.com/containerd/containerd v1.4.1-0.20201204210828-e98d7f8eaafc // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gofrs/flock v0.8.0
	github.com/gorilla/mux v1.7.3
	github.com/kr/text v0.2.0 // indirect
	github.com/minio/minio-go/v7 v7.0.6
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/openshift/api v0.0.0-20200929171550-c99a4deebbe5
	github.com/openshift/client-go v0.0.0-20200929181438-91d71ef2122c
	github.com/projectcalico/libcalico-go v1.7.2-0.20201119184045-34d8399da148
	github.com/smartystreets/goconvey v1.6.4
	github.com/stretchr/testify v1.6.1
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200819165624-17cef6e3e9d5
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.3.0
	helm.sh/helm/v3 v3.4.1
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/cli-runtime v0.19.3
	k8s.io/client-go v9.0.0+incompatible
	k8s.io/cluster-bootstrap v0.19.3
	k8s.io/gengo v0.0.0-20200428234225-8167cfdcfc14
	k8s.io/helm v2.12.3+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-proxy v0.19.3
	k8s.io/kubectl v0.19.3
	k8s.io/kubernetes v1.19.3
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/yaml v1.2.0
	yunion.io/x/code-generator v0.0.0-20210727035420-bc4620019c46
	yunion.io/x/jsonutils v0.0.0-20210709075951-798a67800349
	yunion.io/x/log v0.0.0-20201210064738-43181789dc74
	yunion.io/x/onecloud v0.0.0-20210804081451-066d0c6a879d
	yunion.io/x/pkg v0.0.0-20210721081124-55078288ca4c
	yunion.io/x/sqlchemy v0.0.0-20210619142628-653684d2c4f8
)

replace (
	github.com/Azure/go-autorest/autorest => github.com/Azure/go-autorest/autorest v0.11.1

	github.com/docker/docker => github.com/docker/docker v0.0.0-20190731150326-928381b2215c

	// copy from https://github.com/containerd/containerd/blob/master/go.mod
	github.com/gogo/googleapis => github.com/gogo/googleapis v1.3.2
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.5
	github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0
	github.com/ugorji/go => github.com/ugorji/go v1.1.2
	// urfave/cli must be <= v1.22.1 due to a regression: https://github.com/urfave/cli/issues/1092
	github.com/urfave/cli => github.com/urfave/cli v1.22.1
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200224152610-e50cd9704f63
	google.golang.org/grpc => google.golang.org/grpc v1.27.1
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.4.1

	k8s.io/api => k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.3

	k8s.io/apimachinery => github.com/openshift/kubernetes-apimachinery v0.0.0-20200831185207-c0eb43ac4a3e
	k8s.io/apiserver => k8s.io/apiserver v0.19.3
	k8s.io/cli-runtime => github.com/openshift/kubernetes-cli-runtime v0.0.0-20200831185531-852eec47b608
	k8s.io/client-go => github.com/openshift/kubernetes-client-go v0.0.0-20200908071752-9409de4c95e0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.3
	k8s.io/code-generator => k8s.io/code-generator v0.19.3
	k8s.io/component-base => k8s.io/component-base v0.19.3
	k8s.io/cri-api => k8s.io/cri-api v0.19.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.3
	k8s.io/kubectl => github.com/openshift/kubernetes-kubectl v0.0.0-20200922135455-1f5b2cd472a9
	k8s.io/kubelet => k8s.io/kubelet v0.19.3
	k8s.io/kubernetes => github.com/openshift/kubernetes v1.20.0-alpha.0.0.20200922142336-4700daee7399
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.3
	k8s.io/metrics => k8s.io/metrics v0.19.3
	k8s.io/node-api => k8s.io/node-api v0.19.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.3
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.19.3
	k8s.io/sample-controller => k8s.io/sample-controller v0.19.3
)
