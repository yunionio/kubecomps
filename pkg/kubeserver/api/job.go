package api

import (
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobStatusType string

const (
	// JobRunning means the job is still running.
	JobStatusRunning JobStatusType = "Running"
	// JobComplete means the job has completed its execution.
	JobStatusComplete JobStatusType = "Complete"
	// JobFailed means the job has failed its execution.
	JobStatusFailed JobStatusType = "Failed"
)

type JobStatus struct {
	// Short, machine understandable job status code.
	Status JobStatusType `json:"status"`
	// A human-readable description of the status of related job.
	Message string `json:"message"`
}

// Job is a presentation layer view of Kubernetes Job resource. This means it is Job plus additional
// augmented data we can get from other sources
type Job struct {
	ObjectMeta
	TypeMeta

	// Aggregate information about pods belonging to this Job.
	Pods *PodInfo `json:"podsInfo"`

	// Container images of the Job.
	ContainerImages []ContainerImage `json:"containerImages"`

	// Init Container images of the Job.
	InitContainerImages []ContainerImage `json:"initContainerImages"`

	// number of parallel jobs defined.
	Parallelism *int32 `json:"parallelism"`

	// Completions specifies the desired number of successfully finished pods the job should be run with.
	Completions *int32 `json:"completions"`

	// JobStatus contains inferred job status based on job conditions
	JobStatus JobStatus `json:"jobStatus"`
	Status    string    `json:"status"`
}

// JobDetail is a presentation layer view of Kubernetes Job resource. This means
// it is Job plus additional augmented data we can get from other sources
// (like services that target the same pods).
type JobDetail struct {
	Job

	// Detailed information about Pods belonging to this Job.
	PodList []*Pod `json:"pods"`

	// List of events related to this Job.
	EventList []*Event `json:"events"`
}

type JobDetailV2 struct {
	NamespaceResourceDetail

	// Aggregate information about pods belonging to this Job.
	Pods *PodInfo `json:"podsInfo"`

	// Container images of the Job.
	ContainerImages []ContainerImage `json:"containerImages"`

	// Init Container images of the Job.
	InitContainerImages []ContainerImage `json:"initContainerImages"`

	// number of parallel jobs defined.
	Parallelism *int32 `json:"parallelism"`

	// Completions specifies the desired number of successfully finished pods the job should be run with.
	Completions *int32 `json:"completions"`

	// JobStatus contains inferred job status based on job conditions
	JobStatus JobStatus `json:"jobStatus"`
	Status    string    `json:"status"`

	// // Detailed information about Pods belonging to this Job.
	// PodList []*Pod `json:"pods"`

	// // List of events related to this Job.
	// EventList []*Event `json:"events"`
}

// CronJob is a presentation layer view of Kubernetes Cron Job resource.
type CronJob struct {
	ObjectMeta
	TypeMeta
	Schedule     string       `json:"schedule"`
	Suspend      *bool        `json:"suspend"`
	Active       int          `json:"active"`
	LastSchedule *metav1.Time `json:"lastSchedule"`
}

type CronJobDetail struct {
	CronJob

	ConcurrencyPolicy       string   `json:"concurrencyPolicy"`
	StartingDeadLineSeconds *int64   `json:"startingDeadlineSeconds"`
	ActiveJobs              []*Job   `json:"activeJobs"`
	InactiveJobs            []*Job   `json:"inactiveJobs"`
	Events                  []*Event `json:"events"`
}

type CronJobDetailV2 struct {
	NamespaceResourceDetail
	Schedule                interface{} `json:"schedule"`
	Suspend                 interface{} `json:"suspend"`
	Active                  interface{} `json:"active"`
	LastSchedule            interface{} `json:"lastSchedule"`
	ConcurrencyPolicy       interface{} `json:"concurrencyPolicy"`
	StartingDeadLineSeconds interface{} `json:"startingDeadlineSeconds"`

	/*
	 * ActiveJobs              []*Job   `json:"activeJobs"`
	 * InactiveJobs            []*Job   `json:"inactiveJobs"`
	 * Events                  []*Event `json:"events"`
	 */
}

type JobCreateInput struct {
	K8sNamespaceResourceCreateInput
	batch.JobSpec
}

type JobCreateInputV2 struct {
	NamespaceResourceCreateInput
	batch.JobSpec
}

type JobListInput struct {
	ListInputK8SNamespaceBase
	ListInputOwner
	Active *bool `json:"active"`
}

type CronJobCreateInput struct {
	K8sNamespaceResourceCreateInput
	v1beta1.CronJobSpec
}

type CronJobCreateInputV2 struct {
	NamespaceResourceCreateInput
	v1beta1.CronJobSpec
}
