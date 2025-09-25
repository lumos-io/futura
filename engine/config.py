"""
Configuration management for Futura Engine.
"""

import os
from dataclasses import dataclass
from typing import Optional


@dataclass
class ServerConfig:
    """Main server configuration."""
    port: int = 50051
    max_workers: int = 50
    enable_reflection: bool = True
    enable_health_check: bool = True


@dataclass
class RLServerConfig:
    """RL Server specific configuration."""
    model_store_base_uri: str = "s3://futura-models"
    clickhouse_dsn: str = "clickhouse://localhost:9000/default"
    default_horizon_hours: int = 24
    model_cleanup_hours: int = 168  # 7 days
    enable_drift_detection: bool = True
    drift_check_interval_hours: int = 6


@dataclass
class AgentCoordinatorConfig:
    """Agent Coordinator specific configuration."""
    max_training_job_age_hours: int = 24
    cleanup_interval_hours: int = 1
    heartbeat_timeout_minutes: int = 10
    default_training_timeout_hours: int = 12


@dataclass
class RecommendationServiceConfig:
    """Recommendation Service specific configuration."""
    default_cpu_request_mcpu: int = 1000  # 1 CPU
    default_memory_request_mib: int = 512  # 512 MiB
    max_scale_up_ratio: float = 2.0
    max_scale_down_ratio: float = 0.5
    min_cooldown_seconds: int = 300  # 5 minutes


@dataclass
class FuturaEngineConfig:
    """Complete engine configuration."""
    server: ServerConfig
    rl_server: RLServerConfig
    agent_coordinator: AgentCoordinatorConfig
    recommendation_service: RecommendationServiceConfig

    @classmethod
    def from_env(cls) -> 'FuturaEngineConfig':
        """Create configuration from environment variables."""
        return cls(
            server=ServerConfig(
                port=int(os.getenv('FUTURA_PORT', 50051)),
                max_workers=int(os.getenv('FUTURA_MAX_WORKERS', 50)),
                enable_reflection=os.getenv('FUTURA_ENABLE_REFLECTION', 'true').lower() == 'true',
                enable_health_check=os.getenv('FUTURA_ENABLE_HEALTH_CHECK', 'true').lower() == 'true'
            ),
            rl_server=RLServerConfig(
                model_store_base_uri=os.getenv('FUTURA_MODEL_STORE_URI', 's3://futura-models'),
                clickhouse_dsn=os.getenv('FUTURA_CLICKHOUSE_DSN', 'clickhouse://localhost:9000/default'),
                default_horizon_hours=int(os.getenv('FUTURA_DEFAULT_HORIZON_HOURS', 24)),
                model_cleanup_hours=int(os.getenv('FUTURA_MODEL_CLEANUP_HOURS', 168)),
                enable_drift_detection=os.getenv('FUTURA_ENABLE_DRIFT_DETECTION', 'true').lower() == 'true',
                drift_check_interval_hours=int(os.getenv('FUTURA_DRIFT_CHECK_INTERVAL_HOURS', 6))
            ),
            agent_coordinator=AgentCoordinatorConfig(
                max_training_job_age_hours=int(os.getenv('FUTURA_MAX_TRAINING_JOB_AGE_HOURS', 24)),
                cleanup_interval_hours=int(os.getenv('FUTURA_CLEANUP_INTERVAL_HOURS', 1)),
                heartbeat_timeout_minutes=int(os.getenv('FUTURA_HEARTBEAT_TIMEOUT_MINUTES', 10)),
                default_training_timeout_hours=int(os.getenv('FUTURA_DEFAULT_TRAINING_TIMEOUT_HOURS', 12))
            ),
            recommendation_service=RecommendationServiceConfig(
                default_cpu_request_mcpu=int(os.getenv('FUTURA_DEFAULT_CPU_REQUEST_MCPU', 1000)),
                default_memory_request_mib=int(os.getenv('FUTURA_DEFAULT_MEMORY_REQUEST_MIB', 512)),
                max_scale_up_ratio=float(os.getenv('FUTURA_MAX_SCALE_UP_RATIO', 2.0)),
                max_scale_down_ratio=float(os.getenv('FUTURA_MAX_SCALE_DOWN_RATIO', 0.5)),
                min_cooldown_seconds=int(os.getenv('FUTURA_MIN_COOLDOWN_SECONDS', 300))
            )
        )

    @classmethod
    def default(cls) -> 'FuturaEngineConfig':
        """Create default configuration."""
        return cls(
            server=ServerConfig(),
            rl_server=RLServerConfig(),
            agent_coordinator=AgentCoordinatorConfig(),
            recommendation_service=RecommendationServiceConfig()
        )