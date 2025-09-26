# Futura Engine

The ML-powered Kubernetes optimization engine that implements reinforcement learning-based autoscaling decisions.

## Architecture

The engine consists of three main gRPC services:

### 🎯 RecommendationService (MPA Server)

- **Purpose**: Orchestrates optimization decisions and safety policies
- **Called by**: Kubernetes Operator
- **Functions**:
  - `SyncClusterOptimizationConfig` - Configure cluster settings (cloud provider, budget, instance types)
  - `SyncServiceLevelObjective` - Define SLOs (P95 latency, error rates, throughput)
  - `GetRecommendation` - Get optimization recommendations for workloads
  - `ReportExecutionOutcome` - Receive feedback for learning

### 🧠 RLServer (Control Plane)

- **Purpose**: ML model serving and lifecycle management
- **Functions**:
  - `GetAction` - Online inference from RL models
  - `EnsureModel` - Bootstrap/load models into memory
  - `TriggerTrain` - Kick off training jobs
  - `ListModels` / `GetModelMetadata` - Model management
  - `ReportOutcome` - Accept outcomes for model learning

### 🔧 AgentCoordinator (Training Orchestration)

- **Purpose**: Manages ephemeral training job lifecycle
- **Functions**:
  - `RegisterAgent` - Register training pods
  - `FetchTrainingSpec` - Get training configuration
  - `ReportProgress` / `ReportResult` - Training lifecycle management
  - `Heartbeat` - Agent liveness tracking
  - `CancelTraining` - Training job cancellation

## Quick Start

### Development Setup

```bash
# Install dependencies
uv install

# Run all services (default)
uv run main.py

# Run specific services
uv run server.py --services recommendation+rl --port 50051
uv run server.py --services coordinator --port 50052
```

### Configuration

The engine supports configuration via environment variables:

```bash
# Server configuration
FUTURA_PORT=50051
FUTURA_MAX_WORKERS=50

# RL Server configuration
FUTURA_MODEL_STORE_URI=s3://futura-models
FUTURA_CLICKHOUSE_DSN=clickhouse://localhost:9000/default

# Agent Coordinator configuration
FUTURA_MAX_TRAINING_JOB_AGE_HOURS=24
FUTURA_CLEANUP_INTERVAL_HOURS=1

# Recommendation Service configuration
FUTURA_DEFAULT_CPU_REQUEST_MCPU=1000
FUTURA_DEFAULT_MEMORY_REQUEST_MIB=512
```

### Service Configurations

#### All-in-One (Default)

```bash
uv run main.py --port 50051
```

Runs all three services in a single process.

#### Distributed Setup

```bash
# Terminal 1: RL Server + Agent Coordinator
uv run server.py --services rl+coordinator --port 50051

# Terminal 2: Recommendation Service (connects to external RL Server)
uv run server.py --services recommendation --port 50052 --rl-server-address localhost:50051
```

## Integration with Controller Reference

The engine adapts the **PPO-based multidimensional autoscaling approach** from the USENIX ATC'23 research paper implementation in the `controller/` folder:

### 🧠 **RL Algorithm Implementation**

- **State Space** (`rl_models/state_action_space.py`):

  - System metrics: CPU, memory, disk I/O utilization
  - Application metrics: Request rate, latency, throughput
  - Resource allocation: Current limits and replica counts
  - Feature normalization and extraction from Prometheus data

- **Action Space** (Multidimensional Scaling):

  - **Vertical Scaling**: CPU/memory limit adjustments (256m/256Mi steps)
  - **Horizontal Scaling**: Replica count changes (+1/-1)
  - **No Action**: Stability preference with intelligent action selection

- **Reward Function** (`rl_models/reward_functions.py`):
  - **v1 Formula**: `R = α × ResourceUtilization + (1-α) × DataProcessingRate - penalties`
  - **v2 Formula**: `R = ResourceUtilization × DataProcessingRate - penalties`
  - **SLO-Aware**: Enhanced version prioritizing SLO compliance
  - **Oscillation Penalties**: Prevents thrashing behavior (scale up then down)
  - **Performance Penalties**: Penalizes latency increases and processing lag

### 🔄 **From Controller Reference**

- **PPO Policy Network** → Simulated with intelligent heuristics and action probabilities
- **Environment Interface** → Feature extraction from Kubernetes and Prometheus
- **Reward Calculation** → Direct implementation of paper's reward functions
- **State Normalization** → Bounded feature vectors for stable training
- **Action History** → Tracking for oscillation detection and reward calculation

### 🚀 **Production Enhancements**

- **gRPC Services** → Production-ready API interfaces following proto definitions
- **Multi-App Support** → Per-application feature extractors and model instances
- **Safety Policies** → Resource bounds and constraint enforcement
- **Model Lifecycle** → Versioning, checkpointing, hot-reloading infrastructure
- **Training Orchestration** → Kubernetes Job-based distributed training
- **Audit Trail** → Decision tracking with confidence scores and reasoning

## API Examples

### Get Recommendation

```python
import grpc
from proto.gen.engine import engine_pb2_grpc, engine_pb2

# Connect to engine
channel = grpc.insecure_channel('localhost:50051')
client = engine_pb2_grpc.RecommendationServiceStub(channel)

# Request recommendation
request = engine_pb2.RecommendationRequest(
    app=engine_pb2.AppRef(
        api_key="cluster-123",
        namespace="default",
        app_name="web-app",
        kind=engine_pb2.DEPLOYMENT
    ),
    snapshot=engine_pb2.MetricSnapshot(
        values={
            "cpu_utilization": 0.85,
            "memory_utilization": 0.72,
            "p95_latency_ms": 450.0,
            "request_rate": 1200.0
        }
    )
)

response = client.GetRecommendation(request)
print(f"Decision: {response.decision_id}")
print(f"Action: {response.plan.type}")
print(f"Confidence: {response.confidence}")
```

### Trigger Training

```python
rl_client = engine_pb2_grpc.RLServerStub(channel)

train_request = engine_pb2.TrainRequest(
    app=engine_pb2.AppRef(
        api_key="cluster-123",
        namespace="default",
        app_name="web-app"
    ),
    reason="drift",
    horizon_hours=48,
    hparams={
        "learning_rate": "0.001",
        "batch_size": "128",
        "epochs": "100"
    }
)

train_response = rl_client.TriggerTrain(train_request)
print(f"Training ID: {train_response.training_id}")
```

## Deployment

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: futura-engine
spec:
  replicas: 1
  selector:
    matchLabels:
      app: futura-engine
  template:
    metadata:
      labels:
        app: futura-engine
    spec:
      containers:
        - name: engine
          image: futura/engine:latest
          ports:
            - containerPort: 50051
          env:
            - name: FUTURA_MODEL_STORE_URI
              value: "s3://futura-models"
            - name: FUTURA_CLICKHOUSE_DSN
              value: "clickhouse://clickhouse:9000/futura"
```

### Docker

```bash
docker build -t futura/engine .
docker run -p 50051:50051 futura/engine
```

## Development

### Project Structure

```
engine/
├── services/           # gRPC service implementations
│   ├── recommendation_service.py
│   ├── rl_server.py
│   └── agent_coordinator.py
├── controller/         # Reference RL implementation
├── server.py          # Main server orchestration
├── config.py          # Configuration management
├── main.py           # Entry point
└── pyproject.toml    # Dependencies
```

### Testing

```bash
# Run tests
uv run pytest

# Test specific service
uv run pytest tests/test_recommendation_service.py
```

## Contributing

1. The `controller/` folder contains reference RL implementation - do not modify
2. Adapt RL algorithms from controller into the engine services
3. Follow the protobuf definitions in `proto/engine/engine.proto`
4. Maintain compatibility with the documented sequence diagrams in `docs/`
