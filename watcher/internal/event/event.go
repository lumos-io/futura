package event

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

// Event represent an event got from k8s api server
// Events from different endpoints need to be casted to watcherEvent
// before being able to be handled by handler
type Event struct {
	Namespace  string
	Kind       string
	ApiVersion string
	Component  string
	Host       string
	Reason     string
	Status     string
	Name       string
	Obj        runtime.Object
	OldObj     runtime.Object
}

var m = map[string]string{
	"created": "Normal",
	"deleted": "Danger",
	"updated": "Warning",
}

// Message returns event message in standard format.
// included as a part of event packege to enhance code resuablity across handlers.
func (e *Event) Message() (msg string) {
	// using switch over if..else, since the format could vary based on the kind of the object in future.
	switch e.Kind {
	case "namespace":
		msg = fmt.Sprintf(
			"A namespace `%s` has been `%s`",
			e.Name,
			e.Reason,
		)
	case "node":
		msg = fmt.Sprintf(
			"A node `%s` has been `%s`",
			e.Name,
			e.Reason,
		)
	case "cluster role":
		msg = fmt.Sprintf(
			"A cluster role `%s` has been `%s`",
			e.Name,
			e.Reason,
		)
	case "NodeReady":
		msg = fmt.Sprintf(
			"Node `%s` is Ready : \nNodeReady",
			e.Name,
		)
	case "NodeNotReady":
		msg = fmt.Sprintf(
			"Node `%s` is Not Ready : \nNodeNotReady",
			e.Name,
		)
	case "NodeRebooted":
		msg = fmt.Sprintf(
			"Node `%s` Rebooted : \nNodeRebooted",
			e.Name,
		)
	case "Backoff":
		msg = fmt.Sprintf(
			"Pod `%s` in `%s` Crashed : \nCrashLoopBackOff %s",
			e.Name,
			e.Namespace,
			e.Reason,
		)
	default:
		msg = fmt.Sprintf(
			"A `%s` in namespace `%s` has been `%s`:\n`%s`",
			e.Kind,
			e.Namespace,
			e.Reason,
			e.Name,
		)
	}
	return msg
}
