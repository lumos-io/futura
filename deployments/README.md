# Futura Deployments

This directory contains Helm charts for deploying all Futura services to Kubernetes.

## Available Charts

### 1. futura-operator

Kubernetes operator managing SLOs and cluster optimization.

**Location:** `deployments/futura-operator/`

**Key Features:**

- Custom Resource Definitions (ServiceLevelObjective, ClusterOptimizationConfig)
- HPA and VPA integration
- RBAC with kube-rbac-proxy
- Metrics endpoint

**Installation:**

```bash
helm install futura-operator ./deployments/futura-operator \
  --namespace futura-system \
  --create-namespace
```

### 2. futura-apis

REST API backend with embedded React frontend.

**Location:** `deployments/futura-apis/`

**Key Features:**

- HTTP/REST API (port 8080)
- OAuth integration (Google, GitHub)
- Ingress support
- SSE endpoints for real-time updates
- Embedded frontend serving

**Installation:**

```bash
helm install futura-apis ./deployments/futura-apis \
  --namespace futura-system \
  --create-namespace \
  --set database.host=postgres \
  --set database.password=<password> \
  --set redis.servers={redis:6379}
```

### 3. futura-engine

Optimization engine with RL-based recommendations.

**Location:** `deployments/futura-engine/`

**Key Features:**

- gRPC service (port 8080)
- Recommendation Service
- RL Server
- Agent Coordinator
- Two deployment modes: "all" or "split"
- Model cache persistence
- Training job RBAC

**Installation:**

```bash
helm install futura-engine ./deployments/futura-engine \
  --namespace futura-system \
  --create-namespace \
  --set config.clickhouse.url=http://clickhouse:8123
```

### 4. futura-pipeline

Data ingestion pipeline (Kafka → ClickHouse).

**Location:** `deployments/futura-pipeline/`

**Key Features:**

- 4 microservices: collect, validate, enrich, store
- Independent scaling per service
- Kafka consumer integration
- ClickHouse writer
- HPA support per service

**Installation:**

```bash
helm install futura-pipeline ./deployments/futura-pipeline \
  --namespace futura-system \
  --create-namespace \
  --set config.kafka.brokers={kafka:9092} \
  --set config.clickhouse.url=http://clickhouse:8123
```

## Installation Order

For a complete Futura deployment, install in this order:

```bash
# 1. Create namespace
kubectl create namespace futura-system

# 2. Install operator (manages SLOs and workload optimization)
helm install futura-operator ./deployments/futura-operator \
  --namespace futura-system

# 3. Install APIs (frontend + backend)
helm install futura-apis ./deployments/futura-apis \
  --namespace futura-system \
  --set database.host=postgres \
  --set redis.servers={redis:6379}

# 4. Install engine (optimization recommendations)
helm install futura-engine ./deployments/futura-engine \
  --namespace futura-system \
  --set config.clickhouse.url=http://clickhouse:8123

# 5. Install pipeline (data ingestion)
helm install futura-pipeline ./deployments/futura-pipeline \
  --namespace futura-system \
  --set config.kafka.brokers={kafka:9092} \
  --set config.clickhouse.url=http://clickhouse:8123
```

## Quick Start

For local testing with Kind:

```bash
# Start Kind cluster
kind create cluster --name futura-test

# Install all services with default values
for chart in operator apis engine pipeline; do
  helm install futura-$chart ./deployments/futura-$chart \
    --namespace futura-system \
    --create-namespace \
    --wait
done
```

## Service Dependencies

```
┌──────────────────────────────────────────┐
│              Frontend (Browser)           │
└────────────────┬─────────────────────────┘
                 │ HTTP/SSE
                 ↓
         ┌───────────────┐
         │  futura-apis  │ (REST API + Frontend)
         └───────┬───────┘
                 │ gRPC
       ┌─────────┴
       ↓
┌──────────────┐
│ futura-      │
│ engine       │
└──────┬───────┘
       │ SQL
       ↓
┌──────────────────────────────────┐
│         ClickHouse               │
└──────────────────────────────────┘
       ↑
       │ Writes
┌──────┴───────┐
│ futura-      │
│ pipeline     │
└──────┬───────┘
       │ Kafka
       ↑
┌──────┴───────┐
│ Watchers     │ (eBPF DaemonSet)
│ (eBPF)       │
└──────────────┘
```

## External Dependencies

All charts require these external services:

### Required

- **PostgreSQL**: User/cluster data (APIs)
- **ClickHouse**: Metrics storage (Engine, Pipeline)
- **Redis**: API key storage (APIs, Pipeline)
- **Kafka**: Event streaming (Pipeline)

### Optional

- **Unleash**: Feature flags (APIs)

## Configuration Management

### Using Values Files

Create environment-specific values:

```bash
# Production
helm install futura-apis ./deployments/futura-apis \
  -f environments/production/apis-values.yaml

# Staging
helm install futura-apis ./deployments/futura-apis \
  -f environments/staging/apis-values.yaml

# Development
helm install futura-apis ./deployments/futura-apis \
  -f environments/development/apis-values.yaml
```

### Using Secrets

All charts support Kubernetes secrets for sensitive data:

```bash
# Create secrets
kubectl create secret generic postgres-credentials \
  --from-literal=postgres-password=<password> \
  -n futura-system

kubectl create secret generic clickhouse-credentials \
  --from-literal=clickhouse-password=<password> \
  -n futura-system

```

## Upgrading

Upgrade all services:

```bash
for chart in operator apis engine pipeline; do
  helm upgrade futura-$chart ./deployments/futura-$chart \
    --namespace futura-system
done
```

Upgrade single service:

```bash
helm upgrade futura-apis ./deployments/futura-apis \
  --namespace futura-system \
  -f custom-values.yaml
```

## Uninstalling

Remove all services:

```bash
for chart in pipeline engine apis operator; do
  helm uninstall futura-$chart --namespace futura-system
done

# Remove namespace
kubectl delete namespace futura-system
```

## Monitoring and Observability

All services expose:

- **Health endpoints**: Liveness and readiness probes
- **Prometheus metrics**: Annotated pods for scraping
- **Structured logging**: JSON logs with configurable levels

### Prometheus Integration

All pods include annotations:

```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "<port>"
prometheus.io/path: "/metrics"
```

### Grafana Dashboards

Recommended dashboards:

- Service overview (request rates, latencies, errors)
- Resource usage (CPU, memory, disk)
- Kafka consumer lag (pipeline)
- ClickHouse write performance

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n futura-system
```

### View Logs

```bash
# All pods
kubectl logs -n futura-system -l app.kubernetes.io/name=futura-apis -f

# Specific pod
kubectl logs -n futura-system <pod-name> -f
```

### Common Issues

1. **ImagePullBackOff**: Check image repository and credentials
2. **CrashLoopBackOff**: Check logs for startup errors
3. **Connection refused**: Verify service names and ports
4. **Secret not found**: Ensure secrets are created in correct namespace

## Development

For local chart development:

```bash
# Lint charts
helm lint ./deployments/futura-apis

# Template and preview
helm template futura-apis ./deployments/futura-apis \
  --namespace futura-system \
  -f test-values.yaml

# Dry-run install
helm install futura-apis ./deployments/futura-apis \
  --namespace futura-system \
  --dry-run --debug
```

## Contributing

When adding new services:

1. Create chart directory: `deployments/futura-<service>/`
2. Add standard files: `Chart.yaml`, `values.yaml`, `README.md`
3. Create templates in `templates/` directory
4. Include: ServiceAccount, Deployment, Service, HPA, PDB
5. Document configuration in README
6. Update this README with service information

## License

Copyright © 2025 Futura Team
