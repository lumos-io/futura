package models

import "fmt"

type PodMetric struct {
	Timestamp     string `json:"timestamp"`
	Namespace     string `json:"namespace"`
	Pod           string `json:"pod"`
	Container     string `json:"container"`
	CPU_millicore int64  `json:"cpu_millicore"`
	Memory_bytes  int64  `json:"memory_bytes"`
}

func ParseCPU(cpuStr string) int64 {
	var val int64
	fmt.Sscanf(cpuStr, "%dm", &val)
	return val
}

func ParseMemory(memStr string) int64 {
	var val int64
	fmt.Sscanf(memStr, "%dMi", &val)
	return val * 1024 * 1024
}
