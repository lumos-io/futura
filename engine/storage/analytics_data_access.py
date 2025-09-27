"""
Analytics data access layer for Futura Engine.

This module replaces the Prometheus adapter functionality from the controller
with ClickHouse queries to extract metrics and features for RL training
and inference.
"""

import logging
from typing import Dict, List, Optional, Tuple, Any
from datetime import datetime, timedelta
from dataclasses import dataclass

from .clickhouse_client import ClickHouseClient

logger = logging.getLogger(__name__)


@dataclass
class ApplicationMetrics:
    """Metrics for a specific application instance."""
    # Resource utilization (normalized 0-1)
    cpu_utilization: float
    memory_utilization: float
    disk_io_usage: float  # MB/s

    # Application performance
    request_rate: float  # requests/second
    p95_latency_ms: float
    error_rate: float  # 0-1

    # Resource allocation
    cpu_limit_mcpu: int  # millicores
    memory_limit_mib: int  # MiB
    num_replicas: int

    # Network
    network_rx_bytes_rate: float  # bytes/second
    network_tx_bytes_rate: float  # bytes/second

    # Additional metrics
    restart_count: int
    ready_replicas: int
    available_replicas: int


class AnalyticsDataAccess:
    """
    Data access layer for Kubernetes metrics and telemetry.

    Replaces the Prometheus adapter from the controller with ClickHouse queries
    to extract features needed for RL training and inference.
    """

    def __init__(self, clickhouse_client: ClickHouseClient):
        self.client = clickhouse_client

    async def get_application_metrics(
        self,
        cluster_id: int,
        namespace: str,
        app_name: str,
        owner_kind: str = "Deployment",
        window_minutes: int = 5
    ) -> Optional[ApplicationMetrics]:
        """
        Extract comprehensive application metrics for RL state representation.

        This replaces the controller's observe_states() method with ClickHouse queries.
        """
        try:
            # Get time window
            end_time = datetime.utcnow()
            start_time = end_time - timedelta(minutes=window_minutes)

            # Get Kubernetes object info (deployment/statefulset/etc)
            k8s_info = await self._get_kubernetes_object_info(
                cluster_id, namespace, app_name, owner_kind
            )

            if not k8s_info:
                logger.warning(
                    f"No Kubernetes object found for {namespace}/{app_name}")
                return None

            # Get container resource limits and requests
            container_info = await self._get_container_resource_info(
                cluster_id, namespace, app_name, start_time, end_time
            )

            # Get container metrics (CPU, memory, network)
            container_metrics = await self._get_container_metrics_aggregated(
                cluster_id, namespace, app_name, start_time, end_time
            )

            # Combine all data into ApplicationMetrics
            metrics = await self._build_application_metrics(
                k8s_info, container_info, container_metrics
            )

            logger.debug(f"Extracted metrics for {namespace}/{app_name}: "
                         f"CPU={metrics.cpu_utilization:.2f}, "
                         f"Memory={metrics.memory_utilization:.2f}, "
                         f"Replicas={metrics.num_replicas}")

            return metrics

        except Exception as e:
            logger.error(
                f"Failed to get application metrics for {namespace}/{app_name}: {str(e)}")
            return None

    async def _get_kubernetes_object_info(
        self,
        cluster_id: int,
        namespace: str,
        app_name: str,
        owner_kind: str
    ) -> Optional[Dict[str, Any]]:
        """Get Kubernetes object information (Deployment, StatefulSet, etc)."""

        query = """
        SELECT
            replicas,
            ready_replicas,
            available_replicas,
            updated_replicas,
            current_replicas,
            status,
            labels,
            hpa_max_replicas,
            hpa_min_replicas
        FROM kubernetes_objects
        WHERE cluster_id = {cluster_id}
        AND namespace = '{namespace}'
        AND name = '{app_name}'
        AND kind = '{owner_kind}'
        ORDER BY timestamp DESC
        LIMIT 1
        """

        results = await self.client.execute_query(
            query,
            params={
                "cluster_id": cluster_id,
                "namespace": namespace,
                "app_name": app_name,
                "owner_kind": owner_kind
            },
            database="analytics"
        )

        return results[0] if results else None

    async def _get_container_resource_info(
        self,
        cluster_id: int,
        namespace: str,
        app_name: str,
        start_time: datetime,
        end_time: datetime
    ) -> Dict[str, Any]:
        """Get container resource limits and requests."""

        # TODO: Add cluster_id and namespace filtering once available in schema
        query = """
        SELECT
            container_name,
            cpu_limits,
            memory_limits,
            cpu_requests,
            memory_requests
        FROM kubernetes_containers kc
        JOIN kubernetes_objects ko ON kc.uid = ko.uid
        WHERE ko.cluster_id = {cluster_id}
        AND ko.namespace = '{namespace}'
        AND ko.name = '{app_name}'
        AND kc.timestamp >= '{start_time}'
        AND kc.timestamp <= '{end_time}'
        ORDER BY kc.timestamp DESC
        LIMIT 10
        """

        results = await self.client.execute_query(
            query,
            params={
                "cluster_id": cluster_id,
                "namespace": namespace,
                "app_name": app_name,
                "start_time": start_time.isoformat(),
                "end_time": end_time.isoformat()
            },
            database="analytics"
        )

        # Aggregate resource info across containers
        total_cpu_limits = 0
        total_memory_limits = 0
        container_count = 0

        for container in results:
            try:
                # Parse Kubernetes quantity strings (e.g., "500m", "1Gi")
                cpu_limit = self._parse_cpu_quantity(
                    container.get("cpu_limits", "0"))
                memory_limit = self._parse_memory_quantity(
                    container.get("memory_limits", "0"))

                total_cpu_limits += cpu_limit
                total_memory_limits += memory_limit
                container_count += 1
            except Exception as e:
                logger.warning(
                    f"Failed to parse container resources: {str(e)}")

        return {
            "total_cpu_limit_mcpu": total_cpu_limits,
            "total_memory_limit_mib": total_memory_limits,
            "container_count": container_count
        }

    async def _get_container_metrics_aggregated(
        self,
        cluster_id: int,
        namespace: str,
        app_name: str,
        start_time: datetime,
        end_time: datetime
    ) -> Dict[str, float]:
        """Get aggregated container metrics (CPU, memory, network)."""

        # TODO: Join with kubernetes_objects to filter by app_name once schema allows
        query = """
        SELECT
            -- CPU metrics (nano cores to millicores)
            avg(cpu_usage_nano_cores) / 1000000 as avg_cpu_usage_mcpu,
            max(cpu_usage_nano_cores) / 1000000 as max_cpu_usage_mcpu,
            quantile(0.95)(cpu_usage_nano_cores) / 1000000 as p95_cpu_usage_mcpu,

            -- Memory metrics (bytes to MiB)
            avg(memory_usage_bytes) / 1024 / 1024 as avg_memory_usage_mib,
            max(memory_usage_bytes) / 1024 / 1024 as max_memory_usage_mib,
            quantile(0.95)(memory_usage_bytes) / 1024 / 1024 as p95_memory_usage_mib,

            avg(memory_working_set_bytes) / 1024 / 1024 as avg_memory_working_set_mib,

            -- Storage metrics (bytes to MB)
            avg(rootfs_used_bytes) / 1024 / 1024 as avg_rootfs_used_mb,

            count() as sample_count
        FROM kubelet_container_metrics
        WHERE timestamp >= '{start_time}'
        AND timestamp <= '{end_time}'
        HAVING sample_count > 0
        """

        results = await self.client.execute_query(
            query,
            params={
                "start_time": start_time.isoformat(),
                "end_time": end_time.isoformat()
            },
            database="analytics"
        )

        if not results:
            # Return default metrics if no data found
            logger.warning(
                f"No container metrics found for {namespace}/{app_name}")
            return {
                "avg_cpu_usage_mcpu": 0,
                "p95_cpu_usage_mcpu": 0,
                "avg_memory_usage_mib": 0,
                "p95_memory_usage_mib": 0,
                "avg_memory_working_set_mib": 0,
                "avg_rootfs_used_mb": 0
            }

        return results[0]

    async def get_network_metrics(
        self,
        cluster_id: int,
        namespace: str,
        app_name: str,
        window_minutes: int = 5
    ) -> Dict[str, float]:
        """Get network metrics for application pods."""

        end_time = datetime.utcnow()
        start_time = end_time - timedelta(minutes=window_minutes)

        # TODO: Filter by application once schema allows proper joins
        query = """
        SELECT
            -- Network throughput (bytes/second over the window)
            (max(rx_bytes) - min(rx_bytes)) / {window_seconds} as avg_rx_bytes_per_sec,
            (max(tx_bytes) - min(tx_bytes)) / {window_seconds} as avg_tx_bytes_per_sec,

            -- Error rates
            (max(rx_errors) - min(rx_errors)) / {window_seconds} as rx_errors_per_sec,
            (max(tx_errors) - min(tx_errors)) / {window_seconds} as tx_errors_per_sec,

            count() as sample_count
        FROM kubelet_network_metrics
        WHERE timestamp >= '{start_time}'
        AND timestamp <= '{end_time}'
        GROUP BY pod_uid, interface_name
        HAVING sample_count >= 2
        """

        results = await self.client.execute_query(
            query,
            params={
                "start_time": start_time.isoformat(),
                "end_time": end_time.isoformat(),
                "window_seconds": window_minutes * 60
            },
            database="analytics"
        )

        if not results:
            return {
                "avg_rx_bytes_per_sec": 0,
                "avg_tx_bytes_per_sec": 0,
                "rx_errors_per_sec": 0,
                "tx_errors_per_sec": 0
            }

        # Aggregate across all pods/interfaces
        total_rx = sum(r["avg_rx_bytes_per_sec"] for r in results)
        total_tx = sum(r["avg_tx_bytes_per_sec"] for r in results)
        total_rx_errors = sum(r["rx_errors_per_sec"] for r in results)
        total_tx_errors = sum(r["tx_errors_per_sec"] for r in results)

        return {
            "avg_rx_bytes_per_sec": total_rx,
            "avg_tx_bytes_per_sec": total_tx,
            "rx_errors_per_sec": total_rx_errors,
            "tx_errors_per_sec": total_tx_errors
        }

    async def get_application_slo_metrics(
        self,
        cluster_id: int,
        namespace: str,
        app_name: str,
        window_minutes: int = 5
    ) -> Dict[str, float]:
        """
        Get application SLO metrics (latency, error rate, throughput).

        TODO: This requires custom application metrics to be ingested into ClickHouse.
        """
        # TODO: Implement when custom application metrics are available
        # This would query application-specific metrics like:
        # - P95 latency from application metrics or traces
        # - Error rate from application logs/metrics
        # - Request throughput from application metrics
        # - Integration with Prometheus, OpenTelemetry, or custom metrics endpoints

        logger.debug(
            f"SLO metrics not yet implemented for {namespace}/{app_name}")

        return {
            "p95_latency_ms": 100.0,  # Default values until real metrics available
            "error_rate": 0.01,       # Default 1% error rate
            "request_rate": 50.0      # Default 50 RPS
        }

    async def _build_application_metrics(
        self,
        k8s_info: Dict[str, Any],
        container_info: Dict[str, Any],
        container_metrics: Dict[str, float]
    ) -> ApplicationMetrics:
        """Build ApplicationMetrics from collected data."""

        # Resource utilization calculations
        cpu_limit_mcpu = container_info.get("total_cpu_limit_mcpu", 1000)
        memory_limit_mib = container_info.get("total_memory_limit_mib", 512)

        cpu_usage_mcpu = container_metrics.get("p95_cpu_usage_mcpu", 0)
        memory_usage_mib = container_metrics.get("p95_memory_usage_mib", 0)

        cpu_utilization = min(
            1.0, cpu_usage_mcpu / max(cpu_limit_mcpu, 1)) if cpu_limit_mcpu > 0 else 0
        memory_utilization = min(
            1.0, memory_usage_mib / max(memory_limit_mib, 1)) if memory_limit_mib > 0 else 0

        # TODO: Get real SLO metrics from application monitoring
        slo_metrics = {
            "p95_latency_ms": 100.0,
            "error_rate": 0.01,
            "request_rate": 50.0
        }

        return ApplicationMetrics(
            cpu_utilization=cpu_utilization,
            memory_utilization=memory_utilization,
            disk_io_usage=container_metrics.get("avg_rootfs_used_mb", 0),
            request_rate=slo_metrics["request_rate"],
            p95_latency_ms=slo_metrics["p95_latency_ms"],
            error_rate=slo_metrics["error_rate"],
            cpu_limit_mcpu=cpu_limit_mcpu,
            memory_limit_mib=memory_limit_mib,
            num_replicas=k8s_info.get("ready_replicas", 1),
            network_rx_bytes_rate=0,  # TODO: Implement from network metrics
            network_tx_bytes_rate=0,  # TODO: Implement from network metrics
            restart_count=0,  # TODO: Extract from container info
            ready_replicas=k8s_info.get("ready_replicas", 1),
            available_replicas=k8s_info.get("available_replicas", 1)
        )

    def _parse_cpu_quantity(self, cpu_str: str) -> int:
        """Parse Kubernetes CPU quantity to millicores."""
        if not cpu_str or cpu_str == "0":
            return 0

        cpu_str = cpu_str.strip()

        try:
            if cpu_str.endswith('m'):
                # Millicores (e.g., "500m")
                return int(cpu_str[:-1])
            elif cpu_str.endswith('n'):
                # Nanocores (e.g., "500000000n")
                return int(cpu_str[:-1]) // 1000000
            else:
                # Cores (e.g., "1", "0.5")
                return int(float(cpu_str) * 1000)
        except (ValueError, TypeError):
            logger.warning(f"Failed to parse CPU quantity: {cpu_str}")
            return 1000  # Default to 1 core

    def _parse_memory_quantity(self, memory_str: str) -> int:
        """Parse Kubernetes memory quantity to MiB."""
        if not memory_str or memory_str == "0":
            return 0

        memory_str = memory_str.strip()

        try:
            if memory_str.endswith('Ki'):
                return int(memory_str[:-2]) // 1024  # KiB to MiB
            elif memory_str.endswith('Mi'):
                return int(memory_str[:-2])  # MiB
            elif memory_str.endswith('Gi'):
                return int(memory_str[:-2]) * 1024  # GiB to MiB
            elif memory_str.endswith('Ti'):
                return int(memory_str[:-2]) * 1024 * 1024  # TiB to MiB
            elif memory_str.endswith('k'):
                return int(memory_str[:-1]) // 1024  # KB to MiB (approx)
            elif memory_str.endswith('M'):
                return int(memory_str[:-1])  # MB to MiB (approx)
            elif memory_str.endswith('G'):
                return int(memory_str[:-1]) * 1000  # GB to MiB (approx)
            else:
                # Bytes
                return int(memory_str) // 1024 // 1024  # Bytes to MiB
        except (ValueError, TypeError):
            logger.warning(f"Failed to parse memory quantity: {memory_str}")
            return 512  # Default to 512 MiB

    async def get_historical_metrics_for_training(
        self,
        cluster_id: int,
        namespace: str,
        app_name: str,
        hours_back: int = 24,
        sample_interval_minutes: int = 5
    ) -> List[Tuple[datetime, ApplicationMetrics]]:
        """
        Get historical metrics for RL training data collection.

        This replaces the ClickHouse query functionality needed for training
        data preparation.
        """
        end_time = datetime.utcnow()
        start_time = end_time - timedelta(hours=hours_back)

        # TODO: Implement efficient time-series query
        # For now, return empty list as this requires more complex aggregation
        logger.warning(
            "Historical metrics collection not yet fully implemented")

        return []

    async def check_cluster_capacity(
        self,
        cluster_id: int
    ) -> Dict[str, Any]:
        """
        Check cluster resource capacity and utilization.

        Used for capacity-aware scaling decisions.
        """
        query = """
        SELECT
            -- Node capacity aggregation
            sum(toFloat64OrDefault(cpu, '0')) as total_cpu_cores,
            sum(toFloat64OrDefault(memory, '0')) / 1024 / 1024 / 1024 as total_memory_gib,

            -- Available resources
            sum(toFloat64OrDefault(cpu, '0')) as available_cpu_cores,
            sum(toFloat64OrDefault(memory, '0')) / 1024 / 1024 / 1024 as available_memory_gib,

            count() as node_count
        FROM kubernetes_allocatable_resources kar
        JOIN kubernetes_objects ko ON kar.uid = ko.uid
        WHERE ko.cluster_id = {cluster_id}
        AND ko.kind = 'Node'
        ORDER BY kar.timestamp DESC
        LIMIT 50
        """

        results = await self.client.execute_query(
            query,
            params={"cluster_id": cluster_id},
            database="analytics"
        )

        if not results:
            return {
                "total_cpu_cores": 0,
                "total_memory_gib": 0,
                "available_cpu_cores": 0,
                "available_memory_gib": 0,
                "node_count": 0,
                "cpu_utilization": 0,
                "memory_utilization": 0
            }

        capacity = results[0]
        # TODO: Calculate actual utilization by aggregating kubelet_container_metrics data
        capacity["cpu_utilization"] = 0.0  # Requires aggregating actual pod usage
        capacity["memory_utilization"] = 0.0  # Requires aggregating actual pod usage

        return capacity
