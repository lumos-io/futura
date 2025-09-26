import logging
from typing import Dict, Optional
from datetime import datetime
from proto.gen.engine import engine_pb2, engine_pb2_grpc
from google.protobuf import empty_pb2
import grpc

# Import storage layers for ClickHouse integration
from storage.clickhouse_client import ClickHouseClient, EngineDataAccess
from storage.slo_data_access import SLODataAccess


logger = logging.getLogger(__name__)


class RecommendationService(engine_pb2_grpc.RecommendationServiceServicer):
    """
    MPA Server (engine) - Orchestrates optimization decisions and safety policies.

    This service is called by the Kubernetes Operator to get recommendations
    and report execution outcomes. It may internally consult the RLServer for
    ML-powered decisions.
    """

    def __init__(
        self,
        rl_server_client: Optional[engine_pb2_grpc.RLServerStub] = None,
        clickhouse_client: Optional[ClickHouseClient] = None
    ):
        self.rl_server_client = rl_server_client
        self.cluster_configs: Dict[str,
                                   engine_pb2.ClusterOptimizationConfigRequest] = {}
        self.service_slos: Dict[str, engine_pb2.SyncSLORequest] = {}

        # ClickHouse integration for data persistence
        self.clickhouse_client = clickhouse_client
        self.engine_data: Optional[EngineDataAccess] = None
        self.slo_data: Optional[SLODataAccess] = None

        if self.clickhouse_client:
            self.engine_data = EngineDataAccess(self.clickhouse_client)
            self.slo_data = SLODataAccess(self.clickhouse_client)

    async def SyncClusterOptimizationConfig(
        self,
        request: engine_pb2.ClusterOptimizationConfigRequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.ClusterOptimizationConfigResponse:
        """
        Configure cluster optimization settings including cloud provider,
        budget constraints, instance types, and spot instance preferences.
        """
        logger.info(
            f"Syncing cluster config for API key: {request.api_key[:8]}...")

        try:
            # Store configuration for future use
            self.cluster_configs[request.api_key] = request

            # Validate configuration
            if not request.cloud_provider:
                return engine_pb2.ClusterOptimizationConfigResponse(
                    success=False,
                    message="Cloud provider is required"
                )

            if not request.region:
                return engine_pb2.ClusterOptimizationConfigResponse(
                    success=False,
                    message="Region is required"
                )

            # Store in ClickHouse if available
            if self.slo_data:
                success = await self.slo_data.store_cluster_optimization_config(
                    api_key=request.api_key,
                    cluster_id=request.cluster_id or request.api_key,
                    cloud_provider=request.cloud_provider,
                    region=request.region,
                    cost_sensitivity=request.cost_sensitivity,
                    monthly_budget=request.monthly_budget,
                    preferred_instance_types=list(request.preferred_instance_types),
                    allow_spot=request.allow_spot,
                    max_spot_percentage=request.max_spot_percentage
                )

                if not success:
                    logger.warning("Failed to persist cluster config to ClickHouse")

            logger.info(
                f"Cluster config synced successfully for provider: {request.cloud_provider}, region: {request.region}")

            return engine_pb2.ClusterOptimizationConfigResponse(
                success=True,
                message=f"Configuration synced for {request.cloud_provider} in {request.region}"
            )

        except Exception as e:
            logger.error(f"Error syncing cluster config: {str(e)}")
            return engine_pb2.ClusterOptimizationConfigResponse(
                success=False,
                message=f"Internal error: {str(e)}"
            )

    async def SyncServiceLevelObjective(
        self,
        request: engine_pb2.SyncSLORequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.SyncSLOResponse:
        """
        Sync Service Level Objectives for a specific service including
        P95 latency targets, error rate limits, and throughput requirements.
        """
        logger.info(f"Syncing SLO for service: {request.service_name}")

        try:
            # Store SLO for future recommendations
            slo_key = f"{request.api_key}:{request.service_name}"
            self.service_slos[slo_key] = request

            # Validate SLO parameters
            if not request.service_name:
                return engine_pb2.SyncSLOResponse(
                    success=False,
                    message="Service name is required"
                )

            if not request.target_p95_latency:
                return engine_pb2.SyncSLOResponse(
                    success=False,
                    message="P95 latency target is required"
                )

            # Store in ClickHouse if available
            if self.slo_data:
                success = await self.slo_data.store_service_level_objective(
                    api_key=request.api_key,
                    cluster_id=request.cluster_id or request.api_key,
                    namespace=request.namespace,
                    service_name=request.service_name,
                    target_p95_latency=request.target_p95_latency,
                    target_error_rate=request.target_error_rate,
                    target_throughput=request.target_throughput,
                    priority=request.priority
                )

                if not success:
                    logger.warning(f"Failed to persist SLO to ClickHouse for {request.service_name}")

            logger.info(
                f"SLO synced for {request.service_name}: P95={request.target_p95_latency}, error_rate={request.target_error_rate}")

            return engine_pb2.SyncSLOResponse(
                success=True,
                message=f"SLO synced for service {request.service_name}"
            )

        except Exception as e:
            logger.error(f"Error syncing SLO: {str(e)}")
            return engine_pb2.SyncSLOResponse(
                success=False,
                message=f"Internal error: {str(e)}"
            )

    def GetAppRecommendation(
        self,
        request: engine_pb2.RecommendationAppRequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.RecommendationAppResponse:
        """
        Get optimization recommendation for a specific app workload.
        This is the main decision endpoint called by the Kubernetes Operator for app-level optimizations.
        """
        logger.info(
            f"Getting recommendation for app: {request.app.app_name} in namespace: {request.app.namespace}")

        try:
            # Generate a unique decision ID for tracking
            import uuid
            decision_id = str(uuid.uuid4())

            # Get cluster config for this API key
            cluster_config = self.cluster_configs.get(request.app.api_key)
            if not cluster_config:
                logger.warning(
                    f"No cluster config found for API key: {request.app.api_key[:8]}...")

            # Get SLO for this service
            slo_key = f"{request.app.api_key}:{request.app.app_name}"
            service_slo = self.service_slos.get(slo_key)

            # If we have an RL server client, consult it for ML-powered recommendations
            action_plan = None
            model_version = "baseline-v1"
            confidence = 0.5
            audit_reasons = ["Using baseline heuristic policy"]

            if self.rl_server_client:
                try:
                    # Prepare features for RL server
                    features = {}
                    if request.snapshot and request.snapshot.values:
                        features = dict(request.snapshot.values)

                    # Build GetAppAction request
                    rl_request = engine_pb2.GetAppActionRequest(
                        app=request.app,
                        features=features
                    )

                    # Get action from RL server
                    rl_response = self.rl_server_client.GetAppAction(rl_request)

                    action_plan = rl_response.plan
                    model_version = rl_response.model_version
                    confidence = rl_response.confidence
                    audit_reasons = list(rl_response.audit_reasons)

                    logger.info(
                        f"Got RL recommendation: model={model_version}, confidence={confidence}")

                except Exception as rl_error:
                    logger.warning(
                        f"Failed to get RL recommendation, falling back to baseline: {str(rl_error)}")
                    audit_reasons.append(f"RL server error: {str(rl_error)}")

            # If no ML recommendation, use baseline heuristic
            if not action_plan:
                action_plan = self._generate_baseline_recommendation(
                    request, service_slo, cluster_config)
                audit_reasons.append(
                    "Used baseline heuristic due to no ML model available")

            # Apply safety policy constraints
            effective_policy = self._get_safety_policy(cluster_config)
            action_plan = self._apply_safety_constraints(
                action_plan, effective_policy)

            if not request.dry_run:
                logger.info(
                    f"Recommendation ready for execution: decision_id={decision_id}")
            else:
                logger.info(
                    f"Dry run recommendation generated: decision_id={decision_id}")
                audit_reasons.append("DRY RUN - no execution")

            return engine_pb2.RecommendationAppResponse(
                plan=[action_plan] if action_plan else [],
                decision_id=decision_id,
                model_version=model_version,
                confidence=confidence,
                audit_reasons=audit_reasons,
                effective_policy=effective_policy
            )

        except Exception as e:
            logger.error(f"Error generating recommendation: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return engine_pb2.RecommendationAppResponse()

    async def ReportExecutionAppOutcome(
        self,
        request: engine_pb2.ExecutionAppOutcome,
        context: grpc.ServicerContext
    ) -> empty_pb2.Empty:
        """
        Receive execution outcome from the Operator for learning and auditing.
        This provides feedback to improve future recommendations.
        """
        logger.info(
            f"Received execution outcome for decision: {request.decision_id}")

        try:
            # Log the outcome
            status = "SUCCESS" if request.success else "FAILED"
            logger.info(
                f"Execution {status} for {request.app.app_name}: {request.note}")

            # If we have post-action metrics, log them
            if request.post_action_metrics and request.post_action_metrics.values:
                logger.info(
                    f"Post-action metrics: {dict(request.post_action_metrics.values)}")

            # Persist to ClickHouse for historical analysis
            if self.engine_data:
                try:
                    post_metrics = {}
                    if request.post_action_metrics and request.post_action_metrics.values:
                        post_metrics = dict(request.post_action_metrics.values)

                    success = await self.engine_data.store_execution_outcome(
                        cluster_id=request.app.api_key,
                        namespace=request.app.namespace,
                        app_name=request.app.app_name,
                        workload_kind="Deployment",  # Default assumption
                        decision_id=request.decision_id,
                        success=request.success,
                        note=request.note,
                        post_action_metrics=post_metrics,
                        reported_at=datetime.utcnow()
                    )

                    if success:
                        logger.info(f"Persisted outcome to ClickHouse for decision {request.decision_id}")
                    else:
                        logger.warning(f"Failed to persist outcome to ClickHouse for decision {request.decision_id}")

                except Exception as db_error:
                    logger.warning(f"Failed to persist outcome to ClickHouse: {str(db_error)}")

            # Forward to RL server for learning if available
            if self.rl_server_client:
                try:
                    await self.rl_server_client.ReportAppOutcome(request)
                    logger.info("Outcome forwarded to RL server for learning")
                except Exception as rl_error:
                    logger.warning(
                        f"Failed to forward outcome to RL server: {str(rl_error)}")

            return empty_pb2.Empty()

        except Exception as e:
            logger.error(f"Error processing execution outcome: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return empty_pb2.Empty()

    def GetClusterRecommendation(
        self,
        request: engine_pb2.RecommendationClusterRequest,
        context: grpc.ServicerContext
    ) -> engine_pb2.RecommendationClusterResponse:
        """
        Get optimization recommendation for cluster-level resources.
        This handles Karpenter-style node provisioning recommendations.
        """
        logger.info(
            f"Getting cluster recommendation for API key: {request.cluster.api_key[:8]}...")

        try:
            # Generate a unique decision ID for tracking
            import uuid
            decision_id = str(uuid.uuid4())

            # Get cluster config for this API key
            cluster_config = self.cluster_configs.get(request.cluster.api_key)
            if not cluster_config:
                logger.warning(
                    f"No cluster config found for API key: {request.cluster.api_key[:8]}...")

            # For now, use placeholder cluster recommendation logic
            # TODO: Implement proper Karpenter-style algorithms
            cluster_plan = self._generate_baseline_cluster_recommendation(
                request, cluster_config)

            audit_reasons = ["Using placeholder cluster optimization algorithm"]

            if not request.dry_run:
                logger.info(
                    f"Cluster recommendation ready for execution: decision_id={decision_id}")
            else:
                logger.info(
                    f"Dry run cluster recommendation generated: decision_id={decision_id}")
                audit_reasons.append("DRY RUN - no execution")

            return engine_pb2.RecommendationClusterResponse(
                plan=cluster_plan,
                decision_id=decision_id,
                model_version="cluster-baseline-v1",
                confidence=0.6,
                audit_reasons=audit_reasons
            )

        except Exception as e:
            logger.error(f"Error generating cluster recommendation: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return engine_pb2.RecommendationClusterResponse()

    async def ReportExecutionClusterOutcome(
        self,
        request: engine_pb2.ExecutionClusterOutcome,
        context: grpc.ServicerContext
    ) -> empty_pb2.Empty:
        """
        Receive execution outcome from the Operator for cluster-level changes.
        """
        logger.info(
            f"Received cluster execution outcome for decision: {request.decision_id}")

        try:
            # Log the outcome
            status = "SUCCESS" if request.success else "FAILED"
            logger.info(
                f"Cluster execution {status}: {request.note}")

            # If we have post-action metrics, log them
            if request.post_action_metrics and request.post_action_metrics.values:
                logger.info(
                    f"Post-action cluster metrics: {dict(request.post_action_metrics.values)}")

            # Persist to ClickHouse for historical analysis
            if self.engine_data:
                try:
                    post_metrics = {}
                    if request.post_action_metrics and request.post_action_metrics.values:
                        post_metrics = dict(request.post_action_metrics.values)

                    # Store cluster outcome (adapt the method or create new one)
                    success = await self.engine_data.store_execution_outcome(
                        cluster_id=request.cluster.api_key,
                        namespace="",  # Cluster-level has no namespace
                        app_name="cluster",
                        workload_kind="Cluster",
                        decision_id=request.decision_id,
                        success=request.success,
                        note=request.note,
                        post_action_metrics=post_metrics,
                        reported_at=datetime.utcnow()
                    )

                    if success:
                        logger.info(f"Persisted cluster outcome to ClickHouse for decision {request.decision_id}")
                    else:
                        logger.warning(f"Failed to persist cluster outcome to ClickHouse for decision {request.decision_id}")

                except Exception as db_error:
                    logger.warning(f"Failed to persist cluster outcome to ClickHouse: {str(db_error)}")

            return empty_pb2.Empty()

        except Exception as e:
            logger.error(f"Error processing cluster execution outcome: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return empty_pb2.Empty()

    def _generate_baseline_recommendation(
        self,
        request: engine_pb2.RecommendationAppRequest,
        service_slo: Optional[engine_pb2.SyncSLORequest],
        cluster_config: Optional[engine_pb2.ClusterOptimizationConfigRequest]
    ) -> engine_pb2.AppActionPlan:
        """Generate a baseline heuristic recommendation when ML is not available."""

        # Simple heuristic: if we have recent metrics, make conservative adjustments
        if request.snapshot and request.snapshot.values:
            metrics = dict(request.snapshot.values)

            # Check CPU utilization
            cpu_util = metrics.get("cpu_utilization", 0.5)
            memory_util = metrics.get("memory_utilization", 0.5)

            # Conservative scaling logic
            if cpu_util > 0.8 or memory_util > 0.8:
                # Scale up resources
                return engine_pb2.AppActionPlan(
                    type="VPA_RECOMMEND",
                    confidence=0.7,
                    reason="High resource utilization detected",
                    vpa_recommend=engine_pb2.VpaRecommendAction(
                        container="app",  # Default container name
                        # 20% increase, capped
                        cpu_request_mcpu=int(1000 * min(1.5, cpu_util * 1.2)),
                        # 20% increase, capped
                        memory_mib=int(512 * min(1.5, memory_util * 1.2)),
                        mode="recommendation"
                    )
                )
            elif cpu_util < 0.3 and memory_util < 0.3:
                # Scale down resources
                return engine_pb2.AppActionPlan(
                    type="VPA_RECOMMEND",
                    confidence=0.6,
                    reason="Low resource utilization detected",
                    vpa_recommend=engine_pb2.VpaRecommendAction(
                        container="app",
                        # Small increase from current
                        cpu_request_mcpu=int(1000 * max(0.1, cpu_util * 1.1)),
                        # Small increase from current
                        memory_mib=int(512 * max(0.1, memory_util * 1.1)),
                        mode="recommendation"
                    )
                )

        # No action needed
        return engine_pb2.AppActionPlan(
            type="NO_ACTION",
            confidence=0.5,
            reason="No significant resource pressure detected"
        )

    def _get_safety_policy(self, cluster_config: Optional[engine_pb2.ClusterOptimizationConfigRequest]) -> engine_pb2.SafetyPolicy:
        """Get the effective safety policy for recommendations."""

        # Default conservative safety policy
        default_policy = engine_pb2.SafetyPolicy(
            max_increase_ratio=1.5,  # Allow up to 50% increase
            max_decrease_ratio=0.8,  # Allow up to 20% decrease
            min_action_cooldown_seconds=300  # 5 minute cooldown
        )

        # Add default resource bounds
        cpu_limit = engine_pb2.ResourceLimit(min="100m", max="2")
        memory_limit = engine_pb2.ResourceLimit(min="128Mi", max="4Gi")

        default_policy.resource_bounds["cpu"] = cpu_limit
        default_policy.resource_bounds["memory"] = memory_limit

        return default_policy

    def _apply_safety_constraints(
        self,
        action_plan: engine_pb2.AppActionPlan,
        safety_policy: engine_pb2.SafetyPolicy
    ) -> engine_pb2.AppActionPlan:
        """Apply safety policy constraints to the action plan."""

        # For now, return the plan as-is
        # TODO: Implement actual safety constraint logic
        # - Check resource bounds
        # - Validate increase/decrease ratios
        # - Ensure cooldown periods

        return action_plan

    def _generate_baseline_cluster_recommendation(
        self,
        request: engine_pb2.RecommendationClusterRequest,
        cluster_config: Optional[engine_pb2.ClusterOptimizationConfigRequest]
    ) -> engine_pb2.ClusterActionPlan:
        """Generate a baseline cluster recommendation (placeholder for Karpenter-style logic)."""

        # Placeholder cluster recommendation logic
        # TODO: Implement proper Karpenter-style algorithms that consider:
        # - Current node utilization
        # - Pending pods that can't be scheduled
        # - Cost optimization based on instance types
        # - Spot vs on-demand preferences
        # - Node diversity for availability

        # For now, return a simple cluster provisioning recommendation
        instance_types = ["m5.large", "m5.xlarge"]
        if cluster_config and cluster_config.preferred_instance_types:
            instance_types = list(cluster_config.preferred_instance_types)

        capacity_type = "on-demand"
        if cluster_config and cluster_config.allow_spot:
            capacity_type = "spot"

        provision_action = engine_pb2.ClusterProvisionAction(
            instance_types=instance_types,
            count=1,  # Conservative default
            capacity_type=capacity_type
        )

        return engine_pb2.ClusterActionPlan(
            confidence=0.6,
            reason="Placeholder cluster recommendation - needs proper Karpenter algorithm",
            details=provision_action
        )
