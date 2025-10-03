# Futura Pipeline Helm Chart

Data ingestion pipeline for Futura - Kafka consumers processing events into ClickHouse.

## Overview

The Pipeline consists of 4 microservices that form a stream processing pipeline:

1. **Collect**: gRPC endpoint receiving events from watchers
2. **Validate**: Kafka consumer validating event structure
3. **Enrich**: Kafka consumer enriching events with additional metadata
4. **Store**: Kafka consumer writing events to ClickHouse

Each service can be scaled independently based on load.

**Collect Service Port:** 50051 (gRPC)

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- Kafka cluster
- ClickHouse database
- Redis (for API key validation)

## Installation

### Basic Installation

```bash
helm install futura-pipeline ./deployments/futura-pipeline \
  --namespace futura-system \
  --create-namespace
```

### With Custom Values

```bash
helm install futura-pipeline ./deployments/futura-pipeline \
  --namespace futura-system \
  --create-namespace \
  --set config.kafka.brokers={kafka-0:9092,kafka-1:9092} \
  --set config.clickhouse.url=http://clickhouse-service:8123
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Pipeline image repository | `davideberdin/futura-pipeline` |
| `image.tag` | Image tag | `latest` |
| `collect.enabled` | Enable collect service | `true` |
| `collect.replicas` | Collect replicas | `1` |
| `validate.enabled` | Enable validate service | `true` |
| `validate.replicas` | Validate replicas | `2` |
| `enrich.enabled` | Enable enrich service | `true` |
| `enrich.replicas` | Enrich replicas | `2` |
| `store.enabled` | Enable store service | `true` |
| `store.replicas` | Store replicas | `2` |
| `config.kafka.brokers` | Kafka broker addresses | `["kafka:9092"]` |
| `config.clickhouse.url` | ClickHouse HTTP URL | `http://clickhouse:8123` |
| `config.redis.servers` | Redis server addresses | `["redis:6379"]` |
| `config.log.level` | Log level | `info` |

### Example Production Values

Create `production-values.yaml`:

```yaml
image:
  repository: myregistry/futura-pipeline
  tag: "v0.2.0"

# Scale collect service for high throughput
collect:
  replicas: 3
  resources:
    limits:
      cpu: 2000m
      memory: 2Gi
    requests:
      cpu: 500m
      memory: 512Mi

# Scale validation for high event rates
validate:
  replicas: 5
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 250m
      memory: 256Mi

# Scale enrichment for complex transformations
enrich:
  replicas: 5
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 250m
      memory: 256Mi

# Scale store for high write throughput
store:
  replicas: 4
  resources:
    limits:
      cpu: 2000m
      memory: 2Gi
    requests:
      cpu: 500m
      memory: 512Mi

config:
  kafka:
    brokers:
      - kafka-0.kafka-headless:9092
      - kafka-1.kafka-headless:9092
      - kafka-2.kafka-headless:9092
    saslEnabled: true
    saslMechanism: SCRAM-SHA-512
    saslUsername: pipeline-user
    existingSecret: kafka-credentials

  clickhouse:
    url: http://clickhouse-cluster:8123
    database: events_prod
    username: pipeline
    existingSecret: clickhouse-credentials

  redis:
    servers:
      - redis-master:6379
    existingSecret: redis-credentials

  log:
    level: info

# Enable autoscaling
autoscaling:
  validate:
    enabled: true
    minReplicas: 3
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70
  enrich:
    enabled: true
    minReplicas: 3
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70
  store:
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
            - futura-pipeline
        topologyKey: kubernetes.io/hostname
```

Install with production values:

```bash
helm install futura-pipeline ./deployments/futura-pipeline \
  -f production-values.yaml \
  --namespace futura-system
```

## Upgrading

```bash
helm upgrade futura-pipeline ./deployments/futura-pipeline \
  --namespace futura-system \
  -f production-values.yaml
```

## Uninstalling

```bash
helm uninstall futura-pipeline --namespace futura-system
```

## Architecture

### Data Flow

```
Watchers (eBPF)
    ↓ (gRPC)
Collect Service (Pipeline)
    ↓ (Kafka: raw-events)
Validate Service
    ↓ (Kafka: validated-events)
Enrich Service
    ↓ (Kafka: enriched-events)
Store Service
    ↓ (HTTP)
ClickHouse
```

### Kafka Topics

The pipeline uses the following Kafka topics:
- `raw-events`: Events from collect service
- `validated-events`: Validated events
- `enriched-events`: Enriched events ready for storage

### Scaling Strategy

Each service scales independently:

- **Collect**: Scale based on watcher count and event rate
- **Validate**: Scale based on Kafka consumer lag
- **Enrich**: Scale based on enrichment complexity
- **Store**: Scale based on ClickHouse write throughput

## Testing the Deployment

After installation, verify the services:

```bash
# Check all pods
kubectl get pods -n futura-system -l app.kubernetes.io/name=futura-pipeline

# Check collect service
kubectl get svc -n futura-system -l app.kubernetes.io/component=collect

# View logs for each service
kubectl logs -n futura-system -l app.kubernetes.io/component=collect -f
kubectl logs -n futura-system -l app.kubernetes.io/component=validate -f
kubectl logs -n futura-system -l app.kubernetes.io/component=enrich -f
kubectl logs -n futura-system -l app.kubernetes.io/component=store -f

# Port forward collect service for testing
kubectl port-forward -n futura-system svc/futura-pipeline-collect 50051:50051
```

## Kafka Configuration

### SASL Authentication

For Kafka clusters with SASL authentication:

```yaml
config:
  kafka:
    brokers:
      - kafka:9092
    saslEnabled: true
    saslMechanism: SCRAM-SHA-512  # or PLAIN
    saslUsername: pipeline-user
    existingSecret: kafka-credentials
    existingSecretUsernameKey: username
    existingSecretPasswordKey: password
```

Create the secret:

```bash
kubectl create secret generic kafka-credentials \
  --from-literal=username=pipeline-user \
  --from-literal=password=<password> \
  -n futura-system
```

### TLS/SSL

For TLS-enabled Kafka (requires code changes in pipeline):

```yaml
config:
  kafka:
    brokers:
      - kafka:9093
    tlsEnabled: true
```

## Troubleshooting

### Pods not starting

Check logs:
```bash
kubectl logs -n futura-system <pod-name>
```

Common issues:
- Kafka connection failure
- ClickHouse connection failure
- Invalid configuration

### Kafka connection errors

Verify Kafka connectivity:
```bash
kubectl exec -n futura-system <pipeline-pod> -- \
  nc -zv <kafka-host> 9092
```

### High consumer lag

Check Kafka consumer group lag:
```bash
kafka-consumer-groups.sh --bootstrap-server kafka:9092 \
  --group pipeline-validate \
  --describe
```

If lag is high:
- Increase replicas for the lagging service
- Enable autoscaling
- Check for slow downstream systems (ClickHouse)

### ClickHouse write errors

Verify ClickHouse connectivity:
```bash
kubectl exec -n futura-system <pipeline-store-pod> -- \
  curl -v http://<clickhouse-host>:8123/ping
```

Check ClickHouse load:
- Query `system.metrics` for write queue size
- Check disk I/O on ClickHouse nodes
- Scale store service replicas

## Performance Tuning

### For High Event Rates

```yaml
collect:
  replicas: 5
  resources:
    limits:
      cpu: 4000m
      memory: 4Gi

validate:
  replicas: 10
  resources:
    limits:
      cpu: 2000m
      memory: 2Gi

enrich:
  replicas: 10
  resources:
    limits:
      cpu: 2000m
      memory: 2Gi

store:
  replicas: 8
  resources:
    limits:
      cpu: 4000m
      memory: 4Gi
```

### Resource Optimization

For cost-sensitive environments:

```yaml
collect:
  replicas: 1
  resources:
    limits:
      cpu: 500m
      memory: 512Mi

validate:
  replicas: 1
enrich:
  replicas: 1
store:
  replicas: 1
```

## Monitoring

Key metrics to monitor:

- **Collect Service**:
  - gRPC request rate
  - Event ingestion rate
  - Kafka produce lag

- **Validate/Enrich Services**:
  - Kafka consumer lag
  - Processing latency
  - Invalid event rate

- **Store Service**:
  - ClickHouse write rate
  - Write latency
  - Failed writes

## Development

For local development with port forwarding:

```bash
# Forward collect service
kubectl port-forward -n futura-system svc/futura-pipeline-collect 50051:50051

# Test with grpcurl
grpcurl -plaintext -d '{"event": {...}}' \
  localhost:50051 collector.v1.CollectorService/CollectEvent
```

## License

Copyright © 2025 Futura Team
