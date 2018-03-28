package mutator

import (
	"fmt"

	"github.com/appscode/go/types"
	mon_api "github.com/appscode/kube-mon/api"
	"github.com/appscode/kutil"
	core_util "github.com/appscode/kutil/core/v1"
	meta_util "github.com/appscode/kutil/meta"
	"github.com/cloudflare/cfssl/log"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	cs "github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

// SetDefaultValues provides the defaulting that is performed in mutating stage of creating/updating a MongoDB database
func SetDefaultValues(client kubernetes.Interface, extClient cs.KubedbV1alpha1Interface, mongodb *api.MongoDB) (runtime.Object, error) {
	if mongodb.Spec.Version == "" {
		return nil, fmt.Errorf(`object 'Version' is missing in '%v'`, mongodb.Spec)
	}

	if mongodb.Spec.Replicas == nil {
		mongodb.Spec.Replicas = types.Int32P(1)
	}

	if err := fuseDormantDB(extClient, mongodb); err != nil {
		return nil, err
	}

	// Set Default DatabaseSecretName
	if mongodb.Spec.DatabaseSecret == nil {
		if err := checkSecret(client, mongodb); err != nil {
			return nil, err
		}
		mongodb.Spec.DatabaseSecret = &core.SecretVolumeSource{
			SecretName: fmt.Sprintf("%v-auth", mongodb.Name),
		}
	}

	// If monitoring spec is given without port,
	// set default Listening port
	setMonitoringPort(mongodb)

	return mongodb, nil
}

// fuseDormantDB takes values from Similar Dormant Database
func fuseDormantDB(extClient cs.KubedbV1alpha1Interface, mongodb *api.MongoDB) error {
	// Check if DormantDatabase exists or not
	dormantDb, err := extClient.DormantDatabases(mongodb.Namespace).Get(mongodb.Name, metav1.GetOptions{})
	if err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
		return nil
	}

	// Check DatabaseKind
	if dormantDb.Labels[api.LabelDatabaseKind] != api.ResourceKindMongoDB {
		return errors.New(fmt.Sprintf(`invalid MongoDB: "%v". Exists DormantDatabase "%v" of different Kind`, mongodb.Name, dormantDb.Name))
	}

	// Check Origin Spec
	drmnOriginSpec := dormantDb.Spec.Origin.Spec.MongoDB
	originalSpec := mongodb.Spec

	// If DatabaseSecret of new object is not given,
	// Take dormantDatabaseSecretName
	if originalSpec.DatabaseSecret == nil {
		originalSpec.DatabaseSecret = drmnOriginSpec.DatabaseSecret
	} else {
		drmnOriginSpec.DatabaseSecret = originalSpec.DatabaseSecret
	}

	// Skip checking doNotPause
	drmnOriginSpec.DoNotPause = originalSpec.DoNotPause

	// Skip checking Monitoring
	drmnOriginSpec.Monitor = originalSpec.Monitor

	// Skip Checking BackUP Scheduler
	drmnOriginSpec.BackupSchedule = originalSpec.BackupSchedule

	if !meta_util.Equal(drmnOriginSpec, &originalSpec) {
		diff := meta_util.Diff(drmnOriginSpec, &originalSpec)
		log.Errorf("mongodb spec mismatches with OriginSpec in DormantDatabases. Diff: %v", diff)
		return errors.New(fmt.Sprintf("mongodb spec mismatches with OriginSpec in DormantDatabases. Diff: %v", diff))
	}

	if _, err := meta_util.GetString(mongodb.Annotations, api.AnnotationInitialized); err == kutil.ErrNotFound &&
		mongodb.Spec.Init != nil &&
		mongodb.Spec.Init.SnapshotSource != nil {
		mongodb.Annotations = core_util.UpsertMap(mongodb.Annotations, map[string]string{
			api.AnnotationInitialized: "",
		})
	}

	// Delete  Matching dormantDatabase in Controller

	return nil
}

// Assign Default Monitoring Port if MonitoringSpec Exists
// and the AgentVendor is Prometheus.
func setMonitoringPort(mongodb *api.MongoDB) {
	if mongodb.Spec.Monitor != nil &&
		mongodb.GetMonitoringVendor() == mon_api.VendorPrometheus {
		if mongodb.Spec.Monitor.Prometheus == nil {
			mongodb.Spec.Monitor.Prometheus = &mon_api.PrometheusSpec{}
		}
		if mongodb.Spec.Monitor.Prometheus.Port == 0 {
			mongodb.Spec.Monitor.Prometheus.Port = api.PrometheusExporterPortNumber
		}
	}
}

func checkSecret(client kubernetes.Interface, mongodb *api.MongoDB) error {
	secretName := fmt.Sprintf("%v-auth", mongodb.Name)
	secret, err := client.CoreV1().Secrets(mongodb.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}
	if secret.Labels[api.LabelDatabaseKind] != api.ResourceKindMongoDB ||
		secret.Labels[api.LabelDatabaseName] != mongodb.Name {
		return fmt.Errorf(`intended secret "%v" already exists`, secretName)
	}
	return nil
}
