import grpc
from google.protobuf import empty_pb2, timestamp_pb2
from proto.gen.engine import engine_pb2, engine_pb2_grpc
from typing import Dict, Optional, Set, List
import logging
from datetime import datetime, timedelta, timezone

logger = logging.getLogger(__name__)


class AgentCoordinator(engine_pb2_grpc.AgentCoordinatorServicer):
    """
    Agent Coordinator - Training job protocol for ephemeral trainer pods.

    This service manages the lifecycle of training agents/pods, coordinates
    training specifications, receives progress reports, and handles results.
    Training agents are ephemeral Kubernetes Jobs that pull assignments and
    push results back.
    """

    def __init__(self):
        # Training job registry
        # training_id -> spec
        self.training_specs: Dict[str, engine_pb2.TrainingSpec] = {}
        # agent_id -> registration info
        self.registered_agents: Dict[str, dict] = {}
        # training_id -> progress list
        self.training_progress: Dict[str,
                                     List[engine_pb2.TrainingProgress]] = {}
        # training_id -> final result
        self.training_results: Dict[str, engine_pb2.TrainingResult] = {}
        # agent_id -> last heartbeat time
        self.agent_heartbeats: Dict[str, datetime] = {}

        # Training job status tracking
        # training_id -> status (pending, running, completed, failed, cancelled)
        self.training_status: Dict[str, str] = {}
        self.cancelled_jobs: Set[str] = set()  # cancelled training_ids

        # Configuration
        self.default_clickhouse_dsn = "clickhouse://clickhouse:9000/futura"
        self.default_output_base_uri = "s3://futura-checkpoints"

    def RegisterAgent(
        self,
        request: engine_pb2.AgentRegistration,
        context: grpc.ServicerContext
    ) -> engine_pb2.AgentRegistrationAck:
        """
        Register a training agent pod for a specific training job.
        """
        logger.info(
            f"Registering agent {request.agent_id} for training {request.training_id}")

        try:
            # Check if training job exists and is valid
            if request.training_id not in self.training_specs:
                logger.warning(f"Training job {request.training_id} not found")
                return engine_pb2.AgentRegistrationAck(
                    accepted=False,
                    message=f"Training job {request.training_id} not found or expired"
                )

            # Check if training job is already completed or cancelled
            status = self.training_status.get(request.training_id, "pending")
            if status in ["completed", "failed", "cancelled"]:
                logger.warning(
                    f"Training job {request.training_id} is already {status}")
                return engine_pb2.AgentRegistrationAck(
                    accepted=False,
                    message=f"Training job is already {status}"
                )

            # Register the agent
            agent_info = {
                "agent_id": request.agent_id,
                "training_id": request.training_id,
                "version": request.version,
                "labels": dict(request.labels) if request.labels else {},
                "registered_at": datetime.now(timezone.utc),
                "status": "registered"
            }

            self.registered_agents[request.agent_id] = agent_info

            # Update training status to running
            self.training_status[request.training_id] = "running"

            # Initialize progress tracking
            if request.training_id not in self.training_progress:
                self.training_progress[request.training_id] = []

            logger.info(
                f"Agent {request.agent_id} registered successfully for training {request.training_id}")

            return engine_pb2.AgentRegistrationAck(
                accepted=True,
                message=f"Agent registered for training {request.training_id}"
            )

        except Exception as e:
            logger.error(f"Error registering agent: {str(e)}")
            return engine_pb2.AgentRegistrationAck(
                accepted=False,
                message=f"Registration error: {str(e)}"
            )

    def FetchTrainingSpec(
        self,
        request: engine_pb2.TrainingPollRequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.TrainingSpec:
        """
        Agent polls to fetch the training specification (hyperparams, data window, output URI).
        """
        logger.info(
            f"Agent {request.agent_id} polling for training spec: {request.training_id}")

        try:
            # Verify agent is registered
            if request.agent_id not in self.registered_agents:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details(f"Agent {request.agent_id} not registered")
                return engine_pb2.TrainingSpec()

            agent_info = self.registered_agents[request.agent_id]
            if agent_info["training_id"] != request.training_id:
                context.set_code(grpc.StatusCode.PERMISSION_DENIED)
                context.set_details(
                    f"Agent not authorized for training {request.training_id}")
                return engine_pb2.TrainingSpec()

            # Check if training is cancelled
            if request.training_id in self.cancelled_jobs:
                context.set_code(grpc.StatusCode.CANCELLED)
                context.set_details(
                    f"Training {request.training_id} has been cancelled")
                return engine_pb2.TrainingSpec()

            # Get training specification
            if request.training_id not in self.training_specs:
                context.set_code(grpc.StatusCode.NOT_FOUND)
                context.set_details(
                    f"Training spec for {request.training_id} not found")
                return engine_pb2.TrainingSpec()

            spec = self.training_specs[request.training_id]
            logger.info(
                f"Returning training spec for {request.training_id}: {spec.horizon_hours}h horizon")

            return spec

        except Exception as e:
            logger.error(f"Error fetching training spec: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Error fetching spec: {str(e)}")
            return engine_pb2.TrainingSpec()

    def ReportProgress(
        self,
        request: engine_pb2.TrainingProgress,
        context: grpc.ServicerContext
    ) -> empty_pb2.Empty:
        """
        Agent reports periodic training progress (metrics, scalar logs).
        """
        logger.info(
            f"Progress from agent {request.agent_id}: step {request.step}, progress {request.progress_pct:.1f}%")

        try:
            # Verify agent
            if request.agent_id not in self.registered_agents:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details(f"Agent {request.agent_id} not registered")
                return empty_pb2.Empty()

            # Store progress
            if request.training_id not in self.training_progress:
                self.training_progress[request.training_id] = []

            self.training_progress[request.training_id].append(request)

            # Log key metrics
            logger.info(f"Training {request.training_id}: step={request.step}, "
                        f"loss={request.train_loss:.4f}, reward={request.eval_reward:.4f}")

            # Update agent status
            if request.agent_id in self.registered_agents:
                self.registered_agents[request.agent_id]["last_progress"] = datetime.now(
                    timezone.utc)

            return empty_pb2.Empty()

        except Exception as e:
            logger.error(f"Error processing training progress: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Progress processing error: {str(e)}")
            return empty_pb2.Empty()

    def ReportResult(
        self,
        request: engine_pb2.TrainingResult,
        context: grpc.ServicerContext
    ) -> empty_pb2.Empty:
        """
        Agent reports final training result (success/failure + checkpoint URI).
        """
        status = "SUCCESS" if request.success else "FAILED"
        logger.info(f"Training result from agent {request.agent_id}: {status}")

        try:
            # Verify agent
            if request.agent_id not in self.registered_agents:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details(f"Agent {request.agent_id} not registered")
                return empty_pb2.Empty()

            # Store result
            self.training_results[request.training_id] = request

            # Update training status
            if request.success:
                self.training_status[request.training_id] = "completed"
                logger.info(
                    f"Training {request.training_id} completed successfully")
                logger.info(f"Model checkpoint: {request.checkpoint_uri}")
                logger.info(f"Final eval reward: {request.eval_reward:.4f}")

                # Log final metrics
                if request.metrics:
                    metrics_str = ", ".join(
                        [f"{k}={v:.4f}" for k, v in request.metrics.items()])
                    logger.info(f"Final metrics: {metrics_str}")

            else:
                self.training_status[request.training_id] = "failed"
                logger.error(
                    f"Training {request.training_id} failed: {request.error_message}")

            # Update agent status
            if request.agent_id in self.registered_agents:
                self.registered_agents[request.agent_id]["status"] = "completed"
                self.registered_agents[request.agent_id]["completed_at"] = datetime.now(
                    timezone.utc)

            # TODO: In a real implementation, notify the RL Server about the new model
            # This would involve updating the model registry and potentially
            # triggering hot-reload of the new model version

            return empty_pb2.Empty()

        except Exception as e:
            logger.error(f"Error processing training result: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Result processing error: {str(e)}")
            return empty_pb2.Empty()

    def Heartbeat(
        self,
        request: engine_pb2.AgentHeartbeat,
        context: grpc.ServicerContext
    ) -> empty_pb2.Empty:
        """
        Agent heartbeat for liveness tracking.
        """
        logger.debug(f"Heartbeat from agent {request.agent_id}")

        try:
            # Verify agent
            if request.agent_id not in self.registered_agents:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details(f"Agent {request.agent_id} not registered")
                return empty_pb2.Empty()

            # Update heartbeat timestamp
            self.agent_heartbeats[request.agent_id] = datetime.now(
                timezone.utc)

            # Log system info if provided
            if request.sysinfo:
                logger.debug(
                    f"Agent {request.agent_id} system info: {dict(request.sysinfo)}")

            return empty_pb2.Empty()

        except Exception as e:
            logger.error(f"Error processing heartbeat: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Heartbeat processing error: {str(e)}")
            return empty_pb2.Empty()

    def CancelTraining(
        self,
        request: engine_pb2.CancelTrainingRequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.CancelTrainingAck:
        """
        Cancel a training job. Agents should poll or receive this via side channel.
        """
        logger.info(
            f"Cancelling training {request.training_id}: {request.reason}")

        try:
            # Check if training exists
            if request.training_id not in self.training_specs:
                return engine_pb2.CancelTrainingAck(
                    acknowledged=False,
                    message=f"Training {request.training_id} not found"
                )

            # Mark as cancelled
            self.cancelled_jobs.add(request.training_id)
            self.training_status[request.training_id] = "cancelled"

            # Find and update associated agents
            cancelled_agents = []
            for agent_id, agent_info in self.registered_agents.items():
                if agent_info["training_id"] == request.training_id:
                    agent_info["status"] = "cancelled"
                    agent_info["cancelled_at"] = datetime.now(timezone.utc)
                    cancelled_agents.append(agent_id)

            logger.info(
                f"Training {request.training_id} cancelled, affected agents: {cancelled_agents}")

            return engine_pb2.CancelTrainingAck(
                acknowledged=True,
                message=f"Training {request.training_id} cancelled successfully"
            )

        except Exception as e:
            logger.error(f"Error cancelling training: {str(e)}")
            return engine_pb2.CancelTrainingAck(
                acknowledged=False,
                message=f"Cancellation error: {str(e)}"
            )

    def create_training_spec(
        self,
        training_id: str,
        app: engine_pb2.AppRef,
        horizon_hours: int = 24,
        base_version: str = "",
        hparams: Optional[Dict[str, str]] = None
    ) -> engine_pb2.TrainingSpec:
        """
        Create a training specification for a new training job.
        This is typically called by the RL Server when TriggerTrain is invoked.
        """
        # Generate output URI for this training job
        app_path = f"{app.namespace}-{app.app_name}".replace("/", "-")
        output_uri = f"{self.default_output_base_uri}/{app_path}/{training_id}"

        # Set training time window (default to last N hours)
        now = timestamp_pb2.Timestamp()
        now.GetCurrentTime()

        start_time = timestamp_pb2.Timestamp()
        start_time.seconds = now.seconds - (horizon_hours * 3600)

        # Create training spec
        spec = engine_pb2.TrainingSpec(
            training_id=training_id,
            app=app,
            horizon_hours=horizon_hours,
            base_version=base_version,
            hparams=hparams or {},
            output_uri=output_uri,
            clickhouse_dsn=self.default_clickhouse_dsn,
            start_at=start_time,
            end_at=now
        )

        # Store the spec
        self.training_specs[training_id] = spec
        self.training_status[training_id] = "pending"

        logger.info(
            f"Created training spec {training_id} for {app.namespace}/{app.app_name}")
        return spec

    def get_training_status(self, training_id: str) -> Optional[str]:
        """Get the current status of a training job."""
        return self.training_status.get(training_id)

    def get_training_progress(self, training_id: str) -> List[engine_pb2.TrainingProgress]:
        """Get all progress reports for a training job."""
        return self.training_progress.get(training_id, [])

    def get_training_result(self, training_id: str) -> Optional[engine_pb2.TrainingResult]:
        """Get the final result of a completed training job."""
        return self.training_results.get(training_id)

    def cleanup_expired_training_jobs(self, max_age_hours: int = 24):
        """Clean up old training jobs and agent registrations."""
        cutoff_time = datetime.now(timezone.utc) - \
            timedelta(hours=max_age_hours)

        # Find expired training jobs
        expired_jobs = []
        for training_id, spec in self.training_specs.items():
            if spec.end_at.ToDatetime() < cutoff_time:
                expired_jobs.append(training_id)

        # Clean up expired jobs
        for training_id in expired_jobs:
            logger.info(f"Cleaning up expired training job: {training_id}")

            # Remove from all tracking dictionaries
            self.training_specs.pop(training_id, None)
            self.training_status.pop(training_id, None)
            self.training_progress.pop(training_id, None)
            self.training_results.pop(training_id, None)
            self.cancelled_jobs.discard(training_id)

        # Clean up expired agent registrations
        expired_agents = []
        for agent_id, agent_info in self.registered_agents.items():
            if agent_info["registered_at"] < cutoff_time:
                expired_agents.append(agent_id)

        for agent_id in expired_agents:
            logger.info(f"Cleaning up expired agent registration: {agent_id}")
            self.registered_agents.pop(agent_id, None)
            self.agent_heartbeats.pop(agent_id, None)

        if expired_jobs or expired_agents:
            logger.info(
                f"Cleanup completed: removed {len(expired_jobs)} training jobs, {len(expired_agents)} agents")
