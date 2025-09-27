"""
Karpenter-style cluster scaling algorithms for Futura Engine.

This module implements intelligent cluster scaling decisions based on:
- Pending pod scheduling constraints
- Current cluster capacity and utilization
- Cost optimization principles
- Multi-cloud instance selection
- Bin packing algorithms for efficient resource allocation
"""

import logging
from typing import Dict, List, Optional, Tuple, Any
from dataclasses import dataclass
from datetime import datetime, timedelta
import math

from proto.gen.engine import engine_pb2
from storage.cluster_scaling_data_access import (
    ClusterScalingDataAccess, NodeInfo, PendingPod, ClusterCapacity
)

logger = logging.getLogger(__name__)


@dataclass
class InstanceTypeSpec:
    """Specification for a cloud instance type."""
    instance_type: str
    instance_family: str
    cpu_cores: float
    memory_gb: float
    storage_gb: float
    network_gbps: float
    cost_per_hour: float
    availability_zones: List[str]
    supported_capacity_types: List[str]  # ["on-demand", "spot", "preemptible"]


@dataclass
class ScalingDecision:
    """Result of a scaling analysis."""
    action_type: str  # "provision_nodes", "deprovision_nodes", "no_action"
    confidence: float
    reason: str
    urgency: float  # 0.0 to 1.0
    cost_impact: float  # USD per hour change

    # Node groups to provision/deprovision
    node_groups: List[engine_pb2.NodeGroupProvision]
    nodes_to_remove: List[str]

    # Additional analysis
    pods_that_will_schedule: int
    efficiency_gain: float
    waste_reduction: float
    disruption_risk: float


class KarpenterStyleScaler:
    """
    Implements Karpenter-style cluster scaling algorithms.

    Key principles:
    1. Just-in-time provisioning based on pending pods
    2. Bin packing optimization for efficient resource usage
    3. Cost-aware instance selection
    4. Multi-zone placement for availability
    5. Graceful deprovisioning of underutilized nodes
    """

    def __init__(self, data_access: ClusterScalingDataAccess):
        self.data_access = data_access

        # Default instance type catalog (would be loaded from cloud provider APIs)
        self.instance_catalog = self._load_instance_catalog()

    def _load_instance_catalog(self) -> Dict[str, InstanceTypeSpec]:
        """Load available instance types and their specifications."""

        # TODO: Load instance catalog from cloud provider APIs (AWS EC2, GCP Compute, Azure VM, etc.)
        catalog = {
            # AWS instances
            "m5.large": InstanceTypeSpec(
                instance_type="m5.large", instance_family="m5",
                cpu_cores=2, memory_gb=8, storage_gb=0, network_gbps=1.25,
                cost_per_hour=0.096, availability_zones=["us-east-1a", "us-east-1b", "us-east-1c"],
                supported_capacity_types=["on-demand", "spot"]
            ),
            "m5.xlarge": InstanceTypeSpec(
                instance_type="m5.xlarge", instance_family="m5",
                cpu_cores=4, memory_gb=16, storage_gb=0, network_gbps=2.5,
                cost_per_hour=0.192, availability_zones=["us-east-1a", "us-east-1b", "us-east-1c"],
                supported_capacity_types=["on-demand", "spot"]
            ),
            "c5.large": InstanceTypeSpec(
                instance_type="c5.large", instance_family="c5",
                cpu_cores=2, memory_gb=4, storage_gb=0, network_gbps=1.25,
                cost_per_hour=0.085, availability_zones=["us-east-1a", "us-east-1b", "us-east-1c"],
                supported_capacity_types=["on-demand", "spot"]
            ),
            "r5.large": InstanceTypeSpec(
                instance_type="r5.large", instance_family="r5",
                cpu_cores=2, memory_gb=16, storage_gb=0, network_gbps=1.25,
                cost_per_hour=0.126, availability_zones=["us-east-1a", "us-east-1b", "us-east-1c"],
                supported_capacity_types=["on-demand", "spot"]
            ),

            # Kind instances (for testing)
            "kind-worker": InstanceTypeSpec(
                instance_type="kind-worker", instance_family="kind",
                cpu_cores=2, memory_gb=4, storage_gb=20, network_gbps=1.0,
                cost_per_hour=0.0, availability_zones=["kind-zone"],
                supported_capacity_types=["on-demand"]
            ),
            "kind-worker-large": InstanceTypeSpec(
                instance_type="kind-worker-large", instance_family="kind",
                cpu_cores=4, memory_gb=8, storage_gb=40, network_gbps=1.0,
                cost_per_hour=0.0, availability_zones=["kind-zone"],
                supported_capacity_types=["on-demand"]
            )
        }

        return catalog

    async def analyze_scaling_decision(
        self,
        cluster_id: int,
        config: Optional[engine_pb2.ClusterOptimizationConfigRequest] = None
    ) -> ScalingDecision:
        """
        Analyze cluster state and determine optimal scaling actions.
        """
        logger.info(f"Analyzing scaling decision for cluster {cluster_id}")

        try:
            # Gather cluster telemetry data
            nodes = await self.data_access.get_cluster_nodes(cluster_id)
            pending_pods = await self.data_access.get_pending_pods(cluster_id)
            capacity = await self.data_access.get_cluster_capacity(cluster_id)
            cost_analysis = await self.data_access.get_cost_analysis(cluster_id)

            logger.info(f"Cluster state: {len(nodes)} nodes, {len(pending_pods)} pending pods, "
                        f"{capacity.cpu_utilization:.1%} CPU utilization")

            # 1. Check if we need to provision nodes for pending pods
            if pending_pods:
                return await self._analyze_provisioning_needs(
                    nodes, pending_pods, capacity, config, cost_analysis
                )

            # 2. Check if we can deprovision underutilized nodes
            if len(nodes) > 1:  # Don't scale down to 0 nodes
                return await self._analyze_deprovisioning_opportunities(
                    nodes, capacity, config, cost_analysis
                )

            # 3. No action needed
            return ScalingDecision(
                action_type="no_action",
                confidence=0.8,
                reason="Cluster is optimally sized - no pending pods and good utilization",
                urgency=0.0,
                cost_impact=0.0,
                node_groups=[],
                nodes_to_remove=[],
                pods_that_will_schedule=0,
                efficiency_gain=0.0,
                waste_reduction=0.0,
                disruption_risk=0.0
            )

        except Exception as e:
            logger.error(f"Failed to analyze scaling decision: {str(e)}")
            return ScalingDecision(
                action_type="no_action",
                confidence=0.0,
                reason=f"Analysis failed: {str(e)}",
                urgency=0.0,
                cost_impact=0.0,
                node_groups=[],
                nodes_to_remove=[],
                pods_that_will_schedule=0,
                efficiency_gain=0.0,
                waste_reduction=0.0,
                disruption_risk=0.1
            )

    async def _analyze_provisioning_needs(
        self,
        nodes: List[NodeInfo],
        pending_pods: List[PendingPod],
        capacity: ClusterCapacity,
        config: Optional[engine_pb2.ClusterOptimizationConfigRequest],
        cost_analysis: Dict[str, Any]
    ) -> ScalingDecision:
        """Analyze need for provisioning new nodes."""

        logger.info(
            f"Analyzing provisioning needs for {len(pending_pods)} pending pods")

        # Group pending pods by resource requirements and constraints
        pod_groups = self._group_pending_pods_by_requirements(pending_pods)

        # Find optimal instance types for each pod group using bin packing
        node_groups = []
        total_pods_scheduled = 0
        total_cost_impact = 0.0

        for group_requirements, pods_in_group in pod_groups.items():
            logger.debug(
                f"Processing pod group: {group_requirements}, {len(pods_in_group)} pods")

            # Find best instance types for this group
            optimal_instances = self._find_optimal_instance_types(
                group_requirements, len(pods_in_group), config
            )

            for instance_config in optimal_instances:
                node_group = self._create_node_group_provision(
                    instance_config, pods_in_group, config
                )
                node_groups.append(node_group)

                total_pods_scheduled += len(pods_in_group)
                instance_spec = self.instance_catalog.get(
                    instance_config["instance_type"])
                if instance_spec:
                    total_cost_impact += instance_spec.cost_per_hour * \
                        instance_config["count"]

        # Calculate urgency based on how long pods have been pending
        urgency = self._calculate_urgency(pending_pods)

        # Estimate efficiency gains
        efficiency_gain = min(0.3, len(pending_pods) /
                              max(capacity.total_pods, 1))

        return ScalingDecision(
            action_type="provision_nodes",
            confidence=0.85,
            reason=f"Need to provision {len(node_groups)} node groups for {len(pending_pods)} pending pods",
            urgency=urgency,
            cost_impact=total_cost_impact,
            node_groups=node_groups,
            nodes_to_remove=[],
            pods_that_will_schedule=total_pods_scheduled,
            efficiency_gain=efficiency_gain,
            waste_reduction=0.0,
            disruption_risk=0.05  # Low risk for adding nodes
        )

    async def _analyze_deprovisioning_opportunities(
        self,
        nodes: List[NodeInfo],
        capacity: ClusterCapacity,
        config: Optional[engine_pb2.ClusterOptimizationConfigRequest],
        cost_analysis: Dict[str, Any]
    ) -> ScalingDecision:
        """Analyze opportunities to remove underutilized nodes."""

        logger.info(
            f"Analyzing deprovisioning opportunities for {len(nodes)} nodes")

        # Find nodes that are candidates for removal
        removal_candidates = []
        total_cost_savings = 0.0

        for node in nodes:
            # Skip nodes with high utilization
            if node.cpu_utilization > 0.5 or node.memory_utilization > 0.5:
                continue

            # Skip nodes with many pods (would cause disruption)
            if node.pod_count > 5:
                continue

            # Skip very recently launched nodes (give them time to be useful)
            if node.launch_time and (datetime.utcnow() - node.launch_time).total_seconds() < 600:
                continue

            removal_candidates.append(node.name)
            total_cost_savings += node.cost_per_hour

        # Don't remove too many nodes at once
        max_removals = max(1, len(nodes) // 4)  # Remove at most 25% of nodes
        removal_candidates = removal_candidates[:max_removals]

        if not removal_candidates:
            return ScalingDecision(
                action_type="no_action",
                confidence=0.7,
                reason="No underutilized nodes found for removal",
                urgency=0.0,
                cost_impact=0.0,
                node_groups=[],
                nodes_to_remove=[],
                pods_that_will_schedule=0,
                efficiency_gain=0.0,
                waste_reduction=0.0,
                disruption_risk=0.0
            )

        # Calculate efficiency improvements
        waste_reduction = len(removal_candidates) / len(nodes)
        efficiency_gain = waste_reduction * 0.2  # Removing waste improves efficiency

        return ScalingDecision(
            action_type="deprovision_nodes",
            confidence=0.75,
            reason=f"Can remove {len(removal_candidates)} underutilized nodes to save costs",
            urgency=0.3,  # Cost optimization is moderately urgent
            cost_impact=-total_cost_savings,  # Negative cost = savings
            node_groups=[],
            nodes_to_remove=removal_candidates,
            pods_that_will_schedule=0,
            efficiency_gain=efficiency_gain,
            waste_reduction=waste_reduction,
            disruption_risk=0.15  # Some risk when removing nodes
        )

    def _group_pending_pods_by_requirements(
        self,
        pending_pods: List[PendingPod]
    ) -> Dict[Tuple, List[PendingPod]]:
        """Group pending pods by similar resource requirements and constraints."""

        groups = {}

        for pod in pending_pods:
            # Create a key based on resource requirements and constraints
            cpu_mcpu = self._parse_cpu_to_mcpu(pod.required_cpu)
            memory_mib = self._parse_memory_to_mib(pod.required_memory)

            # Round to common sizes for better grouping
            cpu_group = self._round_to_common_cpu(cpu_mcpu)
            memory_group = self._round_to_common_memory(memory_mib)

            # Include constraints in grouping
            constraints_key = (
                tuple(sorted(pod.node_selector_requirements)),
                tuple(sorted(pod.affinity_requirements)),
                tuple(sorted(pod.unsatisfied_tolerations)),
                pod.failure_domain
            )

            group_key = (cpu_group, memory_group, constraints_key)

            if group_key not in groups:
                groups[group_key] = []
            groups[group_key].append(pod)

        logger.debug(
            f"Grouped {len(pending_pods)} pods into {len(groups)} groups")
        return groups

    def _find_optimal_instance_types(
        self,
        group_requirements: Tuple,
        pod_count: int,
        config: Optional[engine_pb2.ClusterOptimizationConfigRequest]
    ) -> List[Dict[str, Any]]:
        """Find optimal instance types for a group of pods using bin packing."""

        cpu_requirement, memory_requirement, constraints = group_requirements

        # Filter instance types based on config preferences
        candidate_instances = []
        for instance_type, spec in self.instance_catalog.items():
            # Check if instance type is in preferred list
            if config and config.preferred_instance_types:
                if instance_type not in config.preferred_instance_types:
                    continue

            # Check if instance can fit the pod requirements
            # Reserve some capacity for system overhead (10% CPU, 20% memory)
            usable_cpu = spec.cpu_cores * 1000 * 0.9  # Convert to mcpu and reserve 10%
            usable_memory = spec.memory_gb * 1024 * 0.8  # Convert to MiB and reserve 20%

            if usable_cpu >= cpu_requirement and usable_memory >= memory_requirement:
                # Calculate efficiency score (lower is better)
                cpu_waste = (usable_cpu - cpu_requirement) / usable_cpu
                memory_waste = (
                    usable_memory - memory_requirement) / usable_memory
                cost_per_pod = spec.cost_per_hour / \
                    max(1, usable_cpu / cpu_requirement,
                        usable_memory / memory_requirement)

                efficiency_score = (cpu_waste + memory_waste) / \
                    2 + (cost_per_pod / 0.1)  # Normalize cost

                candidate_instances.append({
                    "instance_type": instance_type,
                    "spec": spec,
                    "efficiency_score": efficiency_score,
                    "pods_per_node": min(
                        int(usable_cpu / cpu_requirement),
                        int(usable_memory / memory_requirement)
                    ),
                    "cost_per_pod": cost_per_pod
                })

        if not candidate_instances:
            logger.warning(
                f"No suitable instance types found for requirements: {group_requirements}")
            return []

        # Sort by efficiency (best first)
        candidate_instances.sort(key=lambda x: x["efficiency_score"])

        # Use bin packing to find optimal configuration
        return self._bin_pack_instances(candidate_instances, pod_count, config)

    def _bin_pack_instances(
        self,
        candidates: List[Dict[str, Any]],
        pod_count: int,
        config: Optional[engine_pb2.ClusterOptimizationConfigRequest]
    ) -> List[Dict[str, Any]]:
        """Use bin packing algorithm to optimize instance selection."""

        # Simple first-fit decreasing algorithm
        # In production, could use more sophisticated algorithms

        selected_instances = []
        remaining_pods = pod_count

        # Try to use the most efficient instances first
        for candidate in candidates:
            if remaining_pods <= 0:
                break

            pods_per_node = candidate["pods_per_node"]
            nodes_needed = math.ceil(remaining_pods / pods_per_node)

            # Limit to reasonable number of nodes per instance type
            max_nodes_per_type = 10
            nodes_to_add = min(nodes_needed, max_nodes_per_type)

            if nodes_to_add > 0:
                selected_instances.append({
                    "instance_type": candidate["instance_type"],
                    "count": nodes_to_add,
                    "pods_per_node": pods_per_node,
                    "total_pods": nodes_to_add * pods_per_node
                })

                remaining_pods -= nodes_to_add * pods_per_node

        # If we still have remaining pods, add one more of the best instance type
        if remaining_pods > 0 and candidates:
            best_candidate = candidates[0]
            selected_instances.append({
                "instance_type": best_candidate["instance_type"],
                "count": 1,
                "pods_per_node": best_candidate["pods_per_node"],
                "total_pods": min(remaining_pods, best_candidate["pods_per_node"])
            })

        return selected_instances

    def _create_node_group_provision(
        self,
        instance_config: Dict[str, Any],
        pods: List[PendingPod],
        config: Optional[engine_pb2.ClusterOptimizationConfigRequest]
    ) -> engine_pb2.NodeGroupProvision:
        """Create a NodeGroupProvision proto message."""

        instance_type = instance_config["instance_type"]
        instance_spec = self.instance_catalog.get(instance_type)

        # Determine capacity type based on config
        capacity_type = "on-demand"
        if config and config.allow_spot and instance_spec:
            if "spot" in instance_spec.supported_capacity_types:
                capacity_type = "spot"

        # Select availability zone
        availability_zone = ""
        if instance_spec and instance_spec.availability_zones:
            # Simple selection
            availability_zone = instance_spec.availability_zones[0]

        # Generate group name
        group_name = f"{instance_type}-{capacity_type}-{len(pods)}pods"

        return engine_pb2.NodeGroupProvision(
            name=group_name,
            instance_types=[instance_type],
            count=instance_config["count"],
            capacity_type=capacity_type,
            availability_zone=availability_zone,
            labels={
                "futura.io/provisioned": "true",
                "futura.io/instance-type": instance_type,
                "futura.io/capacity-type": capacity_type
            },
            taints=[],
            reason=f"Provision {instance_config['count']} {instance_type} nodes for {len(pods)} pending pods",
            # Limit to first 5
            target_workloads=[
                f"{pod.namespace}/{pod.name}" for pod in pods[:5]]
        )

    def _calculate_urgency(self, pending_pods: List[PendingPod]) -> float:
        """Calculate urgency score based on how long pods have been pending."""
        if not pending_pods:
            return 0.0

        now = datetime.utcnow()
        urgency_scores = []

        for pod in pending_pods:
            time_pending = (now - pod.last_failure_time).total_seconds()

            # Convert to urgency score (0-1)
            if time_pending < 60:  # Less than 1 minute
                score = 1.0
            elif time_pending < 300:  # Less than 5 minutes
                score = 0.8
            elif time_pending < 900:  # Less than 15 minutes
                score = 0.6
            elif time_pending < 1800:  # Less than 30 minutes
                score = 0.4
            else:  # More than 30 minutes
                score = 0.9  # Very urgent for long-pending pods

            urgency_scores.append(score)

        return sum(urgency_scores) / len(urgency_scores)

    def _parse_cpu_to_mcpu(self, cpu_str: str) -> int:
        """Parse CPU requirement to millicores."""
        if not cpu_str:
            return 100  # Default minimum

        try:
            if cpu_str.endswith('m'):
                return int(cpu_str[:-1])
            else:
                return int(float(cpu_str) * 1000)
        except (ValueError, TypeError):
            return 100

    def _parse_memory_to_mib(self, memory_str: str) -> int:
        """Parse memory requirement to MiB."""
        if not memory_str:
            return 128  # Default minimum

        try:
            if memory_str.endswith('Mi'):
                return int(memory_str[:-2])
            elif memory_str.endswith('Gi'):
                return int(memory_str[:-2]) * 1024
            else:
                return 128
        except (ValueError, TypeError):
            return 128

    def _round_to_common_cpu(self, cpu_mcpu: int) -> int:
        """Round CPU requirement to common sizes for better grouping."""
        common_sizes = [100, 250, 500, 1000, 2000, 4000, 8000]

        for size in common_sizes:
            if cpu_mcpu <= size:
                return size

        return cpu_mcpu

    def _round_to_common_memory(self, memory_mib: int) -> int:
        """Round memory requirement to common sizes for better grouping."""
        common_sizes = [128, 256, 512, 1024, 2048, 4096, 8192]

        for size in common_sizes:
            if memory_mib <= size:
                return size

        return memory_mib
