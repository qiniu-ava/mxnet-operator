package framework

import (
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	BusyBoxImage = "busybox"

	jobTimeout = 15 * time.Minute
)

// CreateAndWaitPod creates a pod and waits until it is running
func CreateAndWaitPod(kubecli kubernetes.Interface, ns string, pod *v1.Pod, timeout time.Duration) (*v1.Pod, error) {
	_, err := kubecli.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		return nil, err
	}

	var retPod *v1.Pod
	err = wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		retPod, err = kubecli.CoreV1().Pods(ns).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		switch retPod.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodPending:
			return false, nil
		default:
			return false, fmt.Errorf("unexpected pod status.phase: %v", retPod.Status.Phase)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to wait pod running: %v", err)
	}

	return retPod, nil
}

// DeleteAndWaitPod deletes a pod and waits util it is completely disappeared.
func DeleteAndWaitPod(kubecli kubernetes.Interface, ns, name string, timeout time.Duration) error {
	err := kubecli.CoreV1().Pods(ns).Delete(name, metav1.NewDeleteOptions(1))
	if err != nil {
		return err
	}
	err = wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		_, err := kubecli.CoreV1().Pods(ns).Get(name, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		return fmt.Errorf("fail to wait operator pod (%s) gone from API: %v", name, err)
	}
	return nil
}
