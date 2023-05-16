module yunion.io/x/kubecomps

go 1.18

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/ceph/go-ceph v0.0.0-20181217221554-e32f9f0f2e94
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/flock v0.8.0
	github.com/gorilla/mux v1.7.3
	github.com/minio/minio-go/v7 v7.0.6
	github.com/openshift/api v0.0.0-20200929171550-c99a4deebbe5
	github.com/openshift/client-go v0.0.0-20200929181438-91d71ef2122c
	github.com/projectcalico/libcalico-go v1.7.2-0.20201119184045-34d8399da148
	github.com/smartystreets/goconvey v1.7.2
	github.com/stretchr/testify v1.8.1
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200819165624-17cef6e3e9d5
	golang.org/x/crypto v0.8.0
	golang.org/x/sync v0.1.0
	gopkg.in/yaml.v2 v2.4.0
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
	sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/yaml v1.2.0
	yunion.io/x/code-generator v0.0.0-20230130032150-a6851cfe4737
	yunion.io/x/jsonutils v1.0.1-0.20230428104347-7c2fdff8e8e7
	yunion.io/x/log v1.0.1-0.20230411060016-feb3f46ab361
	yunion.io/x/onecloud v0.3.10-0-alpha2.0.20230516033511-40d691f4970e
	yunion.io/x/pkg v1.0.1-0.20230504073602-0a74096f836a
	yunion.io/x/sqlchemy v1.1.2-0.20230512065832-5323af46107e
)

require (
	cloud.google.com/go v0.65.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.1 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.0 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.0 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/ClickHouse/clickhouse-go v1.5.4 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20170808103936-bb23615498cd // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig/v3 v3.1.0 // indirect
	github.com/Masterminds/squirrel v1.4.0 // indirect
	github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5 // indirect
	github.com/Microsoft/hcsshim v0.8.10-0.20200715222032-5eafd1556990 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535 // indirect
	github.com/beevik/etree v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cloudflare/golz4 v0.0.0-20150217214814-ef862a3cdc58 // indirect
	github.com/containerd/cgroups v0.0.0-20200531161412-0dbf7f05ba59 // indirect
	github.com/containerd/containerd v1.3.4 // indirect
	github.com/containerd/continuity v0.0.0-20200107194136-26c1120b8d41 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deislabs/oras v0.8.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/cli v0.0.0-20200130152716-5d0cf8839492 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.4.2-0.20200309214505-aa6a9891b09c // indirect
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/docker/spdystream v0.0.0-20160310174837-449fdfce4d96 // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/gin-gonic/gin v1.7.7 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.19.3 // indirect
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.3 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-plus/errors v1.0.0 // indirect
	github.com/golang-plus/uuid v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181017120253-0766667cb4d1 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kelseyhightower/envconfig v0.0.0-20180517194557-dd1402a4d99d // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lib/pq v1.8.0 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/ma314smith/signedxml v0.0.0-20210628192057-abc5b481ae1c // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mattn/go-sqlite3 v1.14.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/minio/md5-simd v1.1.0 // indirect
	github.com/minio/minio-go/v6 v6.0.33 // indirect
	github.com/minio/sha256-simd v0.1.1 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.0 // indirect
	github.com/moby/term v0.0.0-20200312100748-672ec06f55cd // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v1.0.0-rc92 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rs/xid v1.2.1 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shirou/gopsutil/v3 v3.22.10 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/texttheater/golang-levenshtein v0.0.0-20180516184445-d188e65d659e // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/tredoe/osutil/v2 v2.0.0-rc.16 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/vishvananda/netlink v1.1.0 // indirect
	github.com/vishvananda/netns v0.0.0-20211101163701-50045581ed74 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.etcd.io/etcd/api/v3 v3.5.0 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.0 // indirect
	go.etcd.io/etcd/client/v3 v3.5.0 // indirect
	go.opencensus.io v0.22.4 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/term v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	golang.org/x/tools v0.6.0 // indirect
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20230215201556-9c5414ab4bde // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
	google.golang.org/grpc v1.38.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.29.1 // indirect
	gopkg.in/gorp.v1 v1.7.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.19.3 // indirect
	k8s.io/apiserver v0.19.4 // indirect
	k8s.io/cloud-provider v0.19.3 // indirect
	k8s.io/component-base v0.19.4 // indirect
	k8s.io/klog/v2 v2.3.0 // indirect
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6 // indirect
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73 // indirect
	moul.io/http2curl/v2 v2.3.0 // indirect
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/kustomize v2.0.3+incompatible // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.0.1 // indirect
	yunion.io/x/cloudmux v0.3.10-0-alpha.1.0.20230515072204-34461417d90f // indirect
	yunion.io/x/executor v0.0.0-20211018100936-39a2cd966656 // indirect
	yunion.io/x/s3cli v0.0.0-20190917004522-13ac36d8687e // indirect
	yunion.io/x/structarg v0.0.0-20220312084958-9c6c79c7d1c6 // indirect
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0

	github.com/opencontainers/runc => github.com/opencontainers/runc v0.1.1
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
