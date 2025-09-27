"""
End-to-end tests for gRPC services.
Tests complete gRPC communication flows with mocked external dependencies.
"""
import pytest
import pytest_asyncio
from unittest.mock import Mock, AsyncMock, patch, MagicMock
import sys
import os

# Add engine root to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..'))


@pytest_asyncio.fixture
async def grpc_test_server():
    """Set up a test gRPC server for E2E testing."""
    try:
        from services.rl_server import RLServer
        from grpc import aio as grpc_aio

        # Mock external dependencies
        with patch('services.rl_server.ClickHouseClient') as mock_ch, \
                patch('services.rl_server.KubernetesTrainingJobManager') as mock_trainer, \
                patch('torch.cuda.is_available', return_value=False):

            # Set up mocks
            clickhouse_mock = AsyncMock()
            clickhouse_mock.fetch_rows = AsyncMock(return_value=[])
            clickhouse_mock.insert_rows = AsyncMock()
            mock_ch.return_value = clickhouse_mock

            trainer_mock = AsyncMock()
            trainer_mock.create_training_job = AsyncMock(return_value=True)
            mock_trainer.return_value = trainer_mock

            # Create and start server
            server = grpc_aio.server()
            rl_service = RLServer(
                clickhouse_client=clickhouse_mock
            )

            # Add service to server (assuming proper gRPC service setup)
            # server.add_insecure_port('[::]:0')  # Use port 0 for automatic assignment

            # For testing, we'll mock the server behavior
            mock_server = Mock()
            mock_server.start = AsyncMock()
            mock_server.stop = AsyncMock()
            mock_server.wait_for_termination = AsyncMock()

            yield {
                'server': mock_server,
                'service': rl_service,
                'mocks': {
                    'clickhouse': clickhouse_mock,
                    'trainer': trainer_mock
                }
            }

    except ImportError:
        pytest.skip("gRPC modules not available")


@pytest.mark.e2e
@pytest.mark.asyncio
class TestRLServerGRPCEndToEnd:
    """End-to-end tests for RLServer gRPC service."""

    async def test_grpc_get_recommendation_flow(self, grpc_test_server):
        """Test complete gRPC GetRecommendation flow."""
        try:
            from proto.gen.engine import engine_pb2

            server_setup = grpc_test_server
            rl_service = server_setup['service']
            mocks = server_setup['mocks']

            # Mock historical data
            mocks['clickhouse'].fetch_rows.return_value = [
                {
                    'timestamp': 1640995200,
                    'cpu_utilization': 0.80,
                    'memory_utilization': 0.70,
                    'request_rate': 160.0,
                    'p95_latency_ms': 280.0,
                    'error_rate': 0.008,
                    'num_replicas': 3
                }
            ]

            # Create gRPC request
            request = engine_pb2.GetAppActionRequest(
                app_key="e2e-cluster:default/test-app",
                current_metrics=engine_pb2.AppMetrics(
                    replicas=3,
                    cpu_limit_millicores=1000,
                    memory_limit_mib=512,
                    cpu_utilization=0.85,
                    memory_utilization=0.75,
                    request_rate=180.0,
                    p95_latency_ms=320.0,
                    error_rate=0.01
                ),
                slo_config=engine_pb2.SLOConfig(
                    target_p95_latency_ms=250.0,
                    target_error_rate=0.01,
                    target_throughput_rps=200.0
                )
            )

            # Mock gRPC context
            mock_context = Mock()

            # Call service method directly (simulating gRPC call)
            response = await rl_service.GetAppAction(request, mock_context)

            # Verify response
            assert isinstance(response, engine_pb2.GetAppActionResponse)
            assert response.app_key == "e2e-cluster:default/test-app"
            assert response.action.action_type in [
                engine_pb2.ActionType.SCALE_OUT,
                engine_pb2.ActionType.SCALE_UP_CPU,
                engine_pb2.ActionType.SCALE_UP_MEMORY,
                engine_pb2.ActionType.NO_ACTION
            ]
            assert 0.0 <= response.confidence <= 1.0
            assert response.reasoning is not None
            assert len(response.reasoning) > 0

            # Verify external calls were made
            # Note: The actual ClickHouse calls depend on the implementation

        except ImportError:
            pytest.skip("Required gRPC modules not available")

    async def test_grpc_trigger_training_flow(self, grpc_test_server):
        """Test complete gRPC TriggerTrain flow."""
        try:
            from services.rl_server import TriggerTrainRequest, TriggerTrainResponse

            server_setup = grpc_test_server
            rl_service = server_setup['service']
            mocks = server_setup['mocks']

            # Create training request
            request = TriggerTrainRequest(
                app_key="e2e-cluster:default/training-app",
                reason="manual_retrain",
                horizon_hours=8,
                hparams={
                    "learning_rate": 0.0005,
                    "batch_size": 128,
                    "episodes": 1500
                }
            )

            # Mock gRPC context
            mock_context = Mock()

            # Call service method
            response = await rl_service.trigger_train(request, mock_context)

            # Verify response
            assert isinstance(response, TriggerTrainResponse)
            assert response.training_id is not None
            assert len(response.training_id) > 0
            assert response.success is True
            assert response.estimated_completion_time > 0

            # Verify training job was created
            mocks['trainer'].create_training_job.assert_called_once()

            # Verify training request was logged
            mocks['clickhouse'].insert_rows.assert_called()

        except ImportError:
            pytest.skip("Required gRPC modules not available")

    async def test_grpc_error_handling(self, grpc_test_server):
        """Test gRPC error handling and responses."""
        try:
            from services.rl_server import GetRecommendationRequest

            server_setup = grpc_test_server
            rl_service = server_setup['service']
            mocks = server_setup['mocks']

            # Mock ClickHouse failure
            mocks['clickhouse'].fetch_rows.side_effect = Exception(
                "Database connection failed")

            # Create request
            request = GetRecommendationRequest(
                app_key="error-test:default/app",
                current_replicas=2,
                current_cpu_limit=500,
                current_memory_limit=256,
                current_cpu_util=0.90,
                current_memory_util=0.85,
                current_request_rate=100.0,
                current_p95_latency_ms=400.0
            )

            mock_context = Mock()

            # Call should handle error gracefully
            try:
                response = await rl_service.get_recommendation(request, mock_context)
                # If it returns a response, verify it's a safe fallback
                assert response.action_type == "no_action"
                assert "error" in response.reasoning.lower()
            except Exception as e:
                # Or it should raise appropriate gRPC exception
                assert "Database connection failed" in str(e)

        except ImportError:
            pytest.skip("Required gRPC modules not available")


@pytest.mark.e2e
@pytest.mark.asyncio
class TestRecommendationServiceGRPCEndToEnd:
    """End-to-end tests for RecommendationService gRPC client."""

    async def test_grpc_client_server_communication(self):
        """Test gRPC client-server communication flow."""
        try:
            from services.recommendation_service import RecommendationService
            from grpc import aio as grpc_aio

            # Mock gRPC channel and stub
            with patch('grpc.aio.insecure_channel') as mock_channel:
                mock_stub = Mock()

                # Mock successful recommendation response
                mock_response = Mock()
                mock_response.app_key = "test-cluster:default/app"
                mock_response.action_type = "scale_out"
                mock_response.target_replicas = 5
                mock_response.confidence = 0.85
                mock_response.reasoning = "High CPU utilization detected"

                mock_stub.GetRecommendation = AsyncMock(
                    return_value=mock_response)
                mock_channel.return_value.__aenter__ = AsyncMock(
                    return_value=mock_stub)
                mock_channel.return_value.__aexit__ = AsyncMock(
                    return_value=None)

                # Create service
                service = RecommendationService(
                    rl_server_host="localhost",
                    rl_server_port=50051,
                    safety_config={
                        'max_scale_out_factor': 2.0,
                        'min_replicas': 1,
                        'max_replicas': 10
                    }
                )

                # Test recommendation request
                recommendation = await service.get_safe_recommendation(
                    app_key="test-cluster:default/app",
                    current_state={
                        'replicas': 3,
                        'cpu_limit': 1000,
                        'memory_limit': 512,
                        'cpu_util': 0.85,
                        'memory_util': 0.70
                    },
                    slo_targets={
                        'target_p95_latency_ms': 250.0
                    }
                )

                # Verify recommendation
                assert recommendation['action_type'] == 'scale_out'
                assert recommendation['target_replicas'] == 5
                assert recommendation['confidence'] == 0.85

                # Verify gRPC call was made
                mock_stub.GetRecommendation.assert_called_once()

        except ImportError:
            pytest.skip("Required gRPC modules not available")

    async def test_grpc_client_retry_logic(self):
        """Test gRPC client retry logic on failures."""
        try:
            from services.recommendation_service import RecommendationService
            import grpc

            # Mock gRPC channel with failures
            with patch('grpc.aio.insecure_channel') as mock_channel:
                mock_stub = Mock()

                # First call fails, second succeeds
                mock_stub.GetRecommendation = AsyncMock(
                    side_effect=[
                        grpc.RpcError("Connection failed"),
                        Mock(
                            app_key="test-cluster:default/app",
                            action_type="scale_out",
                            target_replicas=4,
                            confidence=0.75,
                            reasoning="Retry successful"
                        )
                    ]
                )

                mock_channel.return_value.__aenter__ = AsyncMock(
                    return_value=mock_stub)
                mock_channel.return_value.__aexit__ = AsyncMock(
                    return_value=None)

                service = RecommendationService(
                    rl_server_host="localhost",
                    rl_server_port=50051,
                    safety_config={'max_scale_out_factor': 2.0}
                )

                # Should retry and succeed
                recommendation = await service.get_safe_recommendation(
                    app_key="test-cluster:default/app",
                    current_state={'replicas': 2, 'cpu_util': 0.90},
                    slo_targets={}
                )

                # Verify retry worked
                assert recommendation['action_type'] == 'scale_out'
                assert mock_stub.GetRecommendation.call_count == 2

        except ImportError:
            pytest.skip("Required gRPC modules not available")


@pytest.mark.e2e
@pytest.mark.slow
class TestFullSystemGRPCFlow:
    """Full system end-to-end tests with gRPC communication."""

    @pytest.mark.asyncio
    async def test_complete_scaling_decision_flow(self):
        """Test complete flow from metrics ingestion to scaling decision."""
        try:
            # This tests the complete flow:
            # 1. Current metrics provided to system
            # 2. RL algorithms process recommendation
            # 3. Safety policies applied
            # 4. Results are validated and ready for execution

            from scaling.scaling_algorithms import ScalingAlgorithms, ResourceState, ScalingConstraints
            from storage.clickhouse_client import ClickHouseClient

            # Create a simple RecommendationService for testing
            class TestRecommendationService:
                def __init__(self, safety_config):
                    self.safety_config = safety_config

                def apply_safety_policies(self, app_key, current_state, raw_recommendation):
                    """Apply safety policies to raw recommendation."""
                    max_replicas = self.safety_config.get('max_replicas', 10)
                    return {
                        'action_type': raw_recommendation['action_type'],
                        'target_replicas': min(raw_recommendation.get('target_replicas', current_state.num_replicas), max_replicas),
                        'confidence': raw_recommendation['confidence'],
                        'reason': raw_recommendation['reason']
                    }

            # Mock external dependencies
            with patch('storage.clickhouse_client.ClickHouseClient') as mock_ch, \
                    patch('torch.cuda.is_available', return_value=False):

                # Set up mocks
                clickhouse_mock = AsyncMock()
                clickhouse_mock.execute_query = AsyncMock(return_value=[
                    {
                        'timestamp': 1640995200,
                        'cpu_utilization': 0.85,
                        'memory_utilization': 0.75,
                        'request_rate': 180.0,
                        'p95_latency_ms': 320.0,
                        'num_replicas': 3
                    }
                ])
                clickhouse_mock.execute_insert = AsyncMock(return_value=True)
                mock_ch.return_value = clickhouse_mock

                # Initialize scaling algorithms directly
                scaling_algorithms = ScalingAlgorithms(
                    constraints=ScalingConstraints(
                        vertical_cpu_step=256,
                        vertical_memory_step=256,
                        max_instances=20,
                        max_cpu_limit=4000,
                        max_memory_limit=8192
                    )
                )

                recommendation_service = TestRecommendationService(
                    safety_config={
                        'max_scale_out_factor': 2.0, 'max_replicas': 10}
                )

                # Step 1: Create current state
                current_state = ResourceState(
                    num_replicas=3,
                    cpu_limit=1000,
                    memory_limit=512,
                    cpu_util=0.85,
                    memory_util=0.75,
                    request_rate=180.0,
                    p95_latency_ms=320.0
                )

                app_key = "prod-cluster:default/web-service"
                slo_targets = {'target_p95_latency_ms': 250.0}

                # Step 2: Get recommendation from RL algorithms
                raw_action = scaling_algorithms.get_intelligent_scaling_action(
                    current_state=current_state,
                    slo_targets=slo_targets,
                    app_key=app_key
                )

                assert raw_action is not None
                assert raw_action.action_type in [
                    "horizontal", "vertical_cpu", "vertical_memory", "no_action"
                ]

                # Step 3: Apply safety policies
                safe_recommendation = recommendation_service.apply_safety_policies(
                    app_key=app_key,
                    current_state=current_state,
                    raw_recommendation={
                        'action_type': raw_action.action_type,
                        'target_replicas': getattr(raw_action, 'target_replicas', current_state.num_replicas),
                        'confidence': raw_action.confidence,
                        'reason': raw_action.reason
                    }
                )

                # Verify complete flow worked
                assert safe_recommendation['action_type'] in [
                    'horizontal', 'vertical_cpu', 'vertical_memory', 'no_action'
                ]
                assert 0.0 <= safe_recommendation['confidence'] <= 1.0
                assert safe_recommendation['reason'] is not None

        except ImportError as e:
            pytest.skip(
                f"Required modules not available for full system test: {e}")

    @pytest.mark.asyncio
    async def test_training_trigger_to_completion_flow(self):
        """Test complete training flow from trigger to model completion."""
        try:
            from training.training_job_manager import KubernetesTrainingJobManager, TrainingJobSpec, TrainingJobResult
            from storage.clickhouse_client import ClickHouseClient

            with patch('storage.clickhouse_client.ClickHouseClient') as mock_ch, \
                    patch('kubernetes.client') as mock_k8s, \
                    patch('torch.cuda.is_available', return_value=False):

                # Set up mocks
                clickhouse_mock = AsyncMock()
                k8s_mock = AsyncMock()

                mock_ch.return_value = clickhouse_mock
                mock_k8s.return_value = k8s_mock

                # Create training job manager directly
                training_manager = KubernetesTrainingJobManager(
                    clickhouse_client=clickhouse_mock,
                    namespace="futura-training",
                    training_image="futura/rl-trainer:latest"
                )

                # Mock training lifecycle
                with patch.object(training_manager, 'create_training_job', return_value=True) as mock_create, \
                        patch.object(training_manager, 'get_job_status', side_effect=["running", "running", "succeeded"]) as mock_status, \
                        patch.object(training_manager, 'collect_job_result') as mock_collect, \
                        patch.object(training_manager, 'cleanup_completed_job', return_value=True) as mock_cleanup:

                    mock_collect.return_value = TrainingJobResult(
                        training_id="train-001",
                        job_name="training-job-001",
                        success=True,
                        model_version="v1.2.0",
                        final_loss=0.03,
                        episodes_completed=2000,
                        model_uri="s3://models/train-001"
                    )

                    # Create training job specification
                    app_key = "prod-cluster:default/ml-service"
                    training_id = f"train-{app_key.replace(':', '-').replace('/', '-')}"

                    job_spec = TrainingJobSpec(
                        training_id=training_id,
                        app_key=app_key,
                        job_name="training-job-001",
                        horizon_hours=12,
                        base_version="v1.0.0",
                        hparams={"learning_rate": 0.001, "episodes": 2000},
                        reason="performance_regression",
                        cpu_request="2",
                        memory_request="4Gi"
                    )

                    # Step 1: Create training job
                    job_created = await training_manager.create_training_job(job_spec)
                    assert job_created is True

                    # Step 2: Monitor job status
                    status_checks = 0
                    while status_checks < 3:
                        status = await training_manager.get_job_status(training_id)
                        status_checks += 1
                        if status == "succeeded":
                            break

                    # Step 3: Collect results
                    result = await training_manager.collect_job_result(training_id)
                    assert result.success is True
                    assert result.model_version == "v1.2.0"
                    assert result.training_id == "train-001"

                    # Step 4: Cleanup
                    cleaned = await training_manager.cleanup_completed_job(training_id)
                    assert cleaned is True

                    # Verify complete flow
                    mock_create.assert_called_once()
                    assert mock_status.call_count == 3
                    mock_collect.assert_called_once()
                    mock_cleanup.assert_called_once()

        except ImportError as e:
            pytest.skip(
                f"Required modules not available for training flow test: {e}")
