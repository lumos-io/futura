"""
Configuration management for Futura Engine.
"""

import os
import tomllib
from dataclasses import dataclass
from typing import Optional


@dataclass
class ServerConfig:
    """Main server configuration."""
    host: str = "0.0.0.0"
    port: int = 8080
    max_workers: int = 50
    log_level: str = "INFO"


@dataclass
class ClickHouseConfig:
    """ClickHouse database configuration."""
    url: str = "http://localhost:8123"
    engine_db: str = "engine"
    analytics_db: str = "analytics"


@dataclass
class TrainingConfig:
    """Training job configuration."""
    namespace: str = "futura-training"
    image: str = "futura/rl-trainer:latest"
    model_storage_uri: str = "s3://futura-models"
    default_timeout_hours: int = 12


@dataclass
class RLConfig:
    """RL Server specific configuration."""
    default_horizon_hours: int = 24
    model_cleanup_hours: int = 168
    enable_drift_detection: bool = True
    drift_check_interval_hours: int = 6


@dataclass
class AgentCoordinatorConfig:
    """Agent Coordinator specific configuration."""
    max_training_job_age_hours: int = 24
    cleanup_interval_hours: int = 1
    heartbeat_timeout_minutes: int = 10


@dataclass
class RecommendationConfig:
    """Recommendation Service specific configuration."""
    default_cpu_request_mcpu: int = 1000
    default_memory_request_mib: int = 512
    max_scale_up_ratio: float = 2.0
    max_scale_down_ratio: float = 0.5
    min_cooldown_seconds: int = 300


@dataclass
class EngineConfig:
    """Complete engine configuration."""
    service: str = "all"  # "all", "recommendation", "rl", "agent-coordinator"
    rl_server_address: Optional[str] = None
    server: ServerConfig = None
    clickhouse: ClickHouseConfig = None
    training: TrainingConfig = None
    rl: RLConfig = None
    agent_coordinator: AgentCoordinatorConfig = None
    recommendation: RecommendationConfig = None

    @classmethod
    def from_toml(cls, path: str = "config.toml") -> 'EngineConfig':
        """Load configuration from TOML file."""
        with open(path, "rb") as f:
            data = tomllib.load(f)

        return cls(
            service=data.get("service", "all"),
            rl_server_address=data.get("rl_server_address"),
            server=ServerConfig(**data.get("server", {})),
            clickhouse=ClickHouseConfig(**data.get("clickhouse", {})),
            training=TrainingConfig(**data.get("training", {})),
            rl=RLConfig(**data.get("rl", {})),
            agent_coordinator=AgentCoordinatorConfig(
                **data.get("agent_coordinator", {})),
            recommendation=RecommendationConfig(
                **data.get("recommendation", {}))
        )

    @classmethod
    def default(cls) -> 'EngineConfig':
        """Create default configuration."""
        return cls(
            service="all",
            server=ServerConfig(),
            clickhouse=ClickHouseConfig(),
            training=TrainingConfig(),
            rl=RLConfig(),
            agent_coordinator=AgentCoordinatorConfig(),
            recommendation=RecommendationConfig()
        )
