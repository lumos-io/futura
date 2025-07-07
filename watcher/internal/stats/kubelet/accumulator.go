package kubelet

import (
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	"github.com/rs/zerolog/log"
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
	metadata              Metadata
	metricGroupsToCollect map[MetricGroup]bool
	allNetworkInterfaces  map[MetricGroup]bool
	time                  time.Time
	mbs                   *metadata.MetricsBuilder
}

func addUptimeMetric(mb *metadata.MetricsBuilder, startTime v1.Time, currentTime time.Time) {
	if !startTime.IsZero() {
		value := int64(time.Since(startTime.Time).Seconds())
		mb.UptimeMetrics.SetUptime(value)
		mb.UptimeMetrics.SetCurrentTime(currentTime)
	}
}

func (a *metricDataAccumulator) nodeStats(s stats.NodeStats) {
	if !a.metricGroupsToCollect[NodeMetricGroup] {
		return
	}

	currentTime := a.time

	addUptimeMetric(a.mbs, s.StartTime, currentTime)
	addCPUMetrics(a.mbs, s.CPU, currentTime, resources{}, 0)
	addMemoryMetrics(a.mbs, s.Memory, currentTime, resources{}, 0)
	addFilesystemMetrics(a.mbs, s.Fs, currentTime)
	addNetworkMetrics(a.mbs, s.Network, currentTime, a.allNetworkInterfaces[NodeMetricGroup])

	a.mbs.Resource.SetK8sNodeName(s.NodeName)

	// rb.SetK8NodeName(s.NodeName)
	// a.m = append(a.m, a.mbs.NodeMetricsBuilder.Emit(
	// metadata.WithStartTimeOverride(s.StartTime.Time),
	// metadata.WithResource(rb.Emit()),
	// ))
}

func (a *metricDataAccumulator) podStats(s stats.PodStats) {
	if !a.metricGroupsToCollect[PodMetricGroup] {
		return
	}

	currentTime := a.time
	addUptimeMetric(a.mbs, s.StartTime, currentTime)
	addCPUMetrics(a.mbs, s.CPU, currentTime, a.metadata.podResources[s.PodRef.UID], a.metadata.nodeInfo.CPUCapacity)
	addMemoryMetrics(a.mbs, s.Memory, currentTime, a.metadata.podResources[s.PodRef.UID], a.metadata.nodeInfo.MemoryCapacity)
	addFilesystemMetrics(a.mbs, s.EphemeralStorage, currentTime)
	addNetworkMetrics(a.mbs, s.Network, currentTime, a.allNetworkInterfaces[PodMetricGroup])

	a.mbs.Resource.SetK8sPodUID(s.PodRef.UID)
	a.mbs.Resource.SetK8sPodName(s.PodRef.Name)
	a.mbs.Resource.SetK8sNamespaceName(s.PodRef.Namespace)
}

func (a *metricDataAccumulator) containerStats(sPod stats.PodStats, s stats.ContainerStats) {
	if !a.metricGroupsToCollect[ContainerMetricGroup] {
		return
	}

	rb := a.mbs.Resource
	if err := getContainerResource(rb, sPod, s, a.metadata); err != nil {
		log.Logger.Warn().Str("pod", sPod.PodRef.Name).Str("container", s.Name).Err(err).Msg("Failed to fetch container metrics")
		return
	}

	currentTime := a.time
	resourceKey := sPod.PodRef.UID + s.Name
	addUptimeMetric(a.mbs, s.StartTime, currentTime)
	addCPUMetrics(a.mbs, s.CPU, currentTime, a.metadata.containerResources[resourceKey], a.metadata.nodeInfo.CPUCapacity)
	addMemoryMetrics(a.mbs, s.Memory, currentTime, a.metadata.containerResources[resourceKey], a.metadata.nodeInfo.MemoryCapacity)
	addFilesystemMetrics(a.mbs, s.Rootfs, currentTime)
}

func (a *metricDataAccumulator) volumeStats(sPod stats.PodStats, s stats.VolumeStats) {
	if !a.metricGroupsToCollect[VolumeMetricGroup] {
		return
	}

	rb := a.mbs.Resource
	if err := getVolumeResourceOptions(rb, sPod, s, a.metadata); err != nil {
		log.Logger.Warn().Str("pod", sPod.PodRef.Name).Str("container", s.Name).Err(err).Msg("Failed to gather additional volume metadata. Skipping metric collection.")
		return
	}

	currentTime := a.time
	addVolumeMetrics(a.mbs, s, currentTime)
}
