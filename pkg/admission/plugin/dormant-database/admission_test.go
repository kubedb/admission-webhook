package dormant_database

import (
	"net/http"
	"os"
	"testing"

	"github.com/appscode/kutil/meta"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	ext_fake "github.com/kubedb/apimachinery/client/clientset/versioned/fake"
	"github.com/kubedb/apimachinery/client/clientset/versioned/scheme"
	"github.com/kubedb/kubedb-server/pkg/admission/util"
	admission "k8s.io/api/admission/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

func init() {
	scheme.AddToScheme(clientsetscheme.Scheme)
	os.Setenv(util.EnvSvcAccountName, "kubedb-operator")
}

func (a *DormantDatabaseValidator) _initialize() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.initialized = true
	a.client = fake.NewSimpleClientset()
	a.extClient = ext_fake.NewSimpleClientset()

	return nil
}

func initialReq() admission.AdmissionRequest {
	return admission.AdmissionRequest{
		Kind: metav1.GroupVersionKind{
			Group:   api.SchemeGroupVersion.Group,
			Version: api.SchemeGroupVersion.Version,
			Kind:    api.ResourceKindDormantDatabase,
		},
	}
}

func initialReqWithObj() admission.AdmissionRequest {
	req := admission.AdmissionRequest{
		Kind: metav1.GroupVersionKind{
			Group:   api.SchemeGroupVersion.Group,
			Version: api.SchemeGroupVersion.Version,
			Kind:    api.ResourceKindDormantDatabase,
		},
	}

	obj := validDormantDatabase()
	objJS, err := meta.MarshalToJson(&obj, api.SchemeGroupVersion)
	if err != nil {
		panic(err)
	}
	req.Object.Raw = objJS
	return req
}

func TestDormantDatabaseValidator_Admit_CreateDormantDatabaseByOperator(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReqWithObj()
	req.Operation = admission.Create
	req.UserInfo = userIsOperator()

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_CreateDormantDatabaseByUser(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReqWithObj()
	req.Operation = admission.Create
	req.UserInfo = userIsHooman()

	response := validator.Admit(&req)
	if response.Allowed == true || response.Result.Code == http.StatusInternalServerError {
		t.Errorf("expected: 'Allowed=false'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_EditStatusByOperator(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReqWithObj()
	req.Operation = admission.Update
	req.UserInfo = userIsOperator()

	oldObj := editStatus(validDormantDatabase())
	oldObjJS, err := meta.MarshalToJson(&oldObj, api.SchemeGroupVersion)
	if err != nil {
		t.Errorf(err.Error())
	}
	req.OldObject.Raw = oldObjJS

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_EditStatusByUser(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReqWithObj()
	req.Operation = admission.Update
	req.UserInfo = userIsHooman()

	oldObj := editStatus(validDormantDatabase())
	oldObjJS, err := meta.MarshalToJson(&oldObj, api.SchemeGroupVersion)
	if err != nil {
		t.Errorf(err.Error())
	}
	req.OldObject.Raw = oldObjJS

	response := validator.Admit(&req)
	if response.Allowed == true || response.Result.Code == http.StatusInternalServerError {
		t.Errorf("expected: 'Allowed=false', but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_EditSpecOriginByUser(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReqWithObj()
	req.Operation = admission.Update
	req.UserInfo = userIsHooman()

	oldObj := editSpecOrigin(validDormantDatabase())
	oldObjJS, err := meta.MarshalToJson(&oldObj, api.SchemeGroupVersion)
	if err != nil {
		t.Errorf(err.Error())
	}
	req.OldObject.Raw = oldObjJS

	response := validator.Admit(&req)
	if response.Allowed == true || response.Result.Code == http.StatusInternalServerError {
		t.Errorf("expected: 'Allowed=false'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_EditSpecResumeByOperator(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReqWithObj()
	req.Operation = admission.Update
	req.UserInfo = userIsOperator()

	oldObj := editSpecResume(validDormantDatabase())
	oldObjJS, err := meta.MarshalToJson(&oldObj, api.SchemeGroupVersion)
	if err != nil {
		t.Errorf(err.Error())
	}
	req.OldObject.Raw = oldObjJS

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_EditSpecResumeByUser(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReqWithObj()
	req.Operation = admission.Update
	req.UserInfo = userIsHooman()

	oldObj := editSpecResume(validDormantDatabase())
	oldObjJS, err := meta.MarshalToJson(&oldObj, api.SchemeGroupVersion)
	if err != nil {
		t.Errorf(err.Error())
	}
	req.OldObject.Raw = oldObjJS

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_EditSpecWipeOutByOperator(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReqWithObj()
	req.Operation = admission.Update
	req.UserInfo = userIsOperator()

	oldObj := editSpecWipeOut(validDormantDatabase())
	oldObjJS, err := meta.MarshalToJson(&oldObj, api.SchemeGroupVersion)
	if err != nil {
		t.Errorf(err.Error())
	}
	req.OldObject.Raw = oldObjJS

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_EditSpecWipeOutByUser(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReqWithObj()
	req.Operation = admission.Update
	req.UserInfo = userIsHooman()

	oldObj := editSpecWipeOut(validDormantDatabase())
	oldObjJS, err := meta.MarshalToJson(&oldObj, api.SchemeGroupVersion)
	if err != nil {
		t.Errorf(err.Error())
	}
	req.OldObject.Raw = oldObjJS

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_DeleteWithoutWipingByOperator(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	obj := validDormantDatabase()
	if _, err := validator.extClient.KubedbV1alpha1().DormantDatabases("default").Create(&obj); err != nil {
		t.Errorf(err.Error())
	}

	req := initialReq()
	req.Operation = admission.Delete
	req.UserInfo = userIsOperator()

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_DeleteWithoutWipingByUser(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	obj := validDormantDatabase()
	if _, err := validator.extClient.KubedbV1alpha1().DormantDatabases("default").Create(&obj); err != nil {
		t.Errorf(err.Error())
	}

	req := initialReq()
	req.Operation = admission.Delete
	req.UserInfo = userIsHooman()

	response := validator.Admit(&req)
	if response.Allowed == true || response.Result.Code == http.StatusInternalServerError {
		t.Errorf("expected: 'Allowed=false'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_DeleteWithWipingByUser(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	obj := validDormantDatabase()
	obj = editSpecWipeOut(obj)
	obj.Status.Phase = api.DormantDatabasePhaseWipedOut
	if _, err := validator.extClient.KubedbV1alpha1().DormantDatabases("default").Create(&obj); err != nil {
		t.Errorf(err.Error())
	}

	req := initialReq()
	req.Operation = admission.Delete
	req.UserInfo = userIsHooman()

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_DeleteNonExistingDormantByOperator(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReq()
	req.Operation = admission.Delete
	req.UserInfo = userIsOperator()

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func TestDormantDatabaseValidator_Admit_DeleteNonExistingDormantByUser(t *testing.T) {
	validator := DormantDatabaseValidator{}
	validator._initialize()

	req := initialReq()
	req.Operation = admission.Delete
	req.UserInfo = userIsHooman()

	response := validator.Admit(&req)
	if response.Allowed != true {
		t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
	}
}

func validDormantDatabase() api.DormantDatabase {
	return api.DormantDatabase{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.ResourceKindDormantDatabase,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
			Labels: map[string]string{
				api.LabelDatabaseKind: api.ResourceKindMongoDB,
			},
		},
		Spec: api.DormantDatabaseSpec{
			Origin: api.Origin{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
					Labels: map[string]string{
						api.LabelDatabaseKind: api.ResourceKindMongoDB,
					},
					Annotations: map[string]string{
						api.AnnotationInitialized: "",
					},
				},
				Spec: api.OriginSpec{
					MongoDB: &api.MongoDBSpec{},
				},
			},
		},
	}
}

func editSpecOrigin(old api.DormantDatabase) api.DormantDatabase {
	old.Spec.Origin.Annotations = nil
	return old
}

func editStatus(old api.DormantDatabase) api.DormantDatabase {
	old.Status = api.DormantDatabaseStatus{
		Phase: api.DormantDatabasePhasePaused,
	}
	return old
}

func editSpecWipeOut(old api.DormantDatabase) api.DormantDatabase {
	old.Spec.WipeOut = true
	return old
}

func editSpecResume(old api.DormantDatabase) api.DormantDatabase {
	old.Spec.Resume = true
	return old
}

func userIsOperator() authenticationv1.UserInfo {
	return authenticationv1.UserInfo{
		Username: "system:serviceaccount:default:kubedb-operator",
		Groups: []string{
			"system:serviceaccounts",
			"system:serviceaccounts:kube-system",
			"system:authenticated",
		},
	}
}

func userIsHooman() authenticationv1.UserInfo {
	return authenticationv1.UserInfo{
		Username: "minikube-user",
		Groups: []string{
			"system:masters",
			"system:authenticated",
		},
	}
}
