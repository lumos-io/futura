package kubelet

import (
	"time"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	"github.com/opisvigilant/futura/watcher/utils"

	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

type getNetworkDataFunc func(s *stats.NetworkStats) (rx *uint64, tx *uint64)

type getInterfaceDataFunc func(s *stats.InterfaceStats) (rx *uint64, tx *uint64)

func addNetworkMetrics(mb *metadata.MetricsBuilder, s *stats.NetworkStats, currentTime time.Time, allInterfaces bool) {
	if s == nil {
		return
	}

	if allInterfaces {
		for i := range s.Interfaces {
			stat := &s.Interfaces[i]
			rx, tx := getInterfaceIO(stat)
			mb.NetworkMetrics.SetCurrentTime(currentTime)
			if rx != nil {
				mb.NetworkMetrics.SetIoRX(utils.PointerToUint64(rx))
			}
			if tx != nil {
				mb.NetworkMetrics.SetIoTx(utils.PointerToUint64(tx))
			}

			rx, tx = getInterfaceErrors(stat)
			if rx != nil {
				mb.NetworkMetrics.SetErrorsRx(utils.PointerToUint64(rx))
			}
			if tx != nil {
				mb.NetworkMetrics.SetErrorsTx(utils.PointerToUint64(tx))
			}
		}
		// Because stats.NetworkStats.Interfaces contains metrics for all interfaces, including default,
		// we don't need to iterate over stats.NetworkStats.InterfaceStats for it, hence we return here
		return
	}

	rx, tx := getNetworkIO(s)
	mb.NetworkMetrics.SetCurrentTime(currentTime)
	if rx != nil {
		mb.NetworkMetrics.SetIoRX(utils.PointerToUint64(rx))
	}
	if tx != nil {
		mb.NetworkMetrics.SetIoTx(utils.PointerToUint64(tx))
	}

	rx, tx = getNetworkErrors(s)
	if rx != nil {
		mb.NetworkMetrics.SetErrorsRx(utils.PointerToUint64(rx))
	}
	if tx != nil {
		mb.NetworkMetrics.SetErrorsTx(utils.PointerToUint64(tx))
	}
}

func getNetworkIO(s *stats.NetworkStats) (*uint64, *uint64) {
	return s.RxBytes, s.TxBytes
}

func getNetworkErrors(s *stats.NetworkStats) (*uint64, *uint64) {
	return s.RxErrors, s.TxErrors
}

func getInterfaceIO(s *stats.InterfaceStats) (*uint64, *uint64) {
	return s.RxBytes, s.TxBytes
}

func getInterfaceErrors(s *stats.InterfaceStats) (*uint64, *uint64) {
	return s.RxErrors, s.TxErrors
}
