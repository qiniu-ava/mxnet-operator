package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type MXJobSpec struct {
	// ReplicaSpecs specifies the MX scheduler replicas to run.
	Scheduler *MXReplicaSpec `json:"scheduler,omitempty"`
	// ReplicaSpecs specifies the MX server replicas to run.
	Server *MXReplicaSpec `json:"server,omitempty"`
	// ReplicaSpecs specifies the MX worker replicas to run.
	Worker *MXReplicaSpec `json:"worker,omitempty"`
}

type MXJobStatus struct {
	// Phase is the MXJob running phase
	Phase MXJobPhase `json:"phase"`

	// ReplicaStatuses specifies the status of each MX replica.
	ReplicaStatuses *MXReplicaStatuses `json:"replicaStatuses"`

	// Represents time when the MXJob was acknowledged by the MXJob controller.
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Represents time when the MXJob was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Represents is an array of current observed MXJob conditions.
	//Conditions []MXJobCondition `json:"conditions"`
}

// MXReplicaStatuses is the status collection of mxnet replica.
type MXReplicaStatuses struct {
	Scheduler *MXReplicaStatus `json:"scheduler,omitempty"`
	Server    *MXReplicaStatus `json:"server,omitempty"`
	Worker    *MXReplicaStatus `json:"worker,omitempty"`
}

type MXReplicaSpec struct {
	// Replicas is the number of desired replicas.
	// This is a pointer to distinguish between explicit zero and unspecified.
	// Defaults to 1.
	// More info: http://kubernetes.io/docs/user-guide/replication-controller#what-is-a-replication-controller
	// +optional
	Replicas *int32              `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	Template *v1.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,3,opt,name=template"`

	// PsRootPort is the port to use for scheduler.
	PsRootPort *int32 `json:"psRootPort,omitempty" protobuf:"varint,1,opt,name=psRootPort"`
}

// MXReplicaStatus mxnet replica status
type MXReplicaStatus struct {
	// The number of actively running pods.
	Active int32 `json:"active,omitempty""`

	// The number of pods which reached phase Succeeded.
	Succeeded int32 `json:"succeeded,omitempty"`

	// The number of pods which reached phase Failed.
	Failed int32 `json:"failed,omitempty"`
}

// MXReplicaType determines how a set of MX processes are handled.
type MXReplicaType string

const (
	// scheduler mxnet training job replica type
	SCHEDULER MXReplicaType = "scheduler"
	// server mxnet training job replica type
	SERVER MXReplicaType = "server"
	// worker mxnet training job replica type
	WORKER MXReplicaType = "worker"
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
