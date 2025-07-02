package collection

import (
	"time"

	"github.com/opisvigilant/futura/watcher/internal/cluster/clusterresourcequota"
	"github.com/opisvigilant/futura/watcher/internal/cluster/cronjob"
	"github.com/opisvigilant/futura/watcher/internal/cluster/daemonset"
	"github.com/opisvigilant/futura/watcher/internal/cluster/deployment"
	"github.com/opisvigilant/futura/watcher/internal/cluster/gvk"
	"github.com/opisvigilant/futura/watcher/internal/cluster/hpa"
	"github.com/opisvigilant/futura/watcher/internal/cluster/jobs"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"github.com/opisvigilant/futura/watcher/internal/cluster/namespace"
	"github.com/opisvigilant/futura/watcher/internal/cluster/node"
	"github.com/opisvigilant/futura/watcher/internal/cluster/pod"
	"github.com/opisvigilant/futura/watcher/internal/cluster/replicaset"
	"github.com/opisvigilant/futura/watcher/internal/cluster/replicationcontroller"
	"github.com/opisvigilant/futura/watcher/internal/cluster/resourcequota"
	"github.com/opisvigilant/futura/watcher/internal/cluster/statefulset"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"

	quotav1 "github.com/openshift/api/quota/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// TODO: Consider moving some of these constants to
// https://go.opentelemetry.io/collector/blob/main/model/semconv/opentelemetry.go.

// DataCollector emits metrics with CollectMetricData based on the Kubernetes API objects in the metadata store.
type DataCollector struct {
	metadataStore            *metadata.Store
	nodeConditionsToReport   []string
	allocatableTypesToReport []string
}

// NewDataCollector returns a DataCollector.
func NewDataCollector(ms *metadata.Store, nodeConditionsToReport, allocatableTypesToReport []string) *DataCollector {
	return &DataCollector{
		metadataStore:            ms,
		nodeConditionsToReport:   nodeConditionsToReport,
		allocatableTypesToReport: allocatableTypesToReport,
	}
}

func (dc *DataCollector) CollectMetricData(ts time.Time) []*pbcluster.KubernetesObjectMetadata {
	dc.metadataStore.ForEach(gvk.Pod, func(o any) {
		pod.RecordMetrics(o.(*corev1.Pod), ts)
	})
	dc.metadataStore.ForEach(gvk.Node, func(o any) {
		crm := node.CustomMetrics(o.(*corev1.Node), dc.nodeConditionsToReport, dc.allocatableTypesToReport, ts)
		if crm.ScopeMetrics().Len() > 0 {
			crm.MoveTo(customRMs.AppendEmpty())
		}
		node.RecordMetrics(o.(*corev1.Node), ts)
	})
	dc.metadataStore.ForEach(gvk.Namespace, func(o any) {
		namespace.RecordMetrics(o.(*corev1.Namespace), ts)
	})
	dc.metadataStore.ForEach(gvk.ReplicationController, func(o any) {
		replicationcontroller.RecordMetrics(o.(*corev1.ReplicationController), ts)
	})
	dc.metadataStore.ForEach(gvk.ResourceQuota, func(o any) {
		resourcequota.RecordMetrics(o.(*corev1.ResourceQuota), ts)
	})
	dc.metadataStore.ForEach(gvk.Deployment, func(o any) {
		deployment.RecordMetrics(o.(*appsv1.Deployment), ts)
	})
	dc.metadataStore.ForEach(gvk.ReplicaSet, func(o any) {
		replicaset.RecordMetrics(o.(*appsv1.ReplicaSet), ts)
	})
	dc.metadataStore.ForEach(gvk.DaemonSet, func(o any) {
		daemonset.RecordMetrics(o.(*appsv1.DaemonSet), ts)
	})
	dc.metadataStore.ForEach(gvk.StatefulSet, func(o any) {
		statefulset.RecordMetrics(o.(*appsv1.StatefulSet), ts)
	})
	dc.metadataStore.ForEach(gvk.Job, func(o any) {
		jobs.RecordMetrics(o.(*batchv1.Job), ts)
	})
	dc.metadataStore.ForEach(gvk.CronJob, func(o any) {
		cronjob.RecordMetrics(o.(*batchv1.CronJob), ts)
	})
	dc.metadataStore.ForEach(gvk.HorizontalPodAutoscaler, func(o any) {
		hpa.RecordMetrics(o.(*autoscalingv2.HorizontalPodAutoscaler), ts)
	})
	dc.metadataStore.ForEach(gvk.ClusterResourceQuota, func(o any) {
		clusterresourcequota.RecordMetrics(o.(*quotav1.ClusterResourceQuota), ts)
	})

	var m []*pbcluster.KubernetesObjectMetadata

	return m
}
