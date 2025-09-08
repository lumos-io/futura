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

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"

	quotav1 "github.com/openshift/api/quota/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// DataCollector emits metrics with CollectMetricData based on the Kubernetes API objects in the metadata store.
type DataCollector struct {
	metadataStore *metadata.Store
}

// NewDataCollector returns a DataCollector.
func NewDataCollector(ms *metadata.Store) *DataCollector {
	return &DataCollector{
		metadataStore: ms,
	}
}

func (dc *DataCollector) CollectMetricData(ts time.Time) []*pb.KubernetesClusterObject {
	result := []*pb.KubernetesClusterObject{}
	dc.metadataStore.ForEach(gvk.Pod, func(o any) {
		result = append(result, pod.RecordMetrics(o.(*corev1.Pod), ts))
	})
	dc.metadataStore.ForEach(gvk.Node, func(o any) {
		result = append(result, node.RecordMetrics(o.(*corev1.Node), ts))
	})
	dc.metadataStore.ForEach(gvk.Namespace, func(o any) {
		result = append(result, namespace.RecordMetrics(o.(*corev1.Namespace), ts))
	})
	dc.metadataStore.ForEach(gvk.ReplicationController, func(o any) {
		result = append(result, replicationcontroller.RecordMetrics(o.(*corev1.ReplicationController), ts))
	})
	dc.metadataStore.ForEach(gvk.ResourceQuota, func(o any) {
		result = append(result, resourcequota.RecordMetrics(o.(*corev1.ResourceQuota), ts))
	})
	dc.metadataStore.ForEach(gvk.Deployment, func(o any) {
		result = append(result, deployment.RecordMetrics(o.(*appsv1.Deployment), ts))
	})
	dc.metadataStore.ForEach(gvk.ReplicaSet, func(o any) {
		result = append(result, replicaset.RecordMetrics(o.(*appsv1.ReplicaSet), ts))
	})
	dc.metadataStore.ForEach(gvk.DaemonSet, func(o any) {
		result = append(result, daemonset.RecordMetrics(o.(*appsv1.DaemonSet), ts))
	})
	dc.metadataStore.ForEach(gvk.StatefulSet, func(o any) {
		result = append(result, statefulset.RecordMetrics(o.(*appsv1.StatefulSet), ts))
	})
	dc.metadataStore.ForEach(gvk.Job, func(o any) {
		result = append(result, jobs.RecordMetrics(o.(*batchv1.Job), ts))
	})
	dc.metadataStore.ForEach(gvk.CronJob, func(o any) {
		result = append(result, cronjob.RecordMetrics(o.(*batchv1.CronJob), ts))
	})
	dc.metadataStore.ForEach(gvk.HorizontalPodAutoscaler, func(o any) {
		result = append(result, hpa.RecordMetrics(o.(*autoscalingv2.HorizontalPodAutoscaler), ts))
	})
	dc.metadataStore.ForEach(gvk.ClusterResourceQuota, func(o any) {
		result = append(result, clusterresourcequota.RecordMetrics(o.(*quotav1.ClusterResourceQuota), ts))
	})
	return result
}
