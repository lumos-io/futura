# Futura Engine Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the Futura ML-powered Kubernetes optimization engine.

## 🏗️ Architecture Overview

The Futura Engine consists of three main services:

- **Recommendation Service**: Main optimization recommendations and SLO management
- **RL Server**: ML-powered decision making and training job orchestration
- **Agent Coordinator**: Training job coordination and management

## 📁 Directory Structure

```tree
deploy/
├── base/                           # Base Kubernetes manifests
│   ├── namespace.yaml              # Namespaces (futura-engine, futura-training)
│   ├── configmap.yaml              # Configuration for all services
│   ├── secret.yaml                 # Secrets (ClickHouse, S3 credentials)
│   ├── rbac.yaml                   # Service accounts and permissions
│   ├── recommendation-service.yaml # Recommendation service deployment
│   ├── rl-server.yaml             # RL server deployment
│   ├── agent-coordinator.yaml     # Agent coordinator deployment
│   ├── all-services.yaml          # All-in-one deployment option
│   └── kustomization.yaml         # Kustomize base configuration
└── overlays/                      # Environment-specific configurations
    ├── development/                # Development environment
    ├── staging/                    # Staging environment
    └── production/                 # Production environment
```

## 🚀 Quick Start

### Prerequisites

1. **Kubernetes Cluster** (v1.25+)
2. **kubectl** configured to access your cluster
3. **Kustomize** (included in kubectl v1.14+)
4. **ClickHouse** database (see [ClickHouse Setup](#clickhouse-setup))
5. **Container Registry** access for `futura/engine` image

### Option 1: All-in-One Deployment (Recommended for testing)

```bash
# Deploy all services in a single pod
kubectl apply -k deploy/overlays/development

# Or use base all-services deployment
kubectl apply -f deploy/base/namespace.yaml
kubectl apply -f deploy/base/configmap.yaml
kubectl apply -f deploy/base/secret.yaml
kubectl apply -f deploy/base/rbac.yaml
kubectl apply -f deploy/base/all-services.yaml
```

### Option 2: Individual Services (Recommended for production)

```bash
# Deploy individual services with development configuration
kubectl apply -k deploy/overlays/development

# Or deploy base individual services
kubectl apply -k deploy/base
```

### Option 3: Production Deployment

```bash
# Deploy with production configuration
kubectl apply -k deploy/overlays/production
```

## 🔧 Configuration

### Environment Variables

Key configuration options available in ConfigMaps:

| Variable                  | Description                            | Default                    |
| ------------------------- | -------------------------------------- | -------------------------- |
| `CLICKHOUSE_URL`          | ClickHouse server URL                  | `http://clickhouse:8123`   |
| `CLICKHOUSE_ENGINE_DB`    | Engine database name                   | `engine`                   |
| `CLICKHOUSE_ANALYTICS_DB` | Analytics database name                | `analytics`                |
| `TRAINING_NAMESPACE`      | Kubernetes namespace for training jobs | `futura-training`          |
| `TRAINING_IMAGE`          | Container image for training jobs      | `futura/rl-trainer:latest` |
| `MODEL_STORAGE_URI`       | S3 URI for model storage               | `s3://futura-models`       |
| `LOG_LEVEL`               | Logging level                          | `INFO`                     |

### Secrets

Required secrets for production deployment:

```bash
# ClickHouse credentials
kubectl create secret generic futura-engine-secrets \
  --from-literal=CLICKHOUSE_USER=futura_user \
  --from-literal=CLICKHOUSE_PASSWORD=your_password \
  -n futura-engine

# S3 credentials for model storage
kubectl create secret generic futura-engine-secrets \
  --from-literal=AWS_ACCESS_KEY_ID=your_access_key \
  --from-literal=AWS_SECRET_ACCESS_KEY=your_secret_key \
  --from-literal=AWS_REGION=us-west-2 \
  -n futura-engine
```

## 🗄️ ClickHouse Setup

### Option 1: Deploy ClickHouse in Kubernetes

```bash
# Basic ClickHouse deployment (not production-ready)
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clickhouse
  namespace: futura-engine
spec:
  replicas: 1
  selector:
    matchLabels:
      app: clickhouse
  template:
    metadata:
      labels:
        app: clickhouse
    spec:
      containers:
      - name: clickhouse
        image: clickhouse/clickhouse-server:latest
        ports:
        - containerPort: 8123
        - containerPort: 9000
        env:
        - name: CLICKHOUSE_DB
          value: "engine"
        - name: CLICKHOUSE_USER
          value: "futura_user"
        - name: CLICKHOUSE_PASSWORD
          value: "your_password"
---
apiVersion: v1
kind: Service
metadata:
  name: clickhouse
  namespace: futura-engine
spec:
  selector:
    app: clickhouse
  ports:
  - name: http
    port: 8123
    targetPort: 8123
  - name: native
    port: 9000
    targetPort: 9000
EOF
```

### Option 2: External ClickHouse

Update the `CLICKHOUSE_URL` in the ConfigMap to point to your external ClickHouse instance:

```yaml
data:
  CLICKHOUSE_URL: "http://your-clickhouse-server:8123"
```

## 🎯 Service Access

### Internal Access (within cluster)

```bash
# Recommendation Service
grpc://futura-recommendation-service.futura-engine.svc.cluster.local:8080

# RL Server
grpc://futura-rl-server.futura-engine.svc.cluster.local:8081

# Agent Coordinator
grpc://futura-agent-coordinator.futura-engine.svc.cluster.local:8082

# All-in-one service
grpc://futura-engine-all.futura-engine.svc.cluster.local:8080
```

### External Access

For external access, use port-forwarding or LoadBalancer:

```bash
# Port forward to local machine
kubectl port-forward -n futura-engine svc/futura-engine-all 8080:8080

# Access via localhost:8080
futura-engine health --endpoint localhost:8080
```

## 🔍 Health Checks and Monitoring

### Health Check

```bash
# Check if services are running
kubectl get pods -n futura-engine

# Check service health
kubectl exec -n futura-engine deployment/futura-engine-all -- \
  futura-engine health --endpoint localhost:8080
```

### Logs

```bash
# View logs from all services
kubectl logs -n futura-engine deployment/futura-engine-all -f

# View logs from specific service
kubectl logs -n futura-engine deployment/futura-recommendation-service -f
```

### Metrics

Services expose Prometheus metrics on their respective ports:

```bash
# Port forward for metrics
kubectl port-forward -n futura-engine svc/futura-engine-all 8080:8080

# Access metrics
curl http://localhost:8080/metrics
```

## 🔧 CLI Usage Examples

Once deployed, you can interact with the services:

```bash
# Test recommendation service
grpcurl -plaintext localhost:8080 futura.RecommendationService/GetRecommendation

# Test RL server
grpcurl -plaintext localhost:8081 futura.RLServer/GetAction

# Test agent coordinator
grpcurl -plaintext localhost:8082 futura.AgentCoordinator/RegisterAgent
```

## 🎛️ Scaling and Performance

### Horizontal Pod Autoscaling

Production overlay includes HPA for the recommendation service:

```bash
# Check HPA status
kubectl get hpa -n futura-engine

# View HPA details
kubectl describe hpa futura-recommendation-service-hpa -n futura-engine
```

### Resource Allocation

| Service           | CPU Request | Memory Request | CPU Limit | Memory Limit |
| ----------------- | ----------- | -------------- | --------- | ------------ |
| Recommendation    | 200m        | 256Mi          | 500m      | 512Mi        |
| RL Server         | 500m        | 1Gi            | 2         | 4Gi          |
| Agent Coordinator | 100m        | 128Mi          | 300m      | 256Mi        |
| All-in-One        | 1           | 2Gi            | 4         | 8Gi          |

## 🔒 Security

### RBAC Permissions

The engine requires specific permissions for:

- **Training Jobs**: Create/manage Kubernetes Jobs in `futura-training` namespace
- **Scaling**: Read/update Deployments and HPA resources
- **Monitoring**: Access to pod metrics and logs

### Network Security

- All services use ClusterIP by default (internal access only)
- LoadBalancer service available for external access if needed
- Use NetworkPolicies to restrict traffic between namespaces

## 🐛 Troubleshooting

### Common Issues

1. **Image Pull Errors**

   ```bash
   # Check image availability
   kubectl describe pod -n futura-engine <pod-name>
   ```

2. **ClickHouse Connection Issues**

   ```bash
   # Test ClickHouse connectivity
   kubectl exec -n futura-engine deployment/futura-engine-all -- \
     curl -v http://clickhouse:8123/ping
   ```

3. **Training Jobs Not Starting**

   ```bash
   # Check training namespace and RBAC
   kubectl get jobs -n futura-training
   kubectl auth can-i create jobs --as=system:serviceaccount:futura-engine:futura-engine -n futura-training
   ```

4. **Resource Issues**

   ```bash
   # Check resource usage
   kubectl top pods -n futura-engine
   kubectl describe nodes
   ```

### Debug Mode

Enable debug logging:

```bash
# Update ConfigMap
kubectl patch configmap futura-engine-config -n futura-engine \
  --patch '{"data":{"LOG_LEVEL":"DEBUG"}}'

# Restart deployments
kubectl rollout restart deployment -n futura-engine
```

## 🔄 Updates and Rollbacks

### Rolling Updates

```bash
# Update image tag in overlay
kubectl patch deployment futura-engine-all -n futura-engine \
  --patch '{"spec":{"template":{"spec":{"containers":[{"name":"futura-engine","image":"futura/engine:v0.2.0"}]}}}}'

# Check rollout status
kubectl rollout status deployment/futura-engine-all -n futura-engine
```

### Rollbacks

```bash
# View rollout history
kubectl rollout history deployment/futura-engine-all -n futura-engine

# Rollback to previous version
kubectl rollout undo deployment/futura-engine-all -n futura-engine
```

## 📊 Performance Tuning

### Production Recommendations

1. **Resource Allocation**: Adjust CPU/memory based on workload
2. **Replica Count**: Scale recommendation service based on request volume
3. **ClickHouse Optimization**: Use appropriate ClickHouse cluster configuration
4. **Model Storage**: Use high-performance S3 or persistent volumes
5. **Node Affinity**: Place RL server on GPU nodes if available

### Monitoring

Consider deploying monitoring stack:

- **Prometheus**: Metrics collection
- **Grafana**: Visualization
- **Jaeger**: Distributed tracing
- **ELK Stack**: Log aggregation

## 📚 Next Steps

1. **Build and Push Images**: Create `futura/engine` and `futura/rl-trainer` images
2. **Set up CI/CD**: Automate deployments with GitOps
3. **Configure Monitoring**: Set up comprehensive observability
4. **Security Hardening**: Implement NetworkPolicies and security scanning
5. **Backup Strategy**: Configure ClickHouse and model storage backups
