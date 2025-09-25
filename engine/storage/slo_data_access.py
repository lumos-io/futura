"""
SLO and Cluster Configuration data access for Futura Engine.

This module handles persistence and retrieval of Service Level Objectives
and cluster optimization configurations that are not in the current
migration schema but are needed for the engine operation.
"""

import logging
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
from dataclasses import dataclass
import numpy as np

from .clickhouse_client import ClickHouseClient
from .analytics_data_access import AnalyticsDataAccess

logger = logging.getLogger(__name__)

# Import RL components for reward calculation
try:
    import sys
    import os
    sys.path.append(os.path.join(os.path.dirname(__file__), '..'))
    from rl_models.reward_functions import RewardCalculator
    from rl_models.state_action_space import ActionSpace, ActionType
except ImportError as e:
    logger.warning(f"Could not import RL components: {e}")
    RewardCalculator = None
    ActionSpace = None
    ActionType = None

logger = logging.getLogger(__name__)


@dataclass
class ClusterOptimizationConfig:
    """Cluster optimization configuration."""
    api_key: str
    cluster_id: str
    cloud_provider: str
    region: str
    cost_sensitivity: str
    monthly_budget: Optional[str]
    preferred_instance_types: List[str]
    allow_spot: bool
    max_spot_percentage: str
    created_at: datetime
    updated_at: datetime


@dataclass
class ServiceLevelObjective:
    """Service Level Objective definition."""
    api_key: str
    cluster_id: str
    namespace: str
    service_name: str
    target_p95_latency_ms: float
    target_error_rate: float
    target_throughput_rps: Optional[float]
    priority: str
    created_at: datetime
    updated_at: datetime


class SLODataAccess:
    """
    Data access layer for SLO and cluster configuration management.

    Note: These tables are not in the current migrations but are needed
    for the engine functionality. They should be added to the migration schema.
    """

    def __init__(self, clickhouse_client: ClickHouseClient):
        self.client = clickhouse_client

    async def store_cluster_optimization_config(
        self,
        api_key: str,
        cluster_id: str,
        cloud_provider: str,
        region: str,
        cost_sensitivity: str,
        monthly_budget: Optional[str] = None,
        preferred_instance_types: List[str] = None,
        allow_spot: bool = False,
        max_spot_percentage: str = "0"
    ) -> bool:
        """Store cluster optimization configuration."""

        # TODO: Add this table to migrations
        # For now, we'll simulate storage and log a warning

        config = ClusterOptimizationConfig(
            api_key=api_key,
            cluster_id=cluster_id,
            cloud_provider=cloud_provider,
            region=region,
            cost_sensitivity=cost_sensitivity,
            monthly_budget=monthly_budget,
            preferred_instance_types=preferred_instance_types or [],
            allow_spot=allow_spot,
            max_spot_percentage=max_spot_percentage,
            created_at=datetime.utcnow(),
            updated_at=datetime.utcnow()
        )

        logger.info(
            f"Storing cluster config for {cluster_id} in {cloud_provider}/{region}")

        # TODO: Implement actual storage once migration is added
        # CREATE TABLE IF NOT EXISTS cluster_optimization_configs (
        #     api_key String,
        #     cluster_id String,
        #     cloud_provider String,
        #     region String,
        #     cost_sensitivity String,
        #     monthly_budget Nullable(String),
        #     preferred_instance_types Array(String),
        #     allow_spot UInt8,
        #     max_spot_percentage String,
        #     created_at DateTime64(3),
        #     updated_at DateTime64(3)
        # ) ENGINE = ReplacingMergeTree(updated_at)
        # ORDER BY (api_key, cluster_id)

        logger.warning(
            "cluster_optimization_configs table not yet implemented in migrations")
        return True

    async def get_cluster_optimization_config(
        self,
        api_key: str
    ) -> Optional[ClusterOptimizationConfig]:
        """Get cluster optimization configuration by API key."""

        # TODO: Implement when table exists
        logger.warning(
            "cluster_optimization_configs table not yet implemented")

        # Return dummy config for testing
        return ClusterOptimizationConfig(
            api_key=api_key,
            cluster_id="dummy-cluster",
            cloud_provider="aws",
            region="us-east-1",
            cost_sensitivity="medium",
            monthly_budget="1000",
            preferred_instance_types=["m5.large", "m5.xlarge"],
            allow_spot=True,
            max_spot_percentage="30",
            created_at=datetime.utcnow(),
            updated_at=datetime.utcnow()
        )

    async def store_service_level_objective(
        self,
        api_key: str,
        cluster_id: str,
        namespace: str,
        service_name: str,
        target_p95_latency: str,
        target_error_rate: str,
        target_throughput: Optional[str] = None,
        priority: str = "medium"
    ) -> bool:
        """Store Service Level Objective."""

        try:
            # Parse string values to floats
            p95_latency_ms = self._parse_latency_target(target_p95_latency)
            error_rate = float(target_error_rate)
            throughput_rps = self._parse_throughput_target(
                target_throughput) if target_throughput else None

            slo = ServiceLevelObjective(
                api_key=api_key,
                cluster_id=cluster_id,
                namespace=namespace,
                service_name=service_name,
                target_p95_latency_ms=p95_latency_ms,
                target_error_rate=error_rate,
                target_throughput_rps=throughput_rps,
                priority=priority,
                created_at=datetime.utcnow(),
                updated_at=datetime.utcnow()
            )

            logger.info(f"Storing SLO for {namespace}/{service_name}: "
                        f"P95={p95_latency_ms}ms, error_rate={error_rate}")

            # TODO: Implement actual storage once migration is added
            # CREATE TABLE IF NOT EXISTS service_level_objectives (
            #     api_key String,
            #     cluster_id String,
            #     namespace String,
            #     service_name String,
            #     target_p95_latency_ms Float64,
            #     target_error_rate Float64,
            #     target_throughput_rps Nullable(Float64),
            #     priority String,
            #     created_at DateTime64(3),
            #     updated_at DateTime64(3)
            # ) ENGINE = ReplacingMergeTree(updated_at)
            # ORDER BY (api_key, cluster_id, namespace, service_name)

            logger.warning(
                "service_level_objectives table not yet implemented in migrations")
            return True

        except Exception as e:
            logger.error(
                f"Error storing SLO for {namespace}/{service_name}: {str(e)}")
            return False

    async def get_service_level_objective(
        self,
        api_key: str,
        namespace: str,
        service_name: str
    ) -> Optional[ServiceLevelObjective]:
        """Get Service Level Objective for a service."""

        # TODO: Implement when table exists
        logger.warning("service_level_objectives table not yet implemented")

        # Return dummy SLO for testing
        return ServiceLevelObjective(
            api_key=api_key,
            cluster_id="dummy-cluster",
            namespace=namespace,
            service_name=service_name,
            target_p95_latency_ms=250.0,  # 250ms
            target_error_rate=0.01,       # 1%
            target_throughput_rps=100.0,  # 100 RPS
            priority="high",
            created_at=datetime.utcnow(),
            updated_at=datetime.utcnow()
        )

    async def get_all_slos_for_cluster(
        self,
        api_key: str,
        cluster_id: str
    ) -> List[ServiceLevelObjective]:
        """Get all SLOs for a cluster."""

        # TODO: Implement when table exists
        logger.warning("service_level_objectives table not yet implemented")
        return []

    async def delete_service_level_objective(
        self,
        api_key: str,
        namespace: str,
        service_name: str
    ) -> bool:
        """Delete Service Level Objective."""

        # TODO: Implement when table exists
        logger.warning("SLO deletion not yet implemented")
        return True

    def _parse_latency_target(self, latency_str: str) -> float:
        """Parse latency target string to milliseconds."""
        if not latency_str:
            return 100.0  # Default 100ms

        latency_str = latency_str.strip().lower()

        try:
            if latency_str.endswith('ms'):
                return float(latency_str[:-2])
            elif latency_str.endswith('s'):
                return float(latency_str[:-1]) * 1000  # seconds to ms
            elif latency_str.endswith('us'):
                return float(latency_str[:-2]) / 1000  # microseconds to ms
            else:
                # Assume milliseconds if no unit
                return float(latency_str)
        except (ValueError, TypeError):
            logger.warning(
                f"Failed to parse latency target: {latency_str}, using default 100ms")
            return 100.0

    def _parse_throughput_target(self, throughput_str: str) -> Optional[float]:
        """Parse throughput target string to requests per second."""
        if not throughput_str:
            return None

        throughput_str = throughput_str.strip().lower()

        try:
            if throughput_str.endswith('rps'):
                return float(throughput_str[:-3])
            elif throughput_str.endswith('qps'):
                return float(throughput_str[:-3])  # queries per second
            elif throughput_str.endswith('req/s'):
                return float(throughput_str[:-5])
            elif throughput_str.endswith('rpm'):
                # requests per minute to RPS
                return float(throughput_str[:-3]) / 60
            else:
                # Assume RPS if no unit
                return float(throughput_str)
        except (ValueError, TypeError):
            logger.warning(
                f"Failed to parse throughput target: {throughput_str}")
            return None


class TrainingDataCollector:
    """
    Collect and prepare training data for RL models.

    This class helps collect state-action-reward trajectories for training
    the PPO models, replacing the trajectory collection from the controller.
    """

    def __init__(self, clickhouse_client: ClickHouseClient):
        self.client = clickhouse_client
        self.analytics_data = AnalyticsDataAccess(clickhouse_client)

    async def collect_training_trajectories(
        self,
        cluster_id: str,
        namespace: str,
        app_name: str,
        hours_back: int = 24
    ) -> List[Dict[str, Any]]:
        """
        Collect state-action-reward trajectories for training.

        This combines data from:
        - recommendation_decisions (actions taken)
        - execution_outcomes (rewards/results)
        - Historical metrics (states)
        """
        end_time = datetime.utcnow()
        start_time = end_time - timedelta(hours=hours_back)

        # Query decision-outcome pairs with temporal alignment
        query = """
        SELECT
            rd.decision_id,
            rd.model_version,
            rd.confidence,
            rd.plan_vertical,
            rd.plan_replicas,
            rd.ts as decision_time,
            eo.success,
            eo.note,
            eo.post_action_metrics,
            eo.reported_at
        FROM recommendation_decisions rd
        LEFT JOIN execution_outcomes eo ON rd.decision_id = eo.decision_id
        WHERE rd.cluster_id = '{cluster_id}'
        AND rd.namespace = '{namespace}'
        AND rd.app_name = '{app_name}'
        AND rd.ts >= '{start_time}'
        AND rd.ts <= '{end_time}'
        ORDER BY rd.ts ASC
        """

        trajectories = await self.client.execute_query(
            query,
            params={
                "cluster_id": cluster_id,
                "namespace": namespace,
                "app_name": app_name,
                "start_time": start_time.isoformat(),
                "end_time": end_time.isoformat()
            },
            database="engine"
        )

        logger.info(
            f"Collected {len(trajectories)} training trajectories for {namespace}/{app_name}")

        # Enrich with state information from analytics database
        enriched_trajectories = []
        for trajectory in trajectories:
            try:
                # Get state information at decision time
                decision_time = trajectory.get("decision_time")
                if decision_time:
                    # Parse decision time and get metrics around that time
                    dt = datetime.fromisoformat(
                        decision_time.replace('Z', '+00:00'))

                    # Get metrics within 1 minute window around decision time
                    pre_metrics = await self.analytics_data.get_app_metrics_at_time(
                        cluster_id=cluster_id,
                        namespace=namespace,
                        app_name=app_name,
                        target_time=dt,
                        window_minutes=1
                    )

                    # Enrich trajectory with state information
                    enriched_trajectory = trajectory.copy()
                    if pre_metrics:
                        enriched_trajectory.update({
                            "pre_state": pre_metrics,
                            "cpu_utilization": pre_metrics.get("cpu_utilization", 0.5),
                            "memory_utilization": pre_metrics.get("memory_utilization", 0.5),
                            "request_rate": pre_metrics.get("request_rate", 100.0),
                            "p95_latency_ms": pre_metrics.get("p95_latency_ms", 200.0),
                            "error_rate": pre_metrics.get("error_rate", 0.01),
                            "num_replicas": pre_metrics.get("num_replicas", 1)
                        })

                    enriched_trajectories.append(enriched_trajectory)
                else:
                    # Keep trajectory without state information
                    enriched_trajectories.append(trajectory)

            except Exception as e:
                logger.warning(
                    f"Failed to enrich trajectory {trajectory.get('decision_id', 'unknown')}: {str(e)}")
                # Keep original trajectory if enrichment fails
                enriched_trajectories.append(trajectory)

        logger.info(
            f"Enriched {len(enriched_trajectories)} trajectories with state information")
        return enriched_trajectories

    async def calculate_training_rewards(
        self,
        trajectories: List[Dict[str, Any]],
        slo_targets: Optional[Dict[str, float]] = None
    ) -> List[float]:
        """
        Calculate rewards for training trajectories using the reward function
        from the RL models.
        """
        if not RewardCalculator:
            logger.warning(
                "RewardCalculator not available, using simple reward calculation")
            rewards = []
            for trajectory in trajectories:
                success = trajectory.get("success", True)
                base_reward = 1.0 if success else -1.0
                rewards.append(base_reward)
            return rewards

        # Initialize reward calculator
        reward_calc = RewardCalculator(alpha=0.3, slo_weight=0.2)
        rewards = []

        for i, trajectory in enumerate(trajectories):
            try:
                # Extract current state from trajectory
                current_state = {
                    'cpu_util': trajectory.get('cpu_utilization', 0.5),
                    'memory_util': trajectory.get('memory_utilization', 0.5),
                    'latency': trajectory.get('p95_latency_ms', 200.0),
                    'processing_rate': trajectory.get('request_rate', 100.0),
                    # Approximation
                    'ingestion_rate': trajectory.get('request_rate', 100.0),
                    'error_rate': trajectory.get('error_rate', 0.01),
                    'num_replicas': trajectory.get('num_replicas', 1),
                    'cpu_limit': 1000,  # Default values
                    'memory_limit': 512
                }

                # Parse action from trajectory
                action_dict = {}
                if trajectory.get('plan_vertical'):
                    plan = trajectory['plan_vertical']
                    if isinstance(plan, list) and len(plan) > 0:
                        vpa_plan = plan[0]
                        action_dict = {
                            'cpu_request_mcpu': int(vpa_plan.get('cpu_request_mcpu', 0)),
                            'memory_request_mib': int(vpa_plan.get('memory_mib', 0)),
                            'replicas': trajectory.get('plan_replicas', 0)
                        }

                # Get previous state (use current as approximation if not available)
                if i > 0:
                    prev_trajectory = trajectories[i - 1]
                    last_state = {
                        'cpu_util': prev_trajectory.get('cpu_utilization', 0.5),
                        'memory_util': prev_trajectory.get('memory_utilization', 0.5),
                        'latency': prev_trajectory.get('p95_latency_ms', 200.0),
                        'processing_rate': prev_trajectory.get('request_rate', 100.0),
                        'ingestion_rate': prev_trajectory.get('request_rate', 100.0),
                        'error_rate': prev_trajectory.get('error_rate', 0.01)
                    }
                    last_action = {}  # Previous action would need to be extracted similarly
                else:
                    # First trajectory, use current state as previous
                    last_state = current_state.copy()
                    last_action = {}

                # Calculate reward using v1 formula
                reward = reward_calc.calculate_reward_v1(
                    current_state=current_state,
                    action=action_dict,
                    last_action=last_action,
                    last_state=last_state,
                    slo_targets=slo_targets
                )

                # Apply execution success/failure modifier
                success = trajectory.get("success", True)
                if not success:
                    reward -= 1.0  # Large penalty for failed executions
                    logger.debug(
                        f"Applied failure penalty for trajectory {trajectory.get('decision_id')}")

                rewards.append(reward)

                logger.debug(
                    f"Calculated reward {reward:.3f} for trajectory {trajectory.get('decision_id')}")

            except Exception as e:
                logger.error(
                    f"Error calculating reward for trajectory {trajectory.get('decision_id', 'unknown')}: {str(e)}")
                # Fallback to simple reward
                success = trajectory.get("success", True)
                fallback_reward = 1.0 if success else -1.0
                rewards.append(fallback_reward)

        logger.info(
            f"Calculated {len(rewards)} rewards for training trajectories")
        return rewards

    async def prepare_training_dataset(
        self,
        cluster_id: str,
        namespace: str,
        app_name: str,
        hours_back: int = 24
    ) -> Dict[str, np.ndarray]:
        """
        Prepare a complete training dataset with states, actions, and rewards
        in the format needed for RL training.
        """
        try:
            # Collect and enrich trajectories
            trajectories = await self.collect_training_trajectories(
                cluster_id=cluster_id,
                namespace=namespace,
                app_name=app_name,
                hours_back=hours_back
            )

            if not trajectories:
                logger.warning(
                    f"No trajectories found for {namespace}/{app_name}")
                return {
                    'states': np.array([]),
                    'actions': np.array([]),
                    'rewards': np.array([]),
                    'next_states': np.array([])
                }

            # Calculate rewards
            rewards = await self.calculate_training_rewards(trajectories)

            # Prepare state and action arrays
            states = []
            actions = []
            next_states = []
            valid_rewards = []

            for i, (trajectory, reward) in enumerate(zip(trajectories, rewards)):
                try:
                    # Create state vector (normalized features for RL)
                    state = np.array([
                        trajectory.get('cpu_utilization', 0.5),
                        trajectory.get('memory_utilization', 0.5),
                        # Normalize latency
                        min(1.0, trajectory.get('p95_latency_ms', 200.0) / 1000.0),
                        min(1.0, trajectory.get('request_rate',
                            100.0) / 1000.0),   # Normalize RPS
                        trajectory.get('error_rate', 0.01),
                        min(1.0, trajectory.get('num_replicas', 1) /
                            10.0),         # Normalize replicas
                        0.5,  # CPU limit (normalized placeholder)
                        0.5,  # Memory limit (normalized placeholder)
                        0.5,  # Processing rate (placeholder)
                        0.5   # Ingestion rate (placeholder)
                    ], dtype=np.float32)

                    # Create action vector (discrete action encoding)
                    action_encoding = 0  # NO_ACTION by default
                    if trajectory.get('plan_vertical'):
                        # This is a VPA action - encode based on the changes
                        action_encoding = 1  # VERTICAL_CPU_UP (simplified)
                    elif trajectory.get('plan_replicas', 0) > 0:
                        # This is an HPA action
                        action_encoding = 5  # HORIZONTAL_UP (simplified)

                    # Create next state (use current state as approximation if next not available)
                    if i + 1 < len(trajectories):
                        next_trajectory = trajectories[i + 1]
                        next_state = np.array([
                            next_trajectory.get('cpu_utilization', 0.5),
                            next_trajectory.get('memory_utilization', 0.5),
                            min(1.0, next_trajectory.get(
                                'p95_latency_ms', 200.0) / 1000.0),
                            min(1.0, next_trajectory.get(
                                'request_rate', 100.0) / 1000.0),
                            next_trajectory.get('error_rate', 0.01),
                            min(1.0, next_trajectory.get(
                                'num_replicas', 1) / 10.0),
                            0.5, 0.5, 0.5, 0.5
                        ], dtype=np.float32)
                    else:
                        next_state = state.copy()  # Terminal state

                    states.append(state)
                    actions.append(action_encoding)
                    next_states.append(next_state)
                    valid_rewards.append(reward)

                except Exception as e:
                    logger.warning(
                        f"Failed to process trajectory {trajectory.get('decision_id')}: {str(e)}")
                    continue

            # Convert to numpy arrays
            dataset = {
                'states': np.array(states, dtype=np.float32),
                'actions': np.array(actions, dtype=np.int32),
                'rewards': np.array(valid_rewards, dtype=np.float32),
                'next_states': np.array(next_states, dtype=np.float32)
            }

            logger.info(
                f"Prepared training dataset: {len(states)} samples for {namespace}/{app_name}")
            logger.info(
                f"State shape: {dataset['states'].shape}, Action shape: {dataset['actions'].shape}")

            return dataset

        except Exception as e:
            logger.error(
                f"Error preparing training dataset for {namespace}/{app_name}: {str(e)}")
            return {
                'states': np.array([]),
                'actions': np.array([]),
                'rewards': np.array([]),
                'next_states': np.array([])
            }
