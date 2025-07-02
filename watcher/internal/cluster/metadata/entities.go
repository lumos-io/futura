package metadata

import (
	"time"
)

// ResourceID is the Kubernetes UID of the resource. In case of
// containers, this value is the container id.
type ResourceID string

type EntityEventsSlice []string

// MetadataDelta keeps track of changes to metadata on resources.
// The fields on this struct should help determine if there have
// been changes to resource metadata such as Kubernetes labels.
// An example of how this is used. Let's say we are dealing with a
// Pod that has the following labels -
// {"env": "test", "team": "otell", "usser": "bob"}. Now, let's say
// there's an update to one or more labels on the same Pod and the
// labels now look like the following -
// {"env": "test", "team": "otel", "user": "bob"}. The k8sclusterreceiver
// upon receiving the event corresponding to the labels updates will
// generate a MetadataDelta with the following values -
//
//	MetadataToAdd: {"user": "bob"}
//	MetadataToRemove: {"usser": "bob"}
//	MetadataToUpdate: {"team": "otel"}
//
// Apart from Kubernetes labels, the other metadata collected by this
// receiver are also handled in the same manner.
// Type, functionality, and fields not guaranteed to be stable or permanent.
type MetadataDelta struct {
	// MetadataToAdd contains key-value pairs that are newly added to
	// the resource description in the current revision.
	MetadataToAdd map[string]string
	// MetadataToRemove contains key-value pairs that no longer describe
	// a resource and needs to be removed.
	MetadataToRemove map[string]string
	// MetadataToUpdate contains key-value pairs that have been updated
	// in the current revision compared to the previous revisions(s).
	MetadataToUpdate map[string]string
}

// MetadataUpdate provides a delta view of metadata on a resource between
// two revisions of a resource.
// Type, functionality, and fields not guaranteed to be stable or permanent.
type MetadataUpdate struct {
	// ResourceIDKey is the label key of UID label for the resource.
	ResourceIDKey string
	// ResourceID is the Kubernetes UID of the resource. In case of
	// containers, this value is the container id.
	ResourceID ResourceID
	MetadataDelta
}

// GetEntityEvents processes metadata updates and returns entity events that describe the metadata changes.
func GetEntityEvents(oldMetadata, newMetadata map[ResourceID]*KubernetesMetadata, timestamp time.Time, reportingInterval time.Duration) EntityEventsSlice {
	out := new(EntityEventsSlice)

	for id, oldObj := range oldMetadata {
		if _, ok := newMetadata[id]; !ok {
			// An object was present, but no longer is. Create a "delete" event.
			entityEvent := out.AppendEmpty()
			entityEvent.SetTimestamp(timestamp)
			entityEvent.ID().PutStr(oldObj.ResourceIDKey, string(oldObj.ResourceID))
			deleteEvent := entityEvent.SetEntityDelete()
			deleteEvent.SetEntityType(oldObj.EntityType)
		}
	}

	// All "new" are current objects. Create "state" events. "old" state does not matter.
	for _, newObj := range newMetadata {
		entityEvent := out.AppendEmpty()
		entityEvent.SetTimestamp(timestamp)
		entityEvent.ID().PutStr(newObj.ResourceIDKey, string(newObj.ResourceID))
		state := entityEvent.SetEntityState()
		state.SetEntityType(newObj.EntityType)
		if reportingInterval != 0 {
			state.SetInterval(reportingInterval)
		}

		attrs := state.Attributes()
		for k, v := range newObj.Metadata {
			attrs.PutStr(k, v)
		}
	}

	return out
}
