package e2e

import (
	"testing"

	"github.com/qiniu-ava/mxnet-operator/test/e2e/framework"
)

func TestMXJobBehavior(t *testing.T) {
	name := "all-succeed"
	t.Run(name, func(t *testing.T) {
		job := framework.NewTestMXJob(fw.Namespace(), name, framework.BehaviorSucceed, 3, 3)
		if e := framework.CreateMXJob(job); e != nil {
			t.Fatal("failed to create mxjob: ", e)
		}

		if e := framework.WaitForMXJobSucceed(fw.Namespace(), name); e != nil {
			t.Fatal("failed to wait mxjob: ", e)
		}
	})

	name = "all-failed"
	t.Run(name, func(t *testing.T) {
		job := framework.NewTestMXJob(fw.Namespace(), name, framework.BehaviorFail, 3, 3)
		if e := framework.CreateMXJob(job); e != nil {
			t.Fatal("failed to create mxjob: ", e)
		}

		if e := framework.WaitForMXJobFailed(fw.Namespace(), name); e != nil {
			t.Fatal("failed to wait mxjob: ", e)
		}
	})

	name = "worker-failed"
	t.Run(name, func(t *testing.T) {
		job := framework.NewTestMXJob(fw.Namespace(), name, framework.BehaviorWorkerFail, 3, 3)
		if e := framework.CreateMXJob(job); e != nil {
			t.Fatal("failed to create mxjob: ", e)
		}

		if e := framework.WaitForMXJobFailed(fw.Namespace(), name); e != nil {
			t.Fatal("failed to wait mxjob: ", e)
		}
	})

	name = "server-failed"
	t.Run(name, func(t *testing.T) {
		job := framework.NewTestMXJob(fw.Namespace(), name, framework.BehaviorServerFail, 3, 3)
		if e := framework.CreateMXJob(job); e != nil {
			t.Fatal("failed to create mxjob: ", e)
		}

		if e := framework.WaitForMXJobSucceed(fw.Namespace(), name); e != nil {
			t.Fatal("failed to wait mxjob: ", e)
		}
	})

	name = "scheduler-failed"
	t.Run(name, func(t *testing.T) {
		job := framework.NewTestMXJob(fw.Namespace(), name, framework.BehaviorSchedulerFail, 3, 3)
		if e := framework.CreateMXJob(job); e != nil {
			t.Fatal("failed to create mxjob: ", e)
		}

		if e := framework.WaitForMXJobSucceed(fw.Namespace(), name); e != nil {
			t.Fatal("failed to wait mxjob: ", e)
		}
	})
}
