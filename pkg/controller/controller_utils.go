package controller

import (
	"fmt"
	"sync"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	"github.com/golang/glog"
)

const (
	FailedCreateServiceReason     = "FailedCreateService"
	SuccessfulCreateServiceReason = "SuccessfulCreateService"
)

// ServiceControlInterface is an interface that knows how to add or delete Services
// created as an interface to allow testing.
type ServiceControlInterface interface {
	// CreateServices creates new Services according to the spec.
	CreateServices(namespace string, service *v1.Service, object runtime.Object) error
	// CreateServicesWithControllerRef creates new services according to the spec, and sets object as the service's controller.
	CreateServicesWithControllerRef(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error
	// PatchService patches the service.
	PatchService(namespace, name string, data []byte) error
}

func validateControllerRef(controllerRef *metav1.OwnerReference) error {
	if controllerRef == nil {
		return fmt.Errorf("controllerRef is nil")
	}
	if len(controllerRef.APIVersion) == 0 {
		return fmt.Errorf("controllerRef has empty APIVersion")
	}
	if len(controllerRef.Kind) == 0 {
		return fmt.Errorf("controllerRef has empty Kind")
	}
	if controllerRef.Controller == nil || *controllerRef.Controller != true {
		return fmt.Errorf("controllerRef.Controller is not set to true")
	}
	if controllerRef.BlockOwnerDeletion == nil || *controllerRef.BlockOwnerDeletion != true {
		return fmt.Errorf("controllerRef.BlockOwnerDeletion is not set to true")
	}
	return nil
}

// RealServiceControl is the default implementation of ServiceControlInterface.
type RealServiceControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

func (r RealServiceControl) PatchService(namespace, name string, data []byte) error {
	_, err := r.KubeClient.CoreV1().Services(namespace).Patch(name, types.StrategicMergePatchType, data)
	return err
}

func (r RealServiceControl) CreateServices(namespace string, service *v1.Service, object runtime.Object) error {
	return r.createServices(namespace, service, object, nil)
}

func (r RealServiceControl) CreateServicesWithControllerRef(namespace string, service *v1.Service, controllerObject runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := validateControllerRef(controllerRef); err != nil {
		return err
	}
	return r.createServices(namespace, service, controllerObject, controllerRef)
}

func (r RealServiceControl) createServices(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	if labels.Set(service.Labels).AsSelectorPreValidated().Empty() {
		return fmt.Errorf("unable to create Services, no labels")
	}

	newService, err := r.KubeClient.CoreV1().Services(namespace).Create(service)
	if err != nil {
		r.Recorder.Eventf(object, v1.EventTypeWarning, FailedCreateServiceReason, "Error creating: %v", err)
		return fmt.Errorf("unable to create services: %v", err)
	}

	accessor, err := meta.Accessor(object)
	if err != nil {
		glog.Errorf("parentObject does not have ObjectMeta, %v", err)
		return nil
	}
	glog.V(4).Infof("Controller %v created service %v", accessor.GetName(), newService.Name)
	r.Recorder.Eventf(object, v1.EventTypeNormal, SuccessfulCreateServiceReason, "Created service: %v", newService.Name)

	return nil
}

type FakeServiceControl struct {
	sync.Mutex
	Templates       []v1.Service
	ControllerRefs  []metav1.OwnerReference
	DeletePodName   []string
	Patches         [][]byte
	Err             error
	CreateLimit     int
	CreateCallCount int
}

var _ ServiceControlInterface = &FakeServiceControl{}

func (f *FakeServiceControl) PatchService(namespace, name string, data []byte) error {
	f.Lock()
	defer f.Unlock()
	f.Patches = append(f.Patches, data)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakeServiceControl) CreateServices(namespace string, service *v1.Service, object runtime.Object) error {
	f.Lock()
	defer f.Unlock()
	f.CreateCallCount++
	if f.CreateLimit != 0 && f.CreateCallCount > f.CreateLimit {
		return fmt.Errorf("Not creating service, limit %d already reached (create call %d)", f.CreateLimit, f.CreateCallCount)
	}
	f.Templates = append(f.Templates, *service)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

func (f *FakeServiceControl) CreateServicesWithControllerRef(namespace string, service *v1.Service, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	f.Lock()
	defer f.Unlock()
	f.CreateCallCount++
	if f.CreateLimit != 0 && f.CreateCallCount > f.CreateLimit {
		return fmt.Errorf("Not creating service, limit %d already reached (create call %d)", f.CreateLimit, f.CreateCallCount)
	}
	f.Templates = append(f.Templates, *service)
	f.ControllerRefs = append(f.ControllerRefs, *controllerRef)
	if f.Err != nil {
		return f.Err
	}
	return nil
}
