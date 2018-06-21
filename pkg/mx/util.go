package mx

import (
	"github.com/qiniu-ava/mxnet-operator/pkg/apis/qiniu/v1alpha1"
	"k8s.io/api/core/v1"
)

// IsJobFinished checks whether MXJob is in Succeed or Failed condition.
func IsJobFinished(status v1alpha1.MXJobStatus) bool {
	for _, c := range status.Conditions {
		if (c.Type == v1alpha1.MXJobSucceeded || c.Type == v1alpha1.MXJobFailed) && c.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsJobSucceed checks whether MXJob is in Succeed condition.
func IsJobSucceed(status v1alpha1.MXJobStatus) bool {
	for _, c := range status.Conditions {
		if (c.Type == v1alpha1.MXJobSucceeded) && c.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsJobFailed checks whether MXJob is in Failed condition.
func IsJobFailed(status v1alpha1.MXJobStatus) bool {
	for _, c := range status.Conditions {
		if (c.Type == v1alpha1.MXJobFailed) && c.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}
