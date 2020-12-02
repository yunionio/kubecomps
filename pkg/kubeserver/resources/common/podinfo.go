package common

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
)

// GetPodInfo returns aggregate information about a group of pods.
func GetPodInfo(current int32, desired *int32, pods []*v1.Pod) api.PodInfo {
	result := api.PodInfo{
		Current:  current,
		Desired:  desired,
		Warnings: make([]api.Event, 0),
	}

	for _, pod := range pods {
		switch pod.Status.Phase {
		case v1.PodRunning:
			result.Running++
		case v1.PodPending:
			result.Pending++
		case v1.PodFailed:
			result.Failed++
		case v1.PodSucceeded:
			result.Succeeded++
		}
	}

	return result
}
