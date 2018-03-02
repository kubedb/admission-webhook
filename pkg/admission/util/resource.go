package util

import (
	"fmt"
	"strings"

	tapi "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
)

func checkChainKeyUnchanged(key string, mapData map[string]interface{}) bool {
	keys := strings.Split(key, ".")

	newKey := strings.Join(keys[1:], ".")
	if keys[0] == "*" {
		if len(keys) == 1 {
			return true
		}
		for _, val := range mapData {
			if !checkChainKeyUnchanged(newKey, val.(map[string]interface{})) {
				return false
			}
		}
	} else {
		val, ok := mapData[keys[0]]
		if !ok || len(keys) == 1 {
			return !ok
		}
		return checkChainKeyUnchanged(newKey, val.(map[string]interface{}))
	}

	return true
}

func requireChainKeyUnchanged(key string) mergepatch.PreconditionFunc {
	return func(patch interface{}) bool {
		patchMap, ok := patch.(map[string]interface{})
		if !ok {
			fmt.Println("Invalid data")
			return true
		}
		return checkChainKeyUnchanged(key, patchMap)
	}
}

func getPreconditionFunc(kind string) []mergepatch.PreconditionFunc {
	preconditions := []mergepatch.PreconditionFunc{
		mergepatch.RequireKeyUnchanged("apiVersion"),
		mergepatch.RequireKeyUnchanged("kind"),
		mergepatch.RequireMetadataKeyUnchanged("name"),
		mergepatch.RequireMetadataKeyUnchanged("namespace"),
	}
	return preconditions
}

var preconditionSpecField = map[string][]string{
	tapi.ResourceKindElasticsearch: {
		"spec.version",
		"spec.topology.*.prefix",
		"spec.enableSSL",
		"spec.certificateSecret",
		"spec.databaseSecret",
		"spec.storage",
		"spec.nodeSelector",
		"spec.init",
	},
	tapi.ResourceKindPostgres: {
		"spec.version",
		"spec.standby",
		"spec.streaming",
		"spec.archiver",
		"spec.databaseSecret",
		"spec.storage",
		"spec.nodeSelector",
		"spec.init",
	},
	tapi.ResourceKindMySQL: {
		"spec.version",
		"spec.storage",
		"spec.databaseSecret",
		"spec.nodeSelector",
		"spec.init",
	},
	tapi.ResourceKindMongoDB: {
		"spec.version",
		"spec.storage",
		"spec.nodeSelector",
		"spec.init",
	},
	tapi.ResourceKindRedis: {
		"spec.version",
		"spec.storage",
		"spec.nodeSelector",
	},
	tapi.ResourceKindMemcached: {
		"spec.version",
		"spec.nodeSelector",
	},
	tapi.ResourceKindDormantDatabase: {
		"spec.origin",
	},
}

func getConditionalPreconditionFunc(kind string) []mergepatch.PreconditionFunc {
	preconditions := make([]mergepatch.PreconditionFunc, 0)

	if fields, found := preconditionSpecField[kind]; found {
		for _, field := range fields {
			preconditions = append(preconditions,
				requireChainKeyUnchanged(field),
			)
		}
	}

	return preconditions
}

func checkConditionalPrecondition(patchData []byte, fns ...mergepatch.PreconditionFunc) error {
	patch := make(map[string]interface{})
	if err := json.Unmarshal(patchData, &patch); err != nil {
		return err
	}
	for _, fn := range fns {
		if !fn(patch) {
			return mergepatch.NewErrPreconditionFailed(patch)
		}
	}
	return nil
}

func preconditionFailedError() error {
	return errors.New(`At least one of the following was changed:
	apiVersion
	kind
	name
	namespace`)
}

func conditionalPreconditionFailedError(kind string) error {
	str := preconditionSpecField[kind]
	strList := strings.Join(str, "\n\t")
	return fmt.Errorf(`At least one of the following was changed:
	%v`, strList)
}

func ValidateUpdate(modified, oldObj []byte, kind string) error {
	preconditions := getPreconditionFunc(kind)
	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(oldObj, modified, oldObj, preconditions...)
	if err != nil {
		if mergepatch.IsPreconditionFailed(err) {
			return preconditionFailedError()
		}
		return err
	}

	conditionalPreconditions := getConditionalPreconditionFunc(kind)
	if err = checkConditionalPrecondition(patch, conditionalPreconditions...); err != nil {
		if mergepatch.IsPreconditionFailed(err) {
			return conditionalPreconditionFailedError(kind)
		}
		return err
	}
	return nil
}
