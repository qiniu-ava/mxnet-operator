package v1alpha1

import (
	arbv1 "github.com/kubernetes-incubator/kube-arbitrator/pkg/apis/v1alpha1"
	admission "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// FinalizerFinishWebhook is a finalizer for calling finish webhook.
	FinalizerFinishWebhook = "finishWebhook"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MXJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []MXJob `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MXJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              MXJobSpec   `json:"spec"`
	Status            MXJobStatus `json:"status,omitempty"`
}

// MXJobSpec is a desired state description of the MXJob.
type MXJobSpec struct {
	// MXReplicaSpecs is map of MXReplicaType and MXReplicaSpec
	// specifies the MX replicas to run.
	// For example,
	//   {
	//     "scheduler": MXReplicaSpec,
	//     "server": MXReplicaSpec,
	//     "worker": MXReplicaSpe,
	//   }
	MXReplicaSpecs MXReplicaSpecs `json:"replicaSpecs"`

	// StartWebhook describes a notification webhook when MXJob started.
	StartWebhook *Webhook `json:"startWebhook,omitempty"`
	// FinishWebhook describes a notification webhook when MXJob finished.
	FinishWebhook *Webhook `json:"finishWebhook,omitempty"`

	// SchedSpec specifies the parameters for scheduling.
	SchedSpec *arbv1.SchedulingSpecTemplate `json:"schedulingSpec,omitempty"`
}

type Webhook struct {
	// The name of the admission webhook.
	// Name should be fully qualified, e.g., imagepolicy.kubernetes.io, where
	// "imagepolicy" is the name of the webhook, and kubernetes.io is the name
	// of the organization.
	Name string

	// ClientConfig defines how to communicate with the hook.
	// Use common structure in admissionregistration.
	ClientConfig admission.WebhookClientConfig `json:"clientConfig"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Notification struct {
	metav1.TypeMeta `json:",inline"`

	// Request describes the attributes for the notification request.
	Request *NotificationRequest `json:"request,omitempty"`
	// Response describes the attributes for the notification response.
	Response *NotificationResponse `json:"response,omitempty"`
}

// NotificationRequest describes the attributes for the notification webhook request.
type NotificationRequest struct {
	// UID is an identifier for the individual request/response. It allows us to distinguish instances of requests which are
	// otherwise identical (parallel requests, requests when earlier requests did not modify etc)
	// The UID is meant to track the round trip (request/response) between the KAS and the WebHook, not the user request.
	// It is suitable for correlating log entries between the webhook and apiserver, for either auditing or debugging.
	UID types.UID `json:"uid"`
	// Kind is the type of object MXJob.
	Kind metav1.GroupVersionKind `json:"kind"`
	// Name is the name of the object as presented in the request.
	Name string `json:"name"`
	// Namespace is the namespace associated with the request.
	Namespace string `json:"namespace"`

	// Status is the current MXJobStatus at the notification time.
	Status MXJobStatus `json:"status,omitempty"`

	// DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted.
	DeletionTimestamp *metav1.Time `json:"deletionTimestamp,omitempty"`
}

// NotificationResponse describes the attributes for the notification webhook response.
type NotificationResponse struct {
	// UID is an identifier for the individual request/response.
	// This should be copied over from the corresponding NotificationRequest.
	UID types.UID `json:"uid"`

	// Result contains extra details.
	Result *metav1.Status `json:"status,omitempty"`
}

type MXReplicaSpecs struct {
	// ReplicaSpecs specifies the MX scheduler replicas to run.
	Scheduler *MXReplicaSpec `json:"scheduler,omitempty"`
	// ReplicaSpecs specifies the MX server replicas to run.
	Server *MXReplicaSpec `json:"server,omitempty"`
	// ReplicaSpecs specifies the MX worker replicas to run.
	Worker *MXReplicaSpec `json:"worker,omitempty"`
}

type MXJobStatus struct {
	// Phase is the MXJob running phase
	//Phase MXJobPhase `json:"phase"`

	// ReplicaStatuses specifies the status of each MX replica.
	ReplicaStatuses MXReplicaStatuses `json:"replicaStatuses"`

	// Represents time when the MXJob was acknowledged by the MXJob controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Represents time when the MXJob was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Represents is an array of current observed MXJob conditions.
	Conditions []MXJobCondition `json:"conditions"`
}

// MXReplicaStatuses is the status collection of mxnet replica.
type MXReplicaStatuses struct {
	Scheduler *MXReplicaStatus `json:"scheduler,omitempty"`
	Server    *MXReplicaStatus `json:"server,omitempty"`
	Worker    *MXReplicaStatus `json:"worker,omitempty"`
}

type MXReplicaSpec struct {
	// Replicas is the number of desired replicas.
	Replicas *int32              `json:"replicas,omitempty"`
	Template *v1.PodTemplateSpec `json:"template,omitempty"`

	// PsRootPort is the port to use for scheduler.
	PsRootPort *int32 `json:"psRootPort,omitempty"`

	// PsVerbose is the communication loggining level.
	PsVerbose *int32 `json:"psVerbose,omitempty"`
}

// MXReplicaStatus mxnet replica status
type MXReplicaStatus struct {
	// The number of actively running pods.
	Active int32 `json:"active,omitempty""`

	// The number of pods which reached phase Succeeded.
	Succeeded int32 `json:"succeeded,omitempty"`

	// The number of pods which reached phase Failed.
	Failed int32 `json:"failed,omitempty"`

	// The number of pods which is deleted.
	Gone int32 `json:"gone,omitempty"`
}

// MXReplicaType determines how a set of MX processes are handled.
type MXReplicaType string

const (
	// scheduler mxnet training job replica type
	MXReplicaTypeScheduler MXReplicaType = "scheduler"
	// server mxnet training job replica type
	MXReplicaTypeServer MXReplicaType = "server"
	// worker mxnet training job replica type
	MXReplicaTypeWorker MXReplicaType = "worker"
)

// MXJobPhase is the mxnet job phase
type MXJobPhase string

const (
	// MXJobPhaseNone job phase none
	MXJobPhaseNone MXJobPhase = ""
	// MXJobPhaseCreating job phase creating
	MXJobPhaseCreating MXJobPhase = "Creating"
	// MXJobPhaseRunning job phase running
	MXJobPhaseRunning MXJobPhase = "Running"
	// MXJobPhaseCleanUp job phase cleanup
	MXJobPhaseCleanUp MXJobPhase = "CleanUp"
	// MXJobPhaseFailed job phase failed
	MXJobPhaseFailed MXJobPhase = "Failed"
	// MXJobPhaseDone job phase done
	MXJobPhaseDone MXJobPhase = "Done"
)

// MXJobCondition describes the state of the MXJob at a certain point.
type MXJobCondition struct {
	// Type of MXJob condition.
	Type MXJobConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason MXJobReason `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// MXJobConditionType defines all kinds of types of MXJobStatus.
type MXJobConditionType string

const (
	// MXJobCreated means the mxjob has been accepted by the system,
	// but one or more of the pods/services has not been started.
	// This includes time before pods being scheduled and launched.
	MXJobCreated MXJobConditionType = "Created"

	// MXJobInitialized means all sub-resources (e.g. services/pods) of this MXJob
	// have been successfully created.
	// The training is starting to run.
	MXJobInitialized MXJobConditionType = "Initialized"

	// MXJobRunning means all sub-resources (e.g. services/pods) of this MXJob
	// have been successfully scheduled and launched.
	// The training is running without error.
	MXJobRunning MXJobConditionType = "Running"

	// MXJobRestarting means one or more sub-resources (e.g. services/pods) of this MXJob
	// reached phase failed but maybe restarted according to it's restart policy
	// which specified by user in v1.PodTemplateSpec.
	// The training is freezing/pending.
	MXJobRestarting MXJobConditionType = "Restarting"

	// MXJobSucceeded means all sub-resources (e.g. services/pods) of this MXJob
	// reached phase have terminated in success.
	// The training is complete without error.
	MXJobSucceeded MXJobConditionType = "Succeeded"

	// MXJobFailed means one or more sub-resources (e.g. services/pods) of this MXJob
	// reached phase failed with no restarting.
	// The training has failed its execution.
	MXJobFailed MXJobConditionType = "Failed"
)

// MXJobReason defines reasons of the MXJobCondition.
type MXJobReason string

const (
	// MXJobReasonCreated is added in a mxjob when it is created.
	MXJobReasonCreated = "MXJobCreated"
	// MXJobReasonInitialized is added in a mxjob when it is initialized.
	MXJobReasonInitialized = "MXJobInitialized"
	// MXJobReasonSucceeded is added in a mxjob when it is succeeded.
	MXJobReasonSucceeded = "MXJobSucceeded"
	// MXJobReasonRunning is added in a mxjob when it is running.
	MXJobReasonRunning = "MXJobRunning"
	// MXJobReasonFailed is added in a mxjob when it is failed.
	MXJobReasonFailed = "MXJobFailed"
)
