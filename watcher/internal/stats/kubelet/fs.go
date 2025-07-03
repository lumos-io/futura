package kubelet

import (
	"time"

	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
)

func addFilesystemMetrics(mb *metadata.MetricsBuilder, filesystemMetrics metadata.FilesystemMetrics, s *stats.FsStats, currentTime time.Time) {
	if s == nil {
		return
	}

	recordIntDataPoint(mb, filesystemMetrics.Available, s.AvailableBytes, currentTime)
	recordIntDataPoint(mb, filesystemMetrics.Capacity, s.CapacityBytes, currentTime)
	recordIntDataPoint(mb, filesystemMetrics.Usage, s.UsedBytes, currentTime)
}
