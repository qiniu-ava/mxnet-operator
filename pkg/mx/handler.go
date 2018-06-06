package mx

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/qiniu-ava/mxnet-operator/pkg/apis/qiniu/v1alpha1"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/controller"
)

const (
	mxLabelReplicaType = "mx-replica-type"
	mxLabelJobName     = "mxjob-name"
	mxLabelGroupName   = "group-name"
)

var (
	controllerName = "mxnet-operator"

	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

// Handler is a MXJob controller event processor.
type Handler struct {
	// podControl is used to add or delete pods.
	podControl controller.PodControlInterface

	// A TTLCache of pod/services creates/deletes each mxjob expects to see
	// We use MXJob namespace/name + MXReplicaType + pods/services as an expectation key,
	// For example, there is a MXJob with namespace "mx-operator" and name "mxjob-abc":
	// {
	//     "scheduler": {
	//         "replicas": 1,
	//     },
	//     "server": {
	//         "replicas": 3,
	//     }
	//     "worker": {
	//         "replicas": 3,
	//     }
	// }
	// We will create 4 expectations:
	// - "mx-operator/mxjob-abc/scheduler/services", expects 1 adds.
	// - "mx-operator/mxjob-abc/scheduler/pods", expects 1 adds.
	// - "mx-operator/mxjob-abc/server/pods", expects 3 adds.
	// - "mx-operator/mxjob-abc/worker/pods", expects 3 adds.
	expectations controller.ControllerExpectationsInterface

	recorder record.EventRecorder
}

// NewHandler creates a Handler.
func NewHandler() *Handler {
	kubeClientSet, _ := mustNewKubeClient()

	logrus.Debug("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logrus.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerName})

	realPodControl := controller.RealPodControl{
		KubeClient: kubeClientSet,
		Recorder:   recorder,
	}

	return &Handler{
		podControl:   realPodControl,
		expectations: controller.NewControllerExpectations(),
		recorder:     recorder,
	}
}

// Sync synchronizes mxjobs.
func (h *Handler) Sync(ctx context.Context, mxjob *v1alpha1.MXJob, deleted bool) error {
	mxjobKey, err := KeyFunc(mxjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for mxjob object %#v: %v", mxjob, err))
		return err
	}

	startTime := time.Now()
	defer func() {
		logrus.Infof("finished syncing mxjob %s (%v)", mxjobKey, time.Since(startTime))
	}()

	if deleted {
		logrus.Infof("mxjob %s has been deleted", mxjobKey)
		return nil
	}

	// expectations are not used since getPodsForMXJob returns live Pods
	// or sync or not will be decided by h.satisfiedExpectations(mxjob) if list cache is enabled
	mxjobNeedsSync := true

	var reconcileMXJobsErr error
	if mxjobNeedsSync && mxjob.DeletionTimestamp == nil {
		reconcileMXJobsErr = h.reconcileMXJobs(mxjob)
	}

	return reconcileMXJobsErr
}

func (h *Handler) reconcileMXJobs(mxjob *v1alpha1.MXJob) error {
	logrus.Infof("Reconcile MXJobs %s", mxjob.Name)

	pods, err := h.getPodsForMXJob(mxjob)

	if err != nil {
		logrus.Infof("getPodsForMXJob error %v", err)
		return err
	}

	oldStatus := mxjob.Status.DeepCopy()

	// reconcile active pods/services with replicas.
	if mxjob.Spec.MXReplicaSpecs.Scheduler != nil {
		err = h.reconcilePods(mxjob, pods, v1alpha1.MXReplicaTypeScheduler, mxjob.Spec.MXReplicaSpecs.Scheduler)
		if err != nil {
			logrus.Infof("reconcilePods error %v", err)
			return err
		}
	}
	if mxjob.Spec.MXReplicaSpecs.Server != nil {
		err = h.reconcilePods(mxjob, pods, v1alpha1.MXReplicaTypeServer, mxjob.Spec.MXReplicaSpecs.Server)
		if err != nil {
			logrus.Infof("reconcilePods error %v", err)
			return err
		}
	}
	if mxjob.Spec.MXReplicaSpecs.Worker != nil {
		err = h.reconcilePods(mxjob, pods, v1alpha1.MXReplicaTypeWorker, mxjob.Spec.MXReplicaSpecs.Worker)
		if err != nil {
			logrus.Infof("reconcilePods error %v", err)
			return err
		}
	}

	if !reflect.DeepEqual(*oldStatus, mxjob.Status) {
		loggerForMXJob(mxjob).Infof("updating status %+v", mxjob.Status)
		if err := sdk.Update(mxjob); err != nil {
			return err
		}
	} else {
		logrus.Debug("no status change, skip update")
	}

	return nil
}

// getPodsForMXJob returns the set of pods that this mxjob should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned are live Pods from apiserver,
// because sdk do not provide the capability to list from local cache for now.
func (h *Handler) getPodsForMXJob(mxjob *v1alpha1.MXJob) ([]*v1.Pod, error) {
	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: genLabels(mxjob, ""),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	pods := v1.PodList{TypeMeta: metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}}
	err = sdk.List(mxjob.Namespace, &pods, sdk.WithListOptions(&metav1.ListOptions{LabelSelector: labels.Everything().String()}))
	if err != nil {
		return nil, err
	}

	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing Pods (see #42639).
	canAdoptFunc := RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh := v1alpha1.MXJob{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.SchemeGroupVersionKind.Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      mxjob.Name,
				Namespace: mxjob.Namespace,
			},
		}
		err := sdk.Get(&fresh)
		if err != nil {
			return nil, err
		}
		if fresh.UID != mxjob.UID {
			return nil, fmt.Errorf("original MXJob %v/%v is gone: got uid %v, wanted %v", mxjob.Namespace, mxjob.Name, fresh.UID, mxjob.UID)
		}
		return &fresh, nil
	})
	cm := controller.NewPodControllerRefManager(h.podControl, mxjob, selector, v1alpha1.SchemeGroupVersionKind, canAdoptFunc)
	return cm.ClaimPods(podList(pods))
}

// reconcilePods checks and updates pods for each given MXReplicaSpec.
// It will requeue the mxjob in case of an error while creating/deleting pods.
func (h *Handler) reconcilePods(mxjob *v1alpha1.MXJob, pods []*v1.Pod, rtype v1alpha1.MXReplicaType, spec *v1alpha1.MXReplicaSpec) error {

	// Get all pods for the type rt.
	pods = filterPodsForMXReplicaType(pods, rtype)
	loggerForReplica(mxjob, rtype).Infof("reconcile %d pods", len(pods))
	replicas := int(*spec.Replicas)
	diff := replicas - len(pods)

	initializeMXReplicaStatuses(mxjob, rtype)

	if diff > 0 {
		for i := 0; i < diff; i++ {
			loggerForReplica(mxjob, rtype).Infof("creating new pod (%d/%d): %s", i+1, diff, rtype)
			err := h.createNewPod(mxjob, rtype, spec)
			if err != nil {
				return err
			}
		}
	} else if diff < 0 {
		loggerForReplica(mxjob, rtype).Infof("need to delete number of pod: %d", -diff)
		// TODO:
	} else { // sync status
		activePods := controller.FilterActivePods(pods)
		active := int32(len(activePods))
		succeeded, failed := getStatus(pods)
		status := v1alpha1.MXReplicaStatus{Active: active, Succeeded: succeeded, Failed: failed}
		loggerForReplica(mxjob, rtype).Infof("sync status %+v", status)
		updateMXReplicaStatuses(mxjob, rtype, &status)
	}

	return h.updateMXJobStatus(mxjob, rtype, replicas)
}

// createNewPod creates a new pod for the given replica type.
func (h *Handler) createNewPod(mxjob *v1alpha1.MXJob, rt v1alpha1.MXReplicaType, spec *v1alpha1.MXReplicaSpec) error {
	podTemplate := spec.Template.DeepCopy()

	// Set type for the pod.
	labels := genLabels(mxjob, rt)
	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}
	for key, value := range labels {
		podTemplate.Labels[key] = value
	}

	// Generate and add mxnet environment variable.
	mxEnv := genMXConfigEnv(mxjob, rt)
	for i := range podTemplate.Spec.Containers {
		for k, v := range mxEnv {
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, v1.EnvVar{Name: k, Value: v})
		}
	}

	err := h.podControl.CreatePodsWithControllerRef(mxjob.Namespace, podTemplate, mxjob, metav1.NewControllerRef(mxjob, v1alpha1.SchemeGroupVersionKind))
	if err != nil && errors.IsTimeout(err) {
		// Pod is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the pod keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// pod when the expectation expires.
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

// updateMXJobStatus updates the status of the mxjob.
func (h *Handler) updateMXJobStatus(mxjob *v1alpha1.MXJob, rtype v1alpha1.MXReplicaType, replicas int) error {
	// Expect to have `replicas - succeeded` pods alive.
	status := getMXReplicaStatus(mxjob, rtype)
	expected := replicas - int(status.Succeeded)
	running := int(status.Active)
	failed := int(status.Failed)

	if rtype == v1alpha1.MXReplicaTypeWorker {
		// All workers are running, set StartTime.
		if running == replicas {
			now := metav1.Now()
			mxjob.Status.StartTime = &now
		}

		// Some workers are still running, leave a running condition.
		if running > 0 {
			msg := fmt.Sprintf("MXJob %s is running.", mxjob.Name)
			err := h.updateMXJobConditions(mxjob, v1alpha1.MXJobRunning, v1alpha1.MXJobReasonRunning, msg)
			if err != nil {
				loggerForMXJob(mxjob).Infof("Append mxjob condition error: %v", err)
				return err
			}
		}

		// All workers are succeeded, leave a succeeded condition.
		if expected == 0 {
			msg := fmt.Sprintf("MXJob %s is successfully completed.", mxjob.Name)
			now := metav1.Now()
			mxjob.Status.CompletionTime = &now
			err := h.updateMXJobConditions(mxjob, v1alpha1.MXJobSucceeded, v1alpha1.MXJobReasonSucceeded, msg)
			if err != nil {
				loggerForMXJob(mxjob).Infof("Append mxjob condition error: %v", err)
				return err
			}
		}
	}

	// Some workers, servers or schedulers are failed , leave a failed condition.
	if failed > 0 {
		msg := fmt.Sprintf("MXJob %s is failed.", mxjob.Name)
		err := h.updateMXJobConditions(mxjob, v1alpha1.MXJobFailed, v1alpha1.MXJobReasonFailed, msg)
		if err != nil {
			loggerForMXJob(mxjob).Infof("Append mxjob condition error: %v", err)
			return err
		}
	}
	return nil
}

func (h *Handler) updateMXJobConditions(mxjob *v1alpha1.MXJob, conditionType v1alpha1.MXJobConditionType, reason v1alpha1.MXJobReason, message string) error {
	condition := newCondition(conditionType, reason, message)
	setCondition(&mxjob.Status, condition)
	return nil
}

// satisfiedExpectations returns true if the required adds/dels for the given mxjob have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller manager.
func (h *Handler) satisfiedExpectations(mxjob *v1alpha1.MXJob) bool {
	satisfied := false
	mxjobKey, err := KeyFunc(mxjob)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for mxjob object %#v: %v", mxjob, err))
		return false
	}

	if mxjob.Spec.MXReplicaSpecs.Scheduler != nil {
		// Check the expectations of the pods.
		expectationPodsKey := genExpectationPodsKey(mxjobKey, v1alpha1.MXReplicaTypeScheduler)
		satisfied = satisfied || h.expectations.SatisfiedExpectations(expectationPodsKey)
	}

	if mxjob.Spec.MXReplicaSpecs.Server != nil {
		// Check the expectations of the pods.
		expectationPodsKey := genExpectationPodsKey(mxjobKey, v1alpha1.MXReplicaTypeServer)
		satisfied = satisfied || h.expectations.SatisfiedExpectations(expectationPodsKey)
	}

	if mxjob.Spec.MXReplicaSpecs.Worker != nil {
		// Check the expectations of the pods.
		expectationPodsKey := genExpectationPodsKey(mxjobKey, v1alpha1.MXReplicaTypeWorker)
		satisfied = satisfied || h.expectations.SatisfiedExpectations(expectationPodsKey)
	}
	return satisfied
}

func podList(pods v1.PodList) (ret []*v1.Pod) {
	for _, p := range pods.Items {
		pod := p
		ret = append(ret, &pod)
	}
	return
}

func loggerForMXJob(mxjob *v1alpha1.MXJob) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		// We use job to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"job": mxjob.ObjectMeta.Namespace + "/" + mxjob.ObjectMeta.Name,
		"uid": mxjob.ObjectMeta.UID,
	})
}

func loggerForReplica(mxjob *v1alpha1.MXJob, rtype v1alpha1.MXReplicaType) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		// We use job to match the key used in controller.go
		// In controller.go we log the key used with the workqueue.
		"job":          mxjob.ObjectMeta.Namespace + "/" + mxjob.ObjectMeta.Name,
		"uid":          mxjob.ObjectMeta.UID,
		"replica-type": string(rtype),
	})
}

// filterPodsForMXReplicaType returns pods belong to a MXReplicaType.
func filterPodsForMXReplicaType(pods []*v1.Pod, rtype v1alpha1.MXReplicaType) []*v1.Pod {
	var result []*v1.Pod

	mxReplicaSelector := &metav1.LabelSelector{
		MatchLabels: make(map[string]string),
	}

	mxReplicaSelector.MatchLabels[mxLabelReplicaType] = string(rtype)

	for _, pod := range pods {
		selector, _ := metav1.LabelSelectorAsSelector(mxReplicaSelector)
		if !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		result = append(result, pod)
	}
	return result
}

// initializeMXReplicaStatuses initializes the MXReplicaStatus for replica.
func initializeMXReplicaStatuses(mxjob *v1alpha1.MXJob, rtype v1alpha1.MXReplicaType) {
	switch rtype {
	case v1alpha1.MXReplicaTypeScheduler:
		mxjob.Status.ReplicaStatuses.Scheduler = &v1alpha1.MXReplicaStatus{}
	case v1alpha1.MXReplicaTypeServer:
		mxjob.Status.ReplicaStatuses.Server = &v1alpha1.MXReplicaStatus{}
	case v1alpha1.MXReplicaTypeWorker:
		mxjob.Status.ReplicaStatuses.Worker = &v1alpha1.MXReplicaStatus{}
	}
	return
}

// getMXReplicaStatuses retrieves the MXReplicaStatus for replica.
func getMXReplicaStatus(mxjob *v1alpha1.MXJob, rtype v1alpha1.MXReplicaType) *v1alpha1.MXReplicaStatus {
	switch rtype {
	case v1alpha1.MXReplicaTypeScheduler:
		return mxjob.Status.ReplicaStatuses.Scheduler
	case v1alpha1.MXReplicaTypeServer:
		return mxjob.Status.ReplicaStatuses.Server
	case v1alpha1.MXReplicaTypeWorker:
		return mxjob.Status.ReplicaStatuses.Worker
	}
	return nil
}

// updateMXJobReplicaStatuses updates the MXJobReplicaStatuses according to the pod.
// pod phase: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-phase
func updateMXReplicaStatuses(mxjob *v1alpha1.MXJob, rtype v1alpha1.MXReplicaType, replicaStatus *v1alpha1.MXReplicaStatus) {
	switch rtype {
	case v1alpha1.MXReplicaTypeScheduler:
		mxjob.Status.ReplicaStatuses.Scheduler = replicaStatus
	case v1alpha1.MXReplicaTypeServer:
		mxjob.Status.ReplicaStatuses.Server = replicaStatus
	case v1alpha1.MXReplicaTypeWorker:
		mxjob.Status.ReplicaStatuses.Worker = replicaStatus
	}
	return
}

// newCondition creates a new mxjob condition.
func newCondition(conditionType v1alpha1.MXJobConditionType, reason v1alpha1.MXJobReason, message string) v1alpha1.MXJobCondition {
	return v1alpha1.MXJobCondition{
		Type:               conditionType,
		Status:             v1.ConditionTrue,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status v1alpha1.MXJobStatus, condType v1alpha1.MXJobConditionType) *v1alpha1.MXJobCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// setCondition updates the mxjob to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *v1alpha1.MXJobStatus, condition v1alpha1.MXJobCondition) {
	currentCond := getCondition(*status, condition.Type)

	// Do nothing if condition doesn't change
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	// Append the updated condition to the
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// removeCondition removes the mxjob condition with the provided type.
func removementCondition(status *v1alpha1.MXJobStatus, condType v1alpha1.MXJobConditionType) {
	status.Conditions = filterOutCondition(status.Conditions, condType)
}

// filterOutCondition returns a new slice of mxjob conditions without conditions with the provided type.
func filterOutCondition(conditions []v1alpha1.MXJobCondition, condType v1alpha1.MXJobConditionType) []v1alpha1.MXJobCondition {
	var newConditions []v1alpha1.MXJobCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

func genExpectationPodsKey(mxjobKey string, replicaType v1alpha1.MXReplicaType) string {
	return mxjobKey + "/" + strings.ToLower(string(replicaType)) + "/pods"
}

func genMXConfigEnv(mxjob *v1alpha1.MXJob, rtype v1alpha1.MXReplicaType) (ret map[string]string) {
	ret = make(map[string]string)
	// TODO:
	switch rtype {
	case v1alpha1.MXReplicaTypeScheduler:
	case v1alpha1.MXReplicaTypeServer:
	case v1alpha1.MXReplicaTypeWorker:
	}
	return
}

// RecheckDeletionTimestamp returns a CanAdopt() function to recheck deletion.
//
// The CanAdopt() function calls getObject() to fetch the latest value,
// and denies adoption attempts if that object has a non-nil DeletionTimestamp.
func RecheckDeletionTimestamp(getObject func() (metav1.Object, error)) func() error {
	return func() error {
		obj, err := getObject()
		if err != nil {
			return fmt.Errorf("can't recheck DeletionTimestamp: %v", err)
		}
		if obj.GetDeletionTimestamp() != nil {
			return fmt.Errorf("%v/%v has just been deleted at %v", obj.GetNamespace(), obj.GetName(), obj.GetDeletionTimestamp())
		}
		return nil
	}
}

func genLabels(mxjob *v1alpha1.MXJob, rt v1alpha1.MXReplicaType) map[string]string {
	l := map[string]string{
		mxLabelGroupName: v1alpha1.SchemeGroupVersion.Group,
		mxLabelJobName:   mxjob.Name,
	}
	if rt != "" {
		l[mxLabelReplicaType] = string(rt)
	}
	return l
}

// getStatus returns no of succeeded and failed pods running a job
func getStatus(pods []*v1.Pod) (succeeded, failed int32) {
	succeeded = int32(filterPods(pods, v1.PodSucceeded))
	failed = int32(filterPods(pods, v1.PodFailed))
	return
}

// filterPods returns pods based on their phase.
func filterPods(pods []*v1.Pod, phase v1.PodPhase) int {
	result := 0
	for i := range pods {
		if phase == pods[i].Status.Phase {
			result++
		}
	}
	return result
}
