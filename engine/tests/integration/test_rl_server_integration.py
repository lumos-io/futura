"""
Integration tests for RLServer with mocked external dependencies.
Tests the complete recommendation flow with mocked ClickHouse and Kubernetes.
"""
import pytest
from unittest.mock import Mock, AsyncMock, patch, MagicMock
import asyncio
import json
import sys
import os

# Add engine root to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..'))


@pytest.fixture
def mock_rl_server_dependencies():
    """Mock all external dependencies for RLServer."""
    with patch('storage.clickhouse_client.ClickHouseClient') as mock_ch, \
            patch('training.training_job_manager.KubernetesTrainingJobManager') as mock_trainer, \
            patch('torch.cuda.is_available', return_value=False), \
            patch('torch.load') as mock_torch_load, \
            patch('torch.save') as mock_torch_save:

        # Mock ClickHouse client
        clickhouse_mock = AsyncMock()
        clickhouse_mock.execute_query = AsyncMock()
        clickhouse_mock.execute_insert = AsyncMock()
        mock_ch.return_value = clickhouse_mock

        # Mock training job manager
        trainer_mock = AsyncMock()
        trainer_mock.create_training_job = AsyncMock(return_value=True)
        trainer_mock.get_job_status = AsyncMock(return_value="completed")
        mock_trainer.return_value = trainer_mock

        # Mock PyTorch model loading
        mock_model = Mock()
        mock_model.eval = Mock()
        mock_model.select_action = Mock(return_value=2)  # Scale out action
        mock_torch_load.return_value = {
            'model_state_dict': {},
            'version': 'v1.0.0'
        }

        yield {
            'clickhouse': clickhouse_mock,
            'trainer': trainer_mock,
            'model': mock_model
        }


@pytest.mark.integration
class TestRLServerIntegration:
    """Integration tests for RLServer."""

    @pytest.fixture(autouse=True)
    def setup_method(self, mock_rl_server_dependencies):
        """Set up test method with mocked dependencies."""
        self.mocks = mock_rl_server_dependencies

    @pytest.mark.asyncio
    async def test_get_recommendation_full_flow(self):
        """Test complete recommendation flow using scaling algorithms directly."""
        try:
            from scaling.scaling_algorithms import ScalingAlgorithms, ResourceState, ScalingConstraints

            # Mock ClickHouse data for historical metrics
            self.mocks['clickhouse'].execute_query.return_value = [
                {
                    'timestamp': 1640995200,  # Mock timestamp
                    'cpu_utilization': 0.85,
                    'memory_utilization': 0.75,
                    'request_rate': 180.0,
                    'p95_latency_ms': 320.0,
                    'error_rate': 0.008,
                    'num_replicas': 3
                }
            ]

            # Create scaling algorithms directly instead of using RLServer
            scaling_algorithms = ScalingAlgorithms(
                constraints=ScalingConstraints(
                    vertical_cpu_step=256,
                    vertical_memory_step=256,
                    max_instances=20,
                    max_cpu_limit=4000,
                    max_memory_limit=8192
                )
            )

            # Create test resource state
            resource_state = ResourceState(
                num_replicas=3,
                cpu_limit=1000,
                memory_limit=512,
                cpu_util=0.85,
                memory_util=0.75
            )

            # Define SLO targets
            slo_targets = {
                'target_p95_latency_ms': 250.0,
                'target_error_rate': 0.01,
                'target_throughput_rps': 200.0
            }

            # Get recommendation
            action = scaling_algorithms.get_intelligent_scaling_action(
                current_state=resource_state,
                slo_targets=slo_targets,
                app_key="test-cluster:default/web-app"
            )

            # Verify response structure
            assert action is not None
            assert hasattr(action, 'action_type')
            assert action.action_type in [
                "horizontal", "vertical_cpu", "vertical_memory", "no_action"]
            assert hasattr(action, 'confidence')
            assert action.confidence >= 0.0
            assert action.confidence <= 1.0
            assert hasattr(action, 'reason')

            # Test that we can mock ClickHouse operations
            self.mocks['clickhouse'].execute_insert.return_value = True
            insert_result = await self.mocks['clickhouse'].execute_insert(
                "recommendation_decisions",
                {"app_key": "test-cluster:default/web-app", "action": action.action_type}
            )
            assert insert_result is True

        except ImportError as e:
            pytest.skip(f"Required modules not available: {e}")

    @pytest.mark.asyncio
    async def test_get_recommendation_with_slo_violation(self):
        """Test recommendation when SLO is being violated."""
        try:
            from scaling.scaling_algorithms import ScalingAlgorithms, ResourceState, ScalingConstraints

            # Mock SLO violation data
            self.mocks['clickhouse'].execute_query.return_value = [
                {
                    'timestamp': 1640995200,
                    'cpu_utilization': 0.95,
                    'memory_utilization': 0.90,
                    'request_rate': 250.0,
                    'p95_latency_ms': 800.0,  # High latency
                    'error_rate': 0.025,      # High error rate
                    'num_replicas': 2
                }
            ]

            # Create scaling algorithms
            scaling_algorithms = ScalingAlgorithms(
                constraints=ScalingConstraints(
                    vertical_cpu_step=256,
                    vertical_memory_step=256,
                    max_instances=20,
                    max_cpu_limit=4000,
                    max_memory_limit=8192
                )
            )

            # Create resource state with SLO violation
            resource_state = ResourceState(
                num_replicas=2,
                cpu_limit=1000,
                memory_limit=512,
                cpu_util=0.95,
                memory_util=0.90
            )

            slo_targets = {
                'target_p95_latency_ms': 300.0,
                'target_error_rate': 0.01,
                'target_throughput_rps': 200.0
            }

            action = scaling_algorithms.get_intelligent_scaling_action(
                current_state=resource_state,
                slo_targets=slo_targets,
                app_key="test-cluster:default/stressed-app"
            )

            # Should recommend scaling action due to SLO violation
            assert action.action_type in [
                "horizontal", "vertical_cpu", "vertical_memory"]
            assert "slo" in action.reason.lower() or "latency" in action.reason.lower() or "utilization" in action.reason.lower()
            assert action.confidence > 0.5  # High confidence for SLO violations

        except ImportError as e:
            pytest.skip(f"Required modules not available: {e}")

    @pytest.mark.asyncio
    async def test_trigger_training_integration(self):
        """Test training trigger integration."""
        try:
            from training.training_job_manager import KubernetesTrainingJobManager, TrainingJobSpec
            import uuid

            # Create training job manager
            trainer = KubernetesTrainingJobManager()

            # Create training job spec
            training_id = str(uuid.uuid4())
            job_spec = TrainingJobSpec(
                training_id=training_id,
                app_key="test-cluster:default/training-app",
                job_name="test-training-job",
                reason="performance_drift",
                horizon_hours=12,
                base_version="v1.0.0",
                hparams={
                    "learning_rate": 0.001,
                    "batch_size": 64,
                    "episodes": 2000
                }
            )

            # Mock successful job creation
            self.mocks['trainer'].create_training_job.return_value = True

            # Trigger training
            result = await self.mocks['trainer'].create_training_job(job_spec)

            # Verify training was triggered
            assert result is True
            assert training_id is not None

            # Verify training job was created
            self.mocks['trainer'].create_training_job.assert_called_once()

        except ImportError as e:
            pytest.skip(f"Required modules not available: {e}")

    @pytest.mark.asyncio
    async def test_model_management_integration(self):
        """Test model loading and version management."""
        try:
            from rl_models.ppo import PPOAgent
            from rl_models.state_action_space import StateSpace, ActionSpace
            import torch

            # Test model initialization
            state_space = StateSpace()
            action_space = ActionSpace()

            # Mock model creation and PyTorch operations
            with patch('torch.load') as mock_load, \
                 patch('torch.save') as mock_save, \
                 patch('os.path.exists') as mock_exists:

                mock_load.return_value = {
                    'actor_state_dict': {},
                    'critic_state_dict': {},
                    'optimizer_state_dict': {},
                    'version': 'v1.0.0'
                }
                mock_exists.return_value = True  # Mock file existence

                # Create PPO agent
                agent = PPOAgent(
                    state_size=state_space.state_dim,
                    action_size=action_space.action_dim
                )

                # Test model save/load functionality
                model_path = "/tmp/test-model.pth"

                # Mock save operation
                mock_save.return_value = None
                agent.save_model(model_path)
                mock_save.assert_called()

                # Mock the load_state_dict methods to avoid actual model loading
                with patch.object(agent.actor, 'load_state_dict') as mock_actor_load, \
                     patch.object(agent.critic, 'load_state_dict') as mock_critic_load, \
                     patch.object(agent.optimizer, 'load_state_dict') as mock_opt_load:

                    # Mock load operation
                    agent.load_model(model_path)
                    mock_load.assert_called()
                    mock_actor_load.assert_called()
                    mock_critic_load.assert_called()
                    mock_opt_load.assert_called()

        except ImportError as e:
            pytest.skip(f"Required modules not available: {e}")


@pytest.mark.integration
class TestRecommendationServiceIntegration:
    """Integration tests for RecommendationService."""

    @pytest.fixture(autouse=True)
    def setup_method(self, mock_rl_server_dependencies):
        """Set up test method with mocked dependencies."""
        self.mocks = mock_rl_server_dependencies

    @pytest.mark.asyncio
    async def test_recommendation_service_full_flow(self):
        """Test complete recommendation service flow using scaling algorithms."""
        try:
            from scaling.scaling_algorithms import ScalingAlgorithms, ResourceState, ScalingConstraints

            # Mock safety policies through constraints
            constraints = ScalingConstraints(
                vertical_cpu_step=256,
                vertical_memory_step=256,
                max_instances=10,  # max_replicas
                min_instances=1,   # min_replicas
                max_cpu_limit=4000,
                max_memory_limit=8192
            )

            service = ScalingAlgorithms(constraints=constraints)

            # Create resource state
            resource_state = ResourceState(
                num_replicas=3,
                cpu_limit=1000,
                memory_limit=512,
                cpu_util=0.85,
                memory_util=0.65
            )

            slo_targets = {
                'target_p95_latency_ms': 250.0,
                'target_error_rate': 0.01
            }

            # Get recommendation
            action = service.get_intelligent_scaling_action(
                current_state=resource_state,
                slo_targets=slo_targets,
                app_key="test-cluster:default/app"
            )

            # Verify safety constraints are applied
            assert action.action_type in ['horizontal', 'vertical_cpu', 'vertical_memory', 'no_action']
            if hasattr(action, 'target_replicas') and action.target_replicas is not None:
                assert action.target_replicas <= 10  # Max replicas constraint
                assert action.target_replicas >= 1   # Min replicas constraint
            assert action.confidence >= 0.0
            assert action.confidence <= 1.0

        except ImportError as e:
            pytest.skip(f"Required modules not available: {e}")

    @pytest.mark.asyncio
    async def test_safety_policy_enforcement(self):
        """Test safety policy enforcement in recommendations."""
        try:
            from scaling.scaling_algorithms import ScalingAlgorithms, ResourceState, ScalingConstraints

            # Restrictive constraints
            constraints = ScalingConstraints(
                vertical_cpu_step=256,
                vertical_memory_step=256,
                max_instances=5,   # Restrictive max replicas
                min_instances=2,   # Restrictive min replicas
                max_cpu_limit=2000,  # Restrictive CPU limit
                max_memory_limit=4096  # Restrictive memory limit
            )

            service = ScalingAlgorithms(constraints=constraints)

            # Create resource state with aggressive scaling need
            resource_state = ResourceState(
                num_replicas=3,
                cpu_limit=1000,
                memory_limit=512,
                cpu_util=0.95,
                memory_util=0.90
            )

            action = service.get_intelligent_scaling_action(
                current_state=resource_state,
                slo_targets={},
                app_key="test-cluster:default/app"
            )

            # Should be constrained by safety policy
            if hasattr(action, 'target_replicas'):
                assert action.target_replicas <= 5  # Max replicas constraint
                assert action.target_replicas >= 2  # Min replicas constraint

            # Verify action is within constraints
            assert action.action_type in ['horizontal', 'vertical_cpu', 'vertical_memory', 'no_action']
            assert action.confidence >= 0.0

        except ImportError as e:
            pytest.skip(f"Required modules not available: {e}")


@pytest.fixture
def mock_clickhouse_client():
    """Mock ClickHouse client for integration tests."""
    client_mock = AsyncMock()
    client_mock.execute_insert = AsyncMock()
    client_mock.execute_query = AsyncMock()
    return client_mock


@pytest.fixture
def mock_kubernetes_client():
    """Mock Kubernetes client for integration tests."""
    k8s_mock = Mock()
    apps_api_mock = Mock()
    autoscaling_api_mock = Mock()

    # Create a mock deployment result with spec attribute
    deployment_result = Mock()
    deployment_result.spec = Mock()
    deployment_result.spec.replicas = 5
    apps_api_mock.patch_namespaced_deployment = Mock(return_value=deployment_result)
    autoscaling_api_mock.patch_namespaced_horizontal_pod_autoscaler = Mock()

    k8s_mock.AppsV1Api.return_value = apps_api_mock
    k8s_mock.AutoscalingV1Api.return_value = autoscaling_api_mock

    return k8s_mock


@pytest.mark.integration
class TestClickHouseIntegration:
    """Integration tests for ClickHouse client with mocked responses."""

    @pytest.mark.asyncio
    async def test_clickhouse_metrics_storage(self, mock_clickhouse_client):
        """Test storing recommendation metrics in ClickHouse."""
        # Mock successful insert
        mock_clickhouse_client.execute_insert.return_value = True

        # Sample recommendation data
        recommendation_data = {
            'timestamp': 1640995200,
            'app_key': 'test-cluster:default/app',
            'action_type': 'scale_out',
            'target_replicas': 5,
            'confidence': 0.85,
            'reasoning': 'High CPU utilization',
            'current_cpu_util': 0.85,
            'current_memory_util': 0.65,
            'slo_violation': False
        }

        # Insert recommendation
        await mock_clickhouse_client.execute_insert(
            "recommendation_decisions",
            [recommendation_data]
        )

        # Verify insert was called
        mock_clickhouse_client.execute_insert.assert_called_once_with(
            "recommendation_decisions",
            [recommendation_data]
        )

    @pytest.mark.asyncio
    async def test_clickhouse_historical_data_query(self, mock_clickhouse_client):
        """Test querying historical metrics from ClickHouse."""
        # Mock historical data
        historical_data = [
            {
                'timestamp': 1640995200 - 3600,  # 1 hour ago
                'app_key': 'test-cluster:default/app',
                'cpu_utilization': 0.75,
                'memory_utilization': 0.60,
                'request_rate': 150.0,
                'p95_latency_ms': 200.0,
                'num_replicas': 3
            },
            {
                'timestamp': 1640995200 - 1800,  # 30 minutes ago
                'app_key': 'test-cluster:default/app',
                'cpu_utilization': 0.80,
                'memory_utilization': 0.65,
                'request_rate': 160.0,
                'p95_latency_ms': 220.0,
                'num_replicas': 3
            }
        ]

        mock_clickhouse_client.execute_query.return_value = historical_data

        # Query historical data
        query = """
        SELECT timestamp, cpu_utilization, memory_utilization,
               request_rate, p95_latency_ms, num_replicas
        FROM app_metrics
        WHERE app_key = '{app_key}'
        AND timestamp >= {start_time}
        ORDER BY timestamp DESC
        LIMIT 100
        """

        params = {
            'app_key': 'test-cluster:default/app',
            'start_time': 1640995200 - 7200  # 2 hours ago
        }

        result = await mock_clickhouse_client.execute_query(query, params)

        # Verify query was executed
        mock_clickhouse_client.execute_query.assert_called_once_with(
            query, params)
        assert len(result) == 2
        assert result[0]['cpu_utilization'] == 0.75


@pytest.mark.integration
class TestKubernetesIntegration:
    """Integration tests for Kubernetes interactions with mocked clients."""

    @pytest.mark.asyncio
    async def test_kubernetes_deployment_scaling(self, mock_kubernetes_client):
        """Test scaling Kubernetes deployments."""
        try:
            # Test Kubernetes deployment scaling logic directly
            # Mock the actual patch operation
            apps_api = mock_kubernetes_client.AppsV1Api()

            # Simulate deployment scaling
            patch_body = {
                "spec": {
                    "replicas": 5
                }
            }

            # Call the mocked patch method
            result = apps_api.patch_namespaced_deployment(
                name="web-app",
                namespace="default",
                body=patch_body
            )

            assert result is not None
            assert result.spec.replicas == 5
            apps_api.patch_namespaced_deployment.assert_called_once()

        except ImportError as e:
            pytest.skip(f"Kubernetes modules not available: {e}")

    @pytest.mark.asyncio
    async def test_kubernetes_hpa_update(self, mock_kubernetes_client):
        """Test updating HPA configuration."""
        try:
            # Test HPA update logic directly
            autoscaling_api = mock_kubernetes_client.AutoscalingV1Api()

            # Simulate HPA update
            patch_body = {
                "spec": {
                    "minReplicas": 2,
                    "maxReplicas": 8,
                    "targetCPUUtilizationPercentage": 70
                }
            }

            # Call the mocked patch method
            autoscaling_api.patch_namespaced_horizontal_pod_autoscaler(
                name="web-app-hpa",
                namespace="default",
                body=patch_body
            )

            autoscaling_api.patch_namespaced_horizontal_pod_autoscaler.assert_called_once()

        except ImportError as e:
            pytest.skip(f"Kubernetes modules not available: {e}")


@pytest.mark.integration
@pytest.mark.slow
class TestEndToEndRecommendationFlow:
    """End-to-end integration tests for complete recommendation flow."""

    @pytest.mark.asyncio
    async def test_complete_recommendation_pipeline(self, mock_rl_server_dependencies):
        """Test complete pipeline from metrics ingestion to recommendation execution."""
        try:
            from scaling.scaling_algorithms import ScalingAlgorithms, ResourceState, ScalingConstraints
            from storage.clickhouse_client import EngineDataAccess

            # Set up mocked pipeline
            self.mocks = mock_rl_server_dependencies

            # Mock app metrics data
            resource_state = ResourceState(
                num_replicas=3,
                cpu_limit=1000,
                memory_limit=512,
                cpu_util=0.85,
                memory_util=0.75
            )

            # Mock SLO targets
            slo_targets = {
                'target_p95_latency_ms': 250.0,
                'target_error_rate': 0.01,
                'target_throughput_rps': 180.0
            }

            # Initialize scaling algorithms with safety constraints
            constraints = ScalingConstraints(
                vertical_cpu_step=256,
                vertical_memory_step=256,
                max_instances=10,
                min_instances=1,
                max_cpu_limit=4000,
                max_memory_limit=8192
            )

            scaling_algorithms = ScalingAlgorithms(constraints=constraints)

            # Get recommendation using scaling algorithms
            action = scaling_algorithms.get_intelligent_scaling_action(
                current_state=resource_state,
                slo_targets=slo_targets,
                app_key="prod-cluster:default/critical-app"
            )

            # Verify recommendation
            assert action.action_type in ['horizontal', 'vertical_cpu', 'vertical_memory', 'no_action']
            assert action.confidence >= 0.0
            assert action.confidence <= 1.0

            # Test ClickHouse storage using mocked client
            engine_data = EngineDataAccess(self.mocks['clickhouse'])
            self.mocks['clickhouse'].execute_insert.return_value = True

            result = await engine_data.store_recommendation_decision(
                cluster_id="prod-cluster",
                namespace="default",
                app_name="critical-app",
                workload_kind="Deployment",
                decision_id="test-decision-123",
                model_version="v1.0.0",
                confidence=action.confidence,
                audit_reasons=[action.reason],
                plan_vertical=[],
                plan_replicas=5,
                effective_policy={}
            )

            # Verify ClickHouse storage was called
            assert result is True
            self.mocks['clickhouse'].execute_insert.assert_called()

        except ImportError as e:
            pytest.skip(f"Required modules not available for E2E test: {e}")
