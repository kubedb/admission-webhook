package dormant_database

import (
	"fmt"
	"sync"

	hookapi "github.com/appscode/kutil/admission/api"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	cs "github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1"
	"github.com/kubedb/kubedb-server/pkg/admission/util"
	admission "k8s.io/api/admission/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type DormantDatabaseValidator struct {
	client      kubernetes.Interface
	extClient   cs.KubedbV1alpha1Interface
	lock        sync.RWMutex
	initialized bool
}

var _ hookapi.AdmissionHook = &DormantDatabaseValidator{}

func (a *DormantDatabaseValidator) Resource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.kubedb.com",
			Version:  "v1alpha1",
			Resource: "dormantdatabasereviews",
		},
		"dormantdatabasereview"
}

func (a *DormantDatabaseValidator) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.initialized = true

	var err error
	if a.client, err = kubernetes.NewForConfig(config); err != nil {
		return err
	}
	if a.extClient, err = cs.NewForConfig(config); err != nil {
		return err
	}
	return err
}

func (a *DormantDatabaseValidator) Admit(req *admission.AdmissionRequest) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}

	if (req.Operation != admission.Create && req.Operation != admission.Update && req.Operation != admission.Delete) ||
		len(req.SubResource) != 0 ||
		req.Kind.Group != api.SchemeGroupVersion.Group ||
		req.Kind.Kind != api.ResourceKindDormantDatabase {
		status.Allowed = true
		return status
	}

	a.lock.RLock()
	defer a.lock.RUnlock()
	if !a.initialized {
		return hookapi.StatusUninitialized()
	}

	switch req.Operation {
	case admission.Delete:
		// validate the operation made by User
		if !util.IsKubeDBOperator(req.UserInfo) {
			// req.Object.Raw = nil, so read from kubernetes
			obj, err := a.extClient.DormantDatabases(req.Namespace).Get(req.Name, metav1.GetOptions{})
			if err != nil && !kerr.IsNotFound(err) {
				return hookapi.StatusInternalServerError(err)
			} else if err == nil && obj.Status.Phase != api.DormantDatabasePhaseWipedOut {
				return hookapi.StatusBadRequest(fmt.Errorf(`dormant_database "%s" can't be delete. To continue delete, set spec.wipeOut true and retry`, req.Name))
			}
		}
	case admission.Create:
		if !util.IsKubeDBOperator(req.UserInfo) {
			// skip validating kubedb-operator
			return hookapi.StatusBadRequest(fmt.Errorf(`user can't create object with kind dormantdatabase'`))
		}
	case admission.Update:
		if !util.IsKubeDBOperator(req.UserInfo) {
			// validate changes made by user
			if err := util.ValidateUpdate(req.Object.Raw, req.OldObject.Raw, req.Kind.Kind); err != nil {
				return hookapi.StatusForbidden(fmt.Errorf("%v", err))
			}
		}
	}
	status.Allowed = true
	return status
}
