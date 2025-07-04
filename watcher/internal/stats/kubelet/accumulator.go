package kubelet

import (
	"time"

	pbst "github.com/opisvigilant/futura/proto/gen/stats"
	"github.com/rs/zerolog/log"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
)

type MetricGroup string

// Values for MetricGroup enum.
const (
	ContainerMetricGroup = MetricGroup("container")
	PodMetricGroup       = MetricGroup("pod")
	NodeMetricGroup      = MetricGroup("node")
	VolumeMetricGroup    = MetricGroup("volume")
)

// ValidMetricGroups map of valid metrics.
var ValidMetricGroups = map[MetricGroup]bool{
	ContainerMetricGroup: true,
	PodMetricGroup:       true,
	NodeMetricGroup:      true,
	VolumeMetricGroup:    true,
}

type metricDataAccumulator struct {
	metrics               []*pbst.Metric
	metadata              Metadata
	metricGroupsToCollect map[MetricGroup]bool
	allNetworkInterfaces  map[MetricGroup]bool
	time                  time.Time
	mbs                   *metadata.MetricsBuilders
}

func addUptimeMetric(mb *metadata.MetricsBuilder, uptimeMetric metadata.RecordIntDataPointFunc, startTime v1.Time, currentTime time.Time) {
	if !startTime.IsZero() {
		value := int64(time.Since(startTime.Time).Seconds())
		uptimeMetric(mb, currentTime, value)
	}
}

func (a *metricDataAccumulator) nodeStats(s stats.NodeStats) {
	if !a.metricGroupsToCollect[NodeMetricGroup] {
		return
	}

	currentTime := time.Now()
	addUptimeMetric(a.mbs.NodeMetricsBuilder, metadata.NodeUptimeMetrics.Uptime, s.StartTime, currentTime)
	addCPUMetrics(a.mbs.NodeMetricsBuilder, metadata.NodeCPUMetrics, s.CPU, currentTime, resources{}, 0)
	addMemoryMetrics(a.mbs.NodeMetricsBuilder, metadata.NodeMemoryMetrics, s.Memory, currentTime, resources{}, 0)
	addFilesystemMetrics(a.mbs.NodeMetricsBuilder, metadata.NodeFilesystemMetrics, s.Fs, currentTime)
	addNetworkMetrics(a.mbs.NodeMetricsBuilder, metadata.NodeNetworkMetrics, s.Network, currentTime, a.allNetworkInterfaces[NodeMetricGroup])
	// todo s.Runtime.ImageFs
	rb := a.mbs.NodeMetricsBuilder.NewResourceBuilder()
	rb.SetK8sNodeName(s.NodeName)
	a.metrics = append(a.metrics, a.mbs.NodeMetricsBuilder.EmitMetrics()...)
}

func (a *metricDataAccumulator) podStats(s stats.PodStats) {
	if !a.metricGroupsToCollect[PodMetricGroup] {
		return
	}

	currentTime := time.Now()
	addUptimeMetric(a.mbs.PodMetricsBuilder, metadata.PodUptimeMetrics.Uptime, s.StartTime, currentTime)
	addCPUMetrics(a.mbs.PodMetricsBuilder, metadata.PodCPUMetrics, s.CPU, currentTime, a.metadata.podResources[s.PodRef.UID], a.metadata.nodeInfo.CPUCapacity)
	addMemoryMetrics(a.mbs.PodMetricsBuilder, metadata.PodMemoryMetrics, s.Memory, currentTime, a.metadata.podResources[s.PodRef.UID], a.metadata.nodeInfo.MemoryCapacity)
	addFilesystemMetrics(a.mbs.PodMetricsBuilder, metadata.PodFilesystemMetrics, s.EphemeralStorage, currentTime)
	addNetworkMetrics(a.mbs.PodMetricsBuilder, metadata.PodNetworkMetrics, s.Network, currentTime, a.allNetworkInterfaces[PodMetricGroup])

	rb := a.mbs.PodMetricsBuilder.NewResourceBuilder()
	rb.SetK8sPodUID(s.PodRef.UID)
	rb.SetK8sPodName(s.PodRef.Name)
	rb.SetK8sNamespaceName(s.PodRef.Namespace)
	a.metrics = append(a.metrics, a.mbs.PodMetricsBuilder.EmitMetrics()...)
}

func (a *metricDataAccumulator) containerStats(sPod stats.PodStats, s stats.ContainerStats) {
	if !a.metricGroupsToCollect[ContainerMetricGroup] {
		return
	}

	rb := a.mbs.ContainerMetricsBuilder.NewResourceBuilder()
	_, err := getContainerResource(rb, sPod, s, a.metadata)
	if err != nil {
		log.Logger.Warn().Str("pod", sPod.PodRef.Name).Str("container", s.Name).Err(err).Msg("failed to fetch container metrics")
		return
	}

	currentTime := time.Now()
	resourceKey := sPod.PodRef.UID + s.Name
	addUptimeMetric(a.mbs.ContainerMetricsBuilder, metadata.ContainerUptimeMetrics.Uptime, s.StartTime, currentTime)
	addCPUMetrics(a.mbs.ContainerMetricsBuilder, metadata.ContainerCPUMetrics, s.CPU, currentTime, a.metadata.containerResources[resourceKey], a.metadata.nodeInfo.CPUCapacity)
	addMemoryMetrics(a.mbs.ContainerMetricsBuilder, metadata.ContainerMemoryMetrics, s.Memory, currentTime, a.metadata.containerResources[resourceKey], a.metadata.nodeInfo.MemoryCapacity)
	addFilesystemMetrics(a.mbs.ContainerMetricsBuilder, metadata.ContainerFilesystemMetrics, s.Rootfs, currentTime)

	a.metrics = append(a.metrics, a.mbs.ContainerMetricsBuilder.EmitMetrics()...)
}

func (a *metricDataAccumulator) volumeStats(sPod stats.PodStats, s stats.VolumeStats) {
	if !a.metricGroupsToCollect[VolumeMetricGroup] {
		return
	}

	rb := a.mbs.OtherMetricsBuilder.NewResourceBuilder()
	_, err := getVolumeResourceOptions(rb, sPod, s, a.metadata)
	if err != nil {
		log.Logger.Warn().Str("pod", sPod.PodRef.Name).Str("volume", s.Name).Err(err).Msg("Failed to gather additional volume metadata. Skipping metric collection.")
		return
	}

	currentTime := time.Now()
	addVolumeMetrics(a.mbs.OtherMetricsBuilder, metadata.K8sVolumeMetrics, s, currentTime)

	a.metrics = append(a.metrics, a.mbs.OtherMetricsBuilder.EmitMetrics()...)
}
