package namespace

import (
	"strings"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	corev1 "k8s.io/api/core/v1"
)

const (
	// Keys for namespace metadata and entity attributes.
	k8sNamespaceCreationTime = "k8s.namespace.creation_timestamp"
	k8sNamespacePhase        = "k8s.namespace.phase"
)

func RecordMetrics(mb *metadata.MetricsBuilder, ns *corev1.Namespace, ts time.Time) {
	mb.RecordK8sNamespacePhaseDataPoint(ts, int64(namespacePhaseValues[ns.Status.Phase]))
	rb := mb.NewResourceBuilder()
	rb.SetK8sNamespaceUID(string(ns.UID))
	rb.SetK8sNamespaceName(ns.Name)
	mb.EmitForResource(metadata.WithResource(rb.Emit()))
}

var namespacePhaseValues = map[corev1.NamespacePhase]int32{
	corev1.NamespaceActive:      1,
	corev1.NamespaceTerminating: 0,
	// If phase is blank for some reason, send as -1 for unknown.
	corev1.NamespacePhase(""): -1,
}

func GetMetadata(ns *corev1.Namespace) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	meta := map[string]string{}
	meta[metadata.GetOTelNameFromKind("namespace")] = ns.Name
	if ns.Status.Phase == "" {
		meta[k8sNamespacePhase] = "unknown"
	} else {
		meta[k8sNamespacePhase] = strings.ToLower(string(ns.Status.Phase))
	}
	meta[k8sNamespaceCreationTime] = ns.CreationTimestamp.Format(time.RFC3339)
	nsID := metadata.ResourceID(ns.UID)

	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		nsID: {
			EntityType:    "k8s.namespace",
			ResourceIDKey: "k8s.namespace.uid",
			ResourceID:    nsID,
			Metadata:      meta,
		},
	}
}
