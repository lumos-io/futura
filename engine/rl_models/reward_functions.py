"""
Reward functions for RL-based Kubernetes autoscaling.

Based on the research implementation from the controller folder,
this module implements various reward functions that balance
resource utilization, application performance, and SLO compliance.
"""

import numpy as np
from typing import Dict, Any, Optional
import logging

logger = logging.getLogger(__name__)

# Constants from controller implementation
ILLEGAL_PENALTY = 0.5  # Penalty for contradictory actions


class RewardCalculator:
    """
    Calculates rewards for RL-based autoscaling decisions.

    Implements the reward functions from the research paper:
    - Resource utilization optimization
    - Application performance maintenance
    - SLO compliance
    - Action penalty for oscillating behavior
    """

    def __init__(self, alpha: float = 0.3, slo_weight: float = 0.2):
        """
        Initialize reward calculator.

        Args:
            alpha: Weight for resource utilization vs performance (0.3 in paper)
            slo_weight: Weight for SLO compliance component
        """
        self.alpha = alpha
        self.slo_weight = slo_weight

    def calculate_reward_v1(
        self,
        current_state: Dict[str, float],
        action: Dict[str, int],
        last_action: Dict[str, int],
        last_state: Dict[str, float],
        slo_targets: Optional[Dict[str, float]] = None
    ) -> float:
        """
        Calculate reward using the v1 formula from the paper:
        R = alpha * RU + (1-alpha) * DP - penalty

        Where:
        - RU: Resource Utilization score
        - DP: Data Processing rate
        - penalty: Penalty for oscillating actions
        """

        # Resource utilization component (0-1 range)
        resource_util_score = (
            current_state.get('cpu_utilization', 0.5) +
            current_state.get('memory_utilization', 0.5)
        ) / 2.0

        # Data processing rate component (0-1 range)
        ingestion_rate = current_state.get('request_rate', 100.0)
        processing_rate = current_state.get('processing_rate', ingestion_rate)

        if ingestion_rate == 0:
            data_processing_rate = 1.0  # Perfect when no load
        else:
            data_processing_rate = min(1.0, processing_rate / ingestion_rate)

        # Base reward: balance utilization and performance
        reward = (
            self.alpha * resource_util_score +
            (1 - self.alpha) * data_processing_rate
        )

        # Apply penalties for oscillating behavior
        reward -= self._calculate_action_penalty(action, last_action)

        # Apply performance degradation penalty
        reward -= self._calculate_performance_penalty(
            current_state, last_state)

        # Add SLO compliance bonus if targets are specified
        if slo_targets:
            reward += self._calculate_slo_reward(current_state, slo_targets)

        logger.debug(f"Reward calculation: util={resource_util_score:.3f}, "
                     f"perf={data_processing_rate:.3f}, total={reward:.3f}")

        return reward

    def calculate_reward_v2(
        self,
        current_state: Dict[str, float],
        action: Dict[str, int],
        last_action: Dict[str, int],
        last_state: Dict[str, float],
        slo_targets: Optional[Dict[str, float]] = None
    ) -> float:
        """
        Calculate reward using the v2 formula from the paper:
        R = RU * DP - penalty

        This version uses multiplication instead of weighted sum,
        encouraging both high utilization AND good performance.
        """

        # Resource utilization component
        resource_util_score = (
            current_state.get('cpu_utilization', 0.5) +
            current_state.get('memory_utilization', 0.5)
        ) / 2.0

        # Data processing rate component
        ingestion_rate = current_state.get('request_rate', 100.0)
        processing_rate = current_state.get('processing_rate', ingestion_rate)

        if ingestion_rate == 0:
            data_processing_rate = 1.0
        else:
            data_processing_rate = min(1.0, processing_rate / ingestion_rate)

        # Multiplicative reward: requires BOTH good utilization AND performance
        reward = resource_util_score * data_processing_rate

        # Apply penalties
        reward -= self._calculate_action_penalty(action, last_action)
        reward -= self._calculate_performance_penalty(
            current_state, last_state)

        # Add SLO compliance bonus
        if slo_targets:
            reward += self._calculate_slo_reward(current_state, slo_targets)

        return reward

    def calculate_slo_aware_reward(
        self,
        current_state: Dict[str, float],
        action: Dict[str, int],
        last_action: Dict[str, int],
        last_state: Dict[str, float],
        slo_targets: Dict[str, float]
    ) -> float:
        """
        Calculate reward with strong SLO compliance focus.

        This is an enhanced version that prioritizes SLO compliance
        while still encouraging efficient resource usage.
        """

        # Base performance metrics
        resource_util_score = (
            current_state.get('cpu_utilization', 0.5) +
            current_state.get('memory_utilization', 0.5)
        ) / 2.0

        # SLO compliance score (primary objective)
        slo_score = self._calculate_slo_compliance_score(
            current_state, slo_targets)

        # Cost efficiency score (secondary objective)
        cost_efficiency = self._calculate_cost_efficiency(
            current_state, slo_targets)

        # Weighted combination with SLO as primary concern
        reward = (
            0.6 * slo_score +           # SLO compliance (primary)
            0.3 * cost_efficiency +     # Cost efficiency (secondary)
            0.1 * resource_util_score   # Resource utilization (tertiary)
        )

        # Apply penalties
        reward -= self._calculate_action_penalty(action, last_action)

        return reward

    def _calculate_action_penalty(
        self,
        action: Dict[str, int],
        last_action: Dict[str, int]
    ) -> float:
        """
        Calculate penalty for oscillating actions (scale up then down).
        This encourages stable behavior and prevents thrashing.
        """
        penalty = 0.0

        # Horizontal scaling oscillation
        if (action.get('horizontal', 0) * last_action.get('horizontal', 0)) < 0:
            penalty += ILLEGAL_PENALTY
            logger.debug("Horizontal scaling oscillation penalty applied")

        # Vertical CPU scaling oscillation
        if (action.get('vertical_cpu', 0) * last_action.get('vertical_cpu', 0)) < 0:
            penalty += ILLEGAL_PENALTY
            logger.debug("Vertical CPU scaling oscillation penalty applied")

        # Vertical memory scaling oscillation
        if (action.get('vertical_memory', 0) * last_action.get('vertical_memory', 0)) < 0:
            penalty += ILLEGAL_PENALTY
            logger.debug("Vertical memory scaling oscillation penalty applied")

        return penalty

    def _calculate_performance_penalty(
        self,
        current_state: Dict[str, float],
        last_state: Dict[str, float]
    ) -> float:
        """
        Calculate penalty for performance degradation.
        """
        penalty = 0.0

        # Check if processing rate decreased (lag increased)
        current_rate = current_state.get('processing_rate', 0.0)
        last_rate = last_state.get('processing_rate', 0.0)

        if current_rate < last_rate:
            penalty += ILLEGAL_PENALTY
            logger.debug("Performance degradation penalty applied")

        return penalty

    def _calculate_slo_reward(
        self,
        current_state: Dict[str, float],
        slo_targets: Dict[str, float]
    ) -> float:
        """
        Calculate reward bonus for SLO compliance.
        """
        reward = 0.0

        # P95 latency SLO
        target_latency = slo_targets.get('p95_latency_ms')
        if target_latency:
            current_latency = current_state.get(
                'p95_latency_ms', target_latency)
            if current_latency <= target_latency:
                # Bonus for meeting SLO
                reward += self.slo_weight
            else:
                # Penalty for violating SLO
                violation_ratio = current_latency / target_latency
                reward -= self.slo_weight * (violation_ratio - 1.0)

        # Error rate SLO
        target_error_rate = slo_targets.get('error_rate')
        if target_error_rate:
            current_error_rate = current_state.get('error_rate', 0.0)
            if current_error_rate <= target_error_rate:
                reward += self.slo_weight * 0.5
            else:
                violation_ratio = current_error_rate / target_error_rate
                reward -= self.slo_weight * (violation_ratio - 1.0)

        return reward

    def _calculate_slo_compliance_score(
        self,
        current_state: Dict[str, float],
        slo_targets: Dict[str, float]
    ) -> float:
        """
        Calculate comprehensive SLO compliance score (0-1 range).
        """
        scores = []

        # Latency SLO compliance
        target_latency = slo_targets.get('p95_latency_ms')
        if target_latency:
            current_latency = current_state.get(
                'p95_latency_ms', target_latency)
            latency_score = min(1.0, target_latency /
                                max(current_latency, 1.0))
            scores.append(latency_score)

        # Error rate SLO compliance
        target_error_rate = slo_targets.get('error_rate', 0.01)
        current_error_rate = current_state.get('error_rate', 0.0)
        if target_error_rate > 0:
            error_score = min(
                1.0, (target_error_rate - current_error_rate) / target_error_rate)
            error_score = max(0.0, error_score)  # Ensure non-negative
            scores.append(error_score)

        # Throughput SLO compliance
        target_throughput = slo_targets.get('throughput_rps')
        if target_throughput:
            current_throughput = current_state.get('request_rate', 0.0)
            throughput_score = min(1.0, current_throughput / target_throughput)
            scores.append(throughput_score)

        return np.mean(scores) if scores else 1.0

    def _calculate_cost_efficiency(
        self,
        current_state: Dict[str, float],
        slo_targets: Dict[str, float]
    ) -> float:
        """
        Calculate cost efficiency score based on resource usage vs SLO targets.
        Higher score for meeting SLOs with lower resource usage.
        """

        # Get current resource utilization
        cpu_util = current_state.get('cpu_utilization', 0.5)
        memory_util = current_state.get('memory_utilization', 0.5)
        avg_util = (cpu_util + memory_util) / 2.0

        # Get SLO compliance
        slo_score = self._calculate_slo_compliance_score(
            current_state, slo_targets)

        # Cost efficiency: meeting SLOs with minimal resources
        if slo_score >= 0.95:  # Meeting SLOs
            # Reward efficient resource usage
            # Sweet spot around 70% utilization
            efficiency = slo_score * (0.7 + 0.3 * avg_util)
        else:  # Violating SLOs
            # Penalty for both SLO violation and resource waste
            efficiency = slo_score * avg_util

        return efficiency
