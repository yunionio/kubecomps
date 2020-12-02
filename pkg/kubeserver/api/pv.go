package api

import (
	"k8s.io/api/core/v1"
)

// PersistentVolume provides the simplified presentation layer view of Kubernetes Persistent Volume resource.
type PersistentVolume struct {
	ObjectMeta
	TypeMeta
	Capacity      v1.ResourceList                  `json:"capacity"`
	AccessModes   []v1.PersistentVolumeAccessMode  `json:"accessModes"`
	ReclaimPolicy v1.PersistentVolumeReclaimPolicy `json:"reclaimPolicy"`
	StorageClass  string                           `json:"storageClass"`
	Status        v1.PersistentVolumePhase         `json:"status"`
	Claim         string                           `json:"claim"`
	Reason        string                           `json:"reason"`
	Message       string                           `json:"message"`
}

type PersistentVolumeDetailV2 struct {
	ClusterResourceDetail
	Capacity               v1.ResourceList                  `json:"capacity"`
	AccessModes            []v1.PersistentVolumeAccessMode  `json:"accessModes"`
	ReclaimPolicy          v1.PersistentVolumeReclaimPolicy `json:"reclaimPolicy"`
	StorageClass           string                           `json:"storageClass"`
	Status                 v1.PersistentVolumePhase         `json:"status"`
	Claim                  string                           `json:"claim"`
	Reason                 string                           `json:"reason"`
	Message                string                           `json:"message"`
	PersistentVolumeSource v1.PersistentVolumeSource        `json:"persistentVolumeSource"`
	// PersistentVolumeClaim  *PersistentVolumeClaim    `json:"persistentVolumeClaim"`
}

// PersistentVolumeDetail provides the presentation layer view of Kubernetes Persistent Volume resource.
type PersistentVolumeDetail struct {
	PersistentVolume
	PersistentVolumeSource v1.PersistentVolumeSource `json:"persistentVolumeSource"`
	PersistentVolumeClaim  *PersistentVolumeClaim    `json:"persistentVolumeClaim"`
}
