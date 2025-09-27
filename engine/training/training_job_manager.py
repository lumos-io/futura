"""
Kubernetes Training Job Manager for Futura Engine.

This module manages the creation, monitoring, and cleanup of Kubernetes Jobs
for RL model training with ClickHouse result storage.
"""

import logging
import asyncio
from typing import Dict, Optional, Any, List
from datetime import datetime
from dataclasses import dataclass
import json
import os

from kubernetes import client, config
from kubernetes.client.rest import ApiException

# Import storage layers
from storage.clickhouse_client import ClickHouseClient, EngineDataAccess

logger = logging.getLogger(__name__)


@dataclass
class TrainingJobSpec:
    """Specification for a training job."""

    training_id: str
    app_key: str
    job_name: str

    # Training parameters
    horizon_hours: int
    base_version: str
    hparams: Dict[str, Any]
    reason: str

    # Resource requirements
    cpu_request: str = "1"
    memory_request: str = "2Gi"
    cpu_limit: str = "2"
    memory_limit: str = "4Gi"

    # Storage
    output_uri: str = ""
    checkpoint_uri: str = ""

    # Environment
    clickhouse_dsn: str = ""
    training_data_hours: int = 24


@dataclass
class TrainingJobResult:
    """Result of a completed training job."""

    training_id: str
    job_name: str
    success: bool

    # Model output
    model_version: str = ""
    model_uri: str = ""
    checkpoint_uri: str = ""

    # Training metrics
    final_loss: float = 0.0
    final_reward: float = 0.0
    episodes_completed: int = 0
    training_time_seconds: int = 0

    # Logs and metadata
    logs: str = ""
    metrics: Dict[str, float] = None
    error_message: str = ""

    def __post_init__(self):
        if self.metrics is None:
            self.metrics = {}


class KubernetesTrainingJobManager:
    """
    Manages Kubernetes Jobs for RL model training.

    Features:
    - Creates training Jobs with proper resource allocation
    - Monitors job status and collects results
    - Stores training results in ClickHouse
    - Cleans up completed jobs and resources
    """

    def __init__(
        self,
        clickhouse_client: Optional[ClickHouseClient] = None,
        namespace: str = "futura-training",
        training_image: str = "futura/rl-trainer:latest",
        job_ttl_seconds: int = 3600  # 1 hour TTL after completion
    ):
        self.namespace = namespace
        self.training_image = training_image
        self.job_ttl_seconds = job_ttl_seconds

        # ClickHouse integration
        self.clickhouse_client = clickhouse_client
        self.engine_data: Optional[EngineDataAccess] = None
        if self.clickhouse_client:
            self.engine_data = EngineDataAccess(self.clickhouse_client)

        # Kubernetes client setup
        try:
            # Try in-cluster config first
            if 'KUBERNETES_SERVICE_HOST' in os.environ:
                config.load_incluster_config()
                logger.info("Loaded in-cluster Kubernetes config")
            else:
                # Fall back to local kubeconfig
                config.load_kube_config()
                logger.info("Loaded local Kubernetes config")

            self.batch_v1 = client.BatchV1Api()
            self.core_v1 = client.CoreV1Api()

        except Exception as e:
            logger.error(f"Failed to initialize Kubernetes client: {str(e)}")
            self.batch_v1 = None
            self.core_v1 = None

        # Job tracking
        self.active_jobs: Dict[str, TrainingJobSpec] = {}
        self.completed_jobs: Dict[str, TrainingJobResult] = {}

    async def create_training_job(self, spec: TrainingJobSpec) -> bool:
        """
        Create a Kubernetes Job for RL model training.
        """
        if not self.batch_v1:
            logger.error("Kubernetes client not available")
            return False

        try:
            logger.info(f"Creating training job: {spec.job_name}")

            # Create Job manifest
            job_manifest = self._build_job_manifest(spec)

            # Create the Job
            response = self.batch_v1.create_namespaced_job(
                namespace=self.namespace,
                body=job_manifest
            )

            # Track the job
            self.active_jobs[spec.training_id] = spec

            logger.info(f"Successfully created training job: {spec.job_name}")

            # Store job creation in ClickHouse
            if self.engine_data:
                await self._store_job_created(spec)

            return True

        except ApiException as e:
            logger.error(
                f"Failed to create training job {spec.job_name}: {e.reason}")
            return False
        except Exception as e:
            logger.error(
                f"Error creating training job {spec.job_name}: {str(e)}")
            return False

    async def get_job_status(self, training_id: str) -> Optional[str]:
        """Get the current status of a training job."""
        if training_id not in self.active_jobs:
            return None

        spec = self.active_jobs[training_id]

        try:
            job = self.batch_v1.read_namespaced_job(
                name=spec.job_name,
                namespace=self.namespace
            )

            # Check job conditions
            if job.status.succeeded:
                return "succeeded"
            elif job.status.failed:
                return "failed"
            elif job.status.active:
                return "running"
            else:
                return "pending"

        except ApiException as e:
            if e.status == 404:
                return "not_found"
            logger.error(
                f"Error getting job status for {spec.job_name}: {e.reason}")
            return "error"
        except Exception as e:
            logger.error(
                f"Error getting job status for {spec.job_name}: {str(e)}")
            return "error"

    async def collect_job_result(self, training_id: str) -> Optional[TrainingJobResult]:
        """
        Collect the result of a completed training job.
        """
        if training_id not in self.active_jobs:
            return None

        spec = self.active_jobs[training_id]
        status = await self.get_job_status(training_id)

        if status not in ["succeeded", "failed"]:
            return None

        try:
            # Get job details
            job = self.batch_v1.read_namespaced_job(
                name=spec.job_name,
                namespace=self.namespace
            )

            # Get pod logs
            logs = await self._get_job_logs(spec.job_name)

            # Parse result from logs or output location
            result = TrainingJobResult(
                training_id=training_id,
                job_name=spec.job_name,
                success=(status == "succeeded"),
                logs=logs
            )

            # Parse training metrics from logs
            result.metrics = self._parse_training_metrics(logs)
            if result.success:
                result.model_version = f"{spec.base_version}-{training_id[:8]}"
                result.model_uri = f"{spec.output_uri}/model.pkl"
                result.checkpoint_uri = f"{spec.output_uri}/checkpoint.pt"
                result.final_loss = result.metrics.get("final_loss", 0.0)
                result.final_reward = result.metrics.get("final_reward", 0.0)
                result.episodes_completed = int(
                    result.metrics.get("episodes", 0))
            else:
                result.error_message = self._extract_error_from_logs(logs)

            # Store result
            self.completed_jobs[training_id] = result

            # Store in ClickHouse
            if self.engine_data:
                await self._store_job_result(result)

            logger.info(
                f"Collected result for training job {spec.job_name}: success={result.success}")
            return result

        except Exception as e:
            logger.error(
                f"Error collecting job result for {spec.job_name}: {str(e)}")
            return None

    async def cleanup_completed_job(self, training_id: str) -> bool:
        """
        Clean up a completed training job and its resources.
        """
        if training_id not in self.active_jobs:
            return False

        spec = self.active_jobs[training_id]

        try:
            # Delete the Job
            await asyncio.to_thread(
                self.batch_v1.delete_namespaced_job,
                name=spec.job_name,
                namespace=self.namespace,
                propagation_policy="Background"
            )

            # Delete associated ConfigMap (if any)
            try:
                await asyncio.to_thread(
                    self.core_v1.delete_namespaced_config_map,
                    name=f"{spec.job_name}-config",
                    namespace=self.namespace
                )
            except ApiException:
                # ConfigMap might not exist, ignore
                pass

            # Remove from active jobs
            del self.active_jobs[training_id]

            logger.info(f"Cleaned up training job: {spec.job_name}")

            # Store cleanup event in ClickHouse
            if self.engine_data:
                await self._store_job_cleanup(training_id, spec.job_name)

            return True

        except ApiException as e:
            if e.status == 404:
                # Job already gone
                del self.active_jobs[training_id]
                return True
            logger.error(f"Error cleaning up job {spec.job_name}: {e.reason}")
            return False
        except Exception as e:
            logger.error(f"Error cleaning up job {spec.job_name}: {str(e)}")
            return False

    async def list_active_jobs(self) -> List[Dict[str, Any]]:
        """
        List all active training jobs in the namespace.
        Returns list of job information dictionaries.
        """
        try:
            if not self.batch_v1:
                return []

            # List jobs with our label selector
            jobs = self.batch_v1.list_namespaced_job(
                namespace=self.namespace,
                label_selector="app=futura-trainer"
            )

            active_jobs = []
            for job in jobs.items:
                # Only include running/active jobs
                if job.status.active:
                    job_info = {
                        "name": job.metadata.name,
                        "training_id": job.metadata.labels.get("training-id", "unknown"),
                        "status": "running",
                        "created": job.metadata.creation_timestamp.isoformat() if job.metadata.creation_timestamp else None
                    }
                    active_jobs.append(job_info)

            return active_jobs

        except Exception as e:
            logger.error(f"Error listing active jobs: {str(e)}")
            return []

    async def monitor_and_collect_jobs(self):
        """
        Single iteration of job monitoring and result collection.
        This is called periodically by the server's background task.
        """
        try:
            for training_id in list(self.active_jobs.keys()):
                status = await self.get_job_status(training_id)

                if status in ["succeeded", "failed"]:
                    # Collect result
                    result = await self.collect_job_result(training_id)
                    if result:
                        # Schedule cleanup after a delay
                        asyncio.create_task(
                            self._delayed_cleanup(
                                training_id, delay_seconds=300)  # 5 min delay
                        )

                elif status in ["not_found", "error"]:
                    # Job disappeared or errored, clean up tracking
                    logger.warning(
                        f"Training job {training_id} has status {status}, removing from tracking")
                    if training_id in self.active_jobs:
                        del self.active_jobs[training_id]

            # Log active job count
            if self.active_jobs:
                logger.debug(
                    f"Monitoring {len(self.active_jobs)} active training jobs")

        except Exception as e:
            logger.error(f"Error in job monitoring: {str(e)}")

    async def start_background_monitoring(self):
        """
        Start continuous background monitoring of training jobs.
        This runs the monitoring loop continuously.
        """
        logger.info("Starting background training job monitoring")

        while True:
            try:
                await self.monitor_and_collect_jobs()
                # Sleep before next check
                await asyncio.sleep(60)  # Check every minute

            except Exception as e:
                logger.error(f"Error in monitoring loop: {str(e)}")
                await asyncio.sleep(60)

    def _build_job_manifest(self, spec: TrainingJobSpec) -> client.V1Job:
        """Build a Kubernetes Job manifest for training."""

        # Environment variables for the training container
        env_vars = [
            client.V1EnvVar(name="TRAINING_ID", value=spec.training_id),
            client.V1EnvVar(name="APP_KEY", value=spec.app_key),
            client.V1EnvVar(name="HORIZON_HOURS",
                            value=str(spec.horizon_hours)),
            client.V1EnvVar(name="BASE_VERSION", value=spec.base_version),
            client.V1EnvVar(name="HPARAMS", value=json.dumps(spec.hparams)),
            client.V1EnvVar(name="REASON", value=spec.reason),
            client.V1EnvVar(name="OUTPUT_URI", value=spec.output_uri),
            client.V1EnvVar(name="CHECKPOINT_URI", value=spec.checkpoint_uri),
            client.V1EnvVar(name="CLICKHOUSE_DSN", value=spec.clickhouse_dsn),
            client.V1EnvVar(name="TRAINING_DATA_HOURS",
                            value=str(spec.training_data_hours)),
        ]

        # Container specification
        container = client.V1Container(
            name="rl-trainer",
            image=self.training_image,
            env=env_vars,
            resources=client.V1ResourceRequirements(
                requests={
                    "cpu": spec.cpu_request,
                    "memory": spec.memory_request
                },
                limits={
                    "cpu": spec.cpu_limit,
                    "memory": spec.memory_limit
                }
            ),
            # Add volume mounts for model storage if needed
            # volume_mounts=[...]
        )

        # Pod template
        pod_template = client.V1PodTemplateSpec(
            metadata=client.V1ObjectMeta(
                labels={
                    "app": "futura-rl-trainer",
                    "training-id": spec.training_id,
                    "app-key": spec.app_key.replace(":", "-").replace("/", "-")
                }
            ),
            spec=client.V1PodSpec(
                restart_policy="Never",
                containers=[container],
                # Add volumes for model storage if needed
                # volumes=[...]
            )
        )

        # Job specification
        job_spec = client.V1JobSpec(
            template=pod_template,
            backoff_limit=2,  # Retry up to 2 times
            ttl_seconds_after_finished=self.job_ttl_seconds
        )

        # Job manifest
        job = client.V1Job(
            api_version="batch/v1",
            kind="Job",
            metadata=client.V1ObjectMeta(
                name=spec.job_name,
                namespace=self.namespace,
                labels={
                    "app": "futura-rl-trainer",
                    "training-id": spec.training_id
                }
            ),
            spec=job_spec
        )

        return job

    async def _get_job_logs(self, job_name: str) -> str:
        """Get logs from a training job's pods."""
        try:
            # Get pods for this job
            pods = self.core_v1.list_namespaced_pod(
                namespace=self.namespace,
                label_selector=f"job-name={job_name}"
            )

            logs = []
            for pod in pods.items:
                try:
                    pod_logs = self.core_v1.read_namespaced_pod_log(
                        name=pod.metadata.name,
                        namespace=self.namespace,
                        container="rl-trainer"
                    )
                    logs.append(f"--- Pod {pod.metadata.name} ---\n{pod_logs}")
                except ApiException as e:
                    logs.append(
                        f"--- Pod {pod.metadata.name} (failed to get logs: {e.reason}) ---")

            return "\n\n".join(logs)

        except Exception as e:
            logger.error(f"Error getting logs for job {job_name}: {str(e)}")
            return ""

    def _parse_training_metrics(self, logs: str) -> Dict[str, float]:
        """Parse training metrics from job logs."""
        metrics = {}

        # Look for common training output patterns
        for line in logs.split('\n'):
            # Example patterns to parse
            if "Final Loss:" in line:
                try:
                    metrics["final_loss"] = float(
                        line.split("Final Loss:")[1].strip())
                except:
                    pass
            elif "Final Reward:" in line:
                try:
                    metrics["final_reward"] = float(
                        line.split("Final Reward:")[1].strip())
                except:
                    pass
            elif "Episodes Completed:" in line:
                try:
                    metrics["episodes"] = float(
                        line.split("Episodes Completed:")[1].strip())
                except:
                    pass

        return metrics

    def _extract_error_from_logs(self, logs: str) -> str:
        """Extract error message from job logs."""
        error_lines = []
        for line in logs.split('\n'):
            if any(keyword in line.lower() for keyword in ["error", "exception", "failed", "traceback"]):
                error_lines.append(line)

        if error_lines:
            return "\n".join(error_lines[-10:])  # Last 10 error lines
        return "Training failed with unknown error"

    async def _store_job_created(self, spec: TrainingJobSpec):
        """Store job creation event in ClickHouse."""
        try:
            if self.engine_data:
                # Store as a training event
                await self.engine_data.store_training_event(
                    training_id=spec.training_id,
                    app_key=spec.app_key,
                    job_name=spec.job_name,
                    event_type="job_created",
                    status="running",
                    metadata={
                        "horizon_hours": spec.horizon_hours,
                        "base_version": spec.base_version,
                        "reason": spec.reason,
                        "cpu_request": spec.cpu_request,
                        "memory_request": spec.memory_request
                    }
                )
        except Exception as e:
            logger.error(f"Error storing job creation event: {str(e)}")

    async def _store_job_result(self, result: TrainingJobResult):
        """Store job result in ClickHouse."""
        try:
            if self.engine_data:
                await self.engine_data.store_training_result(
                    training_id=result.training_id,
                    job_name=result.job_name,
                    success=result.success,
                    model_version=result.model_version,
                    model_uri=result.model_uri,
                    final_loss=result.final_loss,
                    final_reward=result.final_reward,
                    episodes_completed=result.episodes_completed,
                    training_metrics=result.metrics,
                    error_message=result.error_message,
                    completed_at=datetime.utcnow()
                )
        except Exception as e:
            logger.error(f"Error storing job result: {str(e)}")

    async def _store_job_cleanup(self, training_id: str, job_name: str):
        """Store job cleanup event in ClickHouse."""
        try:
            if self.engine_data:
                await self.engine_data.store_training_event(
                    training_id=training_id,
                    app_key="",
                    job_name=job_name,
                    event_type="job_cleaned_up",
                    status="completed",
                    metadata={"cleaned_up_at": datetime.utcnow().isoformat()}
                )
        except Exception as e:
            logger.error(f"Error storing job cleanup event: {str(e)}")

    async def _delayed_cleanup(self, training_id: str, delay_seconds: int = 300):
        """Clean up a job after a delay."""
        await asyncio.sleep(delay_seconds)
        await self.cleanup_completed_job(training_id)
