"""
Global pytest configuration and fixtures for Futura Engine tests.
"""
import asyncio
import pytest
import os
import sys
from unittest.mock import Mock, AsyncMock, MagicMock
from typing import Dict, Any, Generator

# Add engine root to Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

# Import test dependencies
try:
    from unittest.mock import patch
    import grpc
    from grpc import aio as grpc_aio
except ImportError:
    pytest.skip("Missing test dependencies", allow_module_level=True)


@pytest.fixture(scope="session")
def event_loop():
    """Create an instance of the default event loop for the test session."""
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()


@pytest.fixture
def mock_clickhouse_client():
    """Mock ClickHouse client for testing."""
    client = AsyncMock()
    client.execute = AsyncMock()
    client.fetch_rows = AsyncMock()
    client.insert_rows = AsyncMock()
    client.close = AsyncMock()
    return client


@pytest.fixture
def mock_kubernetes_client():
    """Mock Kubernetes client for testing."""
    client = Mock()

    # Mock CoreV1Api
    core_v1 = Mock()
    core_v1.create_namespaced_config_map = AsyncMock()
    core_v1.delete_namespaced_config_map = AsyncMock()
    core_v1.list_namespaced_pod = AsyncMock()

    # Mock BatchV1Api
    batch_v1 = Mock()
    batch_v1.create_namespaced_job = AsyncMock()
    batch_v1.delete_namespaced_job = AsyncMock()
    batch_v1.read_namespaced_job_status = AsyncMock()
    batch_v1.list_namespaced_job = AsyncMock()

    # Mock AppsV1Api
    apps_v1 = Mock()
    apps_v1.read_namespaced_deployment = AsyncMock()
    apps_v1.patch_namespaced_deployment = AsyncMock()

    # Mock AutoscalingV1Api
    autoscaling_v1 = Mock()
    autoscaling_v1.read_namespaced_horizontal_pod_autoscaler = AsyncMock()
    autoscaling_v1.patch_namespaced_horizontal_pod_autoscaler = AsyncMock()

    client.CoreV1Api.return_value = core_v1
    client.BatchV1Api.return_value = batch_v1
    client.AppsV1Api.return_value = apps_v1
    client.AutoscalingV1Api.return_value = autoscaling_v1

    return client


@pytest.fixture
def mock_torch():
    """Mock PyTorch for testing without GPU dependencies."""
    with patch.dict('sys.modules', {
        'torch': Mock(),
        'torch.nn': Mock(),
        'torch.nn.functional': Mock(),
        'torch.optim': Mock(),
    }):
        yield


@pytest.fixture
def sample_resource_state():
    """Sample ResourceState for testing."""
    from scaling.scaling_algorithms import ResourceState
    return ResourceState(
        num_replicas=3,
        cpu_limit=1000,
        memory_limit=512,
        cpu_util=0.75,
        memory_util=0.75,
        request_rate=150.0,
        p95_latency_ms=200.0,
        processing_rate=150.0,
        ingestion_rate=150.0,
        error_rate=0.005
    )


@pytest.fixture
def sample_slo_targets():
    """Sample SLO targets for testing."""
    return {
        'target_p95_latency_ms': 400.0,
        'target_error_rate': 0.01,
        'target_throughput_rps': 150.0
    }


@pytest.fixture
def mock_grpc_server():
    """Mock gRPC server for testing."""
    server = Mock()
    server.add_insecure_port = Mock(return_value=50051)
    server.start = AsyncMock()
    server.stop = AsyncMock()
    server.wait_for_termination = AsyncMock()
    return server


@pytest.fixture
def mock_grpc_channel():
    """Mock gRPC channel for testing."""
    channel = Mock()
    channel.close = AsyncMock()
    return channel


@pytest.fixture
def sample_training_spec():
    """Sample TrainingJobSpec for testing."""
    from training.training_job_manager import TrainingJobSpec
    return TrainingJobSpec(
        training_id="test-training-001",
        app_key="test-cluster:default/test-app",
        job_name="trainer-test-app-001",
        horizon_hours=6,
        base_version="baseline-v1",
        hparams={
            "learning_rate": 0.001,
            "batch_size": 32,
            "episodes": 100
        },
        reason="test_training",
        output_uri="s3://test-bucket/models/test-app/001",
        checkpoint_uri="s3://test-bucket/checkpoints/test-app/001",
        clickhouse_dsn="http://localhost:8123/engine",
        training_data_hours=24,
        cpu_request="1",
        memory_request="2Gi",
        cpu_limit="2",
        memory_limit="4Gi"
    )


@pytest.fixture
def mock_environment_variables():
    """Mock environment variables for testing."""
    env_vars = {
        'CLICKHOUSE_HOST': 'localhost',
        'CLICKHOUSE_PORT': '8123',
        'CLICKHOUSE_DATABASE': 'test_engine',
        'KUBERNETES_NAMESPACE': 'test-namespace',
        'MODEL_STORAGE_PATH': '/tmp/test-models',
        'TRAINING_IMAGE': 'futura/rl-trainer:test'
    }

    with patch.dict(os.environ, env_vars):
        yield env_vars


@pytest.fixture
def temp_model_dir(tmp_path):
    """Temporary directory for model storage during tests."""
    model_dir = tmp_path / "models"
    model_dir.mkdir()
    return str(model_dir)


@pytest.fixture
def mock_model_checkpoint():
    """Mock model checkpoint data for testing."""
    return {
        'model_state_dict': {'layer1.weight': 'fake_tensor_data'},
        'optimizer_state_dict': {'param_groups': []},
        'epoch': 100,
        'loss': 0.05,
        'version': 'v1.0.0',
        'metadata': {
            'training_duration': 1800,
            'episodes_completed': 1000,
            'final_reward': 0.95
        }
    }


@pytest.fixture(autouse=True)
def mock_external_dependencies():
    """Automatically mock external dependencies for all tests."""
    with patch('kubernetes.config.load_incluster_config'), \
         patch('kubernetes.config.load_kube_config'), \
         patch('grpc.aio.server') as mock_grpc_server, \
         patch('grpc.aio.insecure_channel') as mock_grpc_channel:

        # Configure mocks
        mock_grpc_server.return_value = Mock()
        mock_grpc_channel.return_value = Mock()

        yield


@pytest.fixture
def disable_gpu():
    """Disable GPU detection for testing."""
    with patch('torch.cuda.is_available', return_value=False):
        yield


# Test data generators
def generate_metrics_data(num_points: int = 100) -> list:
    """Generate sample metrics data for testing."""
    import time
    import random

    base_time = int(time.time()) - (num_points * 60)  # Start 100 minutes ago

    data = []
    for i in range(num_points):
        data.append({
            'timestamp': base_time + (i * 60),
            'app_key': f'test-cluster:default/app-{random.randint(1, 3)}',
            'cpu_utilization': random.uniform(0.3, 0.9),
            'memory_utilization': random.uniform(0.2, 0.8),
            'request_rate': random.uniform(50, 200),
            'p95_latency_ms': random.uniform(100, 500),
            'error_rate': random.uniform(0, 0.02),
            'num_replicas': random.randint(2, 8)
        })

    return data


# Custom pytest markers
def pytest_configure(config):
    """Configure custom pytest markers."""
    config.addinivalue_line("markers", "unit: Unit tests")
    config.addinivalue_line("markers", "integration: Integration tests")
    config.addinivalue_line("markers", "e2e: End-to-end tests")
    config.addinivalue_line("markers", "slow: Slow running tests")
    config.addinivalue_line("markers", "requires_k8s: Tests requiring Kubernetes")
    config.addinivalue_line("markers", "requires_clickhouse: Tests requiring ClickHouse")


# Test collection customization
def pytest_collection_modifyitems(config, items):
    """Modify test collection to add markers based on test names."""
    for item in items:
        # Add unit marker to unit tests
        if "unit" in item.nodeid:
            item.add_marker(pytest.mark.unit)

        # Add integration marker to integration tests
        if "integration" in item.nodeid:
            item.add_marker(pytest.mark.integration)

        # Add e2e marker to end-to-end tests
        if "e2e" in item.nodeid:
            item.add_marker(pytest.mark.e2e)

        # Add slow marker to potentially slow tests
        if any(keyword in item.nodeid.lower() for keyword in ['training', 'model', 'grpc']):
            item.add_marker(pytest.mark.slow)