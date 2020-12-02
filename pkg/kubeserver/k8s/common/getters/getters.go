package getters

import (
	"fmt"

	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
	"k8s.io/kubernetes/pkg/util/node"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
)

// code is reference: k8s.io/kubernetes/pkg/printers/internalversion/printers.go

var (
	podSuccessConditions = []metav1.TableRowCondition{{Type: metav1.RowCompleted, Status: metav1.ConditionTrue, Reason: string(v1.PodSucceeded), Message: "The pod has completed successfully."}}
	podFailedConditions  = []metav1.TableRowCondition{{Type: metav1.RowCompleted, Status: metav1.ConditionTrue, Reason: string(v1.PodFailed), Message: "The pod failed."}}
)

func GetPodStatus(pod *v1.Pod) *api.PodStatusV2 {
	restarts := 0
	totalContainers := len(pod.Spec.Containers)
	readyContainers := 0

	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: pod},
	}

	switch pod.Status.Phase {
	case v1.PodSucceeded:
		row.Conditions = podSuccessConditions
	case v1.PodFailed:
		row.Conditions = podFailedConditions
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		restarts += int(container.RestartCount)
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		restarts = 0
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			restarts += int(container.RestartCount)
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
				readyContainers++
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == node.NodeUnreachablePodReason {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}
	ret := &api.PodStatusV2{
		Ready:    fmt.Sprintf("%d/%d", readyContainers, totalContainers),
		Status:   reason,
		Restarts: int64(restarts),
	}
	nodeName := pod.Spec.NodeName
	nominatedNodeName := pod.Status.NominatedNodeName
	podIP := ""
	if len(pod.Status.PodIPs) < 0 {
		podIP = pod.Status.PodIPs[0].IP
	}

	if podIP == "" {
		podIP = "<none>"
	}
	if nodeName == "" {
		nodeName = "<none>"
	}
	if nominatedNodeName == "" {
		nominatedNodeName = "<none>"
	}

	readinessGates := "<none>"
	if len(pod.Spec.ReadinessGates) > 0 {
		trueConditions := 0
		for _, readinessGate := range pod.Spec.ReadinessGates {
			conditionType := readinessGate.ConditionType
			for _, condition := range pod.Status.Conditions {
				if condition.Type == conditionType {
					if condition.Status == v1.ConditionTrue {
						trueConditions++
					}
					break
				}
			}
		}
		readinessGates = fmt.Sprintf("%d/%d", trueConditions, len(pod.Spec.ReadinessGates))
	}
	ret.NodeName = nodeName
	ret.NominatedNodeName = nominatedNodeName
	ret.ReadinessGates = readinessGates
	return ret
}

func GetDeploymentStatus(podInfo *api.PodInfo, obj apps.Deployment) *api.DeploymentStatus {
	desiredReplicas := obj.Spec.Replicas
	updatedReplicas := obj.Status.UpdatedReplicas
	readyReplicas := obj.Status.ReadyReplicas
	availableReplicas := obj.Status.AvailableReplicas
	ret := &api.DeploymentStatus{
		ReadyReplicas:     int64(readyReplicas),
		UpdatedReplicas:   int64(updatedReplicas),
		AvailableReplicas: int64(availableReplicas),
	}
	if desiredReplicas == nil {
		ret.Status = podInfo.GetStatus()
		return ret
	}
	ret.DesiredReplicas = int64(*desiredReplicas)
	status := api.DeploymentStatusObservedWaiting
	if obj.Generation <= obj.Status.ObservedGeneration {
		status = podInfo.GetStatus()
		cond := deploymentutil.GetDeploymentCondition(obj.Status, apps.DeploymentProgressing)
		if cond != nil && cond.Reason == deploymentutil.TimedOutReason {
			status = cond.Reason
			ret.Status = status
			return ret
		}
		if obj.Spec.Replicas != nil && obj.Status.UpdatedReplicas < *obj.Spec.Replicas {
			// new replicas have been updated
			status = api.DeploymentStatusNewReplicaUpdating
			ret.Status = status
			return ret
		}
		if obj.Status.Replicas > obj.Status.UpdatedReplicas {
			// old replicas are pending termination
			status = api.DeploymentStatusOldReplicaTerminating
			ret.Status = status
			return ret
		}
		if obj.Status.AvailableReplicas < obj.Status.UpdatedReplicas {
			status = api.DeploymentStatusAvailableWaiting
			ret.Status = status
			return ret
		}
	}
	ret.Status = status
	return ret
}

func GetStatefulSetStatus(podInfo *api.PodInfo, obj apps.StatefulSet) *api.StatefulSetStatus {
	ret := new(api.StatefulSetStatus)
	if obj.Spec.UpdateStrategy.Type != apps.RollingUpdateStatefulSetStrategyType {
		status := podInfo.GetStatus()
		ret.Status = status
		return ret
	}
	status := podInfo.GetStatus()
	if obj.Status.ObservedGeneration == 0 || obj.Generation > obj.Status.ObservedGeneration {
		status = api.StatefulSetStatusObservedWaiting
	}
	if obj.Spec.Replicas != nil && obj.Status.ReadyReplicas < *obj.Spec.Replicas {
		status = api.StatefulSetStatusPodReadyWaiting
	}
	if obj.Spec.UpdateStrategy.Type == apps.RollingUpdateStatefulSetStrategyType && obj.Spec.UpdateStrategy.RollingUpdate != nil {
		if obj.Spec.Replicas != nil && obj.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
			if obj.Status.UpdatedReplicas < (*obj.Spec.Replicas - *obj.Spec.UpdateStrategy.RollingUpdate.Partition) {
				status = api.StatefulSetStatusNewReplicaUpdating
			}
		}
	}
	if obj.Status.UpdateRevision != obj.Status.CurrentRevision {
		status = api.StatefulSetStatusUpdateWaiting
	}
	ret.Status = status
	return ret
}

func GetDaemonsetStatus(podInfo *api.PodInfo, obj apps.DaemonSet) *api.DaemonSetStatus {
	ret := new(api.DaemonSetStatus)
	if obj.Spec.UpdateStrategy.Type != apps.RollingUpdateDaemonSetStrategyType {
		ret.Status = podInfo.GetStatus()
		return ret
	}
	if obj.Generation <= obj.Status.ObservedGeneration {
		if obj.Status.UpdatedNumberScheduled < obj.Status.DesiredNumberScheduled {
			ret.Status = api.DaemonSetStatusUpdateWaiting
			return ret
		}
		if obj.Status.NumberAvailable < obj.Status.DesiredNumberScheduled {
			ret.Status = api.DaemonSetStatusPodReadyWaiting
			return ret
		}
		ret.Status = podInfo.GetStatus()
		return ret
	}
	ret.Status = api.DaemonSetStatusObservedWaiting
	return ret
}
