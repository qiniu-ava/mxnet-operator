package mx

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/qiniu-ava/mxnet-operator/pkg/apis/qiniu/v1alpha1"
	scontroller "github.com/qiniu-ava/mxnet-operator/pkg/controller"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	webhook "k8s.io/apiserver/pkg/admission/plugin/webhook/config"
	webhookerrors "k8s.io/apiserver/pkg/admission/plugin/webhook/errors"
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
	controllerName          = "mxnet-operator"
	defaultPsRootPort int32 = 9091

	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

// Handler is a MXJob controller event processor.
type Handler struct {
	// podControl is used to add or delete pods.
	podControl controller.PodControlInterface

	// serviceControl is used to add or delete services.
	serviceControl scontroller.ServiceControlInterface

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

	webhookClientManager ClientManager
}

// NewHandler creates a Handler.
func NewHandler() (*Handler, error) {
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

	realServiceControl := scontroller.RealServiceControl{
		KubeClient: kubeClientSet,
		Recorder:   eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerName}),
	}

	cm, err := NewClientManager()
	if err != nil {
		return nil, err
	}
	authInfoResolver, err := webhook.NewDefaultAuthenticationInfoResolver("")
	if err != nil {
		return nil, err
	}
	cm.SetAuthenticationInfoResolver(authInfoResolver)
	cm.SetServiceResolver(webhook.NewDefaultServiceResolver())
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	cm.SetNegotiatedSerializer(serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{
		Serializer: serializer.NewCodecFactory(scheme).LegacyCodec(v1alpha1.SchemeGroupVersion),
	}))

	return &Handler{
		podControl:           realPodControl,
		serviceControl:       realServiceControl,
		expectations:         controller.NewControllerExpectations(),
		recorder:             recorder,
		webhookClientManager: cm,
	}, nil
}

// Sync synchronizes mxjobs.
func (h *Handler) Sync(ctx context.Context, mxjob *v1alpha1.MXJob, deleted bool) error {
	startTime := time.Now()
	defer func() {
		loggerForMXJob(mxjob).Infof("finished syncing mxjob (%v)", time.Since(startTime))
	}()

	if deleted {
		loggerForMXJob(mxjob).Infof("mxjob has been deleted")
		return nil
	}

	// add created condition first met
	if !checkCondition(mxjob.Status, v1alpha1.MXJobCreated) {
		msg := fmt.Sprintf("MXJob %s is created.", mxjob.Name)
		h.updateMXJobConditions(mxjob, v1alpha1.MXJobCreated, v1alpha1.MXJobReasonCreated, msg)
	}

	// if job was finished previously, we don't want to redo the termination
	if isJobFinished(mxjob.Status) {
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
	loggerForMXJob(mxjob).Infof("Reconcile MXJobs %s", mxjob.Name)

	pods, err := h.getPodsForMXJob(mxjob)
	if err != nil {
		loggerForMXJob(mxjob).Infof("getPodsForMXJob error %v", err)
		return err
	}

	oldStatus := mxjob.Status.DeepCopy()

	// reconcile active pods/services with replicas.
	if mxjob.Spec.MXReplicaSpecs.Scheduler != nil {
		err = h.reconcilePods(mxjob, pods, v1alpha1.MXReplicaTypeScheduler, mxjob.Spec.MXReplicaSpecs.Scheduler)
		if err != nil {
			loggerForReplica(mxjob, v1alpha1.MXReplicaTypeScheduler).Infof("reconcilePods error %v", err)
			return err
		}

		services, err := h.getServicesForMXJob(mxjob)
		if err != nil {
			loggerForMXJob(mxjob).Infof("getServicesForMXJob error %v", err)
			return err
		}

		err = h.reconcileServices(mxjob, services, v1alpha1.MXReplicaTypeScheduler, mxjob.Spec.MXReplicaSpecs.Scheduler)
		if err != nil {
			loggerForReplica(mxjob, v1alpha1.MXReplicaTypeScheduler).Infof("reconcileServices error %v", err)
			return err
		}
	}
	if mxjob.Spec.MXReplicaSpecs.Server != nil {
		err = h.reconcilePods(mxjob, pods, v1alpha1.MXReplicaTypeServer, mxjob.Spec.MXReplicaSpecs.Server)
		if err != nil {
			loggerForReplica(mxjob, v1alpha1.MXReplicaTypeServer).Infof("reconcilePods error %v", err)
			return err
		}
	}
	if mxjob.Spec.MXReplicaSpecs.Worker != nil {
		err = h.reconcilePods(mxjob, pods, v1alpha1.MXReplicaTypeWorker, mxjob.Spec.MXReplicaSpecs.Worker)
		if err != nil {
			loggerForReplica(mxjob, v1alpha1.MXReplicaTypeWorker).Infof("reconcilePods error %v", err)
			return err
		}
	}

	// add initialized condition after all sub-resource created
	if !checkCondition(mxjob.Status, v1alpha1.MXJobInitialized) {
		msg := fmt.Sprintf("MXJob %s is initialized.", mxjob.Name)
		h.updateMXJobConditions(mxjob, v1alpha1.MXJobInitialized, v1alpha1.MXJobReasonInitialized, msg)

		h.callHookAsync(context.TODO(), mxjob, true)
	}

	// terminate job when job is finished
	if isJobFinished(mxjob.Status) {
		if err := h.terminateMXJob(mxjob, pods); err != nil {
			loggerForMXJob(mxjob).Infof("terminate mxjob error: %v", err)
			return err
		}

		h.callHookAsync(context.TODO(), mxjob, false)
	}

	if !reflect.DeepEqual(*oldStatus, mxjob.Status) {
		loggerForMXJob(mxjob).Infof("updating status %+v", mxjob.Status)
		if err := sdk.Update(mxjob); err != nil {
			loggerForMXJob(mxjob).Infof("update status error: %v", err)
			return err
		}
	} else {
		loggerForMXJob(mxjob).Debug("no status change, skip update")
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

// getServicesForMXJob returns the set of services that this mxjob should manage.
// It also reconciles ControllerRef by adopting/orphaning.
// Note that the returned are live services from apiserver,
// because sdk do not provide the capability to list from local cache for now.
func (h *Handler) getServicesForMXJob(mxjob *v1alpha1.MXJob) ([]*v1.Service, error) {
	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: genLabels(mxjob, ""),
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	// List all services to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	services := v1.ServiceList{TypeMeta: metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}}
	err = sdk.List(mxjob.Namespace, &services, sdk.WithListOptions(&metav1.ListOptions{LabelSelector: labels.Everything().String()}))
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
	cm := scontroller.NewServiceControllerRefManager(h.serviceControl, mxjob, selector, v1alpha1.SchemeGroupVersionKind, canAdoptFunc)
	return cm.ClaimServices(serviceList(services))
}

// reconcilePods checks and updates pods for each given MXReplicaSpec.
// It will requeue the mxjob in case of an error while creating/deleting pods.
func (h *Handler) reconcilePods(mxjob *v1alpha1.MXJob, pods []*v1.Pod, rtype v1alpha1.MXReplicaType, spec *v1alpha1.MXReplicaSpec) error {

	// Get all pods for the type rt.
	pods = filterPodsForMXReplicaType(pods, rtype)
	loggerForReplica(mxjob, rtype).Infof("reconcile %d pods", len(pods))
	replicas := *spec.Replicas
	diff := replicas - int32(len(pods))

	activePods := controller.FilterActivePods(pods)
	active := int32(len(activePods))

	initializeMXReplicaStatuses(mxjob, rtype)
	status := getMXReplicaStatus(mxjob, rtype)
	gone := status.Gone

	if diff > 0 {
		if checkCondition(mxjob.Status, v1alpha1.MXJobInitialized) {
			// after mxjob have been initialized, if any pod is deleted, will term the entire mxjob
			gone = int32(diff)
		} else {
			var activeLock sync.Mutex
			active += diff

			wait := sync.WaitGroup{}
			wait.Add(int(diff))
			errCh := make(chan error, diff)
			loggerForReplica(mxjob, rtype).Infof("creating %d new pod: %s", diff, rtype)

			for i := int32(0); i < diff; i++ {
				go func() {
					defer wait.Done()
					err := h.createNewPod(mxjob, rtype, spec)
					if err != nil {
						loggerForReplica(mxjob, rtype).Infof("failed to new pod of %s", rtype)
						activeLock.Lock()
						active--
						activeLock.Unlock()
						errCh <- err
					}
				}()
			}
			wait.Wait()

			select {
			case err := <-errCh:
				return err
			default:
			}
		}
	} else if diff < 0 {
		loggerForReplica(mxjob, rtype).Infof("need to delete number of pod: %d", -diff)
		// TODO:
	}

	succeeded, failed := getStatus(pods)
	st := v1alpha1.MXReplicaStatus{Active: active, Succeeded: succeeded, Failed: failed, Gone: gone}
	loggerForReplica(mxjob, rtype).Infof("sync status %+v", st)
	updateMXReplicaStatuses(mxjob, rtype, &st)

	return h.updateMXJobStatus(mxjob, rtype, replicas)
}

// reconcileServices checks and updates services for each given MXReplicaSpec.
func (h *Handler) reconcileServices(mxjob *v1alpha1.MXJob, services []*v1.Service, rtype v1alpha1.MXReplicaType, spec *v1alpha1.MXReplicaSpec) error {
	if rtype != v1alpha1.MXReplicaTypeScheduler {
		loggerForReplica(mxjob, rtype).Infof("reconcile sevices for scheduler only")
		return nil
	}

	loggerForReplica(mxjob, rtype).Infof("reconcile %d services", len(services))
	num := len(services)

	if num == 0 {
		// create only one service for scheduler
		loggerForReplica(mxjob, rtype).Infof("creating new service: %s", rtype)
		labels := genLabels(mxjob, v1alpha1.MXReplicaTypeScheduler)

		port := spec.PsRootPort
		if port == nil {
			port = &defaultPsRootPort
		}
		service := &v1.Service{
			Spec: v1.ServiceSpec{
				Selector: labels,
				Ports: []v1.ServicePort{
					{
						Name: serviceName(mxjob, rtype),
						Port: *port,
					},
				},
			},
		}
		service.Name = serviceName(mxjob, rtype)
		service.Labels = labels

		err := h.serviceControl.CreateServicesWithControllerRef(mxjob.Namespace, service, mxjob, metav1.NewControllerRef(mxjob, v1alpha1.SchemeGroupVersionKind))
		if err != nil && errors.IsTimeout(err) {
			return nil
		} else if err != nil {
			loggerForReplica(mxjob, rtype).Infof("failed to new service of %s", rtype)
			return err
		}
	} else if num > 1 {
		loggerForReplica(mxjob, rtype).Infof("need to delete number of service: %d", num-1)
		// TODO:
	}
	return nil
}

// terminateMXJob deletes all active pods of mxjob and leave others.
func (h *Handler) terminateMXJob(mxjob *v1alpha1.MXJob, pods []*v1.Pod) error {
	loggerForMXJob(mxjob).Infof("terminating mxjob")
	activePods := controller.FilterActivePods(pods)
	num := len(activePods)
	if num == 0 {
		return nil
	}

	wait := sync.WaitGroup{}
	wait.Add(num)
	errCh := make(chan error, num)
	loggerForMXJob(mxjob).Infof("deleting %d active pod", num)

	var statusLock sync.Mutex
	for _, p := range activePods {
		go func(pod *v1.Pod) {
			defer wait.Done()
			err := h.podControl.DeletePod(pod.Namespace, pod.Name, pod)
			if err != nil {
				loggerForMXJob(mxjob).Infof("failed to delete pod: %s/%s", pod.Namespace, pod.Name)
				errCh <- err
			}

			// update replica status
			statusLock.Lock()
			switch v1alpha1.MXReplicaType(pod.Labels[mxLabelReplicaType]) {
			case v1alpha1.MXReplicaTypeScheduler:
				mxjob.Status.ReplicaStatuses.Scheduler.Gone++
				mxjob.Status.ReplicaStatuses.Scheduler.Active--
			case v1alpha1.MXReplicaTypeServer:
				mxjob.Status.ReplicaStatuses.Server.Gone++
				mxjob.Status.ReplicaStatuses.Server.Active--
			case v1alpha1.MXReplicaTypeWorker:
				mxjob.Status.ReplicaStatuses.Worker.Gone++
				mxjob.Status.ReplicaStatuses.Worker.Active--
			}
			statusLock.Unlock()
		}(p)
	}
	wait.Wait()

	select {
	case err := <-errCh:
		return err
	default:
	}
	return nil
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
func (h *Handler) updateMXJobStatus(mxjob *v1alpha1.MXJob, rtype v1alpha1.MXReplicaType, replicas int32) error {
	// Expect to have `replicas - succeeded` pods alive.
	status := getMXReplicaStatus(mxjob, rtype)
	expected := replicas - status.Succeeded
	running := status.Active
	failed := status.Failed
	gone := status.Gone

	if rtype == v1alpha1.MXReplicaTypeWorker {
		// All workers are running, set StartTime.
		if running == replicas {
			now := metav1.Now()
			mxjob.Status.StartTime = &now
		}

		// Some workers are still running, leave a running condition.
		if running > 0 {
			msg := fmt.Sprintf("MXJob %s is running.", mxjob.Name)
			h.updateMXJobConditions(mxjob, v1alpha1.MXJobRunning, v1alpha1.MXJobReasonRunning, msg)
		}

		// All workers are succeeded, leave a succeeded condition.
		if expected == 0 {
			msg := fmt.Sprintf("MXJob %s is successfully completed.", mxjob.Name)
			now := metav1.Now()
			mxjob.Status.CompletionTime = &now
			h.updateMXJobConditions(mxjob, v1alpha1.MXJobSucceeded, v1alpha1.MXJobReasonSucceeded, msg)
		}
	}

	// Some workers, servers or schedulers are failed or gone, leave a failed condition.
	if failed > 0 || gone > 0 {
		// ignore if already succeeded
		if !checkCondition(mxjob.Status, v1alpha1.MXJobSucceeded) {
			msg := fmt.Sprintf("MXJob %s is failed.", mxjob.Name)
			h.updateMXJobConditions(mxjob, v1alpha1.MXJobFailed, v1alpha1.MXJobReasonFailed, msg)
		}
	}
	return nil
}

func (h *Handler) updateMXJobConditions(mxjob *v1alpha1.MXJob, conditionType v1alpha1.MXJobConditionType, reason v1alpha1.MXJobReason, message string) {
	condition := newCondition(conditionType, reason, message)
	setCondition(&mxjob.Status, condition)
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

func (h *Handler) callHookAsync(ctx context.Context, mxjob *v1alpha1.MXJob, startOrFinish bool) {
	job := mxjob.DeepCopy()
	var wh *v1alpha1.Webhook
	if startOrFinish {
		wh = job.Spec.StartWebhook
	} else {
		wh = job.Spec.FinishWebhook
	}

	if wh == nil {
		return
	}

	backoff := wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   1.5,
		Jitter:   0.2,
		Steps:    5,
	}

	go func() {
		err := wait.ExponentialBackoff(backoff, func() (bool, error) {
			err := h.callHook(ctx, wh, job)
			if err == nil {
				return true, nil
			}

			if callErr, ok := err.(*webhookerrors.ErrCallingWebhook); ok {
				loggerForMXJob(job).Warningf("failed calling webhook, failing closed %v: %v", wh.Name, callErr)
				return false, nil // retry
			}

			loggerForMXJob(job).Warningf("failed handling by webhook %v: %v", wh.Name, err)
			return false, err
		})

		loggerForMXJob(job).Infof("finished calling webhook %v, for mxjob start %v: %v", wh.Name, startOrFinish, err)
	}()
}

func (h *Handler) callHook(ctx context.Context, wh *v1alpha1.Webhook, mxjob *v1alpha1.MXJob) error {
	request := v1alpha1.Notification{
		Request: &v1alpha1.NotificationRequest{
			UID: uuid.NewUUID(),
			Kind: metav1.GroupVersionKind{
				Group:   v1alpha1.SchemeGroupVersionKind.Group,
				Version: v1alpha1.SchemeGroupVersionKind.Version,
				Kind:    v1alpha1.SchemeGroupVersionKind.Kind,
			},
			Name:      mxjob.Name,
			Namespace: mxjob.Namespace,
			Status:    mxjob.Status,
		},
	}
	// TODO: webhookerrors contains literal "adminssion", need update
	client, err := h.webhookClientManager.HookClient(wh)
	if err != nil {
		return &webhookerrors.ErrCallingWebhook{WebhookName: wh.Name, Reason: err}
	}
	response := &v1alpha1.Notification{}
	if err := client.Post().Context(ctx).Body(&request).Do().Into(response); err != nil {
		return &webhookerrors.ErrCallingWebhook{WebhookName: wh.Name, Reason: err}
	}

	if response.Response == nil {
		return &webhookerrors.ErrCallingWebhook{WebhookName: wh.Name, Reason: fmt.Errorf("Webhook response was absent")}
	}

	if response.Response.Result.Status == metav1.StatusSuccess {
		return nil
	}
	return webhookerrors.ToStatusErr(wh.Name, response.Response.Result)
}

func podList(pods v1.PodList) (ret []*v1.Pod) {
	for _, p := range pods.Items {
		pod := p
		ret = append(ret, &pod)
	}
	return
}

func serviceList(services v1.ServiceList) (ret []*v1.Service) {
	for _, s := range services.Items {
		service := s
		ret = append(ret, &service)
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

// checkCondition returns whether the condition is true
func checkCondition(status v1alpha1.MXJobStatus, condType v1alpha1.MXJobConditionType) bool {
	cond := getCondition(status, condType)
	return cond != nil && cond.Status == v1.ConditionTrue
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

func isJobFinished(status v1alpha1.MXJobStatus) bool {
	for _, c := range status.Conditions {
		if (c.Type == v1alpha1.MXJobSucceeded || c.Type == v1alpha1.MXJobFailed) && c.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func genExpectationPodsKey(mxjobKey string, replicaType v1alpha1.MXReplicaType) string {
	return mxjobKey + "/" + strings.ToLower(string(replicaType)) + "/pods"
}

func genMXConfigEnv(mxjob *v1alpha1.MXJob, rtype v1alpha1.MXReplicaType) (ret map[string]string) {
	ret = make(map[string]string)
	ret["DMLC_PS_ROOT_URI"] = serviceName(mxjob, v1alpha1.MXReplicaTypeScheduler)
	port := mxjob.Spec.MXReplicaSpecs.Scheduler.PsRootPort
	if port == nil {
		port = &defaultPsRootPort
	}
	ret["DMLC_PS_ROOT_PORT"] = strconv.FormatInt(int64(*port), 10)
	ret["DMLC_NUM_SERVER"] = strconv.FormatInt(int64(*mxjob.Spec.MXReplicaSpecs.Server.Replicas), 10)
	ret["DMLC_NUM_WORKER"] = strconv.FormatInt(int64(*mxjob.Spec.MXReplicaSpecs.Worker.Replicas), 10)
	switch rtype {
	case v1alpha1.MXReplicaTypeScheduler:
		ret["DMLC_ROLE"] = string(v1alpha1.MXReplicaTypeScheduler)
		psVerbose := mxjob.Spec.MXReplicaSpecs.Scheduler.PsVerbose
		if psVerbose != nil {
			ret["PS_VERBOSE"] = strconv.FormatInt(int64(*psVerbose), 10)
		}
	case v1alpha1.MXReplicaTypeServer:
		ret["DMLC_ROLE"] = string(v1alpha1.MXReplicaTypeServer)
		psVerbose := mxjob.Spec.MXReplicaSpecs.Server.PsVerbose
		if psVerbose != nil {
			ret["PS_VERBOSE"] = strconv.FormatInt(int64(*psVerbose), 10)
		}
	case v1alpha1.MXReplicaTypeWorker:
		ret["DMLC_ROLE"] = string(v1alpha1.MXReplicaTypeWorker)
		psVerbose := mxjob.Spec.MXReplicaSpecs.Worker.PsVerbose
		if psVerbose != nil {
			ret["PS_VERBOSE"] = strconv.FormatInt(int64(*psVerbose), 10)
		}
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

func serviceName(mxjob *v1alpha1.MXJob, rt v1alpha1.MXReplicaType) string {
	return mxjob.Name + "-" + string(rt)
}
