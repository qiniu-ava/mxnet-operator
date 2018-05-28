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
	// ReplicaSpecs specifies the MX replicas to run.
	ReplicaSpecs []*MXReplicaSpec `json:"replicaSpecs"`
}
type MXJobStatus struct {
	// Phase is the MXJob running phase
	Phase  MXJobPhase `json:"phase"`
	Reason string     `json:"reason"`

	// State indicates the state of the job.
	State State `json:"state"`

	// ReplicaStatuses specifies the status of each MX replica.
	ReplicaStatuses []*MXReplicaStatus `json:"replicaStatuses"`
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
	PsRootPort    *int32 `json:"psRootPort,omitempty" protobuf:"varint,1,opt,name=psRootPort"`
	MXReplicaType `json:"mxReplicaType"`
}

// MXReplicaStatus mxnet replica status
type MXReplicaStatus struct {
	MXReplicaType `json:"mx_replica_type"`
	// State is the overall state of the replica
	State ReplicaState `json:"state"`
	// ReplicasStates provides the number of replicas in each status.
	ReplicasStates map[ReplicaState]int
}

// MXReplicaType determines how a set of MX processes are handled.
type MXReplicaType string

const (
	// SCHEDULER mxnet training job replica type
	SCHEDULER MXReplicaType = "SCHEDULER"
	// SERVER mxnet training job replica type
	SERVER MXReplicaType = "SERVER"
	// WORKER mxnet training job replica type
	WORKER MXReplicaType = "WORKER"
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

// State mxnet job state
type State string

const (
	// StateUnknown state unknown
	StateUnknown State = "Unknown"
	// StateRunning state running
	StateRunning State = "Running"
	// StateSucceeded state succeeded
	StateSucceeded State = "Succeeded"
	// StateFailed state failed
	StateFailed State = "Failed"
)

// ReplicaState mxnet job replica state
type ReplicaState string

const (
	// ReplicaStateUnknown replica state unknown
	ReplicaStateUnknown ReplicaState = "Unknown"
	// ReplicaStateStarting replica state starting
	//ReplicaStateStarting = "Starting"
	// ReplicaStateRunning replica state running
	ReplicaStateRunning = "Running"
	// ReplicaStateFailed replica state failed
	ReplicaStateFailed = "Failed"
	// ReplicaStateSucceeded replica state succeeded
	ReplicaStateSucceeded = "Succeeded"
)
