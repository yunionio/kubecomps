package constants

const (
	DefaultRegistryMirror = "registry.cn-beijing.aliyuncs.com/yunionio"
	ServiceType           = "k8s"
	ServiceVersion        = ""
)

// version dependencies are named after the k8s version number
const (
	KUBESPRAY_VERSION_1_17_0 = "kubespray"
	K8S_VERSION_1_17_0       = "v1.17.0"
	CALICO_VERSION_1_17_0    = "v3.16.5"
	CNI_VERSION_1_17_0       = "v0.8.6"
)

const (
	KUBESPRAY_VERSION_1_20_0 = "kubespray_2_17_0"
	K8S_VERSION_1_20_0       = "v1.20.0"
	CNI_VERSION_1_20_0       = "v0.9.1"
	CALICO_VERSION_1_20_0    = "v3.19.2"
)

const (
	NGINX_INGRESS_CONTROLLER_1_17_0 = "v0.41.2"
	NGINX_INGRESS_CONTROLLER_1_20_0 = "v1.0.0"
)

const (
	KUBESPRAY_VERSION_1_22_9        = "kubespray_2_19_1"
	K8S_VERSION_1_22_9              = "v1.22.9"
	ETCD_VERSION_1_22_9             = "v3.5.3"
	CNI_VERSION_1_22_9              = "v1.1.1"
	CALICO_VERSION_1_22_9           = "v3.22.3"
	NGINX_INGRESS_CONTROLLER_1_22_9 = "v1.2.1"
	CONTAINERD_VERSION_1_22_9       = "1.6.4"
)

const (
	FILE_URL = "http"
)
