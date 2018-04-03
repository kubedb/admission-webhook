package mysql

import (
	"net/http"
	"testing"

	"github.com/appscode/go/types"
	kubeMon "github.com/appscode/kube-mon/api"
	"github.com/appscode/kutil/meta"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	extFake "github.com/kubedb/apimachinery/client/clientset/versioned/fake"
	"github.com/kubedb/apimachinery/client/clientset/versioned/scheme"
	admission "k8s.io/api/admission/v1beta1"
	authenticationV1 "k8s.io/api/authentication/v1"
	core "k8s.io/api/core/v1"
	storageV1beta1 "k8s.io/api/storage/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clientSetScheme "k8s.io/client-go/kubernetes/scheme"
)

func init() {
	scheme.AddToScheme(clientSetScheme.Scheme)
}

var requestKind = metaV1.GroupVersionKind{
	Group:   api.SchemeGroupVersion.Group,
	Version: api.SchemeGroupVersion.Version,
	Kind:    api.ResourceKindMySQL,
}

func TestMySQLValidator_Admit(t *testing.T) {
	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			validator := MySQLValidator{}

			validator.initialized = true
			validator.extClient = extFake.NewSimpleClientset()
			validator.client = fake.NewSimpleClientset(
				&core.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "foo-auth",
						Namespace: "default",
					},
				},
				&storageV1beta1.StorageClass{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "standard",
					},
				},
			)

			objJS, err := meta.MarshalToJson(&c.object, api.SchemeGroupVersion)
			if err != nil {
				panic(err)
			}
			oldObjJS, err := meta.MarshalToJson(&c.oldObject, api.SchemeGroupVersion)
			if err != nil {
				panic(err)
			}

			req := new(admission.AdmissionRequest)

			req.Kind = c.kind
			req.Name = c.objectName
			req.Namespace = c.namespace
			req.Operation = c.operation
			req.UserInfo = authenticationV1.UserInfo{}
			req.Object.Raw = objJS
			req.OldObject.Raw = oldObjJS

			if c.heatUp {
				if _, err := validator.extClient.KubedbV1alpha1().MySQLs(c.namespace).Create(&c.object); err != nil && !kerr.IsAlreadyExists(err) {
					t.Errorf(err.Error())
				}
			}
			if c.operation == admission.Delete {
				req.Object = runtime.RawExtension{}
			}
			if c.operation != admission.Update {
				req.OldObject = runtime.RawExtension{}
			}

			response := validator.Admit(req)
			if c.result == true {
				if response.Allowed != true {
					t.Errorf("expected: 'Allowed=true'. but got response: %v", response)
				}
			} else if c.result == false {
				if response.Allowed == true || response.Result.Code == http.StatusInternalServerError {
					t.Errorf("expected: 'Allowed=false', but got response: %v", response)
				}
			}
		})
	}

}

var cases = []struct {
	testName   string
	kind       metaV1.GroupVersionKind
	objectName string
	namespace  string
	operation  admission.Operation
	object     api.MySQL
	oldObject  api.MySQL
	heatUp     bool
	result     bool
}{
	{"Create Valid MySQL",
		requestKind,
		"foo",
		"default",
		admission.Create,
		sampleMySQL(),
		api.MySQL{},
		false,
		true,
	},
	{"Create Invalid MySQL",
		requestKind,
		"foo",
		"default",
		admission.Create,
		getAwkwardMySQL(),
		api.MySQL{},
		false,
		false,
	},
	{"Edit MySQL Spec.DatabaseSecret",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecSecret(sampleMySQL()),
		sampleMySQL(),
		false,
		false,
	},
	{"Edit Status",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editStatus(sampleMySQL()),
		sampleMySQL(),
		false,
		true,
	},
	{"Edit Spec.Monitor",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecMonitor(sampleMySQL()),
		sampleMySQL(),
		false,
		true,
	},
	{"Edit Invalid Spec.Monitor",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecInvalidMonitor(sampleMySQL()),
		sampleMySQL(),
		false,
		false,
	},
	{"Edit Spec.DoNotPause",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecDoNotPause(sampleMySQL()),
		sampleMySQL(),
		false,
		true,
	},
	{"Delete MySQL when Spec.DoNotPause=true",
		requestKind,
		"foo",
		"default",
		admission.Delete,
		sampleMySQL(),
		api.MySQL{},
		true,
		false,
	},
	{"Delete MySQL when Spec.DoNotPause=false",
		requestKind,
		"foo",
		"default",
		admission.Delete,
		editSpecDoNotPause(sampleMySQL()),
		api.MySQL{},
		true,
		true,
	},
	{"Delete Non Existing MySQL",
		requestKind,
		"foo",
		"default",
		admission.Delete,
		api.MySQL{},
		api.MySQL{},
		false,
		true,
	},
}

func sampleMySQL() api.MySQL {
	return api.MySQL{
		TypeMeta: metaV1.TypeMeta{
			Kind:       api.ResourceKindMySQL,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels: map[string]string{
				api.LabelDatabaseKind: api.ResourceKindMySQL,
			},
		},
		Spec: api.MySQLSpec{
			Version:    "8.0",
			DoNotPause: true,
			Storage: &core.PersistentVolumeClaimSpec{
				StorageClassName: types.StringP("standard"),
				Resources: core.ResourceRequirements{
					Requests: core.ResourceList{
						core.ResourceStorage: resource.MustParse("100Mi"),
					},
				},
			},
			Init: &api.InitSpec{
				ScriptSource: &api.ScriptSourceSpec{
					VolumeSource: core.VolumeSource{
						GitRepo: &core.GitRepoVolumeSource{
							Repository: "https://github.com/kubedb/mysql-init-scripts.git",
							Directory:  ".",
						},
					},
				},
			},
		},
	}
}

func getAwkwardMySQL() api.MySQL {
	mysql := sampleMySQL()
	mysql.Spec.Version = "3.0"
	return mysql
}

func editSpecSecret(old api.MySQL) api.MySQL {
	old.Spec.DatabaseSecret = &core.SecretVolumeSource{
		SecretName: "foo-auth",
	}
	return old
}

func editStatus(old api.MySQL) api.MySQL {
	old.Status = api.MySQLStatus{
		Phase: api.DatabasePhaseCreating,
	}
	return old
}

func editSpecMonitor(old api.MySQL) api.MySQL {
	old.Spec.Monitor = &kubeMon.AgentSpec{
		Agent: kubeMon.AgentPrometheusBuiltin,
	}
	return old
}

// should be failed because more fields required for COreOS Monitoring
func editSpecInvalidMonitor(old api.MySQL) api.MySQL {
	old.Spec.Monitor = &kubeMon.AgentSpec{
		Agent: kubeMon.AgentCoreOSPrometheus,
	}
	return old
}

func editSpecDoNotPause(old api.MySQL) api.MySQL {
	old.Spec.DoNotPause = false
	return old
}
