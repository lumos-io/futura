#!/usr/bin/env python3
"""
Quick test script for the HPA/VPA scaling algorithms.

This script validates that the scaling algorithms work correctly
with different resource states and SLO configurations.
"""

from scaling_algorithms import ScalingAlgorithms, ResourceState
import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def test_horizontal_scaling():
    """Test horizontal scaling scenarios."""
    print("🧪 Testing Horizontal Scaling...")

    scaling = ScalingAlgorithms()

    # High utilization -> scale out
    high_util_state = ResourceState(
        num_replicas=2,
        cpu_limit=1000,
        memory_limit=512,
        cpu_util=0.95,  # High CPU
        memory_util=0.85,  # High memory
        request_rate=200.0,
        p95_latency_ms=300.0
    )

    action = scaling.should_scale_horizontally(high_util_state, "test-app")
    print(
        f"High utilization action: {action.action_type if action else 'None'}")
    if action:
        print(
            f"  Target replicas: {action.target_replicas}, Reason: {action.reason}")

    # Low utilization -> scale in
    low_util_state = ResourceState(
        num_replicas=5,
        cpu_limit=1000,
        memory_limit=512,
        cpu_util=0.3,  # Low CPU
        memory_util=0.2,  # Low memory
        request_rate=50.0,
        p95_latency_ms=150.0
    )

    action = scaling.should_scale_horizontally(low_util_state, "test-app")
    print(
        f"Low utilization action: {action.action_type if action else 'None'}")
    if action:
        print(
            f"  Target replicas: {action.target_replicas}, Reason: {action.reason}")

    print()


def test_vertical_scaling():
    """Test vertical scaling scenarios."""
    print("🧪 Testing Vertical Scaling...")

    scaling = ScalingAlgorithms()

    # High CPU -> scale up CPU
    high_cpu_state = ResourceState(
        num_replicas=3,  # More than 1 for VPA
        cpu_limit=1000,
        memory_limit=512,
        cpu_util=0.95,
        memory_util=0.4,
        request_rate=150.0,
        p95_latency_ms=250.0
    )

    action = scaling.should_scale_vertically(high_cpu_state, "test-app")
    print(f"High CPU action: {action.action_type if action else 'None'}")
    if action:
        print(
            f"  Target CPU: {action.target_cpu_mcpu}, Reason: {action.reason}")

    # High Memory -> scale up memory
    high_mem_state = ResourceState(
        num_replicas=3,
        cpu_limit=1000,
        memory_limit=512,
        cpu_util=0.4,
        memory_util=0.95,
        request_rate=150.0,
        p95_latency_ms=250.0
    )

    action = scaling.should_scale_vertically(high_mem_state, "test-app")
    print(f"High Memory action: {action.action_type if action else 'None'}")
    if action:
        print(
            f"  Target Memory: {action.target_memory_mib}, Reason: {action.reason}")

    print()


def test_slo_violations():
    """Test SLO violation scenarios."""
    print("🧪 Testing SLO Violations...")

    scaling = ScalingAlgorithms()

    # High latency violation
    slo_violating_state = ResourceState(
        num_replicas=2,
        cpu_limit=1000,
        memory_limit=512,
        cpu_util=0.6,
        memory_util=0.5,
        request_rate=200.0,
        p95_latency_ms=800.0,  # High latency
        error_rate=0.02  # High error rate
    )

    slo_targets = {
        'target_p95_latency_ms': 400.0,
        'target_error_rate': 0.01,
        'target_throughput_rps': 150.0
    }

    action = scaling.get_intelligent_scaling_action(
        current_state=slo_violating_state,
        slo_targets=slo_targets,
        app_key="test-app"
    )

    print(f"SLO violation action: {action.action_type}")
    print(f"  Reason: {action.reason}")
    print(f"  Confidence: {action.confidence}")

    print()


def test_performance_bottlenecks():
    """Test performance bottleneck detection."""
    print("🧪 Testing Performance Bottlenecks...")

    scaling = ScalingAlgorithms()

    # Processing lag with high CPU
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

    action = scaling.get_intelligent_scaling_action(
        current_state=bottleneck_state,
        slo_targets=None,
        app_key="test-app"
    )

    print(f"Processing bottleneck action: {action.action_type}")
    print(f"  Reason: {action.reason}")
    print(f"  Confidence: {action.confidence}")

    print()


def test_no_action_needed():
    """Test scenario where no action is needed."""
    print("🧪 Testing No Action Scenario...")

    scaling = ScalingAlgorithms()

    # Balanced state
    balanced_state = ResourceState(
        num_replicas=3,
        cpu_limit=1000,
        memory_limit=512,
        cpu_util=0.75,  # Good utilization
        memory_util=0.65,  # Good utilization
        request_rate=150.0,
        p95_latency_ms=200.0,  # Good latency
        processing_rate=150.0,
        ingestion_rate=150.0  # Balanced processing
    )

    action = scaling.get_intelligent_scaling_action(
        current_state=balanced_state,
        slo_targets={'target_p95_latency_ms': 400.0,
                     'target_error_rate': 0.01},
        app_key="test-app"
    )

    print(f"Balanced state action: {action.action_type}")
    print(f"  Reason: {action.reason}")

    print()


if __name__ == "__main__":
    print("🚀 Testing HPA/VPA Scaling Algorithms\n")

    test_horizontal_scaling()
    test_vertical_scaling()
    test_slo_violations()
    test_performance_bottlenecks()
    test_no_action_needed()

    print("✅ All scaling algorithm tests completed!")
