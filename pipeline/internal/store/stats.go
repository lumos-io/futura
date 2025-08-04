package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/opisvigilant/futura/go-lib/stream"
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
)

const (
	StoreKubeletNodeMetricsTopic      = "store.kubelet.node.metrics"
	StoreKubeletPodMetricsTopic       = "store.kubelet.pod.metrics"
	StoreKubeletContainerMetricsTopic = "store.kubelet.container.metrics"
	StoreKubeletNetworkMetricsTopic   = "store.kubelet.network.metrics"
	StoreKubeletVolumeMetricsTopic    = "store.kubelet.volume.metrics"
)

type StatsSplitter struct {
	kc stream.Stream
}

func NewStatsSplitter(kc stream.Stream) *StatsSplitter {
	return &StatsSplitter{
		kc: kc,
	}
}

type flatKubeletNodeMetric struct {
	OrganizationID          uint32    `json:"organization_id"`
	ClusterID               int64     `json:"cluster_id"`
	ReceivedAtUnix          int64     `json:"received_at_unix"`
	Timestamp               time.Time `json:"timestamp"`
	NodeName                string    `json:"node_name"`
	StartTime               time.Time `json:"start_time"`
	CPUUsageNanoCores       uint64    `json:"cpu_usage_nano_cores"`
	CPUUsageCoreNanoseconds uint64    `json:"cpu_usage_core_nanoseconds"`
	CPUPsiFullAvg10         float64   `json:"cpu_psi_full_avg10"`
	CPUPsiSomeAvg10         float64   `json:"cpu_psi_some_avg10"`
	MemoryAvailableBytes    uint64    `json:"memory_available_bytes"`
	MemoryUsageBytes        uint64    `json:"memory_usage_bytes"`
	MemoryWorkingSetBytes   uint64    `json:"memory_working_set_bytes"`
	MemoryRSSBytes          uint64    `json:"memory_rss_bytes"`
	MemoryPageFaults        uint64    `json:"memory_page_faults"`
	MemoryMajorPageFaults   uint64    `json:"memory_major_page_faults"`
	MemoryPsiFullAvg10      float64   `json:"memory_psi_full_avg10"`
	MemoryPsiSomeAvg10      float64   `json:"memory_psi_some_avg10"`
	IOPsiFullAvg10          float64   `json:"io_psi_full_avg10"`
	IOPsiSomeAvg10          float64   `json:"io_psi_some_avg10"`
	FSAvailableBytes        uint64    `json:"fs_available_bytes"`
	FSCapacityBytes         uint64    `json:"fs_capacity_bytes"`
	FSUsedBytes             uint64    `json:"fs_used_bytes"`
	SwapAvailableBytes      uint64    `json:"swap_available_bytes"`
	SwapUsageBytes          uint64    `json:"swap_usage_bytes"`
}

type flatKubeletPodMetric struct {
	Timestamp               time.Time `json:"timestamp"`
	PodUID                  string    `json:"pod_uid"`
	PodName                 string    `json:"pod_name"`
	PodNamespace            string    `json:"pod_namespace"`
	StartTime               time.Time `json:"start_time"`
	CPUUsageNanoCores       uint64    `json:"cpu_usage_nano_cores"`
	CPUUsageCoreNanoseconds uint64    `json:"cpu_usage_core_nanoseconds"`
	MemoryAvailableBytes    uint64    `json:"memory_available_bytes"`
	MemoryUsageBytes        uint64    `json:"memory_usage_bytes"`
	MemoryWorkingSetBytes   uint64    `json:"memory_working_set_bytes"`
	NetworkRxBytes          uint64    `json:"network_rx_bytes"`
	NetworkTxBytes          uint64    `json:"network_tx_bytes"`
	ProcessCount            uint64    `json:"process_count"`
	SwapAvailableBytes      uint64    `json:"swap_available_bytes"`
	SwapUsageBytes          uint64    `json:"swap_usage_bytes"`
}

type flatKubeletContainerMetric struct {
	Timestamp               time.Time      `json:"timestamp"`
	PodUID                  string         `json:"pod_uid"`
	ContainerName           string         `json:"container_name"`
	ContainerStartTime      time.Time      `json:"container_start_time"`
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
	Timestamp     time.Time `json:"timestamp"`
	PodUID        string    `json:"pod_uid"`
	InterfaceName string    `json:"interface_name"`
	RXBytes       uint64    `json:"rx_bytes"`
	RXErrors      uint64    `json:"rx_errors"`
	TXBytes       uint64    `json:"tx_bytes"`
	TXErrors      uint64    `json:"tx_errors"`
}

type flatKubeletVolumeMetric struct {
	Timestamp      time.Time `json:"timestamp"`
	PodUID         string    `json:"pod_uid"`
	VolumeName     string    `json:"volume_name"`
	PVCName        string    `json:"pvc_name"`
	PVCNamespace   string    `json:"pvc_namespace"`
	Abnormal       bool      `json:"abnormal"`
	AvailableBytes uint64    `json:"available_bytes"`
	CapacityBytes  uint64    `json:"capacity_bytes"`
	UsedBytes      uint64    `json:"used_bytes"`
	InodesFree     uint64    `json:"inodes_free"`
	Inodes         uint64    `json:"inodes"`
	InodesUsed     uint64    `json:"inodes_used"`
}

func (es *StatsSplitter) Split(ctx context.Context, msg *pbst.KubernetesKubeletStats) error {
	timestamp := time.Now()
	knm := &flatKubeletNodeMetric{
		OrganizationID:          msg.Enrichment.OrganizationId,
		ClusterID:               msg.Enrichment.ClusterId,
		ReceivedAtUnix:          msg.Enrichment.ReceivedAtUnix,
		Timestamp:               timestamp,
		NodeName:                msg.Node.NodeName,
		StartTime:               msg.Node.StartTime.AsTime(),
		CPUUsageNanoCores:       msg.Node.Cpu.UsageNanoCores,
		CPUUsageCoreNanoseconds: msg.Node.Cpu.UsageCoreNanoSeconds,
		CPUPsiFullAvg10:         msg.Node.Cpu.Psi.Full.Avg10,
		CPUPsiSomeAvg10:         msg.Node.Cpu.Psi.Some.Avg10,
		MemoryAvailableBytes:    msg.Node.Memory.AvailableBytes,
		MemoryUsageBytes:        msg.Node.Memory.UsageBytes,
		MemoryWorkingSetBytes:   msg.Node.Memory.WorkingSetBytes,
		MemoryRSSBytes:          msg.Node.Memory.RssBytes,
		MemoryPageFaults:        msg.Node.Memory.PageFaults,
		MemoryMajorPageFaults:   msg.Node.Memory.MajorPageFaults,
		MemoryPsiFullAvg10:      msg.Node.Memory.Psi.Full.Avg10,
		MemoryPsiSomeAvg10:      msg.Node.Memory.Psi.Some.Avg10,
		IOPsiFullAvg10:          msg.Node.Io.Psi.Full.Avg10,
		IOPsiSomeAvg10:          msg.Node.Io.Psi.Some.Avg10,
		FSAvailableBytes:        msg.Node.Fs.AvailableBytes,
		FSCapacityBytes:         msg.Node.Fs.CapacityBytes,
		FSUsedBytes:             msg.Node.Fs.UsedBytes,
		SwapAvailableBytes:      msg.Node.Swap.SwapAvailableBytes,
		SwapUsageBytes:          msg.Node.Swap.SwapUsageBytes,
	}
	b, err := json.Marshal(knm)
	if err != nil {
		return err
	}
	if err := es.kc.Publish(ctx, StoreKubeletNodeMetricsTopic, b); err != nil {
		return err
	}

	var kpm *flatKubeletPodMetric
	var kcm *flatKubeletContainerMetric
	for _, pod := range msg.Pods {

		for _, container := range pod.Containers {
			accelerators := map[string]any{}
			for _, acc := range container.Accelerators {
				d, err := json.Marshal(acc)
				if err != nil {
					return err
				}
				accelerators[acc.Id] = string(d)
			}

			// udf := map[string]any{}
			// for _, f := range container.UserDefinedMetrics {

			// }

			kcm = &flatKubeletContainerMetric{
				Timestamp:               timestamp,
				PodUID:                  pod.PodRef.Uid,
				ContainerName:           container.Name,
				ContainerStartTime:      container.StartTime.AsTime(),
				CPUUsageNanoCores:       container.Cpu.UsageNanoCores,
				CPUUsageCoreNanoseconds: container.Cpu.UsageCoreNanoSeconds,
				MemoryAvailableBytes:    container.Memory.AvailableBytes,
				MemoryUsageBytes:        container.Memory.UsageBytes,
				MemoryWorkingSetBytes:   container.Memory.WorkingSetBytes,
				SwapAvailableBytes:      container.Swap.SwapAvailableBytes,
				SwapUsageBytes:          container.Swap.SwapUsageBytes,
				RootFSAvailableBytes:    container.Rootfs.AvailableBytes,
				RootFSUsedBytes:         container.Rootfs.UsedBytes,
				LogsUsedBytes:           container.Logs.UsedBytes,
				Accelerator:             accelerators,
				// UserMetrics:             container.UserDefinedMetrics,
			}
			b, err = json.Marshal(kcm)
			if err != nil {
				return err
			}
			if err := es.kc.Publish(ctx, StoreKubeletContainerMetricsTopic, b); err != nil {
				return err
			}
		}

		kpm = &flatKubeletPodMetric{
			Timestamp:               timestamp,
			PodUID:                  pod.PodRef.Uid,
			PodName:                 pod.PodRef.Name,
			PodNamespace:            pod.PodRef.Namespace,
			StartTime:               pod.StartTime.AsTime(),
			CPUUsageNanoCores:       pod.Cpu.UsageNanoCores,
			CPUUsageCoreNanoseconds: pod.Cpu.UsageCoreNanoSeconds,
			MemoryUsageBytes:        pod.Memory.UsageBytes,
			MemoryWorkingSetBytes:   pod.Memory.WorkingSetBytes,
			NetworkRxBytes:          pod.Network.InterfaceStats.RxBytes,
			NetworkTxBytes:          pod.Network.InterfaceStats.TxBytes,
			ProcessCount:            pod.ProcessStats.ProcessCount,
			SwapAvailableBytes:      pod.Swap.SwapAvailableBytes,
			SwapUsageBytes:          pod.Swap.SwapUsageBytes,
		}
		b, err = json.Marshal(kpm)
		if err != nil {
			return err
		}
		if err := es.kc.Publish(ctx, StoreKubeletPodMetricsTopic, b); err != nil {
			return err
		}
	}

	return nil
}
