package validator

import (
	"fmt"

	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	adr "github.com/kubedb/apimachinery/pkg/docker"
	amv "github.com/kubedb/apimachinery/pkg/validator"
	dr "github.com/kubedb/redis/pkg/docker"
	"k8s.io/client-go/kubernetes"
)

func ValidateRedis(client kubernetes.Interface, redis *api.Redis, docker *dr.Docker) error {
	if redis.Spec.Version == "" {
		return fmt.Errorf(`object 'Version' is missing in '%v'`, redis.Spec)
	}

	if docker != nil {
		// Set Database Image version
		if err := adr.CheckDockerImageVersion(docker.GetImage(redis), string(redis.Spec.Version)); err != nil {
			return fmt.Errorf(`image %v not found`, docker.GetImageWithTag(redis))
		}
	}

	if redis.Spec.Storage != nil {
		var err error
		if err = amv.ValidateStorage(client, redis.Spec.Storage); err != nil {
			return err
		}
	}

	monitorSpec := redis.Spec.Monitor
	if monitorSpec != nil {
		if err := amv.ValidateMonitorSpec(monitorSpec); err != nil {
			return err
		}

	}
	return nil
}
