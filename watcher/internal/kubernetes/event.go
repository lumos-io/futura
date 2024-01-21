package kubernetes

import (
	"encoding/json"

	"github.com/opisvigilant/futura/watcher/internal/logger"
	"k8s.io/apimachinery/pkg/runtime"
)

// Event represent an event got from k8s api server
// Events from different endpoints need to be casted to watcherEvent
// before being able to be handled by handler
type Event struct {
	Kind        watcherType    `json:"kind"`
	TriggetType triggerType    `json:"triggerType"`
	Obj         runtime.Object `json:"object"`
}

// Message returns event message in standard format.
// included as a part of event packege to enhance code resuablity across handlers.
func (e *Event) Message() (msg string) {
	b, err := json.Marshal(e)
	if err != nil {
		logger.Logger().Err(err).Msg("failed to marshal the event")
	}
	return string(b)
}

func ConvertEvent(e Event) interface{} {
	// convert here
	switch e.Kind {
	case PodType:
		return nil
	case CoreEventType:
		return nil
	case EventType:
		return nil
	case HPAType:
		return nil
	case DaemonSetType:
		return nil
	case StatefulSetType:
		return nil
	case ReplicaSetType:
		return nil
	case ServiceType:
		return nil
	case DeploymentType:
		return nil
	case NamespaceType:
		return nil
	case JobType:
		return nil
	case NodeType:
		return nil
	case IngressType:
		return nil
	// case ServiceAccountType:
	// 	return nil
	// case ClusterRoleType:
	// 	return nil
	// case ClusterRoleBindingType:
	// 	return nil
	// case PersistentVolumeType:
	// 	return nil
	// case SecretType:
	// 	return nil
	// case ConfigMapType:
	// 	return nil
	default:
		return nil
	}
}
