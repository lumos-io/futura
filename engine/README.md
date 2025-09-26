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

- **Purpose**: ML model serving and lifecycle management with real PyTorch models
- **Functions**:
  - `GetAppAction` - Online inference from PyTorch PPO/Meta-PPO models
  - `EnsureModel` - Bootstrap/load PyTorch models into memory
  - `TriggerTrain` - Kick off Kubernetes training jobs
  - `ListModels` / `GetModelMetadata` - Model management
  - `ReportOutcome` - Accept outcomes for model learning
- **Features**:
  - Real PyTorch neural network inference (ActorNetwork/CriticNetwork)
  - Meta-learning support with RNN trajectory embeddings
  - CUDA/CPU automatic device detection
  - Model checkpointing and hot-reloading

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

### 🧠 **PyTorch RL Models (Production-Ready)**

- **PPO Implementation** (`rl_models/ppo.py`):
  - **ActorNetwork**: Policy network outputting action probabilities via softmax
  - **CriticNetwork**: Value function estimating state values for advantage calculation
  - **PPOAgent**: Complete PPO implementation with GAE, clipped objectives, entropy regularization
  - **Features**: Model saving/loading, inference/training modes, Kubernetes action conversion

- **Meta-Learning PPO** (`rl_models/meta_ppo.py`):
  - **MetaActorNetwork**: Policy with RNN trajectory embeddings for fast adaptation
  - **MetaCriticNetwork**: Value function enhanced with episode buffer context
  - **MetaPPOAgent**: Few-shot learning for new workloads via trajectory encoding
  - **Episode Buffer**: Stores best/latest episodes for meta-learning context

- **Neural Network Components** (`rl_models/rnn.py`, `rl_models/blocks.py`):
  - **RNNEmbedding**: Bidirectional GRU for sequence modeling
  - **AttentionBlock**: Causal attention mechanisms for temporal dependencies
  - **TCBlock**: Temporal convolution with exponential dilation
  - **TrajectoryEncoder**: Standalone trajectory-to-embedding conversion

- **State Space** (`rl_models/state_action_space.py`):
  - System metrics: CPU, memory, disk I/O utilization
  - Application metrics: Request rate, latency, throughput
  - Resource allocation: Current limits and replica counts
  - **10-dimensional normalized feature vectors** for neural network input

- **Action Space** (Multidimensional Scaling):
  - **7 discrete actions**: No action, horizontal ±1, vertical CPU ±256m, vertical memory ±256Mi
  - **ActionType enum**: Type-safe action definitions with Kubernetes conversion
  - **Intelligent sampling**: Action probability distribution from policy networks

- **Reward Function** (`rl_models/reward_functions.py`):
  - **v1 Formula**: `R = α × ResourceUtilization + (1-α) × DataProcessingRate - penalties`
  - **v2 Formula**: `R = ResourceUtilization × DataProcessingRate - penalties`
  - **SLO-Aware**: Enhanced version prioritizing SLO compliance
  - **Oscillation Penalties**: Prevents thrashing behavior (scale up then down)
  - **Performance Penalties**: Penalizes latency increases and processing lag

### 🔄 **From Controller Reference → Production Implementation**

- **PPO Policy Network** → **Real PyTorch ActorNetwork/CriticNetwork with neural inference**
- **Meta-Learning** → **MetaPPOAgent with RNN trajectory embeddings**
- **Environment Interface** → Feature extraction from ClickHouse and Kubernetes APIs
- **Reward Calculation** → Direct implementation of paper's reward functions
- **State Normalization** → 10D feature vectors optimized for neural network training
- **Action History** → Tracking for oscillation detection and reward calculation
- **Training Jobs** → Kubernetes Job orchestration with automatic cleanup

### 🚀 **Production Enhancements**

- **Real Neural Networks** → PyTorch ActorNetwork/CriticNetwork replacing heuristics
- **Meta-Learning Support** → Fast adaptation to new workloads via trajectory embeddings
- **CUDA/CPU Support** → Automatic device detection and model placement
- **gRPC Services** → Production-ready API interfaces following proto definitions
- **Multi-App Support** → Per-application PyTorch model instances and feature extractors
- **Safety Policies** → Resource bounds and constraint enforcement
- **Model Lifecycle** → PyTorch checkpointing, versioning, hot-reloading infrastructure
- **Training Orchestration** → Kubernetes Job-based distributed training with real RL algorithms
- **Audit Trail** → Decision tracking with neural network confidence scores and reasoning
- **ClickHouse Integration** → Metrics storage and model metadata persistence

## API Examples

### Get Recommendation (with PyTorch Neural Network)

```python
import grpc
from proto.gen.engine import engine_pb2_grpc, engine_pb2

# Connect to engine
channel = grpc.insecure_channel('localhost:50051')
client = engine_pb2_grpc.RecommendationServiceStub(channel)

# Request recommendation (neural network will process these features)
request = engine_pb2.RecommendationRequest(
    app=engine_pb2.AppRef(
        api_key="cluster-123",
        namespace="default",
        app_name="web-app",
        kind=engine_pb2.DEPLOYMENT
    ),
    snapshot=engine_pb2.MetricSnapshot(
        values={
            "cpu_utilization": 0.85,         # → Neural network input
            "memory_utilization": 0.72,      # → Neural network input
            "p95_latency_ms": 450.0,          # → Neural network input
            "request_rate": 1200.0            # → Neural network input
        }
    )
)

# PyTorch ActorNetwork processes normalized features and outputs action probabilities
response = client.GetRecommendation(request)
print(f"Decision: {response.decision_id}")
print(f"Action: {response.plan.type}")  # From neural network inference
print(f"Confidence: {response.confidence}")  # Neural network confidence score
print(f"Model: {response.model_version}")  # PyTorch model version used
```

### Trigger PyTorch Training

```python
rl_client = engine_pb2_grpc.RLServerStub(channel)

# This creates a Kubernetes Job that runs real PPO/Meta-PPO training
train_request = engine_pb2.TrainRequest(
    app=engine_pb2.AppRef(
        api_key="cluster-123",
        namespace="default",
        app_name="web-app"
    ),
    reason="performance_drift",
    horizon_hours=48,
    hparams={
        "learning_rate": "0.001",      # PyTorch optimizer learning rate
        "batch_size": "128",           # PPO mini-batch size
        "epochs": "100",               # Training epochs
        "gamma": "0.99",               # Discount factor
        "clip_epsilon": "0.2",         # PPO clipping parameter
        "entropy_coeff": "0.01",       # Entropy regularization
        "use_meta_learning": "true"    # Enable Meta-PPO with trajectory embeddings
    }
)

# Creates a Kubernetes Job in the futura-training namespace
train_response = rl_client.TriggerTrain(train_request)
print(f"Training ID: {train_response.training_id}")
print(f"K8s Job: {train_response.job_name}")

# Training job will:
# 1. Pull data from ClickHouse
# 2. Run PPO/Meta-PPO algorithms with real neural networks
# 3. Save PyTorch model checkpoints
# 4. Store results in ClickHouse
# 5. Auto-cleanup after completion
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
│   ├── rl_server.py    # PyTorch model serving
│   └── agent_coordinator.py
├── rl_models/         # PyTorch RL implementations
│   ├── ppo.py         # Standard PPO with ActorNetwork/CriticNetwork
│   ├── meta_ppo.py    # Meta-learning PPO with RNN embeddings
│   ├── rnn.py         # RNN embeddings and SNAIL architecture
│   ├── blocks.py      # Neural network building blocks
│   ├── state_action_space.py  # State/action definitions
│   └── reward_functions.py    # Reward calculation
├── storage/           # ClickHouse integration
├── scaling/           # HPA/VPA algorithms
├── training/          # Kubernetes Job management
├── deploy/            # K8s manifests
├── controller/        # Reference implementation (deprecated)
├── server.py          # Main server orchestration
├── cli.py            # CLI interface
├── main.py           # Entry point
└── pyproject.toml    # Dependencies (includes PyTorch)
```

### Testing

```bash
# Run tests
uv run pytest

# Test specific service
uv run pytest tests/test_recommendation_service.py
```

## Contributing

1. **✅ COMPLETED**: PyTorch RL models integrated from `controller/` → `rl_models/`
2. **Real Neural Networks**: Engine now uses PyTorch ActorNetwork/CriticNetwork for inference
3. **Meta-Learning**: MetaPPOAgent supports fast adaptation via trajectory embeddings
4. **Production Ready**: CUDA/CPU support, model checkpointing, Kubernetes training jobs
5. Follow the protobuf definitions in `proto/engine/engine.proto`
6. Maintain compatibility with the documented sequence diagrams in `docs/`

### Recent Integration

The engine has been upgraded from heuristic simulation to **real PyTorch neural networks**:

- **Before**: `_simulate_policy_output()` with hardcoded heuristics
- **After**: `_run_pytorch_inference()` with trained ActorNetwork/CriticNetwork
- **Models**: PPOAgent and MetaPPOAgent with full training capabilities
- **Features**: Model loading, CUDA support, confidence scoring, trajectory embeddings
