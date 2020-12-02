package api

import (
	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
)

type StorageClassCreateInput struct {
	ClusterResourceCreateInput

	// Provisioner indicates the type of the provisioner.
	Provisioner string `json:"provisioner"`

	// Dynamically provisioned PersistentVolumes of this storage class are
	// created with this reclaimPolicy. Defaults to Delete.
	// +optional
	ReclaimPolicy *v1.PersistentVolumeReclaimPolicy `json:"reclaimPolicy,omitempty"`

	// AllowVolumeExpansion shows whether the storage class allow volume expand
	// +optional
	AllowVolumeExpansion *bool `json:"allowVolumeExpansion,omitempty"`

	// Dynamically provisioned PersistentVolumes of this storage class are
	// created with these mountOptions, e.g. ["ro", "soft"]. Not validated -
	// mount of the PVs will simply fail if one is invalid.
	// +optional
	MountOptions []string `json:"mountOptions,omitempty"`

	// VolumeBindingMode indicates how PersistentVolumeClaims should be
	// provisioned and bound.  When unset, VolumeBindingImmediate is used.
	// This field is only honored by servers that enable the VolumeScheduling feature.
	// +optional
	VolumeBindingMode *storagev1.VolumeBindingMode `json:"volumeBindingMode,omitempty"`

	// Restrict the node topologies where volumes can be dynamically provisioned.
	// Each volume plugin defines its own supported topology specifications.
	// An empty TopologySelectorTerm list means there is no topology restriction.
	// This field is only honored by servers that enable the VolumeScheduling feature.
	// +optional
	AllowedTopologies []v1.TopologySelectorTerm `json:"allowedTopologies,omitempty"`

	// CephRBD is in tree ceph rbd create params:
	// More info: https://kubernetes.io/docs/concepts/storage/storage-classes/#ceph-rbd
	// CephRBD *CephRBDStorageClassCreateInput `json:"cephRBD"`

	// CephCSIRBD is ceph-csi rbd create params
	// More info: https://github.com/ceph/ceph-csi/blob/master/examples/rbd/storageclass.yaml
	CephCSIRBD *CephCSIRBDStorageClassCreateInput `json:"cephCSIRBD"`
}

type CephRBDStorageClassCreateInput struct {
	// Ceph monitors, comma delimited. This parameter is required
	Monitors string `json:"monitors"`
	// Ceph client ID that is capable of creating images in the pool. Default is “admin”.
	AdminId string `json:"adminId"`
	// Secret Name for adminId. This parameter is required. The provided secret must have type “kubernetes.io/rbd”.
	AdminSecretName string `json:"adminSecretName"`
	// The namespace for adminSecretName. Default is “default”.
	AdminSecretNamespace string `json:"adminSecretNamespace"`
	// Ceph RBD pool. Default is “rbd”.
	Pool string `json:"pool"`
	// Ceph client ID that is used to map the RBD image. Default is the same as adminId.
	UserId string `json:"userId"`
	// The name of Ceph Secret for userId to map RBD image. It must exist in the same namespace as PVCs.
	UserSecretName string `json:"userSecretName"`
	// The namespace for userSecretName.
	UserSecretNamespace string `json:"userSecretNamespace"`
	// fsType that is supported by kubernetes. Default: "ext4".
	FsType string `json:"fsType"`
	// Ceph RBD image format, “1” or “2”. Default is “2”.
	ImageFormat string `json:"imageFormat"`
	// This parameter is optional and should only be used if you set imageFormat to “2”. Currently supported features are layering only. Default is “”, and no features are turned on.
	ImageFeatures string `json:"layering"`
}

const (
	StorageClassProvisionerCephCSIRBD = "rbd.csi.ceph.com"
)

type CephCSIRBDStorageClassCreateInput struct {
	// String representing a Ceph cluster to provision storage from.
	ClusterId string `json:"clusterId"`
	Pool      string `json:"pool"`
	// RBD image features, CSI creates image with image-format 2
	// CSI RBD currently supports only `layering` feature.
	ImageFeatures string `json:"imageFeatures"`

	SecretName      string `json:"secretName"`
	SecretNamespace string `json:"secretNamespace"`

	// The secrets have to contain Ceph credentials with required access to the `pool`.
	/*CSIProvisionerSecretName string `json:"csiProvisionerSecretName"`
	CSIProvisionerSecretNamespace string `json:"csiProvisionerSecretNamespace"`
	CSIControllerExpandSecretName string `json:"csiControllerExpandSecretName"`
	CSIControllerExpandSecretNamespace string `json:"csiControllerExpandSecretNamespace"`
	CSINodeStageSecretName string `json:"csiNodeStageSecretName"`
	CSINodeStageSecretNamespace string `json:"csiNodeStageSecretNamespace"`*/
	CSIFsType string `json:"csiFsType"`
	// use rbd-nbd as mounter on supported nodes
	// Mounter string `json:"mounter"`

	// Instruct the plugin it has to encrypt the volume
	// By default it is disabled. Valid values are “true” or “false”.
	// A string is expected here, i.e. “true”, not true.
	// Encrypted string `json:"encrypted"

	// Use external key management system for encryption passphrases by specifying
	// a unique ID matching KMS ConfigMap. The ID is only used for correlation to
	// config map entry.
	// EncryptionKMSId string `json:"encryptionKMSId"
}

// StorageClass is a representation of a kubernetes StorageClass object.
type StorageClass struct {
	ObjectMeta
	TypeMeta

	// provisioner is the driver expected to handle this StorageClass.
	// This is an optionally-prefixed name, like a label key.
	// For example: "kubernetes.io/gce-pd" or "kubernetes.io/aws-ebs".
	// This value may not be empty.
	Provisioner string `json:"provisioner"`

	// parameters holds parameters for the provisioner.
	// These values are opaque to the  system and are passed directly
	// to the provisioner.  The only validation done on keys is that they are
	// not empty.  The maximum number of parameters is
	// 512, with a cumulative max size of 256K
	// +optional
	Parameters map[string]string `json:"parameters"`

	// Is default storage class
	IsDefault bool `json:"isDefault"`
}

// StorageClassDetail provides the presentation layer view of Kubernetes StorageClass resource,
// It is StorageClassDetail plus PersistentVolumes associated with StorageClass.
type StorageClassDetail struct {
	StorageClass
	PersistentVolumes []*PersistentVolume `json:"persistentVolumes"`
}

type StorageClassDetailV2 struct {
	ClusterResourceDetail
	// provisioner is the driver expected to handle this StorageClass.
	// This is an optionally-prefixed name, like a label key.
	// For example: "kubernetes.io/gce-pd" or "kubernetes.io/aws-ebs".
	// This value may not be empty.
	Provisioner string `json:"provisioner"`

	// parameters holds parameters for the provisioner.
	// These values are opaque to the  system and are passed directly
	// to the provisioner.  The only validation done on keys is that they are
	// not empty.  The maximum number of parameters is
	// 512, with a cumulative max size of 256K
	// +optional
	Parameters map[string]string `json:"parameters"`

	// Is default storage class
	IsDefault bool `json:"isDefault"`
}

type StorageClassTestResult struct {
	CephCSIRBD *StorageClassTestResultCephCSIRBD `json:"cephCSIRBD"`
}

type StorageClassTestResultCephCSIRBD struct {
	Pools []string `json:"pools"`
}
