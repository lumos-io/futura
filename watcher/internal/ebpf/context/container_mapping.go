package context

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ContainerInfo holds enriched container metadata for eBPF attribution
type ContainerInfo struct {
	ContainerID string            `json:"container_id"`
	PodName     string            `json:"pod_name"`
	PodUID      string            `json:"pod_uid"`
	Namespace   string            `json:"namespace"`
	NodeName    string            `json:"node_name"`
	AppName     string            `json:"app_name"`     // from app label
	AppVersion  string            `json:"app_version"`  // from version label
	ServiceName string            `json:"service_name"` // from service label
	CgroupID    uint64            `json:"cgroup_id"`
	ProcessIDs  []uint32          `json:"process_ids"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ContainerMapper manages the mapping between cgroups, PIDs, and container metadata
type ContainerMapper struct {
	kubeClient kubernetes.Interface

	// Maps for fast lookups
	cgroupToContainer map[uint64]*ContainerInfo
	pidToContainer    map[uint32]*ContainerInfo
	containerToCgroup map[string]uint64

	// Cache management
	mu          sync.RWMutex
	lastRefresh time.Time
	refreshTTL  time.Duration

	// Node-specific info
	nodeName string
}

// NewContainerMapper creates a new container mapping service
func NewContainerMapper(kubeClient kubernetes.Interface, nodeName string) *ContainerMapper {
	return &ContainerMapper{
		kubeClient:        kubeClient,
		cgroupToContainer: make(map[uint64]*ContainerInfo),
		pidToContainer:    make(map[uint32]*ContainerInfo),
		containerToCgroup: make(map[string]uint64),
		refreshTTL:        30 * time.Second,
		nodeName:          nodeName,
	}
}

// Start begins the container mapping service with periodic refresh
func (cm *ContainerMapper) Start(ctx context.Context) error {
	// Initial population
	if err := cm.RefreshContainerMappings(ctx); err != nil {
		log.Error().Err(err).Msg("Failed initial container mapping refresh")
		return err
	}

	// Periodic refresh
	ticker := time.NewTicker(cm.refreshTTL)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := cm.RefreshContainerMappings(ctx); err != nil {
					log.Error().Err(err).Msg("Failed periodic container mapping refresh")
				}
			}
		}
	}()

	return nil
}

// RefreshContainerMappings rebuilds the mapping tables by querying Kubernetes and cgroup filesystem
func (cm *ContainerMapper) RefreshContainerMappings(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	log.Debug().Msg("Refreshing container mappings")

	// Get all pods on this node
	pods, err := cm.kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", cm.nodeName),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	// Clear existing mappings
	cm.cgroupToContainer = make(map[uint64]*ContainerInfo)
	cm.pidToContainer = make(map[uint32]*ContainerInfo)
	cm.containerToCgroup = make(map[string]uint64)

	// Process each pod
	for _, pod := range pods.Items {
		if err := cm.processPod(&pod); err != nil {
			log.Error().Err(err).
				Str("pod", pod.Name).
				Str("namespace", pod.Namespace).
				Msg("Failed to process pod for container mapping")
			continue
		}
	}

	cm.lastRefresh = time.Now()
	log.Info().
		Int("containers", len(cm.cgroupToContainer)).
		Int("pids", len(cm.pidToContainer)).
		Msg("Container mappings refreshed")

	return nil
}

// processPod extracts container information and builds mappings
func (cm *ContainerMapper) processPod(pod *corev1.Pod) error {
	// Skip pods that aren't running
	if pod.Status.Phase != corev1.PodRunning {
		return nil
	}

	// Extract common pod metadata
	appName := getLabel(pod.Labels, "app", "app.kubernetes.io/name", "k8s-app")
	appVersion := getLabel(pod.Labels, "version", "app.kubernetes.io/version")
	serviceName := getLabel(pod.Labels, "service", "app.kubernetes.io/component")

	// Process each container in the pod
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Running == nil {
			continue // Skip non-running containers
		}

		containerInfo := &ContainerInfo{
			ContainerID: extractContainerID(containerStatus.ContainerID),
			PodName:     pod.Name,
			PodUID:      string(pod.UID),
			Namespace:   pod.Namespace,
			NodeName:    pod.Spec.NodeName,
			AppName:     appName,
			AppVersion:  appVersion,
			ServiceName: serviceName,
			Labels:      pod.Labels,
			Annotations: pod.Annotations,
			CreatedAt:   containerStatus.State.Running.StartedAt.Time,
			UpdatedAt:   time.Now(),
		}

		// Get cgroup ID for this container
		cgroupID, err := cm.getCgroupIDForContainer(containerInfo.ContainerID)
		if err != nil {
			log.Warn().Err(err).
				Str("container_id", containerInfo.ContainerID).
				Msg("Failed to get cgroup ID for container")
			continue
		}

		containerInfo.CgroupID = cgroupID

		// Get PIDs for this container
		pids, err := cm.getPIDsForContainer(containerInfo.ContainerID)
		if err != nil {
			log.Warn().Err(err).
				Str("container_id", containerInfo.ContainerID).
				Msg("Failed to get PIDs for container")
		} else {
			containerInfo.ProcessIDs = pids
		}

		// Update mappings
		cm.cgroupToContainer[cgroupID] = containerInfo
		cm.containerToCgroup[containerInfo.ContainerID] = cgroupID

		// Map all PIDs to this container
		for _, pid := range pids {
			cm.pidToContainer[pid] = containerInfo
		}

		log.Debug().
			Str("container_id", containerInfo.ContainerID).
			Str("pod", containerInfo.PodName).
			Str("app", containerInfo.AppName).
			Uint64("cgroup_id", cgroupID).
			Int("pids", len(pids)).
			Msg("Mapped container")
	}

	return nil
}

// getCgroupIDForContainer reads the cgroup ID from the container's cgroup filesystem
func (cm *ContainerMapper) getCgroupIDForContainer(containerID string) (uint64, error) {
	// Try different cgroup paths (Docker, containerd, CRI-O)
	_ = []string{
		fmt.Sprintf("/sys/fs/cgroup/systemd/docker/%s/cgroup.procs", containerID),
		fmt.Sprintf("/sys/fs/cgroup/systemd/system.slice/docker-%s.scope/cgroup.procs", containerID),
		fmt.Sprintf("/sys/fs/cgroup/systemd/system.slice/containerd.service/cgroup.procs"),
		fmt.Sprintf("/proc/self/cgroup"),
	}

	// For now, we'll use a simplified approach - read from /proc/cgroups
	// In production, this would need more sophisticated cgroup v1/v2 detection
	cgroupFile := fmt.Sprintf("/sys/fs/cgroup/memory/docker/%s/memory.usage_in_bytes", containerID)
	if _, err := os.Stat(cgroupFile); err == nil {
		// Container exists, return a hash of container ID as pseudo cgroup ID
		// TODO: Implement proper cgroup ID extraction
		return simpleHash(containerID), nil
	}

	// Fallback: use container ID hash
	return simpleHash(containerID), nil
}

// getPIDsForContainer gets all process IDs running in the container
func (cm *ContainerMapper) getPIDsForContainer(containerID string) ([]uint32, error) {
	// Try to read from cgroup procs file
	cgroupProcsPath := fmt.Sprintf("/sys/fs/cgroup/systemd/docker/%s/cgroup.procs", containerID)

	if data, err := os.ReadFile(cgroupProcsPath); err == nil {
		return parsePIDsFromProcs(string(data)), nil
	}

	// Fallback: try other cgroup paths
	altPaths := []string{
		fmt.Sprintf("/sys/fs/cgroup/systemd/system.slice/docker-%s.scope/cgroup.procs", containerID),
		fmt.Sprintf("/sys/fs/cgroup/memory/docker/%s/cgroup.procs", containerID),
	}

	for _, path := range altPaths {
		if data, err := os.ReadFile(path); err == nil {
			return parsePIDsFromProcs(string(data)), nil
		}
	}

	// If we can't find the cgroup procs, return empty list
	log.Debug().Str("container_id", containerID).Msg("Could not find cgroup procs for container")
	return []uint32{}, nil
}

// GetContainerByCgroupID retrieves container info by cgroup ID
func (cm *ContainerMapper) GetContainerByCgroupID(cgroupID uint64) (*ContainerInfo, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	container, exists := cm.cgroupToContainer[cgroupID]
	return container, exists
}

// GetContainerByPID retrieves container info by process ID
func (cm *ContainerMapper) GetContainerByPID(pid uint32) (*ContainerInfo, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	container, exists := cm.pidToContainer[pid]
	return container, exists
}

// GetContainerByID retrieves container info by container ID
func (cm *ContainerMapper) GetContainerByID(containerID string) (*ContainerInfo, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cgroupID, exists := cm.containerToCgroup[containerID]
	if !exists {
		return nil, false
	}

	return cm.cgroupToContainer[cgroupID], true
}

// GetAllContainers returns all currently mapped containers
func (cm *ContainerMapper) GetAllContainers() []*ContainerInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	containers := make([]*ContainerInfo, 0, len(cm.cgroupToContainer))
	for _, container := range cm.cgroupToContainer {
		containers = append(containers, container)
	}

	return containers
}

// Helper functions

func getLabel(labels map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, exists := labels[key]; exists {
			return value
		}
	}
	return ""
}

func extractContainerID(fullID string) string {
	// Remove container runtime prefix (docker://, containerd://, etc.)
	parts := strings.Split(fullID, "://")
	if len(parts) > 1 {
		return parts[1]
	}
	return fullID
}

func simpleHash(s string) uint64 {
	var hash uint64
	for _, c := range s {
		hash = hash*31 + uint64(c)
	}
	return hash
}

func parsePIDsFromProcs(data string) []uint32 {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	pids := make([]uint32, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if pid, err := strconv.ParseUint(line, 10, 32); err == nil {
			pids = append(pids, uint32(pid))
		}
	}

	return pids
}
