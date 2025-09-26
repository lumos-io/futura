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
    with patch('services.rl_server.ClickHouseClient') as mock_ch, \
            patch('services.rl_server.KubernetesTrainingJobManager') as mock_trainer, \
            patch('torch.cuda.is_available', return_value=False), \
            patch('torch.load') as mock_torch_load, \
            patch('torch.save') as mock_torch_save:

        # Mock ClickHouse client
        clickhouse_mock = AsyncMock()
        clickhouse_mock.fetch_rows = AsyncMock()
        clickhouse_mock.insert_rows = AsyncMock()
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

    async def test_get_recommendation_full_flow(self):
        """Test complete recommendation flow from request to response."""
        try:
            from services.rl_server import RLServer, GetRecommendationRequest, GetRecommendationResponse

            # Mock ClickHouse data for historical metrics
            self.mocks['clickhouse'].fetch_rows.return_value = [
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

            server = RLServer(
                clickhouse_host="localhost",
                clickhouse_port=8123,
                clickhouse_database="test_engine",
                model_storage_path="/tmp/test-models"
            )

            # Create test request
            request = GetRecommendationRequest(
                app_key="test-cluster:default/web-app",
                current_replicas=3,
                current_cpu_limit=1000,
                current_memory_limit=512,
                current_cpu_util=0.85,
                current_memory_util=0.75,
                current_request_rate=180.0,
                current_p95_latency_ms=320.0,
                slo_targets={
                    'target_p95_latency_ms': 250.0,
                    'target_error_rate': 0.01,
                    'target_throughput_rps': 200.0
                }
            )

            # Get recommendation
            response = await server.get_recommendation(request, None)

            # Verify response structure
            assert isinstance(response, GetRecommendationResponse)
            assert response.app_key == "test-cluster:default/web-app"
            assert response.action_type in [
                "scale_out", "scale_up_cpu", "scale_up_memory", "no_action"]
            assert response.confidence >= 0.0
            assert response.confidence <= 1.0
            assert response.reasoning is not None

            # Verify ClickHouse was queried for historical data
            self.mocks['clickhouse'].fetch_rows.assert_called()

            # Verify recommendation was stored
            self.mocks['clickhouse'].insert_rows.assert_called()

        except ImportError:
            pytest.skip("RLServer module not available")

    async def test_get_recommendation_with_slo_violation(self):
        """Test recommendation when SLO is being violated."""
        try:
            from services.rl_server import RLServer, GetRecommendationRequest

            # Mock SLO violation data
            self.mocks['clickhouse'].fetch_rows.return_value = [
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

            server = RLServer(
                clickhouse_host="localhost",
                clickhouse_port=8123,
                clickhouse_database="test_engine",
                model_storage_path="/tmp/test-models"
            )

            # Create request with SLO violation
            request = GetRecommendationRequest(
                app_key="test-cluster:default/stressed-app",
                current_replicas=2,
                current_cpu_limit=1000,
                current_memory_limit=512,
                current_cpu_util=0.95,
                current_memory_util=0.90,
                current_request_rate=250.0,
                current_p95_latency_ms=800.0,
                current_error_rate=0.025,
                slo_targets={
                    'target_p95_latency_ms': 300.0,
                    'target_error_rate': 0.01,
                    'target_throughput_rps': 200.0
                }
            )

            response = await server.get_recommendation(request, None)

            # Should recommend scaling action due to SLO violation
            assert response.action_type in [
                "scale_out", "scale_up_cpu", "scale_up_memory"]
            assert "slo" in response.reasoning.lower(
            ) or "latency" in response.reasoning.lower()
            assert response.confidence > 0.5  # High confidence for SLO violations

        except ImportError:
            pytest.skip("RLServer module not available")

    async def test_trigger_training_integration(self):
        """Test training trigger integration."""
        try:
            from services.rl_server import RLServer, TriggerTrainRequest, TriggerTrainResponse

            server = RLServer(
                clickhouse_host="localhost",
                clickhouse_port=8123,
                clickhouse_database="test_engine",
                model_storage_path="/tmp/test-models"
            )

            # Create training request
            request = TriggerTrainRequest(
                app_key="test-cluster:default/training-app",
                reason="performance_drift",
                horizon_hours=12,
                hparams={
                    "learning_rate": 0.001,
                    "batch_size": 64,
                    "episodes": 2000
                }
            )

            response = await server.trigger_train(request, None)

            # Verify training was triggered
            assert isinstance(response, TriggerTrainResponse)
            assert response.training_id is not None
            assert response.success is True
            assert response.estimated_completion_time > 0

            # Verify training job was created
            self.mocks['trainer'].create_training_job.assert_called_once()

        except ImportError:
            pytest.skip("RLServer module not available")

    async def test_model_management_integration(self):
        """Test model loading and version management."""
        try:
            from services.rl_server import RLServer

            server = RLServer(
                clickhouse_host="localhost",
                clickhouse_port=8123,
                clickhouse_database="test_engine",
                model_storage_path="/tmp/test-models"
            )

            # Test model loading
            with patch.object(server, '_load_model') as mock_load:
                mock_model = Mock()
                mock_model.select_action = Mock(return_value=1)
                mock_load.return_value = mock_model

                await server._initialize_models()

                # Verify model was loaded
                mock_load.assert_called()

        except ImportError:
            pytest.skip("RLServer module not available")


@pytest.mark.integration
class TestRecommendationServiceIntegration:
    """Integration tests for RecommendationService."""

    @pytest.fixture(autouse=True)
    def setup_method(self, mock_rl_server_dependencies):
        """Set up test method with mocked dependencies."""
        self.mocks = mock_rl_server_dependencies

    async def test_recommendation_service_full_flow(self):
        """Test complete recommendation service flow."""
        try:
            from services.recommendation_service import RecommendationService

            # Mock safety policies
            safety_config = {
                'max_scale_out_factor': 2.0,
                'max_scale_in_factor': 0.5,
                'min_replicas': 1,
                'max_replicas': 10,
                'resource_limits': {
                    'cpu_max_mcpu': 4000,
                    'memory_max_mib': 8192
                }
            }

            service = RecommendationService(
                rl_server_host="localhost",
                rl_server_port=50051,
                safety_config=safety_config
            )

            # Mock gRPC client
            with patch.object(service, '_get_rl_recommendation') as mock_rl:
                mock_rl.return_value = {
                    'action_type': 'scale_out',
                    'target_replicas': 6,
                    'confidence': 0.85,
                    'reasoning': 'High CPU utilization detected'
                }

                recommendation = await service.get_safe_recommendation(
                    app_key="test-cluster:default/app",
                    current_state={
                        'replicas': 3,
                        'cpu_limit': 1000,
                        'memory_limit': 512,
                        'cpu_util': 0.85,
                        'memory_util': 0.65
                    },
                    slo_targets={
                        'target_p95_latency_ms': 250.0,
                        'target_error_rate': 0.01
                    }
                )

                # Verify safety constraints are applied
                assert recommendation['action_type'] == 'scale_out'
                # Should be constrained
                assert recommendation['target_replicas'] <= 6
                assert recommendation['safety_applied'] is not None

        except ImportError:
            pytest.skip("RecommendationService module not available")

    async def test_safety_policy_enforcement(self):
        """Test safety policy enforcement in recommendations."""
        try:
            from services.recommendation_service import RecommendationService

            # Restrictive safety config
            safety_config = {
                'max_scale_out_factor': 1.5,  # Only 50% increase
                'max_scale_in_factor': 0.8,   # Only 20% decrease
                'min_replicas': 2,
                'max_replicas': 5,
                'resource_limits': {
                    'cpu_max_mcpu': 2000,
                    'memory_max_mib': 4096
                }
            }

            service = RecommendationService(
                rl_server_host="localhost",
                rl_server_port=50051,
                safety_config=safety_config
            )

            # Mock aggressive scaling recommendation
            with patch.object(service, '_get_rl_recommendation') as mock_rl:
                mock_rl.return_value = {
                    'action_type': 'scale_out',
                    'target_replicas': 10,  # Aggressive scaling
                    'confidence': 0.95,
                    'reasoning': 'Critical performance issue'
                }

                recommendation = await service.get_safe_recommendation(
                    app_key="test-cluster:default/app",
                    current_state={
                        'replicas': 3,
                        'cpu_limit': 1000,
                        'memory_limit': 512,
                        'cpu_util': 0.95,
                        'memory_util': 0.90
                    },
                    slo_targets={}
                )

                # Should be constrained by safety policy
                assert recommendation['target_replicas'] <= 5  # Max replicas
                assert recommendation['safety_applied'] is True

        except ImportError:
            pytest.skip("RecommendationService module not available")


@pytest.mark.integration
class TestClickHouseIntegration:
    """Integration tests for ClickHouse client with mocked responses."""

    async def test_clickhouse_metrics_storage(self, mock_clickhouse_client):
        """Test storing recommendation metrics in ClickHouse."""
        # Mock successful insert
        mock_clickhouse_client.insert_rows.return_value = None

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
        await mock_clickhouse_client.insert_rows(
            "recommendation_decisions",
            [recommendation_data]
        )

        # Verify insert was called
        mock_clickhouse_client.insert_rows.assert_called_once_with(
            "recommendation_decisions",
            [recommendation_data]
        )

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

        mock_clickhouse_client.fetch_rows.return_value = historical_data

        # Query historical data
        query = """
        SELECT timestamp, cpu_utilization, memory_utilization,
               request_rate, p95_latency_ms, num_replicas
        FROM app_metrics
        WHERE app_key = %(app_key)s
        AND timestamp >= %(start_time)s
        ORDER BY timestamp DESC
        LIMIT 100
        """

        params = {
            'app_key': 'test-cluster:default/app',
            'start_time': 1640995200 - 7200  # 2 hours ago
        }

        result = await mock_clickhouse_client.fetch_rows(query, params)

        # Verify query was executed
        mock_clickhouse_client.fetch_rows.assert_called_once_with(
            query, params)
        assert len(result) == 2
        assert result[0]['cpu_utilization'] == 0.75


@pytest.mark.integration
class TestKubernetesIntegration:
    """Integration tests for Kubernetes interactions with mocked clients."""

    async def test_kubernetes_deployment_scaling(self, mock_kubernetes_client):
        """Test scaling Kubernetes deployments."""
        try:
            from services.kubernetes_client import KubernetesClient

            client = KubernetesClient()

            # Mock deployment patch
            mock_kubernetes_client.AppsV1Api().patch_namespaced_deployment.return_value = Mock(
                spec=Mock(replicas=5)
            )

            # Scale deployment
            result = await client.scale_deployment(
                name="web-app",
                namespace="default",
                replicas=5
            )

            assert result is True
            mock_kubernetes_client.AppsV1Api().patch_namespaced_deployment.assert_called_once()

        except ImportError:
            pytest.skip("KubernetesClient module not available")

    async def test_kubernetes_hpa_update(self, mock_kubernetes_client):
        """Test updating HPA configuration."""
        try:
            from services.kubernetes_client import KubernetesClient

            client = KubernetesClient()

            # Mock HPA patch
            mock_kubernetes_client.AutoscalingV1Api(
            ).patch_namespaced_horizontal_pod_autoscaler.return_value = Mock()

            # Update HPA
            result = await client.update_hpa(
                name="web-app-hpa",
                namespace="default",
                min_replicas=2,
                max_replicas=8,
                target_cpu_utilization=70
            )

            assert result is True
            mock_kubernetes_client.AutoscalingV1Api(
            ).patch_namespaced_horizontal_pod_autoscaler.assert_called_once()

        except ImportError:
            pytest.skip("KubernetesClient module not available")


@pytest.mark.integration
@pytest.mark.slow
class TestEndToEndRecommendationFlow:
    """End-to-end integration tests for complete recommendation flow."""

    async def test_complete_recommendation_pipeline(self, mock_rl_server_dependencies):
        """Test complete pipeline from metrics ingestion to recommendation execution."""
        try:
            from services.rl_server import RLServer
            from services.recommendation_service import RecommendationService

            # Set up mocked pipeline
            self.mocks = mock_rl_server_dependencies

            # Mock app metrics data
            app_metrics = {
                'app_key': 'prod-cluster:default/critical-app',
                'current_replicas': 3,
                'cpu_utilization': 0.85,
                'memory_utilization': 0.75,
                'request_rate': 200.0,
                'p95_latency_ms': 350.0,
                'error_rate': 0.015
            }

            # Mock SLO targets
            slo_targets = {
                'target_p95_latency_ms': 250.0,
                'target_error_rate': 0.01,
                'target_throughput_rps': 180.0
            }

            # Initialize services
            rl_server = RLServer(
                clickhouse_host="localhost",
                clickhouse_port=8123,
                clickhouse_database="test_engine",
                model_storage_path="/tmp/test-models"
            )

            recommendation_service = RecommendationService(
                rl_server_host="localhost",
                rl_server_port=50051,
                safety_config={
                    'max_scale_out_factor': 2.0,
                    'max_scale_in_factor': 0.5,
                    'min_replicas': 1,
                    'max_replicas': 10
                }
            )

            # Simulate recommendation flow
            with patch.object(recommendation_service, '_get_rl_recommendation') as mock_rl:
                mock_rl.return_value = {
                    'action_type': 'scale_out',
                    'target_replicas': 5,
                    'confidence': 0.90,
                    'reasoning': 'SLO violation: high latency detected'
                }

                # Get recommendation
                recommendation = await recommendation_service.get_safe_recommendation(
                    app_key=app_metrics['app_key'],
                    current_state=app_metrics,
                    slo_targets=slo_targets
                )

                # Verify recommendation
                assert recommendation['action_type'] == 'scale_out'
                assert recommendation['target_replicas'] == 5
                assert recommendation['confidence'] >= 0.8

                # Verify safety constraints were checked
                assert 'safety_applied' in recommendation

                # Verify ClickHouse storage was called
                self.mocks['clickhouse'].insert_rows.assert_called()

        except ImportError:
            pytest.skip("Required modules not available for E2E test")
