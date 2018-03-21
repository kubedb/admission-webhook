package elasticsearch

import (
	"net/http"
	"os"
	"testing"

	"github.com/appscode/go/types"
	kubeMon "github.com/appscode/kube-mon/api"
	"github.com/appscode/kutil/meta"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	extFake "github.com/kubedb/apimachinery/client/clientset/versioned/fake"
	"github.com/kubedb/apimachinery/client/clientset/versioned/scheme"
	"github.com/kubedb/kubedb-server/pkg/admission/util"
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
	os.Setenv(util.EnvSvcAccountName, "kubedb-operator")
	os.Setenv("KUBE_NAMESPACE", "kube-system")
}

var requestKind = metaV1.GroupVersionKind{
	Group:   api.SchemeGroupVersion.Group,
	Version: api.SchemeGroupVersion.Version,
	Kind:    api.ResourceKindElasticsearch,
}

func TestElasticsearchValidator_Admit(t *testing.T) {
	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			validator := ElasticsearchValidator{}

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
				if _, err := validator.extClient.KubedbV1alpha1().Elasticsearches(c.namespace).Create(&c.object); err != nil && !kerr.IsAlreadyExists(err) {
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
	object     api.Elasticsearch
	oldObject  api.Elasticsearch
	heatUp     bool
	result     bool
}{
	{"Create Valid Elasticsearch",
		requestKind,
		"foo",
		"default",
		admission.Create,
		sampleElasticsearch(),
		api.Elasticsearch{},
		false,
		true,
	},
	{"Create Invalid Elasticsearch",
		requestKind,
		"foo",
		"default",
		admission.Create,
		getAwkwardElasticsearch(),
		api.Elasticsearch{},
		false,
		false,
	},
	{"Edit Elasticsearch Spec.DatabaseSecret",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecSecret(sampleElasticsearch()),
		sampleElasticsearch(),
		false,
		false,
	},
	{"Edit Status",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editStatus(sampleElasticsearch()),
		sampleElasticsearch(),
		false,
		true,
	},
	{"Edit Spec.Monitor",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecMonitor(sampleElasticsearch()),
		sampleElasticsearch(),
		false,
		true,
	},
	{"Edit Invalid Spec.Monitor",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecInvalidMonitor(sampleElasticsearch()),
		sampleElasticsearch(),
		false,
		false,
	},
	{"Edit Spec.DoNotPause",
		requestKind,
		"foo",
		"default",
		admission.Update,
		editSpecDoNotPause(sampleElasticsearch()),
		sampleElasticsearch(),
		false,
		true,
	},
	{"Delete Elasticsearch when Spec.DoNotPause=true",
		requestKind,
		"foo",
		"default",
		admission.Delete,
		sampleElasticsearch(),
		api.Elasticsearch{},
		true,
		false,
	},
	{"Delete Elasticsearch when Spec.DoNotPause=false",
		requestKind,
		"foo",
		"default",
		admission.Delete,
		editSpecDoNotPause(sampleElasticsearch()),
		api.Elasticsearch{},
		true,
		true,
	},
	{"Delete Non Existing Elasticsearch",
		requestKind,
		"foo",
		"default",
		admission.Delete,
		api.Elasticsearch{},
		api.Elasticsearch{},
		false,
		true,
	},
}

func sampleElasticsearch() api.Elasticsearch {
	return api.Elasticsearch{
		TypeMeta: metaV1.TypeMeta{
			Kind:       api.ResourceKindElasticsearch,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels: map[string]string{
				api.LabelDatabaseKind: api.ResourceKindElasticsearch,
			},
		},
		Spec: api.ElasticsearchSpec{
			Version:    "5.6",
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
							Repository: "https://github.com/kubedb/elasticsearch-init-scripts.git",
							Directory:  ".",
						},
					},
				},
			},
		},
	}
}

func getAwkwardElasticsearch() api.Elasticsearch {
	elasticsearch := sampleElasticsearch()
	elasticsearch.Spec.Version = "3.0"
	return elasticsearch
}

func editSpecSecret(old api.Elasticsearch) api.Elasticsearch {
	old.Spec.DatabaseSecret = &core.SecretVolumeSource{
		SecretName: "foo-auth",
	}
	return old
}

func editStatus(old api.Elasticsearch) api.Elasticsearch {
	old.Status = api.ElasticsearchStatus{
		Phase: api.DatabasePhaseCreating,
	}
	return old
}

func editSpecMonitor(old api.Elasticsearch) api.Elasticsearch {
	old.Spec.Monitor = &kubeMon.AgentSpec{
		Agent: kubeMon.AgentPrometheusBuiltin,
	}
	return old
}

// should be failed because more fields required for COreOS Monitoring
func editSpecInvalidMonitor(old api.Elasticsearch) api.Elasticsearch {
	old.Spec.Monitor = &kubeMon.AgentSpec{
		Agent: kubeMon.AgentCoreOSPrometheus,
	}
	return old
}

func editSpecDoNotPause(old api.Elasticsearch) api.Elasticsearch {
	old.Spec.DoNotPause = false
	return old
}
