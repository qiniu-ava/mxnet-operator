// +build !ignore_autogenerated

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1alpha1

import (
	core_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MXJob) DeepCopyInto(out *MXJob) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MXJob.
func (in *MXJob) DeepCopy() *MXJob {
	if in == nil {
		return nil
	}
	out := new(MXJob)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MXJob) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MXJobCondition) DeepCopyInto(out *MXJobCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MXJobCondition.
func (in *MXJobCondition) DeepCopy() *MXJobCondition {
	if in == nil {
		return nil
	}
	out := new(MXJobCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MXJobList) DeepCopyInto(out *MXJobList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MXJob, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MXJobList.
func (in *MXJobList) DeepCopy() *MXJobList {
	if in == nil {
		return nil
	}
	out := new(MXJobList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MXJobList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MXJobSpec) DeepCopyInto(out *MXJobSpec) {
	*out = *in
	in.MXReplicaSpecs.DeepCopyInto(&out.MXReplicaSpecs)
	if in.StartWebhook != nil {
		in, out := &in.StartWebhook, &out.StartWebhook
		if *in == nil {
			*out = nil
		} else {
			*out = new(Webhook)
			(*in).DeepCopyInto(*out)
		}
	}
	if in.FinishWebhook != nil {
		in, out := &in.FinishWebhook, &out.FinishWebhook
		if *in == nil {
			*out = nil
		} else {
			*out = new(Webhook)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MXJobSpec.
func (in *MXJobSpec) DeepCopy() *MXJobSpec {
	if in == nil {
		return nil
	}
	out := new(MXJobSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MXJobStatus) DeepCopyInto(out *MXJobStatus) {
	*out = *in
	in.ReplicaStatuses.DeepCopyInto(&out.ReplicaStatuses)
	if in.StartTime != nil {
		in, out := &in.StartTime, &out.StartTime
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.Time)
			(*in).DeepCopyInto(*out)
		}
	}
	if in.CompletionTime != nil {
		in, out := &in.CompletionTime, &out.CompletionTime
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.Time)
			(*in).DeepCopyInto(*out)
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]MXJobCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MXJobStatus.
func (in *MXJobStatus) DeepCopy() *MXJobStatus {
	if in == nil {
		return nil
	}
	out := new(MXJobStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MXReplicaSpec) DeepCopyInto(out *MXReplicaSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	if in.Template != nil {
		in, out := &in.Template, &out.Template
		if *in == nil {
			*out = nil
		} else {
			*out = new(core_v1.PodTemplateSpec)
			(*in).DeepCopyInto(*out)
		}
	}
	if in.PsRootPort != nil {
		in, out := &in.PsRootPort, &out.PsRootPort
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	if in.PsVerbose != nil {
		in, out := &in.PsVerbose, &out.PsVerbose
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MXReplicaSpec.
func (in *MXReplicaSpec) DeepCopy() *MXReplicaSpec {
	if in == nil {
		return nil
	}
	out := new(MXReplicaSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MXReplicaSpecs) DeepCopyInto(out *MXReplicaSpecs) {
	*out = *in
	if in.Scheduler != nil {
		in, out := &in.Scheduler, &out.Scheduler
		if *in == nil {
			*out = nil
		} else {
			*out = new(MXReplicaSpec)
			(*in).DeepCopyInto(*out)
		}
	}
	if in.Server != nil {
		in, out := &in.Server, &out.Server
		if *in == nil {
			*out = nil
		} else {
			*out = new(MXReplicaSpec)
			(*in).DeepCopyInto(*out)
		}
	}
	if in.Worker != nil {
		in, out := &in.Worker, &out.Worker
		if *in == nil {
			*out = nil
		} else {
			*out = new(MXReplicaSpec)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MXReplicaSpecs.
func (in *MXReplicaSpecs) DeepCopy() *MXReplicaSpecs {
	if in == nil {
		return nil
	}
	out := new(MXReplicaSpecs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MXReplicaStatus) DeepCopyInto(out *MXReplicaStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MXReplicaStatus.
func (in *MXReplicaStatus) DeepCopy() *MXReplicaStatus {
	if in == nil {
		return nil
	}
	out := new(MXReplicaStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MXReplicaStatuses) DeepCopyInto(out *MXReplicaStatuses) {
	*out = *in
	if in.Scheduler != nil {
		in, out := &in.Scheduler, &out.Scheduler
		if *in == nil {
			*out = nil
		} else {
			*out = new(MXReplicaStatus)
			**out = **in
		}
	}
	if in.Server != nil {
		in, out := &in.Server, &out.Server
		if *in == nil {
			*out = nil
		} else {
			*out = new(MXReplicaStatus)
			**out = **in
		}
	}
	if in.Worker != nil {
		in, out := &in.Worker, &out.Worker
		if *in == nil {
			*out = nil
		} else {
			*out = new(MXReplicaStatus)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MXReplicaStatuses.
func (in *MXReplicaStatuses) DeepCopy() *MXReplicaStatuses {
	if in == nil {
		return nil
	}
	out := new(MXReplicaStatuses)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Notification) DeepCopyInto(out *Notification) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.Request != nil {
		in, out := &in.Request, &out.Request
		if *in == nil {
			*out = nil
		} else {
			*out = new(NotificationRequest)
			(*in).DeepCopyInto(*out)
		}
	}
	if in.Response != nil {
		in, out := &in.Response, &out.Response
		if *in == nil {
			*out = nil
		} else {
			*out = new(NotificationResponse)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Notification.
func (in *Notification) DeepCopy() *Notification {
	if in == nil {
		return nil
	}
	out := new(Notification)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Notification) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotificationRequest) DeepCopyInto(out *NotificationRequest) {
	*out = *in
	out.Kind = in.Kind
	in.Status.DeepCopyInto(&out.Status)
	if in.DeletionTimestamp != nil {
		in, out := &in.DeletionTimestamp, &out.DeletionTimestamp
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.Time)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotificationRequest.
func (in *NotificationRequest) DeepCopy() *NotificationRequest {
	if in == nil {
		return nil
	}
	out := new(NotificationRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotificationResponse) DeepCopyInto(out *NotificationResponse) {
	*out = *in
	if in.Result != nil {
		in, out := &in.Result, &out.Result
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.Status)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotificationResponse.
func (in *NotificationResponse) DeepCopy() *NotificationResponse {
	if in == nil {
		return nil
	}
	out := new(NotificationResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Webhook) DeepCopyInto(out *Webhook) {
	*out = *in
	in.ClientConfig.DeepCopyInto(&out.ClientConfig)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Webhook.
func (in *Webhook) DeepCopy() *Webhook {
	if in == nil {
		return nil
	}
	out := new(Webhook)
	in.DeepCopyInto(out)
	return out
}
