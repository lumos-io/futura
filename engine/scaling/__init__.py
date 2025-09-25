"""
Scaling algorithms package for Futura Engine.

This package provides HPA and VPA scaling algorithms extracted from
the controller's RL environment for intelligent Kubernetes resource scaling.
"""

from .scaling_algorithms import (
    ScalingAlgorithms,
    ScalingAction,
    ResourceState,
    ScalingConstraints
)

__all__ = [
    'ScalingAlgorithms',
    'ScalingAction',
    'ResourceState',
    'ScalingConstraints'
]