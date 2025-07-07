package kubelet

import (
	"time"

	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	"github.com/opisvigilant/futura/watcher/utils"
)

func addFilesystemMetrics(mb *metadata.MetricsBuilder, s *stats.FsStats, currentTime time.Time) {
	if s == nil {
		return
	}

	mb.FilesystemMetrics.SetCurrentTime(currentTime)
	mb.FilesystemMetrics.SetAvailable(utils.PointerToUint64(s.AvailableBytes))
	mb.FilesystemMetrics.SetCapacity(utils.PointerToUint64(s.CapacityBytes))
	mb.FilesystemMetrics.SetUsage(utils.PointerToUint64(s.UsedBytes))
}
