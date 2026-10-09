package models

import (
	"context"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	vcJobManager *SVCJobManager
	_            IClusterModel  = new(SVCJob)
	_            IPodOwnerModel = new(SVCJob)
)

func init() {
	GetVCJobManager()
}

type SVCJobManager struct {
	SNamespaceResourceBaseManager
}

type SVCJob struct {
	SNamespaceResourceBase

	Queue string `width:"256" charset:"utf8" nullable:"true" list:"user"`
}

func GetVCJobManager() *SVCJobManager {
	if vcJobManager == nil {
		vcJobManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SVCJobManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					new(SVCJob),
					"vcjobs_tbl",
					"vcjob",
					"vcjobs",
					api.ResourceNameVolcanoJob,
					"batch.volcano.sh",
					"v1alpha1",
					api.KindNameVolcanoJob,
					&unstructured.Unstructured{},
				),
			}
		}).(*SVCJobManager)
	}
	return vcJobManager
}

func (m *SVCJobManager) ListRemoteObjects(cli *client.ClusterManager) ([]interface{}, error) {
	return listVolcanoRemoteObjects(cli, api.KindNameVolcanoJob, m.SNamespaceResourceBaseManager.ListRemoteObjects)
}

func (m *SVCJobManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.VCJobListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.NamespaceResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.Queue != "" {
		q = q.Equals("queue", input.Queue)
	}
	return q, nil
}

func (m *SVCJobManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.VCJobCreateInput) (*api.VCJobCreateInput, error) {
	nInput, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.NamespaceResourceCreateInput = *nInput
	if err := validateVCJobCreate(input); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SVCJobManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.VCJobCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal job create input")
	}
	ns, err := model.(api.INamespaceGetter).GetNamespaceName()
	if err != nil {
		return nil, err
	}
	return newJobObject(input, ns)
}

func (obj *SVCJob) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detail := api.VCJobDetail{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
	}
	u, err := asUnstructured(k8sObj)
	if err != nil {
		detail.Queue = obj.Queue
		detail.Status = obj.Status
		return detail
	}
	detail.Queue = nestedString(u, "spec", "queue")
	detail.PodGroupName = volcanoPodGroupName(u)
	detail.Status = volcanoPhase(u, "status", "state", "phase")
	detail.MinAvailable = nestedInt64(u, "spec", "minAvailable")
	detail.PriorityClassName = nestedString(u, "spec", "priorityClassName")
	detail.NetworkTopologyMode = nestedString(u, "spec", "networkTopology", "mode")
	detail.HighestTierAllowed = nestedInt64(u, "spec", "networkTopology", "highestTierAllowed")
	detail.Running = nestedInt64(u, "status", "running")
	detail.Pending = nestedInt64(u, "status", "pending")
	return detail
}

func (obj *SVCJob) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if u, ok := extObj.(*unstructured.Unstructured); ok {
		obj.Status = volcanoPhase(u, "status", "state", "phase")
		obj.Queue = nestedString(u, "spec", "queue")
		return nil
	}
	return obj.SNamespaceResourceBase.SetStatusByRemoteObject(ctx, userCred, extObj)
}

func (obj *SVCJob) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	name := obj.GetName()
	namespace := ""
	if u, err := asUnstructured(rawObj); err == nil {
		if u.GetName() != "" {
			name = u.GetName()
		}
		namespace = u.GetNamespace()
	}
	if namespace == "" {
		ns, err := obj.GetNamespaceName()
		if err != nil {
			return []*v1.Pod{}, nil
		}
		namespace = ns
	}
	return volcanoPodsByJob(cli, namespace, name)
}

func (obj *SVCJob) GetDetailsRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	raw, err := obj.SNamespaceResourceBase.GetDetailsRawdata(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return volcanoRawdata(raw, api.VolcanoJobKind), nil
}

func (obj *SVCJob) UpdateRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ns, err := obj.GetNamespace()
	if err != nil {
		return nil, err
	}
	updated, err := replaceVolcanoObject(obj, api.ResourceNameVolcanoJob, ns.GetName(), data, api.VolcanoJobKind, api.VolcanoJobAPIVersion)
	if err != nil {
		return nil, err
	}
	return volcanoRawdata(K8SObjectToJSONObject(updated), api.VolcanoJobKind), nil
}

func validateVCJobCreate(input *api.VCJobCreateInput) error {
	if input.Name == "" {
		return httperrors.NewNotEmptyError("name is empty")
	}
	if strings.TrimSpace(input.Queue) == "" {
		return httperrors.NewNotEmptyError("queue is empty")
	}
	if input.MinAvailable < 1 {
		return httperrors.NewInputParameterError("minAvailable must be >= 1")
	}
	switch input.NetworkTopologyMode {
	case "", "hard", "soft":
	default:
		return httperrors.NewInputParameterError("networkTopologyMode must be empty, hard or soft")
	}
	if input.NetworkTopologyMode != "" && input.HighestTierAllowed < 1 {
		return httperrors.NewInputParameterError("highestTierAllowed must be >= 1")
	}
	if len(input.Tasks) == 0 {
		return httperrors.NewInputParameterError("tasks is empty")
	}
	var replicas int32
	for i := range input.Tasks {
		task := input.Tasks[i]
		if strings.TrimSpace(task.Name) == "" {
			return httperrors.NewNotEmptyError("task name is empty")
		}
		if task.Replicas < 1 {
			return httperrors.NewInputParameterError("task replicas must be >= 1")
		}
		if len(task.Containers) == 0 {
			return httperrors.NewInputParameterError("task containers is empty")
		}
		for j := range task.Containers {
			container := task.Containers[j]
			if strings.TrimSpace(container.Name) == "" {
				return httperrors.NewNotEmptyError("container name is empty")
			}
			if strings.TrimSpace(container.Image) == "" {
				return httperrors.NewNotEmptyError("container image is empty")
			}
		}
		replicas += task.Replicas
	}
	if input.MinAvailable > replicas {
		return httperrors.NewInputParameterError("minAvailable cannot exceed the sum of task replicas")
	}
	return nil
}

func newJobObject(input *api.VCJobCreateInput, namespace string) (*unstructured.Unstructured, error) {
	schedulerName := strings.TrimSpace(input.SchedulerName)
	if schedulerName == "" {
		schedulerName = "volcano"
	}
	priorityClassName := strings.TrimSpace(input.PriorityClassName)
	tasks := make([]interface{}, 0, len(input.Tasks))
	for i := range input.Tasks {
		task := input.Tasks[i]
		restartPolicy := strings.TrimSpace(task.RestartPolicy)
		if restartPolicy == "" {
			restartPolicy = "OnFailure"
		}
		podSpec := map[string]interface{}{
			"restartPolicy": restartPolicy,
			"containers":    jobContainers(task.Containers),
		}
		if priorityClassName != "" {
			podSpec["priorityClassName"] = priorityClassName
		}
		tasks = append(tasks, map[string]interface{}{
			"replicas": int64(task.Replicas),
			"name":     strings.TrimSpace(task.Name),
			"template": map[string]interface{}{
				"spec": podSpec,
			},
		})
	}
	u := &unstructured.Unstructured{Object: map[string]interface{}{}}
	u.SetAPIVersion(api.VolcanoJobAPIVersion)
	u.SetKind(api.VolcanoJobKind)
	u.SetName(input.Name)
	u.SetNamespace(namespace)
	if err := unstructured.SetNestedField(u.Object, int64(input.MinAvailable), "spec", "minAvailable"); err != nil {
		return nil, err
	}
	if err := unstructured.SetNestedField(u.Object, schedulerName, "spec", "schedulerName"); err != nil {
		return nil, err
	}
	if err := unstructured.SetNestedField(u.Object, strings.TrimSpace(input.Queue), "spec", "queue"); err != nil {
		return nil, err
	}
	if priorityClassName != "" {
		if err := unstructured.SetNestedField(u.Object, priorityClassName, "spec", "priorityClassName"); err != nil {
			return nil, err
		}
	}
	if input.RestartOnEvict {
		policies := []interface{}{
			map[string]interface{}{
				"event":  "PodEvicted",
				"action": "RestartJob",
			},
		}
		if err := unstructured.SetNestedSlice(u.Object, policies, "spec", "policies"); err != nil {
			return nil, err
		}
	}
	if mode := strings.TrimSpace(input.NetworkTopologyMode); mode != "" {
		tier := input.HighestTierAllowed
		if tier < 1 {
			tier = 1
		}
		topology := map[string]interface{}{
			"mode":               mode,
			"highestTierAllowed": int64(tier),
		}
		if err := unstructured.SetNestedMap(u.Object, topology, "spec", "networkTopology"); err != nil {
			return nil, err
		}
	}
	if err := unstructured.SetNestedSlice(u.Object, tasks, "spec", "tasks"); err != nil {
		return nil, err
	}
	return u, nil
}

func memoryQuantity(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return value + "G"
	}
	return value
}

func jobContainers(containers []api.VCJobContainerInput) []interface{} {
	out := make([]interface{}, 0, len(containers))
	for i := range containers {
		container := containers[i]
		pullPolicy := strings.TrimSpace(container.ImagePullPolicy)
		if pullPolicy == "" {
			pullPolicy = "IfNotPresent"
		}
		item := map[string]interface{}{
			"name":            strings.TrimSpace(container.Name),
			"image":           strings.TrimSpace(container.Image),
			"imagePullPolicy": pullPolicy,
		}
		if command := strings.Fields(container.Command); len(command) > 0 {
			argv := make([]interface{}, len(command))
			for i := range command {
				argv[i] = command[i]
			}
			item["command"] = argv
		}
		cpu := strings.TrimSpace(container.Cpu)
		cpuLimit := strings.TrimSpace(container.CpuLimit)
		if cpuLimit == "" {
			cpuLimit = cpu
		}
		memory := memoryQuantity(container.Memory)
		memoryLimit := memoryQuantity(container.MemoryLimit)
		if memoryLimit == "" {
			memoryLimit = memory
		}
		requests := map[string]interface{}{}
		limits := map[string]interface{}{}
		if cpu != "" {
			requests["cpu"] = cpu
		}
		if cpuLimit != "" {
			limits["cpu"] = cpuLimit
		}
		if memory != "" {
			requests["memory"] = memory
		}
		if memoryLimit != "" {
			limits["memory"] = memoryLimit
		}
		resources := map[string]interface{}{}
		if len(requests) > 0 {
			resources["requests"] = requests
		}
		if len(limits) > 0 {
			resources["limits"] = limits
		}
		if len(resources) > 0 {
			item["resources"] = resources
		}
		out = append(out, item)
	}
	return out
}
