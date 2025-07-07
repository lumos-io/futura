package metadata

import (
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
)

type MetricsBuilder struct {
	CPUMetrics        *CPUMetricsBuilder
	MemoryMetrics     *MemoryMetricsBuilder
	FilesystemMetrics *FilesystemMetricsBuilder
	NetworkMetrics    *NetworkMetricsBuilder
	VolumeMetrics     *VolumeMetricsBuilder
	UptimeMetrics     *UptimeMetricsBuilder
}

func NewMetricsBuilder() *MetricsBuilder {
	return &MetricsBuilder{
		CPUMetrics:        NewCPUMetricsBuilder(),
		MemoryMetrics:     NewMemoryMetricsBuilder(),
		FilesystemMetrics: NewFilesystemMetricsBuilder(),
		NetworkMetrics:    NewNetworkMetricsBuilder(),
		VolumeMetrics:     NewVolumeMetricsBuilder(),
		UptimeMetrics:     NewUptimeMetricsBuilder(),
	}
}

func (mb *MetricsBuilder) RecordCPUMetricsMetric() {

}

func (mb *MetricsBuilder) RecordUptime() {

}

func (mb *MetricsBuilder) Emit() []*pbst.KubernetesResourceMetric {
	var metrics []*pbst.KubernetesResourceMetric
	if mb.CPUMetrics != nil {
		m := mb.CPUMetrics.BuildMetric()
		if m != nil {
			metrics = append(metrics, m)
		}
	}
	if mb.MemoryMetrics != nil {
		m := mb.MemoryMetrics.BuildMetric()
		if m != nil {
			metrics = append(metrics, m)
		}
	}
	if mb.FilesystemMetrics != nil {
		m := mb.FilesystemMetrics.BuildMetric()
		if m != nil {
			metrics = append(metrics, m)
		}
	}
	if mb.NetworkMetrics != nil {
		m := mb.NetworkMetrics.BuildMetric()
		if m != nil {
			metrics = append(metrics, m)
		}
	}
	if mb.VolumeMetrics != nil {
		m := mb.VolumeMetrics.BuildMetric()
		if m != nil {
			metrics = append(metrics, m)
		}
	}
	if mb.UptimeMetrics != nil {
		m := mb.UptimeMetrics.BuildMetric()
		if m != nil {
			metrics = append(metrics, m)
		}
	}
	return metrics
}

type CPUMetricsBuilder struct {
	resource *pbst.Resource
	metadata *pbst.MetricMetadata
	metric   *pbst.CPUMetrics
}

func NewCPUMetricsBuilder() *CPUMetricsBuilder {
	return &CPUMetricsBuilder{
		resource: &pbst.Resource{},
		metadata: &pbst.MetricMetadata{},
		metric:   &pbst.CPUMetrics{},
	}
}

func (b *CPUMetricsBuilder) SetTime(v int64) {
	b.metric.Time = v
}

func (b *CPUMetricsBuilder) GetTime() int64 {
	return b.metric.Time
}

func (b *CPUMetricsBuilder) SetUsage(v float64) {
	b.metric.Usage = v
}

func (b *CPUMetricsBuilder) GetUsage() float64 {
	return b.metric.Usage
}

func (b *CPUMetricsBuilder) SetUtilization(v float64) {
	b.metric.Utilization = v
}

func (b *CPUMetricsBuilder) GetUtilization() float64 {
	return b.metric.Utilization
}

func (b *CPUMetricsBuilder) SetNodeUtilization(v float64) {
	b.metric.NodeUtilization = v
}

func (b *CPUMetricsBuilder) GetNodeUtilization() float64 {
	return b.metric.NodeUtilization
}

func (b *CPUMetricsBuilder) SetLimitUtilization(v float64) {
	b.metric.LimitUtilization = v
}

func (b *CPUMetricsBuilder) GetLimitUtilization() float64 {
	return b.metric.LimitUtilization
}

func (b *CPUMetricsBuilder) SetRequestUtilization(v float64) {
	b.metric.RequestUtilization = v
}

func (b *CPUMetricsBuilder) GetRequestUtilization() float64 {
	return b.metric.RequestUtilization
}

func (b *CPUMetricsBuilder) BuildMetric() *pbst.KubernetesResourceMetric {
	return &pbst.KubernetesResourceMetric{
		Resource:       b.resource,
		MetricMetadata: b.metadata,
		MetricValue: &pbst.KubernetesResourceMetric_Cpu{
			Cpu: b.metric,
		},
	}
}

type MemoryMetricsBuilder struct {
	resource *pbst.Resource
	metadata *pbst.MetricMetadata
	metric   *pbst.MemoryMetrics
}

func NewMemoryMetricsBuilder() *MemoryMetricsBuilder {
	return &MemoryMetricsBuilder{
		resource: &pbst.Resource{},
		metadata: &pbst.MetricMetadata{},
		metric:   &pbst.MemoryMetrics{},
	}
}

func (b *MemoryMetricsBuilder) SetAvailable(v int64) {
	b.metric.Available = v
}

func (b *MemoryMetricsBuilder) GetAvailable() int64 {
	return b.metric.Available
}

func (b *MemoryMetricsBuilder) SetUsage(v int64) {
	b.metric.Usage = v
}

func (b *MemoryMetricsBuilder) GetUsage() int64 {
	return b.metric.Usage
}

func (b *MemoryMetricsBuilder) SetNodeUtilization(v float64) {
	b.metric.NodeUtilization = v
}

func (b *MemoryMetricsBuilder) GetNodeUtilization() float64 {
	return b.metric.NodeUtilization
}

func (b *MemoryMetricsBuilder) SetLimitUtilization(v float64) {
	b.metric.LimitUtilization = v
}

func (b *MemoryMetricsBuilder) GetLimitUtilization() float64 {
	return b.metric.LimitUtilization
}

func (b *MemoryMetricsBuilder) SetRequestUtilization(v float64) {
	b.metric.RequestUtilization = v
}

func (b *MemoryMetricsBuilder) GetRequestUtilization() float64 {
	return b.metric.RequestUtilization
}

func (b *MemoryMetricsBuilder) SetRss(v int64) {
	b.metric.Rss = v
}

func (b *MemoryMetricsBuilder) GetRss() int64 {
	return b.metric.Rss
}

func (b *MemoryMetricsBuilder) SetWorkingSet(v int64) {
	b.metric.WorkingSet = v
}

func (b *MemoryMetricsBuilder) GetWorkingSet() int64 {
	return b.metric.WorkingSet
}

func (b *MemoryMetricsBuilder) SetPageFaults(v int64) {
	b.metric.PageFaults = v
}

func (b *MemoryMetricsBuilder) GetPageFaults() int64 {
	return b.metric.PageFaults
}

func (b *MemoryMetricsBuilder) SetMajorPageFaults(v int64) {
	b.metric.MajorPageFaults = v
}

func (b *MemoryMetricsBuilder) GetMajorPageFaults() int64 {
	return b.metric.MajorPageFaults
}

func (b *MemoryMetricsBuilder) BuildMetric() *pbst.KubernetesResourceMetric {
	return &pbst.KubernetesResourceMetric{
		Resource:       b.resource,
		MetricMetadata: b.metadata,
		MetricValue: &pbst.KubernetesResourceMetric_Memory{
			Memory: b.metric,
		},
	}
}

type FilesystemMetricsBuilder struct {
	resource *pbst.Resource
	metadata *pbst.MetricMetadata
	metric   *pbst.FilesystemMetrics
}

func NewFilesystemMetricsBuilder() *FilesystemMetricsBuilder {
	return &FilesystemMetricsBuilder{
		resource: &pbst.Resource{},
		metadata: &pbst.MetricMetadata{},
		metric:   &pbst.FilesystemMetrics{},
	}
}

func (b *FilesystemMetricsBuilder) SetAvailable(v int64) {
	b.metric.Available = v
}

func (b *FilesystemMetricsBuilder) GetAvailable() int64 {
	return b.metric.Available
}

func (b *FilesystemMetricsBuilder) SetCapacity(v int64) {
	b.metric.Capacity = v
}

func (b *FilesystemMetricsBuilder) GetCapacity() int64 {
	return b.metric.Capacity
}

func (b *FilesystemMetricsBuilder) SetUsage(v int64) {
	b.metric.Usage = v
}

func (b *FilesystemMetricsBuilder) GetUsage() int64 {
	return b.metric.Usage
}

func (b *FilesystemMetricsBuilder) BuildMetric() *pbst.KubernetesResourceMetric {
	return &pbst.KubernetesResourceMetric{
		Resource:       b.resource,
		MetricMetadata: b.metadata,
		MetricValue: &pbst.KubernetesResourceMetric_Filesystem{
			Filesystem: b.metric,
		},
	}
}

type NetworkMetricsBuilder struct {
	resource *pbst.Resource
	metadata *pbst.MetricMetadata
	metric   *pbst.NetworkMetrics
}

func NewNetworkMetricsBuilder() *NetworkMetricsBuilder {
	return &NetworkMetricsBuilder{
		resource: &pbst.Resource{},
		metadata: &pbst.MetricMetadata{},
		metric:   &pbst.NetworkMetrics{},
	}
}

func (b *NetworkMetricsBuilder) SetIo(v int64) {
	b.metric.Io = v
}

func (b *NetworkMetricsBuilder) GetIo() int64 {
	return b.metric.Io
}

func (b *NetworkMetricsBuilder) SetErrors(v int64) {
	b.metric.Errors = v
}

func (b *NetworkMetricsBuilder) GetErrors() int64 {
	return b.metric.Errors
}

func (b *NetworkMetricsBuilder) BuildMetric() *pbst.KubernetesResourceMetric {
	return &pbst.KubernetesResourceMetric{
		Resource:       b.resource,
		MetricMetadata: b.metadata,
		MetricValue: &pbst.KubernetesResourceMetric_Network{
			Network: b.metric,
		},
	}
}

type VolumeMetricsBuilder struct {
	resource *pbst.Resource
	metadata *pbst.MetricMetadata
	metric   *pbst.VolumeMetrics
}

func NewVolumeMetricsBuilder() *VolumeMetricsBuilder {
	return &VolumeMetricsBuilder{
		resource: &pbst.Resource{},
		metadata: &pbst.MetricMetadata{},
		metric:   &pbst.VolumeMetrics{},
	}
}

func (b *VolumeMetricsBuilder) SetAvailable(v int64) {
	b.metric.Available = v
}

func (b *VolumeMetricsBuilder) GetAvailable() int64 {
	return b.metric.Available
}

func (b *VolumeMetricsBuilder) SetCapacity(v int64) {
	b.metric.Capacity = v
}

func (b *VolumeMetricsBuilder) GetCapacity() int64 {
	return b.metric.Capacity
}

func (b *VolumeMetricsBuilder) SetInodes(v int64) {
	b.metric.Inodes = v
}

func (b *VolumeMetricsBuilder) GetInodes() int64 {
	return b.metric.Inodes
}

func (b *VolumeMetricsBuilder) SetInodesFree(v int64) {
	b.metric.InodesFree = v
}

func (b *VolumeMetricsBuilder) GetInodesFree() int64 {
	return b.metric.InodesFree
}

func (b *VolumeMetricsBuilder) SetInodesUsed(v int64) {
	b.metric.InodesUsed = v
}

func (b *VolumeMetricsBuilder) GetInodesUsed() int64 {
	return b.metric.InodesUsed
}

func (b *VolumeMetricsBuilder) BuildMetric() *pbst.KubernetesResourceMetric {
	return &pbst.KubernetesResourceMetric{
		Resource:       b.resource,
		MetricMetadata: b.metadata,
		MetricValue: &pbst.KubernetesResourceMetric_Volume{
			Volume: b.metric,
		},
	}
}

type UptimeMetricsBuilder struct {
	resource *pbst.Resource
	metadata *pbst.MetricMetadata
	metric   *pbst.UptimeMetrics
}

func NewUptimeMetricsBuilder() *UptimeMetricsBuilder {
	return &UptimeMetricsBuilder{
		resource: &pbst.Resource{},
		metadata: &pbst.MetricMetadata{},
		metric:   &pbst.UptimeMetrics{},
	}
}

func (b *UptimeMetricsBuilder) SetUptime(v int64) {
	b.metric.Uptime = v
}

func (b *UptimeMetricsBuilder) GetUptime() int64 {
	return b.metric.Uptime
}

func (b *UptimeMetricsBuilder) BuildMetric() *pbst.KubernetesResourceMetric {
	return &pbst.KubernetesResourceMetric{
		Resource:       b.resource,
		MetricMetadata: b.metadata,
		MetricValue: &pbst.KubernetesResourceMetric_Uptime{
			Uptime: b.metric,
		},
	}
}
