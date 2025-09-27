"""
Unit tests for training job manager.
Tests job creation, monitoring, and cleanup logic.
"""
from training.training_job_manager import (
    KubernetesTrainingJobManager,
    TrainingJobSpec,
    TrainingJobResult
)
import pytest
from unittest.mock import Mock, AsyncMock, patch, MagicMock
import asyncio
import sys
import os
from kubernetes.client.rest import ApiException

# Add engine root to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..'))


class TestTrainingJobSpec:
    """Test TrainingJobSpec data class."""

    def test_training_job_spec_creation(self, sample_training_spec):
        """Test creating a TrainingJobSpec instance."""
        spec = sample_training_spec

        assert spec.training_id == "test-training-001"
        assert spec.app_key == "test-cluster:default/test-app"
        assert spec.job_name == "trainer-test-app-001"
        assert spec.horizon_hours == 6
        assert spec.base_version == "baseline-v1"
        assert spec.reason == "test_training"
        assert spec.hparams["learning_rate"] == 0.001
        assert spec.cpu_request == "1"
        assert spec.memory_request == "2Gi"

    def test_training_job_spec_validation(self):
        """Test TrainingJobSpec validation."""
        # Test with minimal required fields
        spec = TrainingJobSpec(
            training_id="minimal-001",
            app_key="test:default/app",
            job_name="minimal-job",
            horizon_hours=1,
            base_version="v1",
            hparams={},
            reason="test",
            output_uri="s3://bucket/models",
            checkpoint_uri="s3://bucket/checkpoints",
            clickhouse_dsn="http://localhost:8123",
            training_data_hours=1
        )

        assert spec.training_id == "minimal-001"
        assert spec.hparams == {}

    def test_training_job_spec_resource_defaults(self):
        """Test TrainingJobSpec resource defaults."""
        spec = TrainingJobSpec(
            training_id="defaults-001",
            app_key="test:default/app",
            job_name="defaults-job",
            horizon_hours=1,
            base_version="v1",
            hparams={},
            reason="test",
            output_uri="s3://bucket/models",
            checkpoint_uri="s3://bucket/checkpoints",
            clickhouse_dsn="http://localhost:8123",
            training_data_hours=1
        )

        # Check defaults are applied correctly
        assert spec.cpu_request == "1"
        assert spec.memory_request == "2Gi"
        assert spec.cpu_limit == "2"
        assert spec.memory_limit == "4Gi"


class TestTrainingJobResult:
    """Test TrainingJobResult data class."""

    def test_training_job_result_creation(self):
        """Test creating a TrainingJobResult instance."""
        result = TrainingJobResult(
            training_id="test-001",
            job_name="test-job-001",
            success=True,
            model_version="v1.2.0",
            final_loss=0.05,
            episodes_completed=1000,
            training_time_seconds=1800,
            model_uri="s3://bucket/models/v1.2.0",
            metrics={
                "avg_reward": 0.95,
                "convergence_episode": 800
            }
        )

        assert result.training_id == "test-001"
        assert result.success is True
        assert result.model_version == "v1.2.0"
        assert result.final_loss == 0.05
        assert result.episodes_completed == 1000
        assert result.training_time_seconds == 1800
        assert result.metrics["avg_reward"] == 0.95

    def test_training_job_result_failure(self):
        """Test TrainingJobResult for failed training."""
        result = TrainingJobResult(
            training_id="failed-001",
            job_name="failed-job-001",
            success=False,
            error_message="Training failed due to insufficient data",
            training_time_seconds=300
        )

        assert result.training_id == "failed-001"
        assert result.success is False
        assert result.error_message == "Training failed due to insufficient data"
        assert result.model_version == ""
        assert result.final_loss == 0.0


class TestKubernetesTrainingJobManager:
    """Test KubernetesTrainingJobManager class."""

    def setup_method(self):
        """Set up test fixtures."""
        self.mock_clickhouse = AsyncMock()
        self.mock_k8s_client = Mock()

        # Mock Kubernetes API clients
        self.mock_batch_v1 = Mock()
        self.mock_core_v1 = Mock()

        self.mock_k8s_client.BatchV1Api.return_value = self.mock_batch_v1
        self.mock_k8s_client.CoreV1Api.return_value = self.mock_core_v1

    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    def test_job_manager_initialization(self, mock_load_config, mock_core_api, mock_batch_api):
        """Test KubernetesTrainingJobManager initialization."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        assert manager.namespace == "test-namespace"
        assert manager.training_image == "futura/trainer:test"
        assert manager.job_ttl_seconds == 1800
        assert manager.clickhouse_client == self.mock_clickhouse

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_create_training_job(self, mock_load_config, mock_core_api, mock_batch_api, sample_training_spec):
        """Test creating a training job."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Mock successful job creation
        self.mock_batch_v1.create_namespaced_job.return_value = Mock(
            metadata=Mock(name=sample_training_spec.job_name)
        )

        result = await manager.create_training_job(sample_training_spec)

        assert result is True
        self.mock_batch_v1.create_namespaced_job.assert_called_once()

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_create_training_job_failure(self, mock_load_config, mock_core_api, mock_batch_api, sample_training_spec):
        """Test handling training job creation failure."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Mock job creation failure
        self.mock_batch_v1.create_namespaced_job.side_effect = ApiException(
            status=500, reason="API Error")

        result = await manager.create_training_job(sample_training_spec)

        assert result is False

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_get_job_status(self, mock_load_config, mock_core_api, mock_batch_api):
        """Test getting job status."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Mock job status response
        mock_job = Mock()
        mock_job.status.conditions = [
            Mock(type="Complete", status="True")
        ]
        mock_job.status.succeeded = 1
        mock_job.status.failed = None

        self.mock_batch_v1.read_namespaced_job.return_value = mock_job

        # Add job to active jobs (required for get_job_status)
        from training.training_job_manager import TrainingJobSpec
        test_spec = TrainingJobSpec(
            training_id="test-training-001",
            app_key="test:default/app",
            job_name="test-job",
            horizon_hours=1,
            base_version="v1",
            hparams={},
            reason="test",
            output_uri="s3://bucket/models",
            checkpoint_uri="s3://bucket/checkpoints",
            clickhouse_dsn="http://localhost:8123",
            training_data_hours=1
        )
        manager.active_jobs["test-training-001"] = test_spec

        status = await manager.get_job_status("test-training-001")

        assert status == "succeeded"
        self.mock_batch_v1.read_namespaced_job.assert_called_once()

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_get_job_status_running(self, mock_load_config, mock_core_api, mock_batch_api):
        """Test getting status of running job."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Add a training job to the manager's active jobs
        mock_spec = Mock()
        mock_spec.job_name = "trainer-test-training-001"
        manager.active_jobs["test-training-001"] = mock_spec

        # Mock running job status
        mock_job = Mock()
        mock_job.status.conditions = []
        mock_job.status.succeeded = None
        mock_job.status.failed = None
        mock_job.status.active = 1

        self.mock_batch_v1.read_namespaced_job.return_value = mock_job

        status = await manager.get_job_status("test-training-001")

        assert status == "running"

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_get_job_status_failed(self, mock_load_config, mock_core_api, mock_batch_api):
        """Test getting status of failed job."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Add a training job to the manager's active jobs
        mock_spec = Mock()
        mock_spec.job_name = "trainer-test-training-001"
        manager.active_jobs["test-training-001"] = mock_spec

        # Mock failed job status
        mock_job = Mock()
        mock_job.status.conditions = [
            Mock(type="Failed", status="True")
        ]
        mock_job.status.succeeded = None
        mock_job.status.failed = 1

        self.mock_batch_v1.read_namespaced_job.return_value = mock_job

        status = await manager.get_job_status("test-training-001")

        assert status == "failed"

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_collect_job_result_success(self, mock_load_config, mock_core_api, mock_batch_api):
        """Test collecting successful job result."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Add a training job to the manager's active jobs
        mock_spec = Mock()
        mock_spec.job_name = "trainer-test-training-001"
        mock_spec.base_version = "v1.2"
        mock_spec.output_uri = "s3://bucket/models"
        manager.active_jobs["test-training-001"] = mock_spec

        # Mock job status to return "succeeded"
        with patch.object(manager, 'get_job_status', return_value="succeeded"):
            # Mock job logs
            with patch.object(manager, '_get_job_logs', return_value="Training completed successfully"):
                # Mock ClickHouse query for training result
                self.mock_clickhouse.fetch_rows.return_value = [
                    {
                        'training_id': 'test-training-001',
                        'success': True,
                        'model_version': 'v1.2.0',
                        'final_loss': 0.05,
                        'episodes_completed': 1000,
                        'training_duration': 1800,
                        'output_uri': 's3://bucket/models/v1.2.0',
                        'metrics': '{"avg_reward": 0.95}'
                    }
                ]

                result = await manager.collect_job_result("test-training-001")

                assert result is not None
                assert result.training_id == "test-training-001"
                assert result.success is True
                assert result.model_version == "v1.2-test-tra"  # Expected format
                assert result.final_loss == 0.0  # From parsed logs, not ClickHouse

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_collect_job_result_not_found(self, mock_load_config, mock_core_api, mock_batch_api):
        """Test collecting result when job result not found."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Mock empty ClickHouse query result
        self.mock_clickhouse.fetch_rows.return_value = []

        result = await manager.collect_job_result("nonexistent-training")

        assert result is None

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_cleanup_completed_job(self, mock_load_config, mock_core_api, mock_batch_api):
        """Test cleaning up completed job."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Add a training job to the manager's active jobs
        mock_spec = Mock()
        mock_spec.job_name = "trainer-test-training-001"
        manager.active_jobs["test-training-001"] = mock_spec

        # Mock successful job deletion
        self.mock_batch_v1.delete_namespaced_job.return_value = Mock()
        self.mock_core_v1.delete_namespaced_config_map.return_value = Mock()

        result = await manager.cleanup_completed_job("test-training-001")

        assert result is True
        self.mock_batch_v1.delete_namespaced_job.assert_called_once()
        self.mock_core_v1.delete_namespaced_config_map.assert_called_once()

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_cleanup_job_failure(self, mock_load_config, mock_core_api, mock_batch_api):
        """Test handling job cleanup failure."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Mock job deletion failure
        self.mock_batch_v1.delete_namespaced_job.side_effect = Exception(
            "Delete failed")

        result = await manager.cleanup_completed_job("test-training-001")

        assert result is False

    @pytest.mark.asyncio
    @patch('kubernetes.client.BatchV1Api')
    @patch('kubernetes.client.CoreV1Api')
    @patch('kubernetes.config.load_incluster_config')
    async def test_list_active_jobs(self, mock_load_config, mock_core_api, mock_batch_api):
        """Test listing active training jobs."""
        mock_batch_api.return_value = self.mock_batch_v1
        mock_core_api.return_value = self.mock_core_v1

        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        # Mock job list response
        mock_job_1 = Mock()
        mock_job_1.metadata.name = "trainer-app1-001"
        mock_job_1.metadata.labels = {"app": "futura-trainer", "training-id": "training-001"}
        mock_job_1.metadata.creation_timestamp = None
        mock_job_1.status.active = 1
        mock_job_1.status.succeeded = None
        mock_job_1.status.failed = None

        mock_job_2 = Mock()
        mock_job_2.metadata.name = "trainer-app2-002"
        mock_job_2.metadata.labels = {"app": "futura-trainer", "training-id": "training-002"}
        mock_job_2.metadata.creation_timestamp = None
        mock_job_2.status.active = None
        mock_job_2.status.succeeded = 1
        mock_job_2.status.failed = None

        mock_job_list = Mock()
        mock_job_list.items = [mock_job_1, mock_job_2]

        self.mock_batch_v1.list_namespaced_job.return_value = mock_job_list

        active_jobs = await manager.list_active_jobs()

        assert len(active_jobs) == 1  # Only active job should be returned
        assert active_jobs[0]["training_id"] == "training-001"
        assert active_jobs[0]["status"] == "running"

    def test_job_spec_to_k8s_manifest(self, sample_training_spec):
        """Test conversion of TrainingJobSpec to Kubernetes Job manifest."""
        manager = KubernetesTrainingJobManager(
            clickhouse_client=self.mock_clickhouse,
            namespace="test-namespace",
            training_image="futura/trainer:test",
            job_ttl_seconds=1800
        )

        job_manifest = manager._build_job_manifest(sample_training_spec)

        # Verify job manifest structure
        assert job_manifest.api_version == "batch/v1"
        assert job_manifest.kind == "Job"
        assert job_manifest.metadata.name == sample_training_spec.job_name
        assert job_manifest.metadata.namespace == "test-namespace"

        # Verify job spec
        assert job_manifest.spec.ttl_seconds_after_finished == 1800
        assert job_manifest.spec.backoff_limit == 2

        # Verify container spec
        container = job_manifest.spec.template.spec.containers[0]
        assert container.name == "rl-trainer"
        assert container.image == "futura/trainer:test"
        assert container.resources.requests["cpu"] == "1"
        assert container.resources.requests["memory"] == "2Gi"

        # Verify environment variables
        env_vars = {env.name: env.value for env in container.env}
        assert env_vars["TRAINING_ID"] == sample_training_spec.training_id
        assert env_vars["APP_KEY"] == sample_training_spec.app_key
        assert env_vars["HORIZON_HOURS"] == str(
            sample_training_spec.horizon_hours)


@pytest.mark.asyncio
class TestTrainingJobManagerIntegration:
    """Integration tests for training job manager."""

    async def test_full_job_lifecycle(self, mock_clickhouse_client, sample_training_spec):
        """Test complete job lifecycle from creation to cleanup."""
        with patch('kubernetes.client'), \
                patch('kubernetes.config.load_incluster_config'):

            manager = KubernetesTrainingJobManager(
                clickhouse_client=mock_clickhouse_client,
                namespace="test-namespace",
                training_image="futura/trainer:test",
                job_ttl_seconds=1800
            )

            # Mock the job creation process
            with patch.object(manager, 'create_training_job', return_value=True) as mock_create, \
                    patch.object(manager, 'get_job_status', side_effect=["running", "succeeded"]) as mock_status, \
                    patch.object(manager, 'collect_job_result') as mock_collect, \
                    patch.object(manager, 'cleanup_completed_job', return_value=True) as mock_cleanup:

                # Mock successful result
                mock_result = TrainingJobResult(
                    training_id=sample_training_spec.training_id,
                    job_name=sample_training_spec.job_name,
                    success=True,
                    model_version="v1.0.1",
                    final_loss=0.04,
                    episodes_completed=1200,
                    training_time_seconds=2100
                )
                mock_collect.return_value = mock_result

                # Simulate full lifecycle
                created = await manager.create_training_job(sample_training_spec)
                assert created is True

                # Monitor job status
                status = await manager.get_job_status(sample_training_spec.training_id)
                assert status == "running"

                # Wait for completion (simulate)
                status = await manager.get_job_status(sample_training_spec.training_id)
                assert status == "succeeded"

                # Collect result
                result = await manager.collect_job_result(sample_training_spec.training_id)
                assert result.success is True
                assert result.model_version == "v1.0.1"

                # Cleanup
                cleaned = await manager.cleanup_completed_job(sample_training_spec.training_id)
                assert cleaned is True

                # Verify all methods were called
                mock_create.assert_called_once()
                assert mock_status.call_count == 2
                mock_collect.assert_called_once()
                mock_cleanup.assert_called_once()
