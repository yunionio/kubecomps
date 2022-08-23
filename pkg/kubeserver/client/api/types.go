package api

import (
	apps "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	version2 "k8s.io/apimachinery/pkg/version"
	"k8s.io/kubernetes/pkg/apis/networking"
	"strconv"

	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type KindName = string

type ResourceMap struct {
	GroupVersionResourceKind GroupVersionResourceKind
	Namespaced               bool
}

type GroupVersionResourceKind struct {
	schema.GroupVersionResource
	Kind string
}

var KindToResourceMap = map[string]ResourceMap{
	api.ResourceNameConfigMap: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameConfigMap,
			},
			Kind: api.KindNameConfigMap,
		},
		Namespaced: true,
	},
	api.ResourceNameDaemonSet: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    apps.GroupName,
				Version:  apps.SchemeGroupVersion.Version,
				Resource: api.ResourceNameDaemonSet,
			},
			Kind: api.KindNameDaemonSet,
		},
		Namespaced: true,
	},
	api.ResourceNameDeployment: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    apps.GroupName,
				Version:  apps.SchemeGroupVersion.Version,
				Resource: api.ResourceNameDeployment,
			},
			Kind: api.KindNameDeployment,
		},
		Namespaced: true,
	},
	api.ResourceNameEvent: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameEvent,
			},
			Kind: api.KindNameEvent,
		},
		Namespaced: true,
	},

	api.ResourceNameHorizontalPodAutoscaler: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    autoscalingv1.GroupName,
				Version:  autoscalingv1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameHorizontalPodAutoscaler,
			},
			Kind: api.KindNameHorizontalPodAutoscaler,
		},
		Namespaced: true,
	},
	api.ResourceNameIngress: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    extensionsv1beta1.GroupName,
				Version:  extensionsv1beta1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameIngress,
			},
			Kind: api.KindNameIngress,
		},
		Namespaced: true,
	},
	api.ResourceNameJob: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    batchv1.GroupName,
				Version:  batchv1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameJob,
			},
			Kind: api.KindNameJob,
		},
		Namespaced: true,
	},
	api.ResourceNameCronJob: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    batchv1beta1.GroupName,
				Version:  batchv1beta1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameCronJob,
			},
			Kind: api.KindNameCronJob,
		},
		Namespaced: true,
	},
	api.ResourceNameNamespace: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameNamespace,
			},
			Kind: api.KindNameNamespace,
		},
		Namespaced: false,
	},
	api.ResourceNameNode: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameNode,
			},
			Kind: api.KindNameNode,
		},
		Namespaced: false,
	},
	api.ResourceNamePersistentVolumeClaim: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNamePersistentVolumeClaim,
			},
			Kind: api.KindNamePersistentVolumeClaim,
		},
		Namespaced: true,
	},
	api.ResourceNamePersistentVolume: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNamePersistentVolume,
			},
			Kind: api.KindNamePersistentVolume,
		},
		Namespaced: false,
	},
	api.ResourceNamePod: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNamePod,
			},
			Kind: api.KindNamePod,
		},
		Namespaced: true,
	},
	api.ResourceNameReplicaSet: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    apps.GroupName,
				Version:  apps.SchemeGroupVersion.Version,
				Resource: api.ResourceNameReplicaSet,
			},
			Kind: api.KindNameReplicaSet,
		},
		Namespaced: true,
	},
	api.ResourceNameSecret: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameSecret,
			},
			Kind: api.KindNameSecret,
		},
		Namespaced: true,
	},
	api.ResourceNameService: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameService,
			},
			Kind: api.KindNameService,
		},
		Namespaced: true,
	},
	api.ResourceNameStatefulSet: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    apps.GroupName,
				Version:  apps.SchemeGroupVersion.Version,
				Resource: api.ResourceNameStatefulSet,
			},
			Kind: api.KindNameStatefulSet,
		},
		Namespaced: true,
	},
	api.ResourceNameEndpoint: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameEndpoint,
			},
			Kind: api.KindNameEndpoint,
		},
		Namespaced: true,
	},
	api.ResourceNameStorageClass: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    storagev1.GroupName,
				Version:  storagev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameStorageClass,
			},
			Kind: api.KindNameStorageClass,
		},
		Namespaced: false,
	},

	api.ResourceNameRole: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    rbacv1.GroupName,
				Version:  rbacv1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameRole,
			},
			Kind: api.KindNameRole,
		},
		Namespaced: true,
	},
	api.ResourceNameRoleBinding: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    rbacv1.GroupName,
				Version:  rbacv1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameRoleBinding,
			},
			Kind: api.KindNameRoleBinding,
		},
		Namespaced: true,
	},
	api.ResourceNameClusterRole: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    rbacv1.GroupName,
				Version:  rbacv1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameClusterRole,
			},
			Kind: api.KindNameClusterRole,
		},
		Namespaced: false,
	},
	api.ResourceNameClusterRoleBinding: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    rbacv1.GroupName,
				Version:  rbacv1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameClusterRoleBinding,
			},
			Kind: api.KindNameClusterRoleBinding,
		},
		Namespaced: false,
	},
	api.ResourceNameServiceAccount: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameServiceAccount,
			},
			Kind: api.KindNameServiceAccount,
		},
		Namespaced: true,
	},
	api.ResourceNameLimitRange: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameLimitRange,
			},
			Kind: api.KindNameLimitRange,
		},
		Namespaced: true,
	},
	api.ResourceNameResourceQuota: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: api.ResourceNameResourceQuota,
			},
			Kind: api.KindNameResourceQuota,
		},
		Namespaced: true,
	},
}

func GetResourceKinds() sets.String {
	kinds := sets.NewString()
	for keyPlural := range KindToResourceMap {
		kinds.Insert(keyPlural)
	}
	return kinds
}

func TranslateKindPlural(plural string) string {
	if GetResourceKinds().Has(plural) {
		return plural
	}
	switch plural {
	case "k8s_services":
		return api.ResourceNameService
	case "k8s_nodes":
		return api.ResourceNameNode
	case "k8s_endpoints":
		return api.ResourceNameEndpoint
	case "rbacroles":
		return api.ResourceNameRole
	case "rbacclusterroles":
		return api.ResourceNameClusterRole
	case "rbacrolebindings":
		return api.ResourceNameRoleBinding
	case "rbacclusterrolebindings":
		return api.ResourceNameClusterRoleBinding
	}
	return plural
}

// GetKindToResourceMap TODO: Support more version
func GetKindToResourceMap(ver *version2.Info) map[string]ResourceMap {
	res := KindToResourceMap
	if minor, err := strconv.Atoi(ver.Minor); err == nil && minor >= 21 {
		res[api.ResourceNameIngress] = ResourceMap{
			GroupVersionResourceKind: GroupVersionResourceKind{
				GroupVersionResource: schema.GroupVersionResource{
					Group:    networking.GroupName,
					Version:  corev1.SchemeGroupVersion.Version,
					Resource: api.ResourceNameIngress,
				},
				Kind: api.KindNameIngress,
			},
			Namespaced: true,
		}
		res[api.KindNameCronJob] = ResourceMap{
			GroupVersionResourceKind: GroupVersionResourceKind{
				GroupVersionResource: schema.GroupVersionResource{
					Group:    batchv1beta1.GroupName,
					Version:  batchv1.SchemeGroupVersion.Version,
					Resource: api.ResourceNameCronJob,
				},
				Kind: api.KindNameCronJob,
			},
			Namespaced: true,
		}
	}
	return res
}

func GetResourceMapByVersion(kind string, ver *version2.Info) ResourceMap {
	return GetKindToResourceMap(ver)[kind]
}
