package mongodb

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/appscode/kutil/meta"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	cs "github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1"
	hookapi "github.com/kubedb/kubedb-server/pkg/admission/api"
	mgv "github.com/kubedb/mongodb/pkg/validator"
	admission "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type MongoDBValidator struct {
	client      kubernetes.Interface
	extClient   cs.KubedbV1alpha1Interface
	lock        sync.RWMutex
	initialized bool
}

var _ hookapi.AdmissionHook = &MongoDBValidator{}

func (a *MongoDBValidator) Resource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.kubedb.com",
			Version:  "v1alpha1",
			Resource: "mongodbreviews",
		},
		"mongodbreview"
}

func (a *MongoDBValidator) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
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

func (a *MongoDBValidator) Admit(req *admission.AdmissionRequest) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}

	if (req.Operation != admission.Create && req.Operation != admission.Update && req.Operation != admission.Delete) ||
		len(req.SubResource) != 0 ||
		req.Kind.Group != api.SchemeGroupVersion.Group ||
		req.Kind.Kind != api.ResourceKindMongoDB {
		status.Allowed = true
		return status
	}

	a.lock.RLock()
	defer a.lock.RUnlock()
	if !a.initialized {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
			Message: "not initialized",
		}
		return status
	}

	if req.Operation == admission.Delete {
		obj, err := a.extClient.MongoDBs(req.Namespace).Get(req.Name, metav1.GetOptions{})
		if err == nil && obj.Spec.DoNotPause {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: fmt.Sprintf(`mongodb "%s" can't be paused. To continue delete, unset spec.doNotPause and retry`, req.Name),
			}
			return status
		}
		status.Allowed = true
		return status
	}

	obj, err := meta.UnmarshalToJSON(req.Object.Raw, api.SchemeGroupVersion)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: err.Error(),
		}
		return status
	}

	err = mgv.ValidateMongoDB(a.client, a.extClient, obj.(*api.MongoDB))
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
			Message: err.Error(),
		}
		return status
	}

	status.Allowed = true
	return status
}
