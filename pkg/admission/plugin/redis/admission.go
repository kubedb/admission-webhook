package redis

import (
	"fmt"
	"sync"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	hookapi "github.com/appscode/kutil/admission/api"
	"github.com/appscode/kutil/meta"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	cs "github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1"
	rdv "github.com/kubedb/redis/pkg/validator"
	admission "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type RedisValidator struct {
	client      kubernetes.Interface
	extClient   cs.KubedbV1alpha1Interface
	lock        sync.RWMutex
	initialized bool
}

var _ hookapi.AdmissionHook = &RedisValidator{}

func (a *RedisValidator) Resource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.kubedb.com",
			Version:  "v1alpha1",
			Resource: "redisreviews",
		},
		"redisreview"
}

func (a *RedisValidator) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
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

func (a *RedisValidator) Admit(req *admission.AdmissionRequest) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}

	if (req.Operation != admission.Create && req.Operation != admission.Update && req.Operation != admission.Delete) ||
		len(req.SubResource) != 0 ||
		req.Kind.Group != api.SchemeGroupVersion.Group ||
		req.Kind.Kind != api.ResourceKindRedis {
		status.Allowed = true
		return status
	}

	a.lock.RLock()
	defer a.lock.RUnlock()
	if !a.initialized {
		return hookapi.StatusUninitialized()
	}

	if req.Operation == admission.Delete {
		// req.Object.Raw = nil, so read from kubernetes
		obj, err := a.extClient.Redises(req.Namespace).Get(req.Name, metav1.GetOptions{})
		if err != nil && !kerr.IsNotFound(err) {
			return hookapi.StatusInternalServerError(err)
		} else if err == nil && obj.Spec.DoNotPause {
			return hookapi.StatusBadRequest(fmt.Errorf(`redis "%s" can't be paused. To continue delete, unset spec.doNotPause and retry`, req.Name))
		}
		status.Allowed = true
		return status
	}

	obj, err := meta.UnmarshalToJSON(req.Object.Raw, api.SchemeGroupVersion)
	if err != nil {
		return hookapi.StatusBadRequest(err)
	}

	err = rdv.ValidateRedis(a.client, a.extClient, obj.(*api.Redis))
	if err != nil {
		return hookapi.StatusForbidden(err)
	}

	status.Allowed = true
	return status
}
