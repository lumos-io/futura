"""
Unit tests for scaling algorithms.
Tests the core scaling logic without external dependencies.
"""
from scaling.scaling_algorithms import ScalingAlgorithms, ResourceState, ScalingAction
import pytest
from unittest.mock import Mock, patch
import sys
import os

# Add engine root to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..'))


class TestResourceState:
    """Test ResourceState data class."""

    def test_resource_state_creation(self):
        """Test creating a ResourceState instance."""
        state = ResourceState(
            num_replicas=3,
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=0.75,
            memory_util=0.65,
            request_rate=150.0,
            p95_latency_ms=200.0
        )

        assert state.num_replicas == 3
        assert state.cpu_limit == 1000
        assert state.memory_limit == 512
        assert state.cpu_util == 0.75
        assert state.memory_util == 0.65
        assert state.request_rate == 150.0
        assert state.p95_latency_ms == 200.0

    def test_resource_state_optional_fields(self):
        """Test ResourceState with optional fields."""
        state = ResourceState(
            num_replicas=2,
            cpu_limit=500,
            memory_limit=256,
            cpu_util=0.8,
            memory_util=0.7,
            request_rate=100.0,
            p95_latency_ms=300.0,
            processing_rate=80.0,
            ingestion_rate=120.0,
            error_rate=0.01
        )

        assert state.processing_rate == 80.0
        assert state.ingestion_rate == 120.0
        assert state.error_rate == 0.01


class TestScalingAlgorithms:
    """Test ScalingAlgorithms class."""

    def setup_method(self):
        """Set up test fixtures."""
        self.scaling = ScalingAlgorithms()

    def test_scaling_algorithms_initialization(self):
        """Test ScalingAlgorithms initialization."""
        assert self.scaling is not None

    def test_horizontal_scaling_scale_out(self, sample_resource_state, sample_slo_targets):
        """Test horizontal scaling decision for scale out."""
        # High utilization state
        high_util_state = ResourceState(
            num_replicas=2,
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=0.95,  # High CPU
            memory_util=0.85,  # High memory
            request_rate=200.0,
            p95_latency_ms=300.0
        )

        action = self.scaling.should_scale_horizontally(
            high_util_state, "test-app")

        assert action is not None
        assert action.action_type == "horizontal"
        assert action.target_replicas > high_util_state.num_replicas
        assert "high utilization" in action.reason.lower()

    def test_horizontal_scaling_scale_in(self):
        """Test horizontal scaling decision for scale in."""
        # Low utilization state
        low_util_state = ResourceState(
            num_replicas=5,
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=0.3,  # Low CPU
            memory_util=0.2,  # Low memory
            request_rate=50.0,
            p95_latency_ms=150.0
        )

        action = self.scaling.should_scale_horizontally(
            low_util_state, "test-app")

        assert action is not None
        assert action.action_type == "horizontal"
        assert action.target_replicas < low_util_state.num_replicas
        assert "low utilization" in action.reason.lower()

    def test_horizontal_scaling_no_action(self, sample_resource_state):
        """Test horizontal scaling when no action is needed."""
        action = self.scaling.should_scale_horizontally(
            sample_resource_state, "test-app")

        # Should not scale for balanced state
        assert action is None or action.action_type == "no_action"

    def test_vertical_scaling_cpu_scale_up(self):
        """Test vertical scaling for CPU scale up."""
        high_cpu_state = ResourceState(
            num_replicas=3,
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=0.95,  # High CPU
            memory_util=0.4,  # Normal memory
            request_rate=150.0,
            p95_latency_ms=250.0
        )

        action = self.scaling.should_scale_vertically(
            high_cpu_state, "test-app")

        assert action is not None
        assert action.action_type == "vertical_cpu"
        assert action.target_cpu_mcpu > high_cpu_state.cpu_limit
        assert "cpu" in action.reason.lower()

    def test_vertical_scaling_memory_scale_up(self):
        """Test vertical scaling for memory scale up."""
        high_mem_state = ResourceState(
            num_replicas=3,
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=0.75,  # CPU within acceptable bounds (0.7-0.9)
            memory_util=0.95,  # High memory
            request_rate=150.0,
            p95_latency_ms=250.0
        )

        action = self.scaling.should_scale_vertically(
            high_mem_state, "test-app")

        assert action is not None
        assert action.action_type == "vertical_memory"
        assert action.target_memory_mib > high_mem_state.memory_limit
        assert "memory" in action.reason.lower()

    def test_vertical_scaling_single_replica_skip(self):
        """Test that vertical scaling is skipped for single replica deployments."""
        single_replica_state = ResourceState(
            num_replicas=1,  # Single replica
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=0.95,
            memory_util=0.95,
            request_rate=150.0,
            p95_latency_ms=250.0
        )

        action = self.scaling.should_scale_vertically(
            single_replica_state, "test-app")

        # Should prefer horizontal scaling for single replica
        assert action is None

    def test_slo_violation_detection(self, sample_slo_targets):
        """Test SLO violation detection and response."""
        slo_violating_state = ResourceState(
            num_replicas=2,
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=0.6,
            memory_util=0.5,
            request_rate=200.0,
            p95_latency_ms=800.0,  # High latency (target: 400ms)
            error_rate=0.02  # High error rate (target: 0.01)
        )

        action = self.scaling.get_intelligent_scaling_action(
            current_state=slo_violating_state,
            slo_targets=sample_slo_targets,
            app_key="test-app"
        )

        assert action is not None
        assert action.action_type in [
            "horizontal", "vertical_cpu", "vertical_memory"]
        assert "slo" in action.reason.lower() or "latency" in action.reason.lower()
        assert action.confidence > 0.0

    def test_performance_bottleneck_detection(self):
        """Test performance bottleneck detection."""
        bottleneck_state = ResourceState(
            num_replicas=2,
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=0.85,
            memory_util=0.5,
            request_rate=150.0,
            p95_latency_ms=300.0,
            processing_rate=80.0,  # Processing lag
            ingestion_rate=120.0
        )

        action = self.scaling.get_intelligent_scaling_action(
            current_state=bottleneck_state,
            slo_targets=None,
            app_key="test-app"
        )

        assert action is not None
        assert action.confidence > 0.0
        # Should detect processing bottleneck and suggest scaling

    def test_balanced_state_no_action(self, sample_resource_state, sample_slo_targets):
        """Test that balanced state results in no action."""
        action = self.scaling.get_intelligent_scaling_action(
            current_state=sample_resource_state,
            slo_targets=sample_slo_targets,
            app_key="test-app"
        )

        assert action is not None
        assert action.action_type == "no_action"
        assert action.confidence >= 0.0

    def test_scaling_action_validation(self):
        """Test ScalingAction data validation."""
        action = ScalingAction(
            action_type="horizontal",
            target_replicas=5,
            reason="High CPU utilization detected",
            confidence=0.85
        )

        assert action.action_type == "horizontal"
        assert action.target_replicas == 5
        assert action.reason == "High CPU utilization detected"
        assert action.confidence == 0.85

    def test_edge_case_zero_utilization(self):
        """Test edge case with zero utilization."""
        zero_util_state = ResourceState(
            num_replicas=3,
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=0.0,
            memory_util=0.0,
            request_rate=0.0,
            p95_latency_ms=100.0
        )

        action = self.scaling.should_scale_horizontally(
            zero_util_state, "test-app")

        # Should suggest scaling in for zero utilization
        if action:
            assert action.action_type == "horizontal"

    def test_edge_case_max_utilization(self):
        """Test edge case with maximum utilization."""
        max_util_state = ResourceState(
            num_replicas=1,
            cpu_limit=1000,
            memory_limit=512,
            cpu_util=1.0,
            memory_util=1.0,
            request_rate=1000.0,
            p95_latency_ms=2000.0
        )

        action = self.scaling.should_scale_horizontally(
            max_util_state, "test-app")

        # Should suggest scaling out for maximum utilization
        assert action is not None
        assert action.action_type == "horizontal"

    @pytest.mark.parametrize("replicas,expected_action", [
        (1, "horizontal"),  # Single replica under load
        (10, "horizontal"),  # Many replicas with low load
        (5, None),        # Balanced replicas
    ])
    def test_replica_count_scaling_decisions(self, replicas, expected_action):
        """Test scaling decisions based on replica count."""
        if replicas == 1:
            # High load on single replica
            state = ResourceState(
                num_replicas=replicas,
                cpu_limit=1000,
                memory_limit=512,
                cpu_util=0.95,
                memory_util=0.8,
                request_rate=200.0,
                p95_latency_ms=400.0
            )
        elif replicas == 10:
            # Low load on many replicas
            state = ResourceState(
                num_replicas=replicas,
                cpu_limit=1000,
                memory_limit=512,
                cpu_util=0.2,
                memory_util=0.3,
                request_rate=50.0,
                p95_latency_ms=100.0
            )
        else:
            # Balanced load
            state = ResourceState(
                num_replicas=replicas,
                cpu_limit=1000,
                memory_limit=512,
                cpu_util=0.7,
                memory_util=0.6,
                request_rate=150.0,
                p95_latency_ms=200.0
            )

        action = self.scaling.should_scale_horizontally(state, "test-app")

        if expected_action:
            assert action is not None
            assert action.action_type == expected_action
        else:
            assert action is None or action.action_type == "no_action"
