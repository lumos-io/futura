import sys
import os
sys.path.append(os.path.join(os.path.dirname(__file__), '../../proto/gen/engine'))

import grpc
from google.protobuf import empty_pb2, timestamp_pb2
import engine_pb2
import engine_pb2_grpc
from typing import Dict, Optional, List
import logging
import json
import uuid
from datetime import datetime, timedelta
import numpy as np

# Import RL models
from rl_models.reward_functions import RewardCalculator
from rl_models.state_action_space import StateSpace, ActionSpace, FeatureExtractor, ActionType

# Import storage layers
from storage.clickhouse_client import ClickHouseClient, EngineDataAccess
from storage.analytics_data_access import AnalyticsDataAccess
from storage.slo_data_access import SLODataAccess

# Import scaling algorithms
from scaling.scaling_algorithms import ScalingAlgorithms, ScalingAction, ResourceState, ScalingConstraints

logger = logging.getLogger(__name__)


class RLServer(engine_pb2_grpc.RLServerServicer):
    """
    RL Server (Control plane) - ML model serving and lifecycle management.

    This service provides online inference from RL models, manages model lifecycle,
    handles training orchestration, and maintains model metadata.
    """

    def __init__(self, clickhouse_client: Optional[ClickHouseClient] = None):
        # ClickHouse integration for data persistence and metrics
        self.clickhouse_client = clickhouse_client
        self.engine_data: Optional[EngineDataAccess] = None
        self.analytics_data: Optional[AnalyticsDataAccess] = None
        self.slo_data: Optional[SLODataAccess] = None

        if self.clickhouse_client:
            self.engine_data = EngineDataAccess(self.clickhouse_client)
            self.analytics_data = AnalyticsDataAccess(self.clickhouse_client)
            self.slo_data = SLODataAccess(self.clickhouse_client)

        # In-memory model registry (backed by ClickHouse when available)
        self.models: Dict[str, Dict[str, engine_pb2.ModelMetadata]] = {}  # app_key -> {version -> metadata}
        self.loaded_models: Dict[str, str] = {}  # app_key -> current_loaded_version
        self.training_jobs: Dict[str, dict] = {}  # training_id -> job_info

        # Model storage configuration
        self.model_store_base_uri = "s3://futura-models"  # Could be configurable
        self.clickhouse_dsn = "http://localhost:8123/engine"  # Updated for HTTP client

        # RL components (from controller implementation)
        self.reward_calculator = RewardCalculator(alpha=0.3, slo_weight=0.2)
        self.state_space = StateSpace()
        self.action_space = ActionSpace(scaling_step_cpu=256, scaling_step_memory=256)
        self.feature_extractors: Dict[str, FeatureExtractor] = {}  # app_key -> feature_extractor

        # Scaling algorithms integration
        self.scaling_algorithms = ScalingAlgorithms(
            constraints=ScalingConstraints(
                vertical_cpu_step=256,
                vertical_memory_step=256,
                max_instances=20,
                max_cpu_limit=4000,
                max_memory_limit=8192
            )
        )

        # Action history for reward calculation and oscillation detection
        self.action_history: Dict[str, List[Dict[str, int]]] = {}  # app_key -> action_history
        self.state_history: Dict[str, List[Dict[str, float]]] = {}  # app_key -> state_history

    async def GetAction(
        self,
        request: engine_pb2.GetActionRequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.GetActionResponse:
        """
        Online inference - RL policy picks an action plan based on current features.
        """
        app_key = self._get_app_key(request.app)
        logger.info(f"Getting action for app: {app_key}")

        try:
            # Check if we have a loaded model for this app
            loaded_version = self.loaded_models.get(app_key)
            if not loaded_version:
                logger.warning(f"No model loaded for {app_key}, using baseline policy")
                return self._generate_baseline_action(request)

            # Get model metadata
            model_meta = self.models.get(app_key, {}).get(loaded_version)
            if not model_meta:
                logger.warning(f"Model metadata not found for {app_key}:{loaded_version}")
                return self._generate_baseline_action(request)

            # Extract features from ClickHouse if available, otherwise use provided features
            features = await self._extract_features_from_clickhouse(request) if self.analytics_data else {}

            # Fall back to provided features if ClickHouse extraction failed
            if not features and request.features:
                features = dict(request.features)
                logger.info("Using provided gRPC features")

            # Ensure required features are present with defaults
            features.setdefault("cpu_utilization", 0.5)
            features.setdefault("memory_utilization", 0.5)
            features.setdefault("request_rate", 100.0)
            features.setdefault("p95_latency_ms", 200.0)
            features.setdefault("namespace", request.app.namespace)
            features.setdefault("app_name", request.app.app_name)

            logger.info(f"Running inference with model {loaded_version} on features: {features}")

            # In a real implementation, this would load and run the actual RL model
            # For now, we simulate intelligent behavior based on features
            action_plan = self._run_model_inference(features, request.candidates, model_meta)

            decision_id = str(uuid.uuid4())

            # Store recommendation decision in ClickHouse for audit trail
            if self.engine_data:
                await self._store_decision_in_clickhouse(request, decision_id, loaded_version, action_plan)

            return engine_pb2.GetActionResponse(
                plan=action_plan,
                model_version=loaded_version,
                confidence=0.85,  # Simulate high confidence from trained model
                decision_id=decision_id,
                audit_reasons=[
                    f"RL model {loaded_version} inference",
                    f"Input features: {len(features)} metrics",
                    f"Policy: {model_meta.policy_name}"
                ]
            )

        except Exception as e:
            logger.error(f"Error during inference: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Inference error: {str(e)}")
            return self._generate_baseline_action(request)

    def EnsureModel(
        self,
        request: engine_pb2.AppRef,
        context: grpc.ServicerContext
    ) -> engine_pb2.EnsureModelResponse:
        """
        Ensure a model exists and is loaded in memory. Bootstrap if needed.
        """
        app_key = self._get_app_key(request)
        logger.info(f"Ensuring model for app: {app_key}")

        try:
            # Check if we have any models for this app
            app_models = self.models.get(app_key, {})

            if app_models:
                # Get the latest model
                latest_version = max(app_models.keys(), key=lambda v: app_models[v].updated_at.seconds)
                latest_meta = app_models[latest_version]

                # Load model into memory if not already loaded
                if self.loaded_models.get(app_key) != latest_version:
                    self._load_model_into_memory(app_key, latest_version, latest_meta)
                    logger.info(f"Loaded existing model {latest_version} for {app_key}")

                return engine_pb2.EnsureModelResponse(
                    model_version=latest_version,
                    created=False,
                    meta=latest_meta
                )
            else:
                # Bootstrap a baseline model
                baseline_version = self._bootstrap_baseline_model(app_key)
                logger.info(f"Bootstrapped baseline model {baseline_version} for {app_key}")

                return engine_pb2.EnsureModelResponse(
                    model_version=baseline_version,
                    created=True,
                    meta=self.models[app_key][baseline_version]
                )

        except Exception as e:
            logger.error(f"Error ensuring model: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Model ensure error: {str(e)}")
            return engine_pb2.EnsureModelResponse()

    def TriggerTrain(
        self,
        request: engine_pb2.TrainRequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.TrainResponse:
        """
        Kick off model training. Typically spawns a Kubernetes Job.
        """
        app_key = self._get_app_key(request.app)
        training_id = str(uuid.uuid4())
        job_name = f"trainer-{app_key.replace(':', '-').replace('/', '-')}-{training_id[:8]}"

        logger.info(f"Triggering training for {app_key}: reason={request.reason}, horizon={request.horizon_hours}h")

        try:
            # Store training job metadata
            training_job = {
                "training_id": training_id,
                "app_key": app_key,
                "reason": request.reason,
                "horizon_hours": request.horizon_hours,
                "base_version": request.base_version,
                "hparams": dict(request.hparams) if request.hparams else {},
                "status": "starting",
                "created_at": datetime.utcnow().isoformat(),
                "job_name": job_name
            }

            self.training_jobs[training_id] = training_job

            # In a real implementation, this would create a Kubernetes Job
            # For now, we simulate the job creation
            logger.info(f"Created training job: {job_name}")

            # TODO: Create actual Kubernetes Job with:
            # - Training container with RL code
            # - Environment variables for ClickHouse DSN, output URI, etc.
            # - Volume mounts for checkpoint storage
            # self._create_training_job(training_job)

            return engine_pb2.TrainResponse(
                training_id=training_id,
                job_name=job_name
            )

        except Exception as e:
            logger.error(f"Error triggering training: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Training trigger error: {str(e)}")
            return engine_pb2.TrainResponse()

    def ListModels(
        self,
        request: engine_pb2.ListModelsRequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.ListModelsResponse:
        """
        List available models for an app, ordered newest-first.
        """
        app_key = self._get_app_key(request.app)
        logger.info(f"Listing models for app: {app_key}")

        try:
            app_models = self.models.get(app_key, {})

            # Sort by update timestamp, newest first
            sorted_models = sorted(
                app_models.values(),
                key=lambda m: m.updated_at.seconds,
                reverse=True
            )

            return engine_pb2.ListModelsResponse(models=sorted_models)

        except Exception as e:
            logger.error(f"Error listing models: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Model list error: {str(e)}")
            return engine_pb2.ListModelsResponse()

    def GetModelMetadata(
        self,
        request: engine_pb2.GetModelMetadataRequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.ModelMetadata:
        """
        Get metadata for a specific model version, or latest if version not specified.
        """
        app_key = self._get_app_key(request.app)
        logger.info(f"Getting model metadata for {app_key}, version: {request.model_version or 'latest'}")

        try:
            app_models = self.models.get(app_key, {})

            if not app_models:
                context.set_code(grpc.StatusCode.NOT_FOUND)
                context.set_details(f"No models found for app: {app_key}")
                return engine_pb2.ModelMetadata()

            if request.model_version:
                # Get specific version
                model_meta = app_models.get(request.model_version)
                if not model_meta:
                    context.set_code(grpc.StatusCode.NOT_FOUND)
                    context.set_details(f"Model version {request.model_version} not found")
                    return engine_pb2.ModelMetadata()
                return model_meta
            else:
                # Get latest version
                latest_version = max(app_models.keys(), key=lambda v: app_models[v].updated_at.seconds)
                return app_models[latest_version]

        except Exception as e:
            logger.error(f"Error getting model metadata: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Model metadata error: {str(e)}")
            return engine_pb2.ModelMetadata()

    async def ReportOutcome(
        self,
        request: engine_pb2.ExecutionOutcome,
        context: grpc.ServicerContext
    ) -> empty_pb2.Empty:
        """
        Accept execution outcomes for model learning and improvement.
        This implements the reward calculation from the controller research.
        """
        app_key = self._get_app_key(request.app)
        logger.info(f"Received outcome for {app_key}, decision: {request.decision_id}")

        try:
            # Log the outcome
            status = "SUCCESS" if request.success else "FAILED"
            logger.info(f"Execution {status}: {request.note}")

            # Store outcome in ClickHouse for training data
            if self.engine_data:
                await self._store_outcome_in_clickhouse(request)

            # Calculate reward based on the outcome and current metrics
            self._calculate_and_store_reward(app_key, request)

            # Check if we should trigger retraining based on outcomes
            if not request.success and "capacity" in request.note.lower():
                logger.info("Capacity issue detected, may trigger retraining")
                # Could trigger training with reason="capacity_issue"

            # Detect performance drift and trigger retraining if needed
            await self._check_for_drift_and_retrain(app_key)

            return empty_pb2.Empty()

        except Exception as e:
            logger.error(f"Error processing outcome: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Outcome processing error: {str(e)}")
            return empty_pb2.Empty()

    def _calculate_and_store_reward(
        self,
        app_key: str,
        outcome: engine_pb2.ExecutionOutcome
    ):
        """
        Calculate reward for the executed action using the reward function
        from the controller implementation.
        """
        try:
            # Get action and state history
            if (app_key not in self.action_history or
                app_key not in self.state_history or
                len(self.action_history[app_key]) < 2 or
                len(self.state_history[app_key]) < 2):
                logger.debug(f"Insufficient history for reward calculation for {app_key}")
                return

            # Get current and last state/action
            current_action = self.action_history[app_key][-1]
            last_action = self.action_history[app_key][-2]
            current_state = self.state_history[app_key][-1].copy()
            last_state = self.state_history[app_key][-2]

            # Update current state with post-action metrics if available
            if outcome.post_action_metrics and outcome.post_action_metrics.values:
                post_metrics = dict(outcome.post_action_metrics.values)

                # Map common metric names to state space names
                metric_mapping = {
                    'cpu_utilization': 'cpu_util',
                    'memory_utilization': 'memory_util',
                    'p95_latency_ms': 'latency',
                    'request_rate': 'processing_rate',
                    'error_rate': 'error_rate'
                }

                for metric_name, value in post_metrics.items():
                    state_name = metric_mapping.get(metric_name, metric_name)
                    current_state[state_name] = value

            # Calculate reward using v1 formula (as in controller)
            reward = self.reward_calculator.calculate_reward_v1(
                current_state=current_state,
                action=current_action,
                last_action=last_action,
                last_state=last_state
            )

            # Apply success/failure modifier
            if not outcome.success:
                reward -= 1.0  # Large penalty for failed executions
                logger.info(f"Applied failure penalty for {app_key}: {outcome.note}")

            logger.info(f"Calculated reward for {app_key}: {reward:.3f}")

            # In a real implementation, store this for training:
            # self._store_training_sample(app_key, last_state, current_action, reward, current_state)

        except Exception as e:
            logger.error(f"Error calculating reward for {app_key}: {str(e)}")

    async def _check_for_drift_and_retrain(self, app_key: str):
        """
        Check for performance drift and trigger retraining if needed.
        This implements drift detection logic similar to the controller.
        """
        try:
            if not self.engine_data:
                logger.debug(f"Drift detection check for {app_key} (ClickHouse not available)")
                return

            # Get recent execution success rate
            # Parse app_key to extract components
            parts = app_key.split(":", 1)
            if len(parts) != 2:
                logger.warning(f"Invalid app_key format: {app_key}")
                return

            api_key = parts[0]
            app_parts = parts[1].split("/")
            if len(app_parts) != 2:
                logger.warning(f"Invalid app_key namespace/app format: {app_key}")
                return

            namespace, app_name = app_parts

            # Check success rate over last 24 hours
            success_rate = await self.engine_data.get_outcome_success_rate(
                cluster_id=api_key,
                namespace=namespace,
                app_name=app_name,
                hours=24
            )

            logger.debug(f"Success rate for {app_key}: {success_rate:.2f}")

            # Trigger retraining if success rate is too low
            if success_rate < 0.7:  # Less than 70% success
                logger.warning(f"Low success rate detected for {app_key}: {success_rate:.2f}")
                # TODO: Trigger retraining job with reason="performance_drift"
                # self._trigger_training_for_drift(app_key, success_rate)

        except Exception as e:
            logger.error(f"Error in drift detection for {app_key}: {str(e)}")

    def _get_app_key(self, app_ref: engine_pb2.AppRef) -> str:
        """Generate a unique key for an app reference."""
        return f"{app_ref.api_key}:{app_ref.namespace}/{app_ref.app_name}"

    def _generate_baseline_action(self, request: engine_pb2.GetActionRequest) -> engine_pb2.GetActionResponse:
        """Generate a baseline action when no trained model is available."""

        features = dict(request.features) if request.features else {}
        cpu_util = features.get("cpu_utilization", 0.5)
        memory_util = features.get("memory_utilization", 0.5)

        # Simple heuristic logic
        if cpu_util > 0.8:
            action_plan = engine_pb2.ActionPlan(
                type="VPA_RECOMMEND",
                confidence=0.6,
                reason="Baseline: High CPU utilization",
                vpa_recommend=engine_pb2.VpaRecommendAction(
                    container="app",
                    cpu_request_mcpu=int(1000 * min(2.0, cpu_util * 1.3)),
                    memory_mib=int(512 * max(1.0, memory_util * 1.1)),
                    mode="recommendation"
                )
            )
        elif cpu_util < 0.2:
            action_plan = engine_pb2.ActionPlan(
                type="VPA_RECOMMEND",
                confidence=0.4,
                reason="Baseline: Low CPU utilization",
                vpa_recommend=engine_pb2.VpaRecommendAction(
                    container="app",
                    cpu_request_mcpu=int(1000 * max(0.1, cpu_util * 1.1)),
                    memory_mib=int(512 * max(0.1, memory_util * 1.0)),
                    mode="recommendation"
                )
            )
        else:
            action_plan = engine_pb2.ActionPlan(
                type="NO_ACTION",
                confidence=0.5,
                reason="Baseline: Resource utilization within normal range"
            )

        return engine_pb2.GetActionResponse(
            plan=action_plan,
            model_version="baseline-v1",
            confidence=0.5,
            decision_id=str(uuid.uuid4()),
            audit_reasons=["Baseline heuristic policy", f"CPU: {cpu_util:.2f}, Memory: {memory_util:.2f}"]
        )

    def _bootstrap_baseline_model(self, app_key: str) -> str:
        """Bootstrap a baseline model for a new app."""

        version = "baseline-v1"
        now = timestamp_pb2.Timestamp()
        now.GetCurrentTime()

        # Create baseline model metadata
        model_meta = engine_pb2.ModelMetadata(
            model_version=version,
            policy_name="baseline-heuristic",
            updated_at=now,
            labels={"type": "baseline", "auto_generated": "true"},
            checkpoint_uri=f"{self.model_store_base_uri}/{app_key}/baseline-v1/model.pkl",
            compatible_feature_schema=["cpu_utilization", "memory_utilization", "request_rate", "p95_latency_ms"]
        )

        # Store in registry
        if app_key not in self.models:
            self.models[app_key] = {}
        self.models[app_key][version] = model_meta

        # Load into memory
        self._load_model_into_memory(app_key, version, model_meta)

        return version

    def _load_model_into_memory(self, app_key: str, version: str, model_meta: engine_pb2.ModelMetadata):
        """Load a model into memory for inference."""

        # In a real implementation, this would:
        # 1. Download checkpoint from S3/PVC
        # 2. Load PyTorch model weights
        # 3. Initialize model in inference mode

        logger.info(f"Loading model {version} for {app_key} from {model_meta.checkpoint_uri}")

        # For simulation, just mark as loaded
        self.loaded_models[app_key] = version

    def _run_model_inference(
        self,
        features: Dict[str, float],
        candidates: List[engine_pb2.CandidateProposal],
        model_meta: engine_pb2.ModelMetadata
    ) -> engine_pb2.ActionPlan:
        """
        Run inference on the loaded RL model using the PPO-based approach
        with HPA/VPA scaling algorithms from the controller implementation.
        """

        # Get or create feature extractor for this app context
        app_key = f"{features.get('namespace', 'default')}/{features.get('app_name', 'unknown')}"
        if app_key not in self.feature_extractors:
            self.feature_extractors[app_key] = FeatureExtractor()

        feature_extractor = self.feature_extractors[app_key]

        # Convert raw features to normalized state vector
        normalized_features = self.state_space.extract_features(features)

        # Get current state description
        current_state = self.state_space.get_state_description(normalized_features)

        logger.info(f"Running RL inference for {app_key}")
        logger.debug(f"Current state: {current_state}")

        # Convert to ResourceState for scaling algorithms
        resource_state = self.scaling_algorithms.convert_state_dict_to_resource_state(features)

        # Get SLO targets if available
        slo_targets = self._get_slo_targets_for_inference(features)

        # Get intelligent scaling action using HPA/VPA algorithms
        scaling_action = self.scaling_algorithms.get_intelligent_scaling_action(
            current_state=resource_state,
            slo_targets=slo_targets,
            app_key=app_key
        )

        # If scaling algorithms suggest a specific action, use it
        if scaling_action.action_type != "no_action":
            logger.info(f"Scaling algorithms suggest: {scaling_action.action_type} - {scaling_action.reason}")

            # Convert scaling action to RL action space for policy learning
            action_index = self._convert_scaling_action_to_rl_action(scaling_action)

            # Mark scaling as executed for cooldown tracking
            if scaling_action.action_type in ["horizontal"]:
                self.scaling_algorithms.mark_scaling_executed(app_key, "horizontal")
            elif scaling_action.action_type in ["vertical_cpu", "vertical_memory"]:
                self.scaling_algorithms.mark_scaling_executed(app_key, "vertical")

        else:
            # Fall back to RL policy simulation when scaling algorithms suggest no action
            logger.info("Scaling algorithms suggest no action, using RL policy simulation")
            action_probs = self._simulate_policy_output(normalized_features, current_state)
            action_index = self.action_space.sample_action(action_probs)

        action_type = ActionType(action_index)

        # Convert to Kubernetes resource changes
        k8s_changes = self.action_space.convert_action_to_k8s_changes(action_index, current_state)

        # Store action history for reward calculation
        action_dict = self.action_space.convert_action_to_dict(action_index)
        self._update_action_history(app_key, action_dict, current_state)

        # Build ActionPlan based on selected action
        if scaling_action.action_type != "no_action":
            # Use scaling algorithm's decision
            action_plan = self._convert_scaling_action_to_action_plan(scaling_action, resource_state)
        else:
            # Use RL policy's decision
            action_plan = self._convert_to_action_plan(action_index, k8s_changes, current_state)

        logger.info(f"Selected action: {action_type.name} - {self.action_space.get_action_description(action_index)}")

        return action_plan

    def _simulate_policy_output(
        self,
        normalized_features: np.ndarray,
        current_state: Dict[str, float]
    ) -> np.ndarray:
        """
        Simulate PPO policy network output.
        In production, this would be actual PyTorch model inference.
        """

        # Get current resource utilization and performance metrics
        cpu_util = current_state.get('cpu_util', 0.5)
        memory_util = current_state.get('memory_util', 0.5)
        latency = current_state.get('latency', 100.0)
        processing_rate = current_state.get('processing_rate', 100.0)
        ingestion_rate = current_state.get('ingestion_rate', 100.0)

        # Initialize action probabilities (softmax will be applied)
        action_logits = np.zeros(len(ActionType))

        # Base policy: prefer no action for stability
        action_logits[ActionType.NO_ACTION] = 2.0

        # High CPU utilization → scale up CPU or scale out
        if cpu_util > 0.8:
            action_logits[ActionType.VERTICAL_CPU_UP] += 3.0 * (cpu_util - 0.8)
            action_logits[ActionType.HORIZONTAL_UP] += 2.0 * (cpu_util - 0.8)

        # High memory utilization → scale up memory
        if memory_util > 0.8:
            action_logits[ActionType.VERTICAL_MEMORY_UP] += 3.0 * (memory_util - 0.8)

        # Low utilization → consider scaling down
        if cpu_util < 0.3 and memory_util < 0.3:
            action_logits[ActionType.VERTICAL_CPU_DOWN] += 1.5 * (0.3 - cpu_util)
            action_logits[ActionType.VERTICAL_MEMORY_DOWN] += 1.5 * (0.3 - memory_util)
            if current_state.get('num_replicas', 1) > 1:
                action_logits[ActionType.HORIZONTAL_DOWN] += 1.0 * (0.3 - cpu_util)

        # High latency → scale up resources or replicas
        if latency > 300:  # ms
            latency_pressure = min(2.0, (latency - 300) / 200)
            action_logits[ActionType.VERTICAL_CPU_UP] += 2.0 * latency_pressure
            action_logits[ActionType.HORIZONTAL_UP] += 2.5 * latency_pressure

        # Processing lag (ingestion > processing) → scale up
        if ingestion_rate > processing_rate * 1.2:
            lag_pressure = min(2.0, (ingestion_rate / processing_rate) - 1.0)
            action_logits[ActionType.VERTICAL_CPU_UP] += 1.5 * lag_pressure
            action_logits[ActionType.HORIZONTAL_UP] += 2.0 * lag_pressure

        # Apply resource bounds constraints
        current_replicas = int(current_state.get('num_replicas', 1))
        current_cpu = current_state.get('cpu_limit', 1000)
        current_memory = current_state.get('memory_limit', 512)

        if current_replicas >= 20:  # Max replicas limit
            action_logits[ActionType.HORIZONTAL_UP] = -5.0
        if current_replicas <= 1:  # Min replicas limit
            action_logits[ActionType.HORIZONTAL_DOWN] = -5.0
        if current_cpu >= 4000:  # Max CPU limit (4 cores)
            action_logits[ActionType.VERTICAL_CPU_UP] = -5.0
        if current_cpu <= 200:  # Min CPU limit (200m)
            action_logits[ActionType.VERTICAL_CPU_DOWN] = -5.0
        if current_memory >= 8192:  # Max memory limit (8Gi)
            action_logits[ActionType.VERTICAL_MEMORY_UP] = -5.0
        if current_memory <= 256:  # Min memory limit (256Mi)
            action_logits[ActionType.VERTICAL_MEMORY_DOWN] = -5.0

        # Apply softmax to get probabilities
        action_probs = self._softmax(action_logits)

        logger.debug(f"Action probabilities: {dict(zip([a.name for a in ActionType], action_probs))}")

        return action_probs

    def _softmax(self, logits: np.ndarray, temperature: float = 1.0) -> np.ndarray:
        """Apply softmax with temperature to get action probabilities."""
        scaled_logits = logits / temperature
        exp_logits = np.exp(scaled_logits - np.max(scaled_logits))  # Subtract max for numerical stability
        return exp_logits / np.sum(exp_logits)

    def _convert_to_action_plan(
        self,
        action_index: int,
        k8s_changes: Dict[str, any],
        current_state: Dict[str, float]
    ) -> engine_pb2.ActionPlan:
        """Convert selected action to protobuf ActionPlan."""

        action_type = ActionType(action_index)
        confidence = 0.85  # High confidence from trained model

        if action_type == ActionType.NO_ACTION:
            return engine_pb2.ActionPlan(
                type="NO_ACTION",
                confidence=confidence,
                reason="RL model: Current configuration appears optimal"
            )

        elif action_type in [ActionType.VERTICAL_CPU_UP, ActionType.VERTICAL_CPU_DOWN,
                           ActionType.VERTICAL_MEMORY_UP, ActionType.VERTICAL_MEMORY_DOWN]:
            # Vertical scaling action
            cpu_request = k8s_changes.get('cpu_request_mcpu')
            memory_request = k8s_changes.get('memory_request_mib')

            return engine_pb2.ActionPlan(
                type="VPA_RECOMMEND",
                confidence=confidence,
                reason=f"RL model: {self.action_space.get_action_description(action_index)}",
                vpa_recommend=engine_pb2.VpaRecommendAction(
                    container="app",
                    cpu_request_mcpu=cpu_request or int(current_state.get('cpu_limit', 1000)),
                    memory_mib=memory_request or int(current_state.get('memory_limit', 512)),
                    mode="recommendation"
                )
            )

        elif action_type in [ActionType.HORIZONTAL_UP, ActionType.HORIZONTAL_DOWN]:
            # Horizontal scaling action
            target_replicas = k8s_changes.get('replicas', int(current_state.get('num_replicas', 1)))

            return engine_pb2.ActionPlan(
                type="HPA_SCALE",
                confidence=confidence,
                reason=f"RL model: {self.action_space.get_action_description(action_index)}",
                hpa_scale=engine_pb2.HpaScaleAction(replicas=target_replicas)
            )

        # Fallback
        return engine_pb2.ActionPlan(
            type="NO_ACTION",
            confidence=0.5,
            reason=f"RL model: Unknown action type {action_type}"
        )

    def _update_action_history(
        self,
        app_key: str,
        action_dict: Dict[str, int],
        current_state: Dict[str, float]
    ):
        """Update action and state history for reward calculation."""

        # Initialize history if needed
        if app_key not in self.action_history:
            self.action_history[app_key] = []
        if app_key not in self.state_history:
            self.state_history[app_key] = []

        # Add current action and state
        self.action_history[app_key].append(action_dict)
        self.state_history[app_key].append(current_state.copy())

        # Keep only recent history (last 10 steps)
        max_history = 10
        if len(self.action_history[app_key]) > max_history:
            self.action_history[app_key] = self.action_history[app_key][-max_history:]
        if len(self.state_history[app_key]) > max_history:
            self.state_history[app_key] = self.state_history[app_key][-max_history:]

    async def _extract_features_from_clickhouse(
        self,
        request: engine_pb2.GetActionRequest
    ) -> Dict[str, float]:
        """
        Extract features from ClickHouse analytics data instead of relying
        solely on gRPC-provided features.
        """
        try:
            if not self.analytics_data:
                return {}

            app_ref = request.app
            features = {}

            # Get latest metrics for this app
            latest_metrics = await self.analytics_data.get_latest_app_metrics(
                cluster_id=app_ref.api_key,  # Using api_key as cluster_id
                namespace=app_ref.namespace,
                app_name=app_ref.app_name,
                minutes_back=5  # Last 5 minutes
            )

            if latest_metrics:
                features.update({
                    "cpu_utilization": latest_metrics.get("cpu_utilization", 0.5),
                    "memory_utilization": latest_metrics.get("memory_utilization", 0.5),
                    "request_rate": latest_metrics.get("request_rate", 100.0),
                    "p95_latency_ms": latest_metrics.get("p95_latency_ms", 200.0),
                    "error_rate": latest_metrics.get("error_rate", 0.01),
                    "num_replicas": latest_metrics.get("num_replicas", 1),
                    "cpu_limit": latest_metrics.get("cpu_limit", 1000),
                    "memory_limit": latest_metrics.get("memory_limit", 512)
                })

                logger.info(f"Extracted {len(features)} features from ClickHouse for {app_ref.namespace}/{app_ref.app_name}")
            else:
                logger.warning(f"No recent metrics found in ClickHouse for {app_ref.namespace}/{app_ref.app_name}")
                # TODO: Add more comprehensive metric collection from multiple tables

            # Get additional context from SLO data if available
            if self.slo_data:
                slo = await self.slo_data.get_service_level_objective(
                    api_key=app_ref.api_key,
                    namespace=app_ref.namespace,
                    service_name=app_ref.app_name
                )
                if slo:
                    features.update({
                        "slo_target_latency": slo.target_p95_latency_ms,
                        "slo_target_error_rate": slo.target_error_rate,
                        "slo_priority": 1.0 if slo.priority == "high" else 0.5
                    })

            return features

        except Exception as e:
            logger.error(f"Error extracting features from ClickHouse: {str(e)}")
            return {}

    async def _store_decision_in_clickhouse(
        self,
        request: engine_pb2.GetActionRequest,
        decision_id: str,
        model_version: str,
        action_plan: engine_pb2.ActionPlan
    ):
        """
        Store recommendation decision in ClickHouse for audit trail and training.
        """
        try:
            if not self.engine_data:
                return

            app_ref = request.app

            # Convert action plan to dictionary for storage
            plan_vertical = []
            if action_plan.vpa_recommend:
                plan_vertical.append({
                    "container": action_plan.vpa_recommend.container,
                    "cpu_request_mcpu": str(action_plan.vpa_recommend.cpu_request_mcpu),
                    "memory_mib": str(action_plan.vpa_recommend.memory_mib),
                    "mode": action_plan.vpa_recommend.mode
                })

            plan_replicas = 0
            if action_plan.hpa_scale:
                plan_replicas = action_plan.hpa_scale.replicas

            audit_reasons = [
                f"Model: {model_version}",
                f"Action: {action_plan.type}",
                f"Confidence: {action_plan.confidence:.2f}",
                action_plan.reason
            ]

            # Extract effective policy from request context
            effective_policy = {
                "api_key": app_ref.api_key,
                "namespace": app_ref.namespace,
                "app_name": app_ref.app_name,
                "action_type": action_plan.type
            }

            # Store in ClickHouse
            success = await self.engine_data.store_recommendation_decision(
                cluster_id=app_ref.api_key,  # Using api_key as cluster_id
                namespace=app_ref.namespace,
                app_name=app_ref.app_name,
                workload_kind="Deployment",  # Default assumption
                decision_id=decision_id,
                model_version=model_version,
                confidence=action_plan.confidence,
                audit_reasons=audit_reasons,
                plan_vertical=plan_vertical,
                plan_replicas=plan_replicas,
                effective_policy=effective_policy
            )

            if success:
                logger.info(f"Stored decision {decision_id} in ClickHouse for {app_ref.namespace}/{app_ref.app_name}")
            else:
                logger.warning(f"Failed to store decision {decision_id} in ClickHouse")

        except Exception as e:
            logger.error(f"Error storing decision in ClickHouse: {str(e)}")

    async def _store_outcome_in_clickhouse(
        self,
        outcome: engine_pb2.ExecutionOutcome
    ):
        """
        Store execution outcome in ClickHouse for training data and analytics.
        """
        try:
            if not self.engine_data:
                return

            app_ref = outcome.app

            # Convert post-action metrics to dictionary
            post_metrics = {}
            if outcome.post_action_metrics and outcome.post_action_metrics.values:
                post_metrics = dict(outcome.post_action_metrics.values)

            # Store outcome in ClickHouse
            success = await self.engine_data.store_execution_outcome(
                cluster_id=app_ref.api_key,  # Using api_key as cluster_id
                namespace=app_ref.namespace,
                app_name=app_ref.app_name,
                workload_kind="Deployment",  # Default assumption
                decision_id=outcome.decision_id,
                success=outcome.success,
                note=outcome.note,
                post_action_metrics=post_metrics,
                reported_at=datetime.utcnow()
            )

            if success:
                logger.info(f"Stored outcome for decision {outcome.decision_id} in ClickHouse")
            else:
                logger.warning(f"Failed to store outcome for decision {outcome.decision_id} in ClickHouse")

        except Exception as e:
            logger.error(f"Error storing outcome in ClickHouse: {str(e)}")

    def _get_slo_targets_for_inference(self, features: Dict[str, float]) -> Optional[Dict[str, float]]:
        """Get SLO targets for the current application if available."""
        if 'slo_target_latency' in features:
            return {
                'target_p95_latency_ms': features.get('slo_target_latency', 500.0),
                'target_error_rate': features.get('slo_target_error_rate', 0.01),
                'target_throughput_rps': features.get('slo_target_throughput', 100.0)
            }
        return None

    def _convert_scaling_action_to_rl_action(self, scaling_action: ScalingAction) -> int:
        """Convert a ScalingAction to RL action space index."""
        if scaling_action.action_type == "horizontal":
            return ActionType.HORIZONTAL_UP if scaling_action.target_replicas else ActionType.HORIZONTAL_DOWN
        elif scaling_action.action_type == "vertical_cpu":
            return ActionType.VERTICAL_CPU_UP if scaling_action.target_cpu_mcpu else ActionType.VERTICAL_CPU_DOWN
        elif scaling_action.action_type == "vertical_memory":
            return ActionType.VERTICAL_MEMORY_UP if scaling_action.target_memory_mib else ActionType.VERTICAL_MEMORY_DOWN
        else:
            return ActionType.NO_ACTION

    def _convert_scaling_action_to_action_plan(
        self,
        scaling_action: ScalingAction,
        resource_state: ResourceState
    ) -> engine_pb2.ActionPlan:
        """Convert a ScalingAction to gRPC ActionPlan."""

        if scaling_action.action_type == "horizontal":
            return engine_pb2.ActionPlan(
                type="HPA_SCALE",
                confidence=scaling_action.confidence,
                reason=scaling_action.reason,
                hpa_scale=engine_pb2.HpaScaleAction(
                    replicas=scaling_action.target_replicas or resource_state.num_replicas
                )
            )

        elif scaling_action.action_type in ["vertical_cpu", "vertical_memory"]:
            return engine_pb2.ActionPlan(
                type="VPA_RECOMMEND",
                confidence=scaling_action.confidence,
                reason=scaling_action.reason,
                vpa_recommend=engine_pb2.VpaRecommendAction(
                    container="app",
                    cpu_request_mcpu=scaling_action.target_cpu_mcpu or resource_state.cpu_limit,
                    memory_mib=scaling_action.target_memory_mib or resource_state.memory_limit,
                    mode="recommendation"
                )
            )

        else:
            return engine_pb2.ActionPlan(
                type="NO_ACTION",
                confidence=scaling_action.confidence,
                reason=scaling_action.reason
            )