"""
Cluster scaling data access layer for Futura Engine.

This module provides data access methods for cluster-level scaling decisions,
leveraging telemetry data collected by the watcher including cloud metadata,
scheduling constraints, and cluster resource utilization.
"""

import logging
from typing import Dict, List, Optional, Tuple, Any
from datetime import datetime, timedelta
from dataclasses import dataclass

from .clickhouse_client import ClickHouseClient

logger = logging.getLogger(__name__)


@dataclass
class NodeInfo:
    """Information about a cluster node."""
    uid: str
    name: str
    instance_type: str
    instance_family: str
    capacity_type: str  # "on-demand", "spot", "preemptible"
    availability_zone: str
    cloud_provider: str
    cpu_cores: float
    memory_gb: float
    cost_per_hour: float
    launch_time: datetime

    # Resource allocation
    cpu_allocatable: float
    memory_allocatable_gb: float
    pods_allocatable: int

    # Current utilization
    cpu_utilization: float
    memory_utilization: float
    pod_count: int


@dataclass
class PendingPod:
    """Information about a pod that failed to schedule."""
    namespace: str
    name: str
    required_cpu: str
    required_memory: str
    required_storage: Optional[str]

    # Scheduling constraints
    node_selector_requirements: List[str]
    affinity_requirements: List[str]
    anti_affinity_conflicts: List[str]
    unsatisfied_tolerations: List[str]

    # Failure analysis
    insufficient_resource: str  # "cpu", "memory", "pods", "storage"
    nodes_considered: int
    nodes_rejected_resource: int
    nodes_rejected_affinity: int
    nodes_rejected_taints: int
    nodes_rejected_selector: int

    failure_domain: str
    last_failure_time: datetime


@dataclass
class ClusterCapacity:
    """Overall cluster capacity and utilization."""
    total_nodes: int
    total_cpu_cores: float
    total_memory_gb: float
    total_pods: int

    allocatable_cpu_cores: float
    allocatable_memory_gb: float
    allocatable_pods: int

    used_cpu_cores: float
    used_memory_gb: float
    used_pods: int

    cpu_utilization: float
    memory_utilization: float
    pod_utilization: float

    # Cloud provider breakdown
    providers: Dict[str, int]  # provider -> node count
    instance_types: Dict[str, int]  # instance_type -> node count
    capacity_types: Dict[str, int]  # capacity_type -> node count
    availability_zones: Dict[str, int]  # zone -> node count


class ClusterScalingDataAccess:
    """
    Data access layer for cluster scaling decisions.

    Provides methods to analyze cluster state, pending pods, and make
    intelligent scaling recommendations based on Karpenter principles.
    """

    def __init__(self, clickhouse_client: ClickHouseClient):
        self.client = clickhouse_client

    async def get_cluster_nodes(
        self,
        cluster_id: int,
        window_minutes: int = 5
    ) -> List[NodeInfo]:
        """
        Get comprehensive information about all nodes in the cluster.
        """
        try:
            end_time = datetime.utcnow()
            start_time = end_time - timedelta(minutes=window_minutes)

            # Get node information with cloud metadata
            query = """
            SELECT DISTINCT ON (ko.uid)
                ko.uid,
                ko.name,
                ko.kubelet_version,
                ko.os_type,
                kar.cpu as allocatable_cpu,
                kar.memory as allocatable_memory,
                kar.pods as allocatable_pods,

                -- Cloud metadata
                cm.instance_type,
                cm.instance_family,
                cm.capacity_type,
                cm.availability_zone,
                cm.cloud_provider,
                cm.cpu_cores,
                cm.memory_gb,
                cm.cost_per_hour,
                cm.launch_time

            FROM kubernetes_objects ko
            JOIN kubernetes_allocatable_resources kar ON ko.uid = kar.uid
            LEFT JOIN kubernetes_cloud_metadata cm ON ko.uid = cm.uid
            WHERE ko.cluster_id = {cluster_id}
                AND ko.kind = 'Node'
                AND ko.timestamp >= '{start_time}'
                AND ko.timestamp <= '{end_time}'
            ORDER BY ko.uid, ko.timestamp DESC
            """

            results = await self.client.execute_query(
                query,
                params={
                    "cluster_id": cluster_id,
                    "start_time": start_time.isoformat(),
                    "end_time": end_time.isoformat()
                },
                database="analytics"
            )

            nodes = []
            for row in results:
                # Get current utilization for this node
                utilization = await self._get_node_utilization(
                    cluster_id, row["uid"], window_minutes
                )

                node = NodeInfo(
                    uid=row["uid"],
                    name=row["name"],
                    instance_type=row.get("instance_type", "unknown"),
                    instance_family=row.get("instance_family", "unknown"),
                    capacity_type=row.get("capacity_type", "on-demand"),
                    availability_zone=row.get("availability_zone", "unknown"),
                    cloud_provider=row.get("cloud_provider", "unknown"),
                    cpu_cores=float(row.get("cpu_cores", 0)),
                    memory_gb=float(row.get("memory_gb", 0)),
                    cost_per_hour=float(row.get("cost_per_hour", 0)),
                    launch_time=row.get("launch_time", datetime.utcnow()),

                    cpu_allocatable=self._parse_cpu_quantity(
                        row.get("allocatable_cpu", "0")),
                    memory_allocatable_gb=self._parse_memory_quantity(
                        row.get("allocatable_memory", "0")) / 1024,
                    pods_allocatable=int(row.get("allocatable_pods", 0)),

                    cpu_utilization=utilization["cpu_utilization"],
                    memory_utilization=utilization["memory_utilization"],
                    pod_count=utilization["pod_count"]
                )
                nodes.append(node)

            logger.info(
                f"Retrieved {len(nodes)} nodes for cluster {cluster_id}")
            return nodes

        except Exception as e:
            logger.error(f"Failed to get cluster nodes: {str(e)}")
            return []

    async def get_pending_pods(
        self,
        cluster_id: int,
        window_minutes: int = 30
    ) -> List[PendingPod]:
        """
        Get pods that failed to schedule with detailed constraint analysis.
        """
        try:
            end_time = datetime.utcnow()
            start_time = end_time - timedelta(minutes=window_minutes)

            # Query for FailedScheduling events with scheduling constraints
            query = """
            SELECT
                ke.object_namespace as namespace,
                ke.object_name as name,
                ke.event_message,
                ke.event_timestamp,

                -- Scheduling constraints from our enhanced watcher data
                sc.required_cpu,
                sc.required_memory,
                sc.required_storage,
                sc.node_selector_requirements,
                sc.affinity_requirements,
                sc.anti_affinity_conflicts,
                sc.unsatisfied_tolerations,
                sc.insufficient_resource,
                sc.nodes_considered,
                sc.nodes_rejected_resource,
                sc.nodes_rejected_affinity,
                sc.nodes_rejected_taints,
                sc.nodes_rejected_selector,
                sc.failure_domain

            FROM kubernetes_events ke
            LEFT JOIN kubernetes_scheduling_constraints sc ON ke.uid = sc.event_uid
            WHERE ke.cluster_id = {cluster_id}
                AND ke.event_reason = 'FailedScheduling'
                AND ke.object_kind = 'Pod'
                AND ke.object_timestamp >= '{start_time_unix}'
                AND ke.object_timestamp <= '{end_time_unix}'
            ORDER BY ke.event_timestamp DESC
            LIMIT 100
            """

            results = await self.client.execute_query(
                query,
                params={
                    "cluster_id": cluster_id,
                    "start_time_unix": int(start_time.timestamp()),
                    "end_time_unix": int(end_time.timestamp())
                },
                database="analytics"
            )

            pending_pods = []
            for row in results:
                pod = PendingPod(
                    namespace=row["namespace"],
                    name=row["name"],
                    required_cpu=row.get("required_cpu", ""),
                    required_memory=row.get("required_memory", ""),
                    required_storage=row.get("required_storage"),

                    node_selector_requirements=row.get(
                        "node_selector_requirements", []),
                    affinity_requirements=row.get("affinity_requirements", []),
                    anti_affinity_conflicts=row.get(
                        "anti_affinity_conflicts", []),
                    unsatisfied_tolerations=row.get(
                        "unsatisfied_tolerations", []),

                    insufficient_resource=row.get("insufficient_resource", ""),
                    nodes_considered=row.get("nodes_considered", 0),
                    nodes_rejected_resource=row.get(
                        "nodes_rejected_resource", 0),
                    nodes_rejected_affinity=row.get(
                        "nodes_rejected_affinity", 0),
                    nodes_rejected_taints=row.get("nodes_rejected_taints", 0),
                    nodes_rejected_selector=row.get(
                        "nodes_rejected_selector", 0),

                    failure_domain=row.get("failure_domain", ""),
                    last_failure_time=datetime.fromtimestamp(
                        row["event_timestamp"])
                )
                pending_pods.append(pod)

            logger.info(
                f"Found {len(pending_pods)} pending pods for cluster {cluster_id}")
            return pending_pods

        except Exception as e:
            logger.error(f"Failed to get pending pods: {str(e)}")
            return []

    async def get_cluster_capacity(
        self,
        cluster_id: int
    ) -> ClusterCapacity:
        """
        Get overall cluster capacity and utilization statistics.
        """
        try:
            # Get aggregated cluster capacity
            query = """
            SELECT
                count() as total_nodes,
                sum(toFloat64OrDefault(kar.cpu, '0')) as total_cpu_cores,
                sum(toFloat64OrDefault(kar.memory, '0')) / 1024 / 1024 / 1024 as total_memory_gb,
                sum(toInt64OrDefault(kar.pods, '0')) as total_pods,

                -- Cloud provider breakdown
                groupArray(cm.cloud_provider) as providers,
                groupArray(cm.instance_type) as instance_types,
                groupArray(cm.capacity_type) as capacity_types,
                groupArray(cm.availability_zone) as zones

            FROM kubernetes_objects ko
            JOIN kubernetes_allocatable_resources kar ON ko.uid = kar.uid
            LEFT JOIN kubernetes_cloud_metadata cm ON ko.uid = cm.uid
            WHERE ko.cluster_id = {cluster_id}
                AND ko.kind = 'Node'
            ORDER BY ko.timestamp DESC
            LIMIT 1000
            """

            results = await self.client.execute_query(
                query,
                params={"cluster_id": cluster_id},
                database="analytics"
            )

            if not results:
                return ClusterCapacity(
                    total_nodes=0, total_cpu_cores=0, total_memory_gb=0, total_pods=0,
                    allocatable_cpu_cores=0, allocatable_memory_gb=0, allocatable_pods=0,
                    used_cpu_cores=0, used_memory_gb=0, used_pods=0,
                    cpu_utilization=0, memory_utilization=0, pod_utilization=0,
                    providers={}, instance_types={}, capacity_types={}, availability_zones={}
                )

            data = results[0]

            # TODO: Calculate actual utilization from running pod resource usage in ClickHouse
            total_cpu = data["total_cpu_cores"]
            total_memory = data["total_memory_gb"]
            total_pods = data["total_pods"]

            # TODO: Query kubelet_container_metrics to get actual resource usage from running pods
            used_cpu = total_cpu * 0.6  # Estimated 60% utilization
            used_memory = total_memory * 0.7  # Estimated 70% utilization
            used_pods = total_pods * 0.5  # Estimated 50% utilization

            # Count instances by provider/type/etc
            def count_items(items_list):
                counts = {}
                for item in items_list:
                    if item:
                        counts[item] = counts.get(item, 0) + 1
                return counts

            return ClusterCapacity(
                total_nodes=data["total_nodes"],
                total_cpu_cores=total_cpu,
                total_memory_gb=total_memory,
                total_pods=total_pods,

                allocatable_cpu_cores=total_cpu,  # Simplified
                allocatable_memory_gb=total_memory,  # Simplified
                allocatable_pods=total_pods,  # Simplified

                used_cpu_cores=used_cpu,
                used_memory_gb=used_memory,
                used_pods=used_pods,

                cpu_utilization=used_cpu / max(total_cpu, 1),
                memory_utilization=used_memory / max(total_memory, 1),
                pod_utilization=used_pods / max(total_pods, 1),

                providers=count_items(data.get("providers", [])),
                instance_types=count_items(data.get("instance_types", [])),
                capacity_types=count_items(data.get("capacity_types", [])),
                availability_zones=count_items(data.get("zones", []))
            )

        except Exception as e:
            logger.error(f"Failed to get cluster capacity: {str(e)}")
            return ClusterCapacity(
                total_nodes=0, total_cpu_cores=0, total_memory_gb=0, total_pods=0,
                allocatable_cpu_cores=0, allocatable_memory_gb=0, allocatable_pods=0,
                used_cpu_cores=0, used_memory_gb=0, used_pods=0,
                cpu_utilization=0, memory_utilization=0, pod_utilization=0,
                providers={}, instance_types={}, capacity_types={}, availability_zones={}
            )

    async def _get_node_utilization(
        self,
        cluster_id: int,
        node_uid: str,
        window_minutes: int
    ) -> Dict[str, float]:
        """Get current utilization for a specific node."""

        end_time = datetime.utcnow()
        start_time = end_time - timedelta(minutes=window_minutes)

        # Get node metrics
        query = """
        SELECT
            avg(cpu_usage_nano_cores) / 1000000000 as avg_cpu_usage,
            max(cpu_usage_nano_cores) / 1000000000 as max_cpu_usage,
            avg(memory_usage_bytes) / 1024 / 1024 / 1024 as avg_memory_usage_gb,

            -- Count pods on this node (approximation)
            count(DISTINCT pod_uid) as pod_count

        FROM kubelet_node_metrics knm
        WHERE knm.node_name = (
            SELECT name FROM kubernetes_objects
            WHERE uid = '{node_uid}'
            LIMIT 1
        )
        AND knm.timestamp >= '{start_time}'
        AND knm.timestamp <= '{end_time}'
        """

        results = await self.client.execute_query(
            query,
            params={
                "node_uid": node_uid,
                "start_time": start_time.isoformat(),
                "end_time": end_time.isoformat()
            },
            database="analytics"
        )

        if not results:
            return {
                "cpu_utilization": 0.0,
                "memory_utilization": 0.0,
                "pod_count": 0
            }

        data = results[0]

        # TODO: Get node capacity to calculate utilization percentage
        # For now, assume utilization as provided
        return {
            "cpu_utilization": min(1.0, data.get("avg_cpu_usage", 0.0)),
            # Assume 16GB node
            "memory_utilization": min(1.0, data.get("avg_memory_usage_gb", 0.0) / 16),
            "pod_count": data.get("pod_count", 0)
        }

    def _parse_cpu_quantity(self, cpu_str: str) -> float:
        """Parse Kubernetes CPU quantity to cores."""
        if not cpu_str or cpu_str == "0":
            return 0.0

        cpu_str = cpu_str.strip()

        try:
            if cpu_str.endswith('m'):
                return float(cpu_str[:-1]) / 1000
            elif cpu_str.endswith('n'):
                return float(cpu_str[:-1]) / 1000000000
            else:
                return float(cpu_str)
        except (ValueError, TypeError):
            logger.warning(f"Failed to parse CPU quantity: {cpu_str}")
            return 1.0

    def _parse_memory_quantity(self, memory_str: str) -> float:
        """Parse Kubernetes memory quantity to MiB."""
        if not memory_str or memory_str == "0":
            return 0.0

        memory_str = memory_str.strip()

        try:
            if memory_str.endswith('Ki'):
                return float(memory_str[:-2]) / 1024
            elif memory_str.endswith('Mi'):
                return float(memory_str[:-2])
            elif memory_str.endswith('Gi'):
                return float(memory_str[:-2]) * 1024
            elif memory_str.endswith('Ti'):
                return float(memory_str[:-2]) * 1024 * 1024
            elif memory_str.endswith('k'):
                return float(memory_str[:-1]) / 1024
            elif memory_str.endswith('M'):
                return float(memory_str[:-1])
            elif memory_str.endswith('G'):
                return float(memory_str[:-1]) * 1000
            else:
                return float(memory_str) / 1024 / 1024
        except (ValueError, TypeError):
            logger.warning(f"Failed to parse memory quantity: {memory_str}")
            return 512.0

    async def get_cost_analysis(
        self,
        cluster_id: int,
        window_hours: int = 24
    ) -> Dict[str, Any]:
        """
        Get cluster cost analysis for cost-aware scaling decisions.
        """
        try:
            end_time = datetime.utcnow()
            start_time = end_time - timedelta(hours=window_hours)

            query = """
            SELECT
                cm.cloud_provider,
                cm.capacity_type,
                cm.instance_type,
                count() as node_count,
                avg(cm.cost_per_hour) as avg_cost_per_hour,
                sum(cm.cost_per_hour) as total_cost_per_hour

            FROM kubernetes_objects ko
            JOIN kubernetes_cloud_metadata cm ON ko.uid = cm.uid
            WHERE ko.cluster_id = {cluster_id}
                AND ko.kind = 'Node'
                AND ko.timestamp >= '{start_time}'
            GROUP BY cm.cloud_provider, cm.capacity_type, cm.instance_type
            ORDER BY total_cost_per_hour DESC
            """

            results = await self.client.execute_query(
                query,
                params={
                    "cluster_id": cluster_id,
                    "start_time": start_time.isoformat()
                },
                database="analytics"
            )

            total_hourly_cost = sum(row["total_cost_per_hour"]
                                    for row in results)
            estimated_monthly_cost = total_hourly_cost * 24 * 30

            return {
                "total_hourly_cost": total_hourly_cost,
                "estimated_monthly_cost": estimated_monthly_cost,
                "cost_breakdown": results,
                "analysis_window_hours": window_hours
            }

        except Exception as e:
            logger.error(f"Failed to get cost analysis: {str(e)}")
            return {
                "total_hourly_cost": 0,
                "estimated_monthly_cost": 0,
                "cost_breakdown": [],
                "analysis_window_hours": window_hours
            }
