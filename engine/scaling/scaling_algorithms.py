"""
HPA and VPA Scaling Algorithms for Futura Engine.

This module implements the horizontal and vertical scaling algorithms
extracted from the controller's RL environment, providing the RL agent
with proper scaling capabilities for Kubernetes resources.
"""

import logging
from typing import Dict, Optional, Tuple, Any
from dataclasses import dataclass
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)


@dataclass
class ScalingConstraints:
    """Resource scaling constraints and bounds."""

    # Replica constraints
    min_instances: int = 1
    max_instances: int = 20

    # CPU constraints (millicores)
    min_cpu_limit: int = 128
    max_cpu_limit: int = 2048

    # Memory constraints (MiB)
    min_memory_limit: int = 256
    max_memory_limit: int = 3072

    # Scaling steps
    vertical_cpu_step: int = 128
    vertical_memory_step: int = 128
    horizontal_scaling_step: int = 1

    # Utilization bounds
    lower_bound_util: float = 0.7
    upper_bound_util: float = 0.9


@dataclass
class ScalingAction:
    """Represents a scaling action to be executed."""

    action_type: str  # "horizontal", "vertical_cpu", "vertical_memory", "no_action"
    target_replicas: Optional[int] = None
    target_cpu_mcpu: Optional[int] = None
    target_memory_mib: Optional[int] = None
    reason: str = ""
    confidence: float = 1.0


@dataclass
class ResourceState:
    """Current resource state of the application."""

    # Current allocations
    num_replicas: int
    cpu_limit: int  # millicores
    memory_limit: int  # MiB

    # Utilization metrics (0.0 to 1.0)
    cpu_util: float = 0.5
    memory_util: float = 0.5

    # Performance metrics
    request_rate: float = 100.0
    p95_latency_ms: float = 200.0
    error_rate: float = 0.01

    # Processing metrics (from controller)
    processing_rate: float = 100.0
    ingestion_rate: float = 100.0
    file_discovery_rate: float = 10.0
    disk_io_usage: float = 0.1


class ScalingAlgorithms:
    """
    Implements HPA and VPA scaling algorithms extracted from the controller.

    Provides intelligent scaling decisions based on resource utilization,
    performance metrics, and SLO compliance.
    """

    def __init__(self, constraints: Optional[ScalingConstraints] = None):
        self.constraints = constraints or ScalingConstraints()
        self.last_scaling_time: Dict[str, datetime] = {}

        # Cooldown periods to prevent oscillation
        self.horizontal_cooldown_seconds = 300  # 5 minutes
        self.vertical_cooldown_seconds = 180    # 3 minutes

    def sanity_check_action(
        self,
        current_state: ResourceState,
        action: Dict[str, int]
    ) -> bool:
        """
        Validate that a scaling action is within bounds and safe to execute.

        Based on the controller's sanity_check method.
        """

        # Check horizontal scaling bounds
        if action.get('horizontal', 0) != 0:
            new_replicas = current_state.num_replicas + action['horizontal']
            if new_replicas < self.constraints.min_instances:
                logger.warning(
                    f"Horizontal scaling would go below min instances: {new_replicas} < {self.constraints.min_instances}")
                return False
            if new_replicas > self.constraints.max_instances:
                logger.warning(
                    f"Horizontal scaling would exceed max instances: {new_replicas} > {self.constraints.max_instances}")
                return False

        # Check vertical CPU scaling bounds
        elif action.get('vertical_cpu', 0) != 0:
            new_cpu_limit = current_state.cpu_limit + action['vertical_cpu']
            if new_cpu_limit > self.constraints.max_cpu_limit or new_cpu_limit < self.constraints.min_cpu_limit:
                logger.warning(
                    f"CPU scaling would exceed bounds: {new_cpu_limit} not in [{self.constraints.min_cpu_limit}, {self.constraints.max_cpu_limit}]")
                return False

            # Require at least 2 replicas for vertical scaling (pod eviction)
            if current_state.num_replicas <= 1:
                logger.warning(
                    "Vertical CPU scaling requires at least 2 replicas for pod eviction")
                return False

        # Check vertical memory scaling bounds
        elif action.get('vertical_memory', 0) != 0:
            new_memory_limit = current_state.memory_limit + \
                action['vertical_memory']
            if new_memory_limit > self.constraints.max_memory_limit or new_memory_limit < self.constraints.min_memory_limit:
                logger.warning(
                    f"Memory scaling would exceed bounds: {new_memory_limit} not in [{self.constraints.min_memory_limit}, {self.constraints.max_memory_limit}]")
                return False

            # Require at least 2 replicas for vertical scaling (pod eviction)
            if current_state.num_replicas <= 1:
                logger.warning(
                    "Vertical memory scaling requires at least 2 replicas for pod eviction")
                return False

        return True

    def should_scale_horizontally(
        self,
        current_state: ResourceState,
        app_key: str
    ) -> Optional[ScalingAction]:
        """
        Determine if horizontal scaling (HPA) is needed based on resource utilization.

        Uses the controller's logic for horizontal scaling decisions.
        """

        # Check cooldown period
        if self._is_in_cooldown(app_key, 'horizontal'):
            return None

        current_replicas = current_state.num_replicas
        cpu_util = current_state.cpu_util
        memory_util = current_state.memory_util

        # Scale out conditions
        if (cpu_util > self.constraints.upper_bound_util or
                memory_util > self.constraints.upper_bound_util):

            # Check if we can scale out
            if current_replicas < self.constraints.max_instances:
                target_replicas = min(
                    current_replicas + self.constraints.horizontal_scaling_step,
                    self.constraints.max_instances
                )

                return ScalingAction(
                    action_type="horizontal",
                    target_replicas=target_replicas,
                    reason=f"High utilization detected: CPU={cpu_util:.2f}, Memory={memory_util:.2f}",
                    confidence=0.9
                )

        # Scale in conditions
        elif (cpu_util < self.constraints.lower_bound_util and
              memory_util < self.constraints.lower_bound_util):

            # Check if we can scale in
            if current_replicas > self.constraints.min_instances:
                target_replicas = max(
                    current_replicas - self.constraints.horizontal_scaling_step,
                    self.constraints.min_instances
                )

                return ScalingAction(
                    action_type="horizontal",
                    target_replicas=target_replicas,
                    reason=f"Low utilization detected: CPU={cpu_util:.2f}, Memory={memory_util:.2f}",
                    confidence=0.7
                )

        return None

    def should_scale_vertically(
        self,
        current_state: ResourceState,
        app_key: str
    ) -> Optional[ScalingAction]:
        """
        Determine if vertical scaling (VPA) is needed based on resource utilization.

        Uses the controller's logic for vertical scaling decisions.
        """

        # Check cooldown period
        if self._is_in_cooldown(app_key, 'vertical'):
            return None

        # Require at least 2 replicas for vertical scaling
        if current_state.num_replicas <= 1:
            return None

        cpu_util = current_state.cpu_util
        memory_util = current_state.memory_util

        # CPU vertical scaling
        if cpu_util > self.constraints.upper_bound_util:
            new_cpu_limit = current_state.cpu_limit + self.constraints.vertical_cpu_step
            if new_cpu_limit <= self.constraints.max_cpu_limit:
                return ScalingAction(
                    action_type="vertical_cpu",
                    target_cpu_mcpu=new_cpu_limit,
                    reason=f"High CPU utilization: {cpu_util:.2f}",
                    confidence=0.85
                )

        elif cpu_util < self.constraints.lower_bound_util:
            new_cpu_limit = current_state.cpu_limit - self.constraints.vertical_cpu_step
            if new_cpu_limit >= self.constraints.min_cpu_limit:
                return ScalingAction(
                    action_type="vertical_cpu",
                    target_cpu_mcpu=new_cpu_limit,
                    reason=f"Low CPU utilization: {cpu_util:.2f}",
                    confidence=0.7
                )

        # Memory vertical scaling
        if memory_util > self.constraints.upper_bound_util:
            new_memory_limit = current_state.memory_limit + \
                self.constraints.vertical_memory_step
            if new_memory_limit <= self.constraints.max_memory_limit:
                return ScalingAction(
                    action_type="vertical_memory",
                    target_memory_mib=new_memory_limit,
                    reason=f"High memory utilization: {memory_util:.2f}",
                    confidence=0.85
                )

        elif memory_util < self.constraints.lower_bound_util:
            new_memory_limit = current_state.memory_limit - \
                self.constraints.vertical_memory_step
            if new_memory_limit >= self.constraints.min_memory_limit:
                return ScalingAction(
                    action_type="vertical_memory",
                    target_memory_mib=new_memory_limit,
                    reason=f"Low memory utilization: {memory_util:.2f}",
                    confidence=0.7
                )

        return None

    def get_intelligent_scaling_action(
        self,
        current_state: ResourceState,
        slo_targets: Optional[Dict[str, float]] = None,
        app_key: str = "default"
    ) -> ScalingAction:
        """
        Get the best scaling action based on current state and SLO targets.

        Prioritizes actions based on:
        1. SLO violations (latency, error rate)
        2. Resource utilization efficiency
        3. Processing performance (ingestion vs processing rate)
        """

        # Check for SLO violations first
        if slo_targets:
            slo_action = self._check_slo_violations(current_state, slo_targets)
            if slo_action:
                return slo_action

        # Check for performance bottlenecks
        perf_action = self._check_performance_bottlenecks(
            current_state, app_key)
        if perf_action:
            return perf_action

        # Check for resource utilization scaling
        horizontal_action = self.should_scale_horizontally(
            current_state, app_key)
        if horizontal_action:
            return horizontal_action

        vertical_action = self.should_scale_vertically(current_state, app_key)
        if vertical_action:
            return vertical_action

        # No action needed
        return ScalingAction(
            action_type="no_action",
            reason="Resource utilization and performance within acceptable bounds",
            confidence=0.8
        )

    def _check_slo_violations(
        self,
        current_state: ResourceState,
        slo_targets: Dict[str, float]
    ) -> Optional[ScalingAction]:
        """Check for SLO violations and suggest scaling actions."""

        # High latency violation
        target_latency = slo_targets.get('target_p95_latency_ms', 500.0)
        if current_state.p95_latency_ms > target_latency * 1.2:  # 20% tolerance
            # Prefer horizontal scaling for latency issues
            if current_state.num_replicas < self.constraints.max_instances:
                target_replicas = min(
                    current_state.num_replicas + 1,
                    self.constraints.max_instances
                )
                return ScalingAction(
                    action_type="horizontal",
                    target_replicas=target_replicas,
                    reason=f"SLO violation: P95 latency {current_state.p95_latency_ms:.1f}ms > {target_latency}ms",
                    confidence=0.95
                )

        # High error rate violation
        target_error_rate = slo_targets.get('target_error_rate', 0.01)
        if current_state.error_rate > target_error_rate * 2:  # 2x tolerance
            # Scale out to distribute load
            if current_state.num_replicas < self.constraints.max_instances:
                target_replicas = min(
                    current_state.num_replicas + 1,
                    self.constraints.max_instances
                )
                return ScalingAction(
                    action_type="horizontal",
                    target_replicas=target_replicas,
                    reason=f"SLO violation: Error rate {current_state.error_rate:.3f} > {target_error_rate:.3f}",
                    confidence=0.9
                )

        return None

    def _check_performance_bottlenecks(
        self,
        current_state: ResourceState,
        app_key: str
    ) -> Optional[ScalingAction]:
        """Check for processing performance bottlenecks."""

        # Check for processing lag (ingestion > processing)
        if (current_state.ingestion_rate > 0 and
                current_state.processing_rate < current_state.ingestion_rate * 0.8):  # Processing lag

            # Check if CPU is the bottleneck
            if current_state.cpu_util > 0.8:
                # Try vertical CPU scaling first if possible
                if (current_state.num_replicas > 1 and
                        current_state.cpu_limit < self.constraints.max_cpu_limit):

                    return ScalingAction(
                        action_type="vertical_cpu",
                        target_cpu_mcpu=min(
                            current_state.cpu_limit + self.constraints.vertical_cpu_step,
                            self.constraints.max_cpu_limit
                        ),
                        reason=f"Processing lag detected: {current_state.processing_rate:.1f} < {current_state.ingestion_rate:.1f} RPS",
                        confidence=0.8
                    )

                # Otherwise scale horizontally
                elif current_state.num_replicas < self.constraints.max_instances:
                    return ScalingAction(
                        action_type="horizontal",
                        target_replicas=min(
                            current_state.num_replicas + 1,
                            self.constraints.max_instances
                        ),
                        reason=f"Processing lag with high CPU: {current_state.cpu_util:.2f}",
                        confidence=0.85
                    )

        return None

    def _is_in_cooldown(self, app_key: str, scaling_type: str) -> bool:
        """Check if scaling is in cooldown period to prevent oscillation."""

        last_scaling_key = f"{app_key}:{scaling_type}"
        if last_scaling_key not in self.last_scaling_time:
            return False

        cooldown_seconds = (
            self.horizontal_cooldown_seconds if scaling_type == 'horizontal'
            else self.vertical_cooldown_seconds
        )

        time_since_last = datetime.utcnow(
        ) - self.last_scaling_time[last_scaling_key]
        return time_since_last.total_seconds() < cooldown_seconds

    def mark_scaling_executed(self, app_key: str, scaling_type: str):
        """Mark that a scaling action was executed for cooldown tracking."""
        last_scaling_key = f"{app_key}:{scaling_type}"
        self.last_scaling_time[last_scaling_key] = datetime.utcnow()

    def convert_state_dict_to_resource_state(self, state_dict: Dict[str, Any]) -> ResourceState:
        """Convert a state dictionary to ResourceState object."""

        return ResourceState(
            num_replicas=int(state_dict.get('num_replicas', 1)),
            cpu_limit=int(state_dict.get('cpu_limit', 1000)),
            memory_limit=int(state_dict.get('memory_limit', 512)),
            cpu_util=float(state_dict.get('cpu_utilization',
                           state_dict.get('cpu_util', 0.5))),
            memory_util=float(state_dict.get(
                'memory_utilization', state_dict.get('memory_util', 0.5))),
            request_rate=float(state_dict.get(
                'request_rate', state_dict.get('rate', 100.0))),
            p95_latency_ms=float(state_dict.get(
                'p95_latency_ms', state_dict.get('latency', 200.0))),
            error_rate=float(state_dict.get('error_rate', 0.01)),
            processing_rate=float(state_dict.get('processing_rate', 100.0)),
            ingestion_rate=float(state_dict.get('ingestion_rate', 100.0)),
            file_discovery_rate=float(
                state_dict.get('file_discovery_rate', 10.0)),
            disk_io_usage=float(state_dict.get('disk_io_usage', 0.1))
        )
