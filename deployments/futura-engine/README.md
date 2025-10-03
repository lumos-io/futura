# Futura Engine Helm Chart

Optimization engine for Futura - RL-based recommendation and scaling services.

## Overview

The Engine service provides intelligent optimization recommendations using reinforcement learning. It includes:
- **Recommendation Service**: Provides workload optimization recommendations
- **RL Server**: Manages RL models and inference
- **Agent Coordinator**: Coordinates training jobs

The chart supports two deployment modes:
- **all**: Single deployment running all services (recommended for small clusters)
- **split**: Separate deployments for each service (recommended for production)

**Service Port:** 8080 (gRPC)

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- ClickHouse database with `engine` and `analytics` databases
- (Optional) S3-compatible storage for model checkpoints

## Installation

### Basic Installation (All Services in One Pod)

```bash
helm install futura-engine ./deployments/futura-engine \
  --namespace futura-system \
  --create-namespace
```

### With Custom Values

```bash
helm install futura-engine ./deployments/futura-engine \
  --namespace futura-system \
  --create-namespace \
  --set config.clickhouse.url=http://clickhouse-service:8123 \
  --set config.training.modelStorageUri=s3://my-bucket/models
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `deploymentMode` | Deployment mode: "all" or "split" | `all` |
| `image.repository` | Engine image repository | `davideberdin/futura-engine` |
| `image.tag` | Image tag | `latest` |
| `service.grpcPort` | gRPC service port | `8080` |
| `config.server.logLevel` | Log level (DEBUG, INFO, WARN, ERROR) | `INFO` |
| `config.clickhouse.url` | ClickHouse HTTP URL | `http://clickhouse:8123` |
| `config.training.namespace` | Namespace for training jobs | `futura-training` |
| `config.training.modelStorageUri` | Model storage location | `s3://futura-models` |
| `persistence.modelCache.enabled` | Enable persistent model cache | `true` |
| `persistence.modelCache.size` | Model cache size | `10Gi` |
| `resources.limits.cpu` | CPU limit (all mode) | `4000m` |
| `resources.limits.memory` | Memory limit (all mode) | `8Gi` |
| `externalService.enabled` | Enable external LoadBalancer | `false` |
| `rbac.create` | Create RBAC for training jobs | `true` |

### Example Production Values

Create `production-values.yaml`:

```yaml
deploymentMode: all

image:
  repository: myregistry/futura-engine
  tag: "v0.2.0"

config:
  server:
    maxWorkers: 100
    logLevel: INFO

  clickhouse:
    url: http://clickhouse-cluster:8123
    engineDb: engine_prod
    analyticsDb: analytics_prod

  training:
    namespace: futura-training-prod
    image: myregistry/futura-trainer:v0.2.0
    modelStorageUri: s3://prod-bucket/futura-models
    defaultTimeoutHours: 24

  rl:
    defaultHorizonHours: 72
    modelCleanupHours: 336  # 14 days
    enableDriftDetection: true
    driftCheckIntervalHours: 12

resources:
  limits:
    cpu: 8000m
    memory: 16Gi
  requests:
    cpu: 2000m
    memory: 4Gi

persistence:
  modelCache:
    enabled: true
    size: 50Gi
    storageClass: fast-ssd

externalService:
  enabled: true
  type: LoadBalancer
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"

nodeSelector:
  workload-type: ml

tolerations:
- key: "ml-workload"
  operator: "Equal"
  value: "true"
  effect: "NoSchedule"
```

Install with production values:

```bash
helm install futura-engine ./deployments/futura-engine \
  -f production-values.yaml \
  --namespace futura-system
```

### Split Deployment Mode

For production environments with high load, use split mode:

```yaml
deploymentMode: split

recommendationService:
  replicas: 3
  resources:
    limits:
      cpu: 2000m
      memory: 4Gi
    requests:
      cpu: 500m
      memory: 1Gi

rlServer:
  replicas: 2
  resources:
    limits:
      cpu: 8000m
      memory: 16Gi
    requests:
      cpu: 2000m
      memory: 4Gi

agentCoordinator:
  replicas: 1
  resources:
    limits:
      cpu: 1000m
      memory: 2Gi
    requests:
      cpu: 250m
      memory: 512Mi

autoscaling:
  enabled: true
  targetCPUUtilizationPercentage: 70
```

## Upgrading

```bash
helm upgrade futura-engine ./deployments/futura-engine \
  --namespace futura-system \
  -f production-values.yaml
```

## Uninstalling

```bash
helm uninstall futura-engine --namespace futura-system
```

## Health Checks

The chart includes TCP health probes:

- **Liveness Probe:** TCP check on port 8080 (initial delay 60s, period 30s)
- **Readiness Probe:** TCP check on port 8080 (initial delay 30s, period 10s)

## RBAC and Permissions

The engine requires permissions to create and manage training jobs:

- Create Kubernetes Jobs in the training namespace
- Read Pod logs for training monitoring
- List and watch Jobs and Pods

These permissions are automatically created when `rbac.create: true` (default).

## Model Storage

The engine supports multiple storage backends for RL models:

### S3-Compatible Storage

```yaml
config:
  training:
    modelStorageUri: s3://my-bucket/models
```

Ensure pods have AWS credentials via:
- IAM roles (EKS IRSA)
- Environment variables
- Mounted credentials file

### PVC Storage

```yaml
config:
  training:
    modelStorageUri: pvc://model-storage

persistence:
  modelCache:
    enabled: true
    size: 100Gi
    storageClass: fast-ssd
```

## Training Jobs

The engine spawns Kubernetes Jobs for RL training:

```yaml
config:
  training:
    namespace: futura-training  # Jobs created in this namespace
    image: futura/rl-trainer:latest
    defaultTimeoutHours: 12
```

Training jobs are automatically:
- Created when new models need training
- Cleaned up after completion (based on `max_training_job_age_hours`)
- Monitored for heartbeats and progress

## Testing the Deployment

After installation, verify the service:

```bash
# Check pods
kubectl get pods -n futura-system -l app.kubernetes.io/name=futura-engine

# Check service
kubectl get svc -n futura-system -l app.kubernetes.io/name=futura-engine

# View logs
kubectl logs -n futura-system -l app.kubernetes.io/name=futura-engine -f

# Port forward for local testing
kubectl port-forward -n futura-system svc/futura-engine 8080:8080

# Test with grpcurl (if installed)
grpcurl -plaintext localhost:8080 list
```

## Architecture

### All Services Mode (Default)

Single deployment running:
- Recommendation Service (handles operator requests)
- RL Server (model inference)
- Agent Coordinator (training job management)

**Pros:**
- Simple deployment
- Lower resource overhead
- Suitable for small/medium clusters

**Cons:**
- Single point of failure
- Cannot scale services independently

### Split Mode (Production)

Separate deployments for each service:
- Multiple Recommendation Service replicas (stateless, highly available)
- Dedicated RL Server (stateful, GPU-optimized)
- Single Agent Coordinator (manages training state)

**Pros:**
- Independent scaling
- High availability
- Better resource utilization

**Cons:**
- More complex
- Higher resource overhead

## Troubleshooting

### Pods not starting

Check logs:
```bash
kubectl logs -n futura-system <pod-name>
```

Common issues:
- ClickHouse connection failure
- Invalid model storage URI
- Missing RBAC permissions

### ClickHouse connection errors

Verify ClickHouse connectivity:
```bash
kubectl exec -n futura-system <engine-pod> -- \
  curl -v http://<clickhouse-host>:8123/ping
```

### Training jobs not starting

Check RBAC permissions:
```bash
kubectl auth can-i create jobs \
  --as=system:serviceaccount:futura-system:futura-engine \
  -n futura-training
```

### Model storage access errors

For S3:
```bash
kubectl exec -n futura-system <engine-pod> -- \
  aws s3 ls s3://your-bucket/
```

For PVC:
```bash
kubectl exec -n futura-system <engine-pod> -- ls -la /app/models
```

## Performance Tuning

### For High Throughput

```yaml
config:
  server:
    maxWorkers: 200

resources:
  limits:
    cpu: 16000m
    memory: 32Gi
  requests:
    cpu: 8000m
    memory: 16Gi
```

### For GPU Workloads (RL Server)

```yaml
deploymentMode: split

rlServer:
  resources:
    limits:
      nvidia.com/gpu: 2
      cpu: 8000m
      memory: 32Gi

nodeSelector:
  nvidia.com/gpu: "true"

tolerations:
- key: "nvidia.com/gpu"
  operator: "Exists"
  effect: "NoSchedule"
```

## Monitoring

The engine exposes Prometheus metrics at `/metrics` on port 8080:

```yaml
podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```

Key metrics:
- `engine_recommendation_requests_total`
- `engine_rl_inference_duration_seconds`
- `engine_training_jobs_active`
- `engine_model_cache_size_bytes`

## Development

For local development with port forwarding:

```bash
# Forward engine service
kubectl port-forward -n futura-system svc/futura-engine 8080:8080

# Test recommendation endpoint
grpcurl -plaintext -d '{"app": {"namespace": "default", "app_name": "nginx"}}' \
  localhost:8080 engine.v1.RecommendationService/GetAppRecommendation
```

## License

Copyright © 2025 Futura Team
