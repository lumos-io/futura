package namespace

import (
	"strings"
	"time"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
)

const (
	// Keys for namespace metadata and entity attributes.
	k8sNamespaceCreationTime = "k8s.namespace.creation_timestamp"
	k8sNamespacePhase        = "k8s.namespace.phase"
)

func RecordMetrics(ns *corev1.Namespace, ts time.Time) *pbcluster.KubernetesObjectMetadata {
	obj := &pbcluster.KubernetesObjectMetadata{
		Timestamp: timestamppb.New(ts),
		Uid:       string(ns.UID),
		Name:      ns.Name,
	}

	mb.RecordK8sNamespacePhaseDataPoint(ts, int64(namespacePhaseValues[ns.Status.Phase]))

	return obj
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
