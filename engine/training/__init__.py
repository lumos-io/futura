"""
Training package for Futura Engine.

This package provides Kubernetes Job management for RL model training
with ClickHouse result storage and automatic cleanup.
"""

from .training_job_manager import (
    KubernetesTrainingJobManager,
    TrainingJobSpec,
    TrainingJobResult
)

__all__ = [
    'KubernetesTrainingJobManager',
    'TrainingJobSpec',
    'TrainingJobResult'
]