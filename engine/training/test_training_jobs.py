#!/usr/bin/env python3
"""
Test script for Kubernetes training job creation and monitoring.

This demonstrates how the training job system works with ClickHouse
result storage and automatic cleanup.
"""

from training_job_manager import KubernetesTrainingJobManager, TrainingJobSpec
import asyncio
import logging
import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def test_training_job_creation():
    """Test creating a training job and monitoring its lifecycle."""
    print("🧪 Testing Kubernetes Training Job Management")

    try:
        # Initialize ClickHouse client (mock for testing)
        clickhouse_client = None  # Would be initialized in production

        # Create training job manager
        job_manager = KubernetesTrainingJobManager(
            clickhouse_client=clickhouse_client,
            namespace="futura-training-test",
            training_image="futura/rl-trainer:test",
            job_ttl_seconds=1800  # 30 minutes
        )

        # Test training job specification
        training_spec = TrainingJobSpec(
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
            reason="initial_training",
            output_uri="s3://futura-test/models/test-app/001",
            checkpoint_uri="s3://futura-test/checkpoints/test-app/001",
            clickhouse_dsn="http://localhost:8123/engine",
            training_data_hours=24,
            cpu_request="1",
            memory_request="2Gi",
            cpu_limit="2",
            memory_limit="4Gi"
        )

        print(f"📋 Training Spec Created:")
        print(f"  Training ID: {training_spec.training_id}")
        print(f"  App Key: {training_spec.app_key}")
        print(f"  Job Name: {training_spec.job_name}")
        print(f"  Horizon: {training_spec.horizon_hours} hours")
        print(
            f"  Resources: {training_spec.cpu_request}/{training_spec.cpu_limit} CPU, {training_spec.memory_request}/{training_spec.memory_limit} memory")
        print(f"  Output URI: {training_spec.output_uri}")

        # Test job creation (will fail if not in Kubernetes cluster)
        print("\n🚀 Attempting to create training job...")

        job_created = await job_manager.create_training_job(training_spec)

        if job_created:
            print("✅ Training job created successfully")

            # Test job monitoring
            print("\n👀 Testing job monitoring...")

            for i in range(3):  # Check status 3 times
                status = await job_manager.get_job_status(training_spec.training_id)
                print(f"  Status check {i+1}: {status}")

                if status in ["succeeded", "failed"]:
                    # Collect result
                    result = await job_manager.collect_job_result(training_spec.training_id)
                    if result:
                        print(f"📊 Job Result:")
                        print(f"  Success: {result.success}")
                        print(f"  Model Version: {result.model_version}")
                        print(f"  Final Loss: {result.final_loss}")
                        print(f"  Episodes: {result.episodes_completed}")
                        if result.error_message:
                            print(f"  Error: {result.error_message}")
                    break

                await asyncio.sleep(5)  # Wait 5 seconds between checks

            # Test cleanup
            print("\n🧹 Testing job cleanup...")
            cleaned_up = await job_manager.cleanup_completed_job(training_spec.training_id)
            if cleaned_up:
                print("✅ Job cleaned up successfully")
            else:
                print("⚠️ Job cleanup failed")

        else:
            print(
                "❌ Training job creation failed (expected if not in Kubernetes cluster)")

    except Exception as e:
        print(f"❌ Test failed with error: {str(e)}")


async def demo_training_workflow():
    """Demonstrate the full training workflow."""
    print("\n" + "="*60)
    print("🎯 DEMO: Full Training Workflow")
    print("="*60)

    print("""
This demonstrates how the Futura engine handles training:

1. 📝 RL Server receives TriggerTrain request
2. 🔧 Creates TrainingJobSpec with proper resource allocation
3. 🚀 Kubernetes Job is created with training container
4. 👀 Background monitoring checks job status periodically
5. 📊 When complete, results are collected and stored in ClickHouse
6. 🧹 Job resources are cleaned up after result storage
7. 🎯 Model version is updated for future inference

Key Benefits:
✅ Scalable training on Kubernetes cluster
✅ Automatic resource management and cleanup
✅ Persistent result storage in ClickHouse
✅ Integration with RL decision making
✅ Handles both initial training and retraining
""")

    # Show example training container environment
    print("🐳 Training Container Environment:")
    env_vars = {
        "TRAINING_ID": "train-001",
        "APP_KEY": "cluster1:production/web-app",
        "HORIZON_HOURS": "12",
        "BASE_VERSION": "v1.2.3",
        "HPARAMS": '{"lr": 0.001, "batch_size": 32}',
        "REASON": "performance_drift",
        "OUTPUT_URI": "s3://futura-models/web-app/train-001",
        "CLICKHOUSE_DSN": "http://clickhouse:8123/engine",
        "TRAINING_DATA_HOURS": "24"
    }

    for key, value in env_vars.items():
        print(f"  {key}={value}")

    print("\n📊 Expected ClickHouse Storage:")
    print("  ├── training_events (job lifecycle)")
    print("  ├── training_results (model metrics)")
    print("  ├── recommendation_decisions (training triggers)")
    print("  └── execution_outcomes (performance feedback)")


if __name__ == "__main__":
    print("🚀 Futura Training Job System Test\n")

    # Run the tests
    asyncio.run(test_training_job_creation())

    # Show the workflow demo
    asyncio.run(demo_training_workflow())

    print("\n✅ Training job system test completed!")
    print("\nNext Steps:")
    print("1. Build and push the futura/rl-trainer container image")
    print("2. Create the futura-training namespace in Kubernetes")
    print("3. Set up ClickHouse with proper migration tables")
    print("4. Configure S3/storage for model artifacts")
    print("5. Test with real training workloads")
