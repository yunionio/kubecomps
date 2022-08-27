package models

import (
	batch "k8s.io/api/batch/v1"
	batch2 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/jsonutils"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	cronJobManager *SCronJobManager
	_              IClusterModel = new(SCronJob)
)

func init() {
	GetCronJobManager()
}

func GetCronJobManager() *SCronJobManager {
	if cronJobManager == nil {
		cronJobManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SCronJobManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SCronJob{},
					"cronjobs_tbl",
					"cronjob",
					"cronjobs",
					api.ResourceNameCronJob,
					"",
					"",
					api.KindNameCronJob,
					new(unstructured.Unstructured),
				),
			}
		}).(*SCronJobManager)
	}
	return cronJobManager
}

type SCronJobManager struct {
	SNamespaceResourceBaseManager
}

type SCronJob struct {
	SNamespaceResourceBase
}

func (m *SCronJobManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.CronJobCreateInputV2)
	res := new(unstructured.Unstructured)
	spec := make(map[string]interface{})
	var (
		err      error
		timeZone string
		objMeta  metav1.ObjectMeta
		//jobSpec  map[string]interface{}
	)
	log.Errorf("data: %v", data.String())
	err = data.Unmarshal(input)
	if err != nil {
		return nil, errors.Wrap(err, "cronjob input unmarshal error")
	}
	objMeta, err = input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, errors.Wrap(err, "cronjob input get meta error")
	}
	objMeta = *api.AddObjectMetaDefaultLabel(&objMeta)

	if len(input.JobTemplate.Spec.Template.Spec.RestartPolicy) == 0 {
		input.JobTemplate.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
	}
	input.JobTemplate.Spec.Template.ObjectMeta = objMeta

	// meta object
	res.SetName(objMeta.Name)
	res.SetNamespace(objMeta.Namespace)
	res.SetLabels(objMeta.Labels)
	res.SetAnnotations(objMeta.Annotations)
	// optional schedule
	if input.Schedule != "" {
		spec["schedule"] = input.Schedule
	}
	// timeZone, for 1.25, optional
	timeZone, _ = data.GetString("timeZone")
	if timeZone != "" {
		spec["timeZone"] = timeZone
	}
	// optional StartingDeadlineSeconds
	if input.StartingDeadlineSeconds != nil {
		spec["startingDeadlineSeconds"] = input.StartingDeadlineSeconds
	}
	if input.ConcurrencyPolicy != "" {
		spec["concurrencyPolicy"] = input.ConcurrencyPolicy
	}
	if input.Suspend != nil {
		spec["suspend"] = *(input.Suspend)
	}
	if input.SuccessfulJobsHistoryLimit != nil {
		spec["successfulJobsHistoryLimit"] = *(input.SuccessfulJobsHistoryLimit)
	}
	if input.FailedJobsHistoryLimit != nil {
		spec["failedJobsHistoryLimit"] = *(input.FailedJobsHistoryLimit)
	}
	// in 1.25/1.17, job template both v1/JobSpec and v1.ObjectMeta
	//jobSpec, err = m.generateJobSpec(&input.CronJobSpec.JobTemplate.Spec)
	//if err != nil {
	//	return nil, errors.Wrap(err, "generate job spec")
	//}
	jobTemplate, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&(input.JobTemplate))
	if err != nil {
		log.Errorf("error when convert to unstruc, %v", err)
	}
	spec["jobTemplate"] = jobTemplate
	//map[string]interface{}{
	//	"metadata": map[string]interface{}{
	//		"name":        objMeta.Name,
	//		"namespace":   objMeta.Namespace,
	//		"labels":      m.generateStrInterface(objMeta.Labels),
	//		"annotations": m.generateStrInterface(objMeta.Annotations),
	//	},
	//	"spec": input.CronJobSpec.JobTemplate.Spec.String(),
	//}
	log.Errorf("%v", jobTemplate)
	err = unstructured.SetNestedMap(res.Object, spec, "spec")
	if err != nil {
		return nil, errors.Wrap(err, "set nested map of unstructured")
	}

	return res, nil
}

func (obj *SCronJob) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	cj := k8sObj.(*unstructured.Unstructured)
	spec, _, _ := unstructured.NestedMap(cj.Object, "spec")
	status, _, _ := unstructured.NestedMap(cj.Object, "status")
	actives, _, _ := unstructured.NestedSlice(status, "active")
	schedule, _, _ := unstructured.NestedString(spec, "schedule")
	suspend, _, _ := unstructured.NestedBool(spec, "suspend")
	lastScheduleTime, _, _ := unstructured.NestedMap(status, "lastScheduleTime")
	concurrencyPolicy, _, _ := unstructured.NestedString(spec, "concurrencyPolicy")
	startingDeadlineSeconds, _, _ := unstructured.NestedInt64(spec, "concurrencyPolicy")

	detail := api.CronJobDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Schedule:                schedule,
		Suspend:                 suspend,
		Active:                  len(actives),
		LastSchedule:            lastScheduleTime,
		ConcurrencyPolicy:       concurrencyPolicy,
		StartingDeadLineSeconds: startingDeadlineSeconds,
	}
	return detail
}

func filterJobsByOwnerUID(UID types.UID, jobs []*batch.Job) (matchingJobs []*batch.Job) {
	for _, j := range jobs {
		for _, i := range j.OwnerReferences {
			if i.UID == UID {
				matchingJobs = append(matchingJobs, j)
				break
			}
		}
	}
	return
}

func filterJobsByState(active bool, jobs []*batch.Job) (matchingJobs []*batch.Job) {
	for _, j := range jobs {
		if active && j.Status.Active > 0 {
			matchingJobs = append(matchingJobs, j)
		} else if !active && j.Status.Active == 0 {
			matchingJobs = append(matchingJobs, j)
		} else {
			//sup
		}
	}
	return
}

// func (obj *SCronJob) GetRawJobs(cronjob *batch2.CronJob) ([]*batch.Job, error) {
// jobs, err := GetJobManager().GetRawJobs(obj.GetCluster(), obj.GetNamespace())
// if err != nil {
// return nil, err
// }
// return filterJobsByOwnerUID(cronjob.GetUID(), jobs), nil
// }

// func (obj *SCronJob) GetJobsByState(active bool) ([]*api.Job, error) {
// jobs, err := obj.GetRawJobs()
// if err != nil {
// return nil, err
// }
// jobs = filterJobsByState(active, jobs)
// return GetJobManager().GetAPIJobs(obj.GetCluster(), jobs)
// }

// func (obj *SCronJob) GetActiveJobs() ([]*api.Job, error) {
// return obj.GetJobsByState(true)
// }

// func (obj *SCronJob) GetInActiveJobs() ([]*api.Job, error) {

// return obj.GetJobsByState(false)
// }

// TriggerCronJob manually triggers a cron job and creates a new job.
func (obj *SCronJob) TriggerCronJob() error {
	kObj, err := obj.GetK8sObject()
	cronJob := kObj.(*batch2.CronJob)

	annotations := make(map[string]string)
	annotations["cronjob.kubernetes.io/instantiate"] = "manual"

	labels := make(map[string]string)
	for k, v := range cronJob.Spec.JobTemplate.Labels {
		labels[k] = v
	}

	//job name cannot exceed DNS1053LabelMaxLength (52 characters)
	var newJobName string
	if len(cronJob.Name) < 42 {
		newJobName = cronJob.Name + "-manual-" + rand.String(3)
	} else {
		newJobName = cronJob.Name[0:41] + "-manual-" + rand.String(3)
	}

	jobToCreate := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        newJobName,
			Namespace:   cronJob.GetNamespace(),
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: cronJob.Spec.JobTemplate.Spec,
	}

	cli, err := obj.GetClusterClient()
	if err != nil {
		return err
	}
	_, err = cli.GetHandler().CreateV2(api.ResourceNameJob, cronJob.GetNamespace(), jobToCreate)
	if err != nil {
		return err
	}

	return nil
}

func (m *SCronJobManager) generateJobSpec(spec *batch.JobSpec) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	if spec.Parallelism != nil {
		res["parallelism"] = *spec.Parallelism
	}
	if spec.Completions != nil {
		res["completions"] = *spec.Completions
	}
	if spec.ActiveDeadlineSeconds != nil {
		res["activeDeadlineSeconds"] = *spec.ActiveDeadlineSeconds
	}
	if spec.BackoffLimit != nil {
		res["backoffLimit"] = *spec.BackoffLimit
	}
	selector, err := m.generateSelector(spec.Selector)
	if err == nil {
		res["selector"] = selector
	}
	pt, err := m.generatePodTemplate(&(spec.Template))
	if err == nil {
		res["template"] = pt
	}

	return res, nil
}

func (m *SCronJobManager) generateSelector(s *metav1.LabelSelector) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	matchLabels := make(map[string]interface{})
	matchExpressions := make([]map[string]interface{}, len(s.MatchExpressions))
	for k, v := range s.MatchLabels {
		matchLabels[k] = v
	}

	for i, matchExpression := range s.MatchExpressions {
		matchExpressions[i]["key"] = matchExpression.Key
		matchExpressions[i]["operator"] = matchExpression.Operator

		values := make([]interface{}, len(matchExpression.Values))
		for j, value := range matchExpression.Values {
			values[j] = value
		}
		matchExpressions[i]["values"] = values
	}

	res["matchExpressions"] = matchExpressions
	res["matchLabels"] = matchLabels
	return res, nil
}

func (m *SCronJobManager) generatePodTemplate(spec *v1.PodTemplateSpec) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	res["metadata"] = map[string]interface{}{
		"name":        spec.Name,
		"namespace":   spec.Namespace,
		"labels":      m.generateStrInterface(spec.Labels),
		"annotations": m.generateStrInterface(spec.Annotations),
	}
	podSpec := spec.Spec.String()
	res["spec"] = podSpec
	return res, nil
}

func (m *SCronJobManager) generateStrInterface(src map[string]string) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range src {
		res[k] = v
	}
	return res
}
