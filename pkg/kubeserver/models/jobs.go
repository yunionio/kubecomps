package models

import (
	"context"

	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	jobManager *SJobManager
	_          IClusterModel = new(SJob)
)

func init() {
	GetJobManager()
}

type SJobManager struct {
	SNamespaceResourceBaseManager
}

type SJob struct {
	SNamespaceResourceBase
}

func GetJobManager() *SJobManager {
	if jobManager == nil {
		jobManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SJobManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					new(SJob),
					"jobs_tbl",
					"job",
					"jobs",
					api.ResourceNameJob,
					batch.GroupName,
					batch.SchemeGroupVersion.Version,
					api.KindNameJob,
					new(batch.Job),
				),
			}
		}).(*SJobManager)
	}
	return jobManager
}

func (m *SJobManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query *jsonutils.JSONDict, input *api.JobCreateInputV2) (*api.JobCreateInputV2, error) {
	nInput, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.NamespaceResourceCreateInput = *nInput
	podTemplate := &input.Template
	if err := ValidatePodTemplate(userCred, input.ClusterId, input.NamespaceId, podTemplate); err != nil {
		return nil, errors.Wrap(err, "validate pod template")
	}
	return input, nil
}

func (m *SJobManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.JobCreateInputV2)
	data.Unmarshal(input)
	objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, err
	}
	return &batch.Job{
		ObjectMeta: objMeta,
		Spec:       input.JobSpec,
	}, nil
}

func (job *SJob) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.JobDetailV2, error) {
	return api.JobDetailV2{}, nil
}

func (obj *SJob) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	job := rawObj.(*batch.Job)
	labelSelector := labels.SelectorFromSet(job.Spec.Selector.MatchLabels)
	return GetPodManager().GetRawPodsBySelector(cli, job.GetNamespace(), labelSelector)
}

func (obj *SJob) GetPodInfo(cli *client.ClusterManager, job *batch.Job) (*api.PodInfo, error) {
	pods, err := obj.GetRawPods(cli, job)
	if err != nil {
		return nil, errors.Wrap(err, "job get raw pods")
	}
	podInfo, err := GetPodInfo(job.Status.Active, job.Spec.Completions, pods)
	if err != nil {
		return nil, errors.Wrap(err, "get job pod info")
	}
	// This pod info for jobs should be get from job status, similar to kubectl describe logic.
	podInfo.Running = job.Status.Active
	podInfo.Succeeded = job.Status.Succeeded
	podInfo.Failed = job.Status.Failed
	return podInfo, nil
}

func (obj *SJob) GetDetails(
	cli *client.ClusterManager,
	base interface{},
	k8sObj runtime.Object,
	isList bool,
) interface{} {
	job := k8sObj.(*batch.Job)
	jobStatus := api.JobStatus{Status: api.JobStatusRunning}
	detail := api.JobDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		ContainerImages:         GetContainerImages(&job.Spec.Template.Spec),
		InitContainerImages:     GetInitContainerImages(&job.Spec.Template.Spec),
		Parallelism:             job.Spec.Parallelism,
		Completions:             job.Spec.Completions,
		JobStatus:               jobStatus,
		Status:                  string(jobStatus.Status),
	}
	for _, condition := range job.Status.Conditions {
		if condition.Type == batch.JobComplete && condition.Status == v1.ConditionTrue {
			jobStatus.Status = api.JobStatusComplete
			break
		} else if condition.Type == batch.JobFailed && condition.Status == v1.ConditionTrue {
			jobStatus.Status = api.JobStatusFailed
			jobStatus.Message = condition.Message
			break
		}
	}
	podInfo, err := obj.GetPodInfo(cli, job)
	if err != nil {
		log.Errorf("Get pod info by job %s error: %v", obj.GetName(), err)
	} else {
		detail.Pods = podInfo
	}
	return detail
}
