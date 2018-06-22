package framework

import (
	"strings"
	"time"

	constants "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
)

const (
	opPodName           = "mxnet-operator"
	testNamespacePrefix = "mxnet-operator-e2e-test-"
	waitTimeout         = 60 * time.Second
)

// Framework is a framework for e2e testing.
type Framework struct {
	opImage    string
	kubeClient kubernetes.Interface
	namespace  string
}

// New creates a Framework.
func New(kubecli kubernetes.Interface, ns, opImage string) *Framework {
	if ns == "" {
		ns = testNamespacePrefix + string(uuid.NewUUID())
		logrus.Info("running tests in generated namespace: ", ns)
	}

	return &Framework{
		opImage:    opImage,
		namespace:  ns,
		kubeClient: kubecli,
	}
}

// Namespace returns the namespace of the framework.
func (t *Framework) Namespace() string {
	return t.namespace
}

// Setup setup the framework.
func (t *Framework) Setup() error {
	if e := t.createTestNamespace(); e != nil {
		return e
	}
	return t.setupOperator()
}

// Teardown cleanup the framework.
func (t *Framework) Teardown() error {
	if e := t.cleanupOperator(); e != nil {
		// delete first and ignore error
		logrus.Info("failed to cleanup mxnet operator: ", e)
	}
	return t.deleteTestNamespace()
}

func (t *Framework) createTestNamespace() error {
	if !strings.HasPrefix(t.namespace, testNamespacePrefix) {
		return nil
	}

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: t.namespace,
		},
	}
	_, err := t.kubeClient.CoreV1().Namespaces().Create(ns)
	if err != nil {
		return err
	}
	return nil
}

func (t *Framework) deleteTestNamespace() error {
	if !strings.HasPrefix(t.namespace, testNamespacePrefix) {
		return nil
	}

	if e := t.kubeClient.CoreV1().Namespaces().Delete(t.namespace, nil); e != nil {
		if apierrors.IsNotFound(e) {
			return nil
		}
	}
	return nil
}

func (t *Framework) setupOperator() error {
	if t.opImage == "" {
		return nil
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opPodName,
			Namespace: t.namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:            "mxnet-operator",
				Image:           t.opImage,
				Command:         []string{"/usr/local/bin/mxnet-operator"},
				ImagePullPolicy: v1.PullIfNotPresent,
				Env: []v1.EnvVar{
					{
						Name:      constants.WatchNamespaceEnvVar,
						ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
					},
				},
			}},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	pod, err := CreateAndWaitPod(t.kubeClient, t.namespace, pod, waitTimeout)
	if err != nil {
		return err
	}
	logrus.Infof("mxnet operator pod is running on node (%s)", pod.Spec.NodeName)
	return nil
}

func (t *Framework) cleanupOperator() error {
	if t.opImage == "" {
		return nil
	}

	if err := DeleteAndWaitPod(t.kubeClient, t.namespace, opPodName, waitTimeout); err != nil {
		return err
	}
	logrus.Info("mxnet operator pod is completely removed")
	return nil
}
