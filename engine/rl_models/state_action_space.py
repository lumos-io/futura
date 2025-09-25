"""
State and Action Space definitions for RL-based Kubernetes autoscaling.

Based on the controller implementation, this module defines:
- State space: System and application metrics
- Action space: Vertical and horizontal scaling actions
- Feature extraction and normalization utilities
"""

from typing import Dict, List, Tuple, Any, Optional
import numpy as np
from enum import IntEnum
import logging

logger = logging.getLogger(__name__)


class ActionType(IntEnum):
    """Action types for multidimensional autoscaling."""
    NO_ACTION = 0
    VERTICAL_CPU_UP = 1
    VERTICAL_CPU_DOWN = 2
    VERTICAL_MEMORY_UP = 3
    VERTICAL_MEMORY_DOWN = 4
    HORIZONTAL_UP = 5
    HORIZONTAL_DOWN = 6


class StateSpace:
    """
    Defines the state space for RL-based autoscaling.

    Based on the controller implementation, includes:
    - System metrics: CPU, memory, disk I/O utilization
    - Application metrics: Request rate, latency, throughput
    - Resource allocation: Current limits and replica count
    """

    def __init__(self):
        # State vector indices (from controller rl_env.py)
        self.state_indices = {
            'cpu_util': 0,
            'memory_util': 1,
            'disk_io_usage': 2,
            'file_discovery_rate': 3,  # Application-specific metric
            'processing_rate': 4,
            'ingestion_rate': 5,
            'latency': 6,
            'num_replicas': 7,
            'cpu_limit': 8,
            'memory_limit': 9,
        }

        # State normalization bounds
        self.state_bounds = {
            'cpu_util': (0.0, 1.0),
            'memory_util': (0.0, 1.0),
            'disk_io_usage': (0.0, 1000.0),  # MB/s
            'file_discovery_rate': (0.0, 1000.0),  # events/s
            'processing_rate': (0.0, 10000.0),  # events/s
            'ingestion_rate': (0.0, 10000.0),  # events/s
            'latency': (0.0, 5000.0),  # ms
            'num_replicas': (1, 50),
            'cpu_limit': (100, 8000),  # millicore
            'memory_limit': (128, 16384),  # MiB
        }

        self.state_dim = len(self.state_indices)

    def extract_features(self, metrics: Dict[str, float]) -> np.ndarray:
        """
        Extract and normalize features from raw metrics.

        Args:
            metrics: Raw metrics dictionary

        Returns:
            Normalized feature vector
        """
        features = np.zeros(self.state_dim)

        for key, index in self.state_indices.items():
            raw_value = metrics.get(key, 0.0)

            # Map common metric names
            if key == 'cpu_util' and key not in metrics:
                raw_value = metrics.get('cpu_utilization', 0.0)
            elif key == 'memory_util' and key not in metrics:
                raw_value = metrics.get('memory_utilization', 0.0)
            elif key == 'processing_rate' and key not in metrics:
                raw_value = metrics.get('request_rate', 0.0)
            elif key == 'latency' and key not in metrics:
                raw_value = metrics.get('p95_latency_ms', 100.0)

            # Normalize to [0, 1] range
            min_val, max_val = self.state_bounds[key]
            normalized_value = (raw_value - min_val) / (max_val - min_val)
            normalized_value = np.clip(normalized_value, 0.0, 1.0)

            features[index] = normalized_value

        return features

    def get_state_description(self, features: np.ndarray) -> Dict[str, float]:
        """
        Convert normalized feature vector back to interpretable metrics.
        """
        description = {}

        for key, index in self.state_indices.items():
            if index < len(features):
                normalized_value = features[index]
                min_val, max_val = self.state_bounds[key]
                raw_value = normalized_value * (max_val - min_val) + min_val
                description[key] = raw_value

        return description


class ActionSpace:
    """
    Defines the action space for RL-based autoscaling.

    Actions include:
    - Vertical scaling: Adjust CPU/memory limits
    - Horizontal scaling: Adjust replica count
    - No action: Keep current configuration
    """

    def __init__(self, scaling_step_cpu: int = 256, scaling_step_memory: int = 256):
        """
        Initialize action space.

        Args:
            scaling_step_cpu: CPU scaling step in millicore (256m default)
            scaling_step_memory: Memory scaling step in MiB (256Mi default)
        """
        self.scaling_step_cpu = scaling_step_cpu
        self.scaling_step_memory = scaling_step_memory
        self.action_dim = len(ActionType)

    def sample_action(self, policy_output: np.ndarray) -> int:
        """
        Sample action from policy output (discrete action space).

        Args:
            policy_output: Action probabilities from policy network

        Returns:
            Sampled action index
        """
        if len(policy_output.shape) > 1:
            # Batch dimension present
            policy_output = policy_output[0]

        # Sample from categorical distribution
        action = np.random.choice(len(policy_output), p=policy_output)
        return action

    def convert_action_to_dict(self, action: int) -> Dict[str, int]:
        """
        Convert discrete action to scaling commands.

        Args:
            action: Action index from ActionType enum

        Returns:
            Action dictionary for environment execution
        """
        action_dict = {
            'vertical_cpu': 0,
            'vertical_memory': 0,
            'horizontal': 0
        }

        if action == ActionType.VERTICAL_CPU_UP:
            action_dict['vertical_cpu'] = 1
        elif action == ActionType.VERTICAL_CPU_DOWN:
            action_dict['vertical_cpu'] = -1
        elif action == ActionType.VERTICAL_MEMORY_UP:
            action_dict['vertical_memory'] = 1
        elif action == ActionType.VERTICAL_MEMORY_DOWN:
            action_dict['vertical_memory'] = -1
        elif action == ActionType.HORIZONTAL_UP:
            action_dict['horizontal'] = 1
        elif action == ActionType.HORIZONTAL_DOWN:
            action_dict['horizontal'] = -1
        # NO_ACTION requires no changes (all zeros)

        return action_dict

    def convert_action_to_k8s_changes(
        self,
        action: int,
        current_state: Dict[str, float]
    ) -> Dict[str, Any]:
        """
        Convert action to actual Kubernetes resource changes.

        Args:
            action: Action index
            current_state: Current resource state

        Returns:
            Kubernetes resource change specification
        """
        changes = {
            'cpu_request_mcpu': None,
            'memory_request_mib': None,
            'replicas': None,
            'action_type': ActionType(action).name
        }

        action_dict = self.convert_action_to_dict(action)

        # Apply vertical CPU scaling
        if action_dict['vertical_cpu'] != 0:
            current_cpu = current_state.get('cpu_limit', 1000)  # millicore
            new_cpu = current_cpu + \
                (action_dict['vertical_cpu'] * self.scaling_step_cpu)
            changes['cpu_request_mcpu'] = max(
                100, min(8000, new_cpu))  # Bounds check

        # Apply vertical memory scaling
        if action_dict['vertical_memory'] != 0:
            current_memory = current_state.get('memory_limit', 512)  # MiB
            new_memory = current_memory + \
                (action_dict['vertical_memory'] * self.scaling_step_memory)
            changes['memory_request_mib'] = max(
                128, min(16384, new_memory))  # Bounds check

        # Apply horizontal scaling
        if action_dict['horizontal'] != 0:
            current_replicas = int(current_state.get('num_replicas', 1))
            new_replicas = current_replicas + action_dict['horizontal']
            changes['replicas'] = max(1, min(50, new_replicas))  # Bounds check

        return changes

    def get_action_description(self, action: int) -> str:
        """Get human-readable description of action."""
        action_type = ActionType(action)

        descriptions = {
            ActionType.NO_ACTION: "No scaling action",
            ActionType.VERTICAL_CPU_UP: f"Scale up CPU by {self.scaling_step_cpu}m",
            ActionType.VERTICAL_CPU_DOWN: f"Scale down CPU by {self.scaling_step_cpu}m",
            ActionType.VERTICAL_MEMORY_UP: f"Scale up memory by {self.scaling_step_memory}Mi",
            ActionType.VERTICAL_MEMORY_DOWN: f"Scale down memory by {self.scaling_step_memory}Mi",
            ActionType.HORIZONTAL_UP: "Scale up replicas by 1",
            ActionType.HORIZONTAL_DOWN: "Scale down replicas by 1"
        }

        return descriptions.get(action_type, f"Unknown action: {action}")


class FeatureExtractor:
    """
    Extract and preprocess features from Kubernetes metrics and Prometheus data.

    This class handles the conversion from raw monitoring data to
    normalized features suitable for RL training and inference.
    """

    def __init__(self):
        self.state_space = StateSpace()
        self.history_window = 5  # Number of historical samples to keep
        self.feature_history = []

    def extract_from_prometheus_data(
        self,
        prometheus_metrics: Dict[str, List[Tuple[float, float]]],
        custom_metrics: Dict[str, List[Tuple[float, float]]]
    ) -> np.ndarray:
        """
        Extract features from Prometheus time series data.

        Args:
            prometheus_metrics: System metrics from Prometheus
            custom_metrics: Application-specific metrics

        Returns:
            Normalized feature vector
        """
        # Aggregate time series data to single values
        aggregated_metrics = {}

        # Process system metrics
        for metric_name, time_series in prometheus_metrics.items():
            if time_series:
                # Use average of recent samples
                values = [float(value) for _, value in time_series]
                aggregated_metrics[metric_name] = np.mean(values)

        # Process custom application metrics
        for metric_name, time_series in custom_metrics.items():
            if time_series:
                values = [float(value) for _, value in time_series]
                aggregated_metrics[metric_name] = np.mean(values)

        # Map to standard metric names expected by state space
        standard_metrics = self._map_prometheus_to_standard_metrics(
            aggregated_metrics)

        # Extract features
        features = self.state_space.extract_features(standard_metrics)

        # Add to history for temporal analysis
        self.feature_history.append(features)
        if len(self.feature_history) > self.history_window:
            self.feature_history.pop(0)

        return features

    def _map_prometheus_to_standard_metrics(
        self,
        prometheus_metrics: Dict[str, float]
    ) -> Dict[str, float]:
        """
        Map Prometheus metric names to standard state space names.
        """
        mapping = {
            # CPU metrics
            'container_cpu_usage_seconds_total': 'cpu_util',
            'cpu_utilization': 'cpu_util',

            # Memory metrics
            'container_memory_usage_bytes': 'memory_util',
            'memory_utilization': 'memory_util',

            # Disk I/O metrics
            'container_fs_usage_bytes': 'disk_io_usage',

            # Network metrics
            'container_network_receive_bytes_total': 'ingestion_rate',
            'container_network_transmit_bytes_total': 'processing_rate',

            # Application metrics
            'request_rate': 'processing_rate',
            'p95_latency_ms': 'latency',
            'error_rate': 'error_rate',
        }

        standard_metrics = {}

        for prom_name, value in prometheus_metrics.items():
            standard_name = mapping.get(prom_name, prom_name)
            standard_metrics[standard_name] = value

        return standard_metrics

    def get_temporal_features(self) -> Optional[np.ndarray]:
        """
        Get temporal features from recent history.

        Returns:
            Stacked feature vectors from recent history, or None if insufficient history
        """
        if len(self.feature_history) < 3:  # Need at least 3 samples for trends
            return None

        # Stack recent features
        temporal_features = np.stack(
            self.feature_history[-3:])  # Last 3 samples

        # Add trend information (difference between recent samples)
        trends = np.diff(temporal_features, axis=0)

        # Combine current features with trend information
        current_features = temporal_features[-1]  # Most recent
        recent_trend = trends[-1] if len(
            trends) > 0 else np.zeros_like(current_features)

        # Concatenate current state with recent trend
        combined_features = np.concatenate([current_features, recent_trend])

        return combined_features
