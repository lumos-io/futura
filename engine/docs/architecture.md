# Futura Engine Architecture

This document provides a comprehensive overview of the Futura Engine's architecture, data flows, and component interactions.

## 🏗️ High-Level Architecture

```mermaid
graph TB
    subgraph "Kubernetes Cluster"
        Op[Futura Operator]
        Pods[Application Pods]
        HPA[HPA Controller]
        VPA[VPA Controller]
    end

    subgraph "Futura Engine"
        RS[RecommendationService<br/>Port 50051]
        RL[RLServer<br/>Port 50051]
        AC[AgentCoordinator<br/>Port 50051]
    end

    subgraph "ML Infrastructure"
        PPO[PPO Models<br/>PyTorch]
        Meta[Meta-PPO Models<br/>PyTorch]
        Train[Training Jobs<br/>Kubernetes]
    end

    subgraph "Data Layer"
        CH[(ClickHouse<br/>Metrics & Models)]
        S3[(S3/PVC<br/>Model Checkpoints)]
    end

    Op --> RS
    RS --> RL
    RL --> AC
    RL --> PPO
    RL --> Meta
    AC --> Train
    Train --> S3
    RL --> CH
    RS --> CH
    Train --> CH

    RS --> HPA
    RS --> VPA
    RS --> Pods
```

## 🎯 Core Services Overview

### RecommendationService (MPA Server)

**Primary Role**: Orchestration and safety enforcement

- **Port**: 50051 (main API endpoint)
- **Purpose**: Receives optimization requests from Kubernetes operator
- **Key Features**:
  - SLO validation and enforcement
  - Safety policy application
  - Intelligent fallback to HPA/VPA algorithms
  - Audit trail generation

### RLServer (Control Plane)

**Primary Role**: ML model serving and lifecycle management

- **Port**: 50051 (can be distributed)
- **Purpose**: PyTorch neural network inference and training orchestration
- **Key Features**:
  - Real-time neural network inference
  - PPO and Meta-PPO model serving
  - CUDA/CPU automatic device detection
  - Model checkpointing and versioning

### AgentCoordinator (Training Orchestration)

**Primary Role**: Training job lifecycle management

- **Port**: 50051 (can be distributed)
- **Purpose**: Manages ephemeral training pods and their coordination
- **Key Features**:
  - Kubernetes Job creation and monitoring
  - Training progress tracking
  - Automatic cleanup and result storage

## 🔄 End-to-End Data Flow

### 1. Application Onboarding Flow

```mermaid
sequenceDiagram
    participant Op as Futura Operator
    participant RS as RecommendationService
    participant RL as RLServer
    participant CH as ClickHouse
    participant PPO as PyTorch Models

    Note over Op: New SLO CRD created
    Op->>RS: SyncServiceLevelObjective(app, slo)
    RS->>CH: Store SLO configuration
    RS->>RL: EnsureModel(app)

    alt No existing model
        RL->>PPO: Bootstrap PPOAgent
        RL->>CH: Store model metadata
        RL->>PPO: Set inference mode
    else Existing model
        RL->>CH: Load model metadata
        RL->>PPO: Load checkpoint
        RL->>PPO: Set inference mode
    end

    RL-->>RS: Model ready (version)
    RS-->>Op: SLO configured successfully

    Note over Op,PPO: Application ready for ML-powered optimization
```

### 2. Real-Time Recommendation Flow

```mermaid
sequenceDiagram
    participant Op as Futura Operator
    participant RS as RecommendationService
    participant RL as RLServer
    participant PPO as PyTorch ActorNetwork
    participant HVA as HPA/VPA Algorithms
    participant CH as ClickHouse

    Note over Op: Resource optimization needed
    Op->>RS: GetRecommendation(app, metrics)
    RS->>CH: Validate SLO constraints
    RS->>RL: GetAppAction(app, features)

    RL->>RL: Extract and normalize features
    RL->>HVA: Check scaling algorithms

    alt HPA/VPA suggests action
        HVA-->>RL: Scaling recommendation
        Note over RL: Use scaling algorithm decision
    else No immediate scaling needed
        RL->>PPO: Neural network inference
        PPO-->>RL: Action probabilities
        Note over RL: Sample action from distribution
    end

    RL->>RL: Convert to Kubernetes changes
    RL-->>RS: ActionPlan + confidence

    RS->>RS: Apply safety policies
    RS->>CH: Store decision audit trail
    RS-->>Op: Recommendation with reasoning

    Note over Op: Execute scaling action
```

### 3. Training Pipeline Flow

```mermaid
sequenceDiagram
    participant RS as RecommendationService
    participant RL as RLServer
    participant AC as AgentCoordinator
    participant K8s as Kubernetes API
    participant Pod as Training Pod
    participant CH as ClickHouse
    participant S3 as S3/PVC Storage

    Note over RS: Performance drift detected
    RS->>RL: TriggerTrain(app, reason, horizon)
    RL->>RL: Generate training spec
    RL->>K8s: Create Kubernetes Job
    K8s->>Pod: Start training container

    Pod->>AC: RegisterAgent(training_id)
    AC->>RL: FetchTrainingSpec(training_id)
    RL-->>AC: Training configuration
    AC-->>Pod: Training spec + data sources

    Pod->>CH: Fetch training data
    Note over Pod: Train PyTorch PPO/Meta-PPO
    Pod->>Pod: Validate model performance
    Pod->>S3: Save model checkpoint
    Pod->>AC: ReportResult(metrics, checkpoint_uri)

    AC->>RL: Training completed
    RL->>CH: Store training results
    RL->>RL: Update model registry
    RL->>K8s: Schedule Job cleanup

    Note over RL: New model ready for inference
```

## 📊 Data Architecture

### ClickHouse Schema Overview

```mermaid
erDiagram
    SLO_CONFIGS {
        string api_key
        string namespace
        string app_name
        float p95_latency_ms
        float error_rate_percent
        float throughput_rps
        timestamp created_at
    }

    CLUSTER_CONFIGS {
        string api_key
        string cloud_provider
        float budget_limit_usd
        string instance_types
        timestamp updated_at
    }

    RECOMMENDATION_DECISIONS {
        string decision_id
        string api_key
        string namespace
        string app_name
        string action_type
        float confidence
        string model_version
        json reasoning
        timestamp created_at
    }

    TRAINING_EVENTS {
        string training_id
        string app_key
        string event_type
        string status
        json metadata
        timestamp created_at
    }

    TRAINING_RESULTS {
        string training_id
        string model_version
        float final_loss
        float avg_reward
        int episodes_trained
        string checkpoint_uri
        timestamp completed_at
    }

    MODEL_REGISTRY {
        string app_key
        string version
        string policy_name
        string checkpoint_uri
        json training_metrics
        boolean is_production
        timestamp created_at
    }

    SLO_CONFIGS ||--o{ RECOMMENDATION_DECISIONS : "constrains"
    CLUSTER_CONFIGS ||--o{ RECOMMENDATION_DECISIONS : "influences"
    TRAINING_EVENTS ||--o| TRAINING_RESULTS : "produces"
    MODEL_REGISTRY ||--o{ RECOMMENDATION_DECISIONS : "serves"
```

### Model Storage Architecture

```mermaid
graph TB
    subgraph "Model Lifecycle"
        Train[Training Job] --> Checkpoint[PyTorch Checkpoint]
        Checkpoint --> S3[S3/PVC Storage]
        S3 --> Registry[Model Registry]
        Registry --> Memory[In-Memory Models]
    end

    subgraph "Storage Locations"
        S3 --> |"Persistent"| Checkpoints["model.pth files"]
        Registry --> |"Metadata"| CH[(ClickHouse)]
        Memory --> |"Inference"| PPO[PPO Agents]
        Memory --> |"Inference"| Meta[Meta-PPO Agents]
    end

    subgraph "Model Types"
        PPO --> |"Standard RL"| Apps1[Regular Apps]
        Meta --> |"Fast Adaptation"| Apps2[New/Variable Apps]
    end
```

## 🚀 Deployment Architectures

### Single Process (Development)

```mermaid
graph TB
    subgraph "Single Process - Port 50051"
        RS[RecommendationService]
        RL[RLServer]
        AC[AgentCoordinator]

        RS --- RL
        RL --- AC
    end

    Operator --> RS
    RL --> ClickHouse
    AC --> K8s[Kubernetes Jobs]
```

**Use Case**: Development, testing, small clusters
**Command**: `uv run main.py --port 50051`

### Distributed (Production)

```mermaid
graph TB
    subgraph "Service Pod 1 - Port 50052"
        RS[RecommendationService]
    end

    subgraph "Service Pod 2 - Port 50051"
        RL[RLServer]
        AC[AgentCoordinator]

        RL --- AC
    end

    Operator --> RS
    RS --> |gRPC| RL
    RL --> ClickHouse
    AC --> K8s[Kubernetes Jobs]
```

**Use Case**: Production, high availability, load distribution
**Commands**:

```bash
# Pod 1
uv run server.py --services recommendation --port 50052 --rl-server-address service2:50051

# Pod 2
uv run server.py --services rl+coordinator --port 50051
```

## 🔧 Component Integration Patterns

### Service Discovery

```mermaid
graph LR
    subgraph "Service Mesh Integration"
        Operator --> |DNS/Service| RS[RecommendationService]
        RS --> |Internal gRPC| RL[RLServer]
        RL --> |Internal gRPC| AC[AgentCoordinator]
    end

    subgraph "Configuration"
        ENV[Environment Variables] --> RS
        ENV --> RL
        ENV --> AC
    end
```

### Error Handling & Resilience

```mermaid
stateDiagram-v2
    [*] --> Healthy

    Healthy --> Degraded: Model loading fails
    Healthy --> Degraded: ClickHouse unavailable

    Degraded --> Fallback: Use HPA/VPA algorithms
    Degraded --> Healthy: Issue resolved

    Fallback --> Healthy: Systems restored
    Fallback --> [*]: Critical failure

    note right of Fallback
        - Use scaling algorithms
        - Heuristic policy
        - Graceful degradation
    end note
```

## 📈 Performance & Scaling

### Inference Performance

```mermaid
graph LR
    subgraph "Inference Path"
        Request --> Features[Feature Extraction<br/>~1ms]
        Features --> GPU[CUDA Inference<br/>~5ms]
        GPU --> Action[Action Conversion<br/>~1ms]
        Action --> Response[~7ms total]
    end

    subgraph "Fallback Path"
        Request2[Request] --> Heuristic[Heuristic Policy<br/>~2ms]
        Heuristic --> Response2[~2ms total]
    end
```

### Horizontal Scaling

```mermaid
graph TB
    subgraph "Load Balancing"
        LB[Load Balancer] --> RS1[RecommendationService 1]
        LB --> RS2[RecommendationService 2]
        LB --> RS3[RecommendationService N]
    end

    subgraph "Shared Backend"
        RS1 --> RL[RLServer Cluster]
        RS2 --> RL
        RS3 --> RL
        RL --> CH[(ClickHouse)]
    end
```

## 🔒 Security Considerations

### Network Security

- **Internal gRPC**: TLS encryption between services
- **ClickHouse**: Authentication and network policies
- **S3/PVC**: IAM roles and encryption at rest

### Model Security

- **Checkpoint Validation**: SHA256 verification
- **Access Control**: RBAC for training jobs
- **Audit Logging**: All model operations logged

### Data Privacy

- **PII Handling**: No personal data in metrics
- **Metric Aggregation**: Only cluster-level analytics
- **Retention Policies**: Automatic data cleanup

## 🎯 Next Steps

- **[Service Documentation](./recommendation-service.md)**: Deep dive into individual services
- **[Operator Integration](./operator-integration.md)**: How external systems interact
- **[PyTorch Models](./pytorch-models.md)**: ML model implementation details
- **[Training Pipeline](./training-pipeline.md)**: End-to-end training workflow
