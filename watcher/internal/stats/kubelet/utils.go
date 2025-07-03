package kubelet

import (
	"time"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
)

func recordIntDataPoint(mb *metadata.MetricsBuilder, recordDataPoint metadata.RecordIntDataPointFunc, value *uint64, currentTime time.Time) {
	if value == nil {
		return
	}
	recordDataPoint(mb, currentTime, int64(*value))
}
