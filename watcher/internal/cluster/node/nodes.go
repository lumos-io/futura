package node

import (
	"fmt"
	"strings"
	"time"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"github.com/opisvigilant/futura/watcher/pkg/strcase"
	"github.com/opisvigilant/futura/watcher/utils"
	conventions "go.opentelemetry.io/otel/semconv/v1.18.0"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// Keys for node metadata and entity attributes. These are NOT used by resource attributes.
	nodeCreationTime       = "node.creation_timestamp"
	k8sNodeConditionPrefix = "k8s.node.condition"
)

// Transform transforms the node to remove the fields that we don't use to reduce RAM utilization.
// IMPORTANT: Make sure to update this function before using new node fields.
func Transform(node *corev1.Node) *corev1.Node {
	newNode := &corev1.Node{
		ObjectMeta: metadata.TransformObjectMeta(node.ObjectMeta),
		Status: corev1.NodeStatus{
			Allocatable: node.Status.Allocatable,
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
				ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
				OSImage:                 node.Status.NodeInfo.OSImage,
				OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
			},
		},
	}
	for _, c := range node.Status.Conditions {
		newNode.Status.Conditions = append(newNode.Status.Conditions, corev1.NodeCondition{
			Type:   c.Type,
			Status: c.Status,
		})
	}
	return newNode
}

func RecordMetrics(node *corev1.Node, ts time.Time) *pb.KubernetesClusterObject {
	obj := &pb.KubernetesClusterObject{
		Timestamp:   timestamppb.New(ts),
		Uid:         string(node.UID),
		Name:        node.Name,
		Kind:        "Node",
		Labels:      node.Labels,
		Annotations: node.Annotations,
	}

	// Node Info
	obj.KubeletVersion = node.Status.NodeInfo.KubeletVersion
	obj.OsType = node.Status.NodeInfo.OperatingSystem
	obj.OsImage = node.Status.NodeInfo.OSImage

	runtime, version := parseContainerRuntime(node.Status.NodeInfo.ContainerRuntimeVersion)
	obj.ContainerRuntime = runtime
	obj.ContainerRuntimeVersion = version

	// Node Conditions
	for _, cond := range node.Status.Conditions {
		obj.Conditions = append(obj.Conditions, &pb.NodeCondition{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		})
	}

	// Allocatable Resources
	alloc := &pb.AllocatableResources{
		Others: make(map[string]string),
	}
	for res, quantity := range node.Status.Allocatable {
		val := quantity.String()
		switch res {
		case corev1.ResourceCPU:
			alloc.Cpu = val
		case corev1.ResourceMemory:
			alloc.Memory = val
		case corev1.ResourcePods:
			alloc.Pods = val
		case corev1.ResourceEphemeralStorage:
			alloc.EphemeralStorage = val
		default:
			alloc.Others[string(res)] = val
		}
	}
	obj.Allocatable = alloc

	// Cloud Metadata
	obj.CloudMetadata = extractCloudMetadata(node)

	return obj
}

func parseContainerRuntime(runtimeStr string) (string, string) {
	// e.g. "docker://20.10.7" or "containerd://1.6.21"
	parts := strings.Split(runtimeStr, "://")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

var nodeConditionValues = map[corev1.ConditionStatus]int64{
	corev1.ConditionTrue:    1,
	corev1.ConditionFalse:   0,
	corev1.ConditionUnknown: -1,
}

func nodeConditionValue(node *corev1.Node, condType corev1.NodeConditionType) int64 {
	status := corev1.ConditionUnknown
	for _, c := range node.Status.Conditions {
		if c.Type == condType {
			status = c.Status
			break
		}
	}
	return nodeConditionValues[status]
}

func GetMetadata(node *corev1.Node) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	meta := utils.MergeStringMaps(map[string]string{}, node.Labels)

	meta[string(conventions.K8SNodeNameKey)] = node.Name
	meta[nodeCreationTime] = node.GetCreationTimestamp().Format(time.RFC3339)

	// Node can have many additional conditions (gke has 18 on v1.29). Bad thresholds/implementations
	// of custom conditions can cause value to oscillate between true/false frequently. So, only sending the node
	// pressure conditions that are set by kubelet to avoid noise.
	// https://pkg.go.dev/k8s.io/api/core/v1#NodeConditionType
	kubeletConditions := map[corev1.NodeConditionType]struct{}{
		corev1.NodeReady:              {},
		corev1.NodeMemoryPressure:     {},
		corev1.NodeDiskPressure:       {},
		corev1.NodePIDPressure:        {},
		corev1.NodeNetworkUnavailable: {},
	}

	for _, c := range node.Status.Conditions {
		if _, ok := kubeletConditions[c.Type]; ok {
			meta[fmt.Sprintf("%s_%s", k8sNodeConditionPrefix, strcase.ToSnake(string(c.Type)))] = strings.ToLower(string(c.Status))
		}
	}

	nodeID := metadata.ResourceID(node.UID)
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		nodeID: {
			EntityType:    "k8s.node",
			ResourceIDKey: string(conventions.K8SNodeUIDKey),
			ResourceID:    nodeID,
			Metadata:      meta,
		},
	}
}

func getContainerRuntimeInfo(rawInfo string) (runtime string, version string) {
	// Kubelet reports container runtime version in the following format:
	// <runtime-name>://<version>
	parts := strings.Split(rawInfo, "://")

	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func getNodeConditionMetric(nodeConditionTypeValue string) string {
	return "k8s.node.condition_" + strcase.ToSnake(nodeConditionTypeValue)
}

func getNodeAllocatableUnit(res corev1.ResourceName) string {
	switch res {
	case corev1.ResourceCPU:
		return "{cpu}"
	case corev1.ResourceMemory, corev1.ResourceEphemeralStorage, corev1.ResourceStorage:
		return "By"
	case corev1.ResourcePods:
		return "{pod}"
	default:
		return fmt.Sprintf("{%s}", string(res))
	}
}

func setNodeAllocatableValue(res corev1.ResourceName, q resource.Quantity) float64 {
	switch res {
	case corev1.ResourceCPU:
		return float64(q.MilliValue()) / 1000.0
	default:
		return float64(q.Value())
	}
}

func getNodeAllocatableMetric(nodeAllocatableTypeValue string) string {
	return "k8s.node.allocatable_" + strcase.ToSnake(nodeAllocatableTypeValue)
}

// extractCloudMetadata extracts cloud provider metadata from node labels and annotations
func extractCloudMetadata(node *corev1.Node) *pb.CloudNodeMetadata {
	metadata := &pb.CloudNodeMetadata{}

	// Extract cloud provider from common labels
	if provider := node.Labels["cloud.google.com/gke-nodepool"]; provider != "" {
		metadata.CloudProvider = "gcp"
	} else if provider := node.Labels["eks.amazonaws.com/nodegroup"]; provider != "" {
		metadata.CloudProvider = "aws"
	} else if provider := node.Labels["kubernetes.azure.com/agentpool"]; provider != "" {
		metadata.CloudProvider = "azure"
	} else if provider := node.Labels["alibabacloud.com/nodepool"]; provider != "" {
		metadata.CloudProvider = "alibaba"
	} else if provider := node.Labels["doks.digitalocean.com/node-pool"]; provider != "" {
		metadata.CloudProvider = "digitalocean"
	} else if _, ok := node.Labels["io.x-k8s.io/kind-node"]; ok {
		metadata.CloudProvider = "kind"
	} else {
		// Try to infer from other labels
		if _, ok := node.Labels["node.kubernetes.io/instance-type"]; ok {
			// This is likely a cloud node, try to detect provider
			if zone := node.Labels["topology.kubernetes.io/zone"]; zone != "" {
				if strings.Contains(zone, "us-") || strings.Contains(zone, "eu-") || strings.Contains(zone, "ap-") {
					if strings.Contains(zone, "-") && len(strings.Split(zone, "-")) >= 3 {
						metadata.CloudProvider = "aws"
					} else if strings.Contains(zone, "-") && len(strings.Split(zone, "-")) == 2 {
						metadata.CloudProvider = "gcp"
					}
				}
			}
		}
	}

	// Extract instance type
	if instanceType := node.Labels["node.kubernetes.io/instance-type"]; instanceType != "" {
		metadata.InstanceType = instanceType
		metadata.InstanceFamily = extractInstanceFamily(instanceType)
	} else if instanceType := node.Labels["beta.kubernetes.io/instance-type"]; instanceType != "" {
		metadata.InstanceType = instanceType
		metadata.InstanceFamily = extractInstanceFamily(instanceType)
	}

	// Extract availability zone
	if zone := node.Labels["topology.kubernetes.io/zone"]; zone != "" {
		metadata.AvailabilityZone = zone
	} else if zone := node.Labels["failure-domain.beta.kubernetes.io/zone"]; zone != "" {
		metadata.AvailabilityZone = zone
	}

	// Extract capacity type (spot vs on-demand)
	if capacityType := node.Labels["karpenter.sh/capacity-type"]; capacityType != "" {
		metadata.CapacityType = capacityType
	} else if capacityType := node.Labels["eks.amazonaws.com/capacityType"]; capacityType != "" {
		metadata.CapacityType = capacityType
	} else if _, ok := node.Labels["cloud.google.com/gke-preemptible"]; ok {
		metadata.CapacityType = "preemptible"
	} else if _, ok := node.Labels["cloud.google.com/gke-spot"]; ok {
		metadata.CapacityType = "spot"
	} else {
		metadata.CapacityType = "on-demand"
	}

	// Extract resource specifications from allocatable
	if cpu := node.Status.Allocatable[corev1.ResourceCPU]; !cpu.IsZero() {
		metadata.CpuCores = float64(cpu.MilliValue()) / 1000.0
	}

	if memory := node.Status.Allocatable[corev1.ResourceMemory]; !memory.IsZero() {
		metadata.MemoryGb = float64(memory.Value()) / (1024 * 1024 * 1024)
	}

	// Extract instance ID from provider ID
	if node.Spec.ProviderID != "" {
		metadata.InstanceId = extractInstanceId(node.Spec.ProviderID)
	}

	// Extract launch time from node creation timestamp
	if !node.CreationTimestamp.IsZero() {
		metadata.LaunchTime = node.CreationTimestamp.Format(time.RFC3339)
	}

	return metadata
}

// extractInstanceFamily extracts the instance family from instance type
func extractInstanceFamily(instanceType string) string {
	if instanceType == "" {
		return ""
	}

	// AWS: m5.large -> m5, c5n.xlarge -> c5n
	if parts := strings.Split(instanceType, "."); len(parts) >= 2 {
		return parts[0]
	}

	// GCP: e2-standard-4 -> e2-standard
	if strings.Contains(instanceType, "-") {
		parts := strings.Split(instanceType, "-")
		if len(parts) >= 2 {
			return strings.Join(parts[:2], "-")
		}
	}

	return instanceType
}

// extractInstanceId extracts instance ID from provider ID
func extractInstanceId(providerID string) string {
	if providerID == "" {
		return ""
	}

	// AWS: aws:///us-west-2a/i-1234567890abcdef0
	if strings.HasPrefix(providerID, "aws://") {
		parts := strings.Split(providerID, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// GCP: gce://project-id/zone/instance-name
	if strings.HasPrefix(providerID, "gce://") {
		parts := strings.Split(providerID, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Azure: azure:///subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm-name
	if strings.HasPrefix(providerID, "azure://") {
		parts := strings.Split(providerID, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	return providerID
}
