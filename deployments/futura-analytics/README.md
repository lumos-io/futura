# Futura Analytics Helm Chart

Analytics service for Futura - gRPC server for metrics aggregation from ClickHouse.

## Overview

The Analytics service provides a gRPC interface for querying aggregated metrics from ClickHouse. It serves as the metrics backend for the Futura APIs service and frontend.

**Service Port:** 50061 (gRPC)

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- ClickHouse database with `events` database and required tables
- Network access to ClickHouse server

## Installation

### Basic Installation

```bash
helm install futura-analytics ./deployments/futura-analytics \
  --namespace futura-system \
  --create-namespace
```

### With Custom Values

```bash
helm install futura-analytics ./deployments/futura-analytics \
  --namespace futura-system \
  --create-namespace \
  --set clickhouse.servers={clickhouse-service:9000} \
  --set clickhouse.password=<your-password>
```

### Using External Secrets

For production deployments, use Kubernetes secrets for sensitive data:

```bash
# Create secret
kubectl create secret generic clickhouse-credentials \
  --from-literal=clickhouse-password=<password> \
  -n futura-system

# Install with secret reference
helm install futura-analytics ./deployments/futura-analytics \
  --namespace futura-system \
  --set clickhouse.existingSecret=clickhouse-credentials \
  --set clickhouse.existingSecretPasswordKey=clickhouse-password
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `2` |
| `image.repository` | Analytics image repository | `davideberdin/futura-analytics` |
| `image.tag` | Image tag | `latest` |
| `service.grpcPort` | gRPC service port | `50061` |
| `clickhouse.servers` | ClickHouse server addresses | `["clickhouse:9000"]` |
| `clickhouse.database` | ClickHouse database name | `events` |
| `clickhouse.username` | ClickHouse username | `user` |
| `clickhouse.password` | ClickHouse password | `password` |
| `analytics.logLevel` | Log level (debug, info, warn, error) | `info` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `autoscaling.enabled` | Enable HPA | `false` |
| `podDisruptionBudget.enabled` | Enable PDB | `true` |

### Example Custom Values

Create `custom-values.yaml`:

```yaml
replicaCount: 3

image:
  repository: myregistry/futura-analytics
  tag: "v0.2.0"

clickhouse:
  servers:
    - clickhouse-0.clickhouse:9000
    - clickhouse-1.clickhouse:9000
  database: events
  username: analytics_user
  existingSecret: clickhouse-creds
  existingSecretPasswordKey: password

analytics:
  logLevel: info

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 250m
    memory: 256Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 15
  targetCPUUtilizationPercentage: 70

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app.kubernetes.io/name
            operator: In
            values:
            - futura-analytics
        topologyKey: kubernetes.io/hostname
```

Install with custom values:

```bash
helm install futura-analytics ./deployments/futura-analytics \
  -f custom-values.yaml \
  --namespace futura-system
```

## Upgrading

```bash
helm upgrade futura-analytics ./deployments/futura-analytics \
  --namespace futura-system \
  -f custom-values.yaml
```

## Uninstalling

```bash
helm uninstall futura-analytics --namespace futura-system
```

## Health Checks

The chart includes gRPC health probes:

- **Liveness Probe:** Checks if the service is running (initial delay 30s, period 10s)
- **Readiness Probe:** Checks if the service is ready to accept traffic (initial delay 10s, period 5s)

## Testing the Deployment

After installation, verify the service:

```bash
# Check pods
kubectl get pods -n futura-system -l app.kubernetes.io/name=futura-analytics

# Check service
kubectl get svc -n futura-system -l app.kubernetes.io/name=futura-analytics

# View logs
kubectl logs -n futura-system -l app.kubernetes.io/name=futura-analytics -f

# Port forward for local testing
kubectl port-forward -n futura-system svc/futura-analytics 50061:50061

# Test with grpcurl (if installed)
grpcurl -plaintext localhost:50061 list
```

## Architecture

The Analytics service:
- Listens on port 50061 for gRPC requests
- Connects to ClickHouse for metrics queries
- Loads configuration from `/opt/analytics/config.toml`
- Supports graceful shutdown on SIGTERM/SIGINT

## Troubleshooting

### Pods not starting

Check logs for configuration errors:
```bash
kubectl logs -n futura-system <pod-name>
```

Common issues:
- Invalid ClickHouse connection string
- Database/tables not created
- Network policy blocking access to ClickHouse

### Connection refused to ClickHouse

Verify ClickHouse connectivity:
```bash
kubectl exec -n futura-system <analytics-pod> -- nc -zv <clickhouse-host> 9000
```

### High CPU/Memory usage

Enable autoscaling or increase resource limits:
```bash
helm upgrade futura-analytics ./deployments/futura-analytics \
  --set autoscaling.enabled=true \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=1Gi
```

## Development

For local development with port forwarding:

```bash
# Forward analytics service
kubectl port-forward -n futura-system svc/futura-analytics 50061:50061

# Update your local config to use localhost:50061
```

## License

Copyright © 2025 Futura Team
