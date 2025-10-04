package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/opisvigilant/futura/backend/pkg/stream"
	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
)

const (
	StoreKubeletNodeMetricsTopic      = "store.kubelet.node.metrics"
	StoreKubeletPodMetricsTopic       = "store.kubelet.pod.metrics"
	StoreKubeletContainerMetricsTopic = "store.kubelet.container.metrics"
	StoreKubeletNetworkMetricsTopic   = "store.kubelet.network.metrics"
	StoreKubeletVolumeMetricsTopic    = "store.kubelet.volume.metrics"
)

type StatsFlattener struct {
	kc stream.Stream
}

func NewStatsFlattener(kc stream.Stream) *StatsFlattener {
	return &StatsFlattener{
		kc: kc,
	}
}

type flatKubeletNodeMetric struct {
	OrganizationID          uint32  `json:"organization_id"`
	ClusterID               int64   `json:"cluster_id"`
	ReceivedAtUnix          int64   `json:"received_at_unix"`
	Timestamp               int64   `json:"timestamp"`
	IdempotencyKey          string  `json:"idempotency_key"`
	NodeName                string  `json:"node_name"`
	StartTime               int64   `json:"start_time"`
	CPUUsageNanoCores       uint64  `json:"cpu_usage_nano_cores"`
	CPUUsageCoreNanoseconds uint64  `json:"cpu_usage_core_nanoseconds"`
	CPUPsiFullAvg10         float64 `json:"cpu_psi_full_avg10"`
	CPUPsiSomeAvg10         float64 `json:"cpu_psi_some_avg10"`
	MemoryAvailableBytes    uint64  `json:"memory_available_bytes"`
	MemoryUsageBytes        uint64  `json:"memory_usage_bytes"`
	MemoryWorkingSetBytes   uint64  `json:"memory_working_set_bytes"`
	MemoryRSSBytes          uint64  `json:"memory_rss_bytes"`
	MemoryPageFaults        uint64  `json:"memory_page_faults"`
	MemoryMajorPageFaults   uint64  `json:"memory_major_page_faults"`
	MemoryPsiFullAvg10      float64 `json:"memory_psi_full_avg10"`
	MemoryPsiSomeAvg10      float64 `json:"memory_psi_some_avg10"`
	IOPsiFullAvg10          float64 `json:"io_psi_full_avg10"`
	IOPsiSomeAvg10          float64 `json:"io_psi_some_avg10"`
	FSAvailableBytes        uint64  `json:"fs_available_bytes"`
	FSCapacityBytes         uint64  `json:"fs_capacity_bytes"`
	FSUsedBytes             uint64  `json:"fs_used_bytes"`
	SwapAvailableBytes      uint64  `json:"swap_available_bytes"`
	SwapUsageBytes          uint64  `json:"swap_usage_bytes"`
}

type flatKubeletPodMetric struct {
	Timestamp               int64  `json:"timestamp"`
	IdempotencyKey          string `json:"idempotency_key"`
	PodUID                  string `json:"pod_uid"`
	PodName                 string `json:"pod_name"`
	PodNamespace            string `json:"pod_namespace"`
	StartTime               int64  `json:"start_time"`
	CPUUsageNanoCores       uint64 `json:"cpu_usage_nano_cores"`
	CPUUsageCoreNanoseconds uint64 `json:"cpu_usage_core_nanoseconds"`
	MemoryAvailableBytes    uint64 `json:"memory_available_bytes"`
	MemoryUsageBytes        uint64 `json:"memory_usage_bytes"`
	MemoryWorkingSetBytes   uint64 `json:"memory_working_set_bytes"`
	NetworkRxBytes          uint64 `json:"network_rx_bytes"`
	NetworkTxBytes          uint64 `json:"network_tx_bytes"`
	ProcessCount            uint64 `json:"process_count"`
	SwapAvailableBytes      uint64 `json:"swap_available_bytes"`
	SwapUsageBytes          uint64 `json:"swap_usage_bytes"`
}

type flatKubeletContainerMetric struct {
	Timestamp               int64          `json:"timestamp"`
	IdempotencyKey          string         `json:"idempotency_key"`
	PodUID                  string         `json:"pod_uid"`
	ContainerName           string         `json:"container_name"`
	ContainerStartTime      int64          `json:"container_start_time"`
	CPUUsageNanoCores       uint64         `json:"cpu_usage_nano_cores"`
	CPUUsageCoreNanoseconds uint64         `json:"cpu_usage_core_nanoseconds"`
	MemoryAvailableBytes    uint64         `json:"memory_available_bytes"`
	MemoryUsageBytes        uint64         `json:"memory_usage_bytes"`
	MemoryWorkingSetBytes   uint64         `json:"memory_working_set_bytes"`
	SwapAvailableBytes      uint64         `json:"swap_available_bytes"`
	SwapUsageBytes          uint64         `json:"swap_usage_bytes"`
	RootFSAvailableBytes    uint64         `json:"rootfs_available_bytes"`
	RootFSUsedBytes         uint64         `json:"rootfs_used_bytes"`
	LogsUsedBytes           uint64         `json:"logs_used_bytes"`
	Accelerator             map[string]any `json:"accelerator"`
	UserMetrics             map[string]any `json:"user_metrics"`
}

type flatKubeletNetworkMetric struct {
	Timestamp      int64  `json:"timestamp"`
	IdempotencyKey string `json:"idempotency_key"`
	PodUID         string `json:"pod_uid"`
	InterfaceName  string `json:"interface_name"`
	RXBytes        uint64 `json:"rx_bytes"`
	RXErrors       uint64 `json:"rx_errors"`
	TXBytes        uint64 `json:"tx_bytes"`
	TXErrors       uint64 `json:"tx_errors"`
}

type flatKubeletVolumeMetric struct {
	Timestamp      int64  `json:"timestamp"`
	IdempotencyKey string `json:"idempotency_key"`
	PodUID         string `json:"pod_uid"`
	VolumeName     string `json:"volume_name"`
	PVCName        string `json:"pvc_name"`
	PVCNamespace   string `json:"pvc_namespace"`
	Abnormal       bool   `json:"abnormal"`
	AvailableBytes uint64 `json:"available_bytes"`
	CapacityBytes  uint64 `json:"capacity_bytes"`
	UsedBytes      uint64 `json:"used_bytes"`
	InodesFree     uint64 `json:"inodes_free"`
	Inodes         uint64 `json:"inodes"`
	InodesUsed     uint64 `json:"inodes_used"`
}

func (es *StatsFlattener) Flatten(ctx context.Context, msg *pb.KubernetesKubeletStats) error {
	if msg == nil {
		return nil
	}

	timestamp := time.Now()

	// --- NODE METRICS ---
	var knm *flatKubeletNodeMetric
	if msg.Node != nil {
		knm = &flatKubeletNodeMetric{
			IdempotencyKey: msg.Metadata.IdempotencyKey,
			OrganizationID: msg.GetEnrichment().GetOrganizationId(),
			ClusterID:      msg.GetEnrichment().GetClusterId(),
			ReceivedAtUnix: msg.GetEnrichment().GetReceivedAtUnix(),
			Timestamp:      timestamp.Unix(),
			NodeName:       msg.Node.GetNodeName(),
			StartTime:      safeTime(msg.Node.GetStartTime()),

			CPUUsageNanoCores:       safeU64(msg.Node.GetCpu().GetUsageNanoCores()),
			CPUUsageCoreNanoseconds: safeU64(msg.Node.GetCpu().GetUsageCoreNanoSeconds()),
			CPUPsiFullAvg10:         safeF64(msg.Node.GetCpu().GetPsi().GetFull().GetAvg10()),
			CPUPsiSomeAvg10:         safeF64(msg.Node.GetCpu().GetPsi().GetSome().GetAvg10()),

			MemoryAvailableBytes:  safeU64(msg.Node.GetMemory().GetAvailableBytes()),
			MemoryUsageBytes:      safeU64(msg.Node.GetMemory().GetUsageBytes()),
			MemoryWorkingSetBytes: safeU64(msg.Node.GetMemory().GetWorkingSetBytes()),
			MemoryRSSBytes:        safeU64(msg.Node.GetMemory().GetRssBytes()),
			MemoryPageFaults:      safeU64(msg.Node.GetMemory().GetPageFaults()),
			MemoryMajorPageFaults: safeU64(msg.Node.GetMemory().GetMajorPageFaults()),
			MemoryPsiFullAvg10:    safeF64(msg.Node.GetMemory().GetPsi().GetFull().GetAvg10()),
			MemoryPsiSomeAvg10:    safeF64(msg.Node.GetMemory().GetPsi().GetSome().GetAvg10()),

			IOPsiFullAvg10: safeF64(msg.Node.GetIo().GetPsi().GetFull().GetAvg10()),
			IOPsiSomeAvg10: safeF64(msg.Node.GetIo().GetPsi().GetSome().GetAvg10()),

			FSAvailableBytes: safeU64(msg.Node.GetFs().GetAvailableBytes()),
			FSCapacityBytes:  safeU64(msg.Node.GetFs().GetCapacityBytes()),
			FSUsedBytes:      safeU64(msg.Node.GetFs().GetUsedBytes()),

			SwapAvailableBytes: safeU64(msg.Node.GetSwap().GetSwapAvailableBytes()),
			SwapUsageBytes:     safeU64(msg.Node.GetSwap().GetSwapUsageBytes()),
		}

		if b, err := json.Marshal(knm); err == nil {
			if err := es.kc.Publish(ctx, StoreKubeletNodeMetricsTopic, b); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// --- POD / CONTAINER / VOLUME / NETWORK METRICS ---
	for _, pod := range msg.GetPods() {
		// Containers
		for _, container := range pod.GetContainers() {
			accelerators := make(map[string]any, len(container.GetAccelerators()))
			for _, acc := range container.GetAccelerators() {
				if acc == nil {
					continue
				}
				if d, err := json.Marshal(acc); err == nil {
					accelerators[acc.GetId()] = string(d)
				} else {
					return err
				}
			}

			kcm := &flatKubeletContainerMetric{
				Timestamp:               timestamp.Unix(),
				IdempotencyKey:          msg.Metadata.IdempotencyKey,
				PodUID:                  pod.GetPodRef().GetUid(),
				ContainerName:           container.GetName(),
				ContainerStartTime:      safeTime(container.GetStartTime()),
				CPUUsageNanoCores:       safeU64(container.GetCpu().GetUsageNanoCores()),
				CPUUsageCoreNanoseconds: safeU64(container.GetCpu().GetUsageCoreNanoSeconds()),
				MemoryAvailableBytes:    safeU64(container.GetMemory().GetAvailableBytes()),
				MemoryUsageBytes:        safeU64(container.GetMemory().GetUsageBytes()),
				MemoryWorkingSetBytes:   safeU64(container.GetMemory().GetWorkingSetBytes()),
				SwapAvailableBytes:      safeU64(container.GetSwap().GetSwapAvailableBytes()),
				SwapUsageBytes:          safeU64(container.GetSwap().GetSwapUsageBytes()),
				RootFSAvailableBytes:    safeU64(container.GetRootfs().GetAvailableBytes()),
				RootFSUsedBytes:         safeU64(container.GetRootfs().GetUsedBytes()),
				LogsUsedBytes:           safeU64(container.GetLogs().GetUsedBytes()),
				Accelerator:             accelerators,
				UserMetrics:             map[string]any{},
			}
			if b, err := json.Marshal(kcm); err == nil {
				if err := es.kc.Publish(ctx, StoreKubeletContainerMetricsTopic, b); err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// Volumes
		for _, volume := range pod.GetVolumes() {
			kvm := &flatKubeletVolumeMetric{
				Timestamp:      timestamp.Unix(),
				IdempotencyKey: msg.Metadata.IdempotencyKey,
				PodUID:         pod.GetPodRef().GetUid(),
				VolumeName:     volume.GetName(),
				AvailableBytes: safeU64(volume.GetFsStats().GetAvailableBytes()),
				CapacityBytes:  safeU64(volume.GetFsStats().GetCapacityBytes()),
				UsedBytes:      safeU64(volume.GetFsStats().GetUsedBytes()),
				InodesFree:     safeU64(volume.GetFsStats().GetInodesFree()),
				Inodes:         safeU64(volume.GetFsStats().GetInodes()),
				InodesUsed:     safeU64(volume.GetFsStats().GetInodesUsed()),
				PVCName:        volume.GetPvcRef().GetName(),
				PVCNamespace:   volume.GetPvcRef().GetNamespace(),
				Abnormal:       volume.GetVolumeHealthStats().GetAbnormal(),
			}
			if b, err := json.Marshal(kvm); err == nil {
				if err := es.kc.Publish(ctx, StoreKubeletVolumeMetricsTopic, b); err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// Network
		if pod.GetNetwork() != nil {
			knm := &flatKubeletNetworkMetric{
				Timestamp:      timestamp.Unix(),
				IdempotencyKey: msg.Metadata.IdempotencyKey,
				PodUID:         pod.GetPodRef().GetUid(),
				InterfaceName:  pod.GetNetwork().GetInterfaceStats().GetName(),
				RXBytes:        safeU64(pod.GetNetwork().GetInterfaceStats().GetRxBytes()),
				RXErrors:       safeU64(pod.GetNetwork().GetInterfaceStats().GetRxErrors()),
				TXBytes:        safeU64(pod.GetNetwork().GetInterfaceStats().GetTxBytes()),
				TXErrors:       safeU64(pod.GetNetwork().GetInterfaceStats().GetTxErrors()),
			}
			if b, err := json.Marshal(knm); err == nil {
				if err := es.kc.Publish(ctx, StoreKubeletNetworkMetricsTopic, b); err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// Pod
		kpm := &flatKubeletPodMetric{
			Timestamp:               timestamp.Unix(),
			IdempotencyKey:          msg.Metadata.IdempotencyKey,
			PodUID:                  pod.GetPodRef().GetUid(),
			PodName:                 pod.GetPodRef().GetName(),
			PodNamespace:            pod.GetPodRef().GetNamespace(),
			StartTime:               safeTime(pod.GetStartTime()),
			CPUUsageNanoCores:       safeU64(pod.GetCpu().GetUsageNanoCores()),
			CPUUsageCoreNanoseconds: safeU64(pod.GetCpu().GetUsageCoreNanoSeconds()),
			MemoryUsageBytes:        safeU64(pod.GetMemory().GetUsageBytes()),
			MemoryWorkingSetBytes:   safeU64(pod.GetMemory().GetWorkingSetBytes()),
			NetworkRxBytes:          safeU64(pod.GetNetwork().GetInterfaceStats().GetRxBytes()),
			NetworkTxBytes:          safeU64(pod.GetNetwork().GetInterfaceStats().GetTxBytes()),
			ProcessCount:            safeU64(pod.GetProcessStats().GetProcessCount()),
			SwapAvailableBytes:      safeU64(pod.GetSwap().GetSwapAvailableBytes()),
			SwapUsageBytes:          safeU64(pod.GetSwap().GetSwapUsageBytes()),
		}
		if b, err := json.Marshal(kpm); err == nil {
			if err := es.kc.Publish(ctx, StoreKubeletPodMetricsTopic, b); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
