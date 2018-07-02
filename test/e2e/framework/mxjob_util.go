package framework

import (
	"fmt"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/qiniu-ava/mxnet-operator/pkg/apis/ava/v1alpha1"
	"github.com/qiniu-ava/mxnet-operator/pkg/mx"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Behavior is the replica behavior of MXJob for testing.
type Behavior int

const (
	BehaviorSucceed Behavior = iota
	BehaviorFail
	BehaviorRandomlySucceedOrFail
	BehaviorNotTerminate
	BehaviorSchedulerFail
	BehaviorWorkerFail
	BehaviorServerFail
)

var (
	commandNotTerminate = []string{"sleep", "1000000"}
	commandFail         = []string{"/bin/sh", "-c", "exit 1"}
	commandSlowFail     = []string{"/bin/sh", "-c", "sleep 5 && exit 1"}
	commandSucceed      = []string{"/bin/sh", "-c", "exit 0"}
	// Bash's $RANDOM generates pseudorandom int in range 0 - 32767.
	// Dividing by 16384 gives roughly 50/50 chance of success.
	commandRandomlySucceedOrFail = []string{"/bin/sh", "-c", "exit $(( $RANDOM / 16384 ))"}
)

// NewTestMXJob creates a MXJob for testing purpose.
func NewTestMXJob(ns, name string, behavior Behavior, servers, workers int32) *v1alpha1.MXJob {
	job := &v1alpha1.MXJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.SchemeGroupVersionKind.Kind,
		},
		Spec: v1alpha1.MXJobSpec{
			MXReplicaSpecs: v1alpha1.MXReplicaSpecs{
				Scheduler: &v1alpha1.MXReplicaSpec{
					Replicas: newInt32(1),
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							RestartPolicy: v1.RestartPolicyNever,
							Containers: []v1.Container{
								{
									Name:  "c",
									Image: BusyBoxImage,
								},
							},
						},
					},
				},
				Server: &v1alpha1.MXReplicaSpec{
					Replicas: newInt32(servers),
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							RestartPolicy: v1.RestartPolicyNever,
							Containers: []v1.Container{
								{
									Name:  "c",
									Image: BusyBoxImage,
								},
							},
						},
					},
				},
				Worker: &v1alpha1.MXReplicaSpec{
					Replicas: newInt32(workers),
					Template: &v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							RestartPolicy: v1.RestartPolicyNever,
							Containers: []v1.Container{
								{
									Name:  "c",
									Image: BusyBoxImage,
								},
							},
						},
					},
				},
			},
		},
	}

	switch behavior {
	case BehaviorSucceed:
		job.Spec.MXReplicaSpecs.Scheduler.Template.Spec.Containers[0].Command = commandSucceed
		job.Spec.MXReplicaSpecs.Server.Template.Spec.Containers[0].Command = commandSucceed
		job.Spec.MXReplicaSpecs.Worker.Template.Spec.Containers[0].Command = commandSucceed
	case BehaviorFail:
		job.Spec.MXReplicaSpecs.Scheduler.Template.Spec.Containers[0].Command = commandFail
		job.Spec.MXReplicaSpecs.Server.Template.Spec.Containers[0].Command = commandFail
		job.Spec.MXReplicaSpecs.Worker.Template.Spec.Containers[0].Command = commandFail
	case BehaviorRandomlySucceedOrFail:
		job.Spec.MXReplicaSpecs.Scheduler.Template.Spec.Containers[0].Command = commandRandomlySucceedOrFail
		job.Spec.MXReplicaSpecs.Server.Template.Spec.Containers[0].Command = commandRandomlySucceedOrFail
		job.Spec.MXReplicaSpecs.Worker.Template.Spec.Containers[0].Command = commandRandomlySucceedOrFail
	case BehaviorNotTerminate:
		job.Spec.MXReplicaSpecs.Scheduler.Template.Spec.Containers[0].Command = commandNotTerminate
		job.Spec.MXReplicaSpecs.Server.Template.Spec.Containers[0].Command = commandNotTerminate
		job.Spec.MXReplicaSpecs.Worker.Template.Spec.Containers[0].Command = commandNotTerminate
	case BehaviorSchedulerFail:
		job.Spec.MXReplicaSpecs.Scheduler.Template.Spec.Containers[0].Command = commandSlowFail
		job.Spec.MXReplicaSpecs.Server.Template.Spec.Containers[0].Command = commandSucceed
		job.Spec.MXReplicaSpecs.Worker.Template.Spec.Containers[0].Command = commandSucceed
	case BehaviorServerFail:
		job.Spec.MXReplicaSpecs.Scheduler.Template.Spec.Containers[0].Command = commandSucceed
		job.Spec.MXReplicaSpecs.Server.Template.Spec.Containers[0].Command = commandSlowFail
		job.Spec.MXReplicaSpecs.Worker.Template.Spec.Containers[0].Command = commandSucceed
	case BehaviorWorkerFail:
		job.Spec.MXReplicaSpecs.Scheduler.Template.Spec.Containers[0].Command = commandSucceed
		job.Spec.MXReplicaSpecs.Server.Template.Spec.Containers[0].Command = commandSucceed
		job.Spec.MXReplicaSpecs.Worker.Template.Spec.Containers[0].Command = commandFail
	}
	return job
}

// CreateMXJob creates the MXJob to APIServer.
func CreateMXJob(mxjob *v1alpha1.MXJob) error {
	return sdk.Create(mxjob)
}

// WaitForMXJobFinish waits for the MXJob to finish.
func WaitForMXJobFinish(ns, name string) error {
	return wait.Poll(1*time.Second, jobTimeout, func() (bool, error) {
		job := &v1alpha1.MXJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.SchemeGroupVersionKind.Kind,
			},
		}
		if err := sdk.Get(job); err != nil {
			return false, err
		}
		return mx.IsJobFinished(job.Status), nil
	})
}

// WaitForMXJobSucceed waits for the MXJob to succeed.
func WaitForMXJobSucceed(ns, name string) error {
	return wait.Poll(1*time.Second, jobTimeout, func() (bool, error) {
		job := &v1alpha1.MXJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.SchemeGroupVersionKind.Kind,
			},
		}
		if err := sdk.Get(job); err != nil {
			return false, err
		}

		if !mx.IsJobFinished(job.Status) {
			return false, nil
		}

		if mx.IsJobFailed(job.Status) {
			return false, fmt.Errorf("mxjob is failed")
		}

		return true, nil
	})
}

// WaitForMXJobFailed waits for the MXJob to fail.
func WaitForMXJobFailed(ns, name string) error {
	return wait.Poll(1*time.Second, jobTimeout, func() (bool, error) {
		job := &v1alpha1.MXJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.SchemeGroupVersionKind.Kind,
			},
		}
		if err := sdk.Get(job); err != nil {
			return false, err
		}

		if !mx.IsJobFinished(job.Status) {
			return false, nil
		}

		if mx.IsJobSucceed(job.Status) {
			return false, fmt.Errorf("mxjob is succeed")
		}

		return true, nil
	})
}

func newInt32(val int32) *int32 {
	p := new(int32)
	*p = val
	return p
}
