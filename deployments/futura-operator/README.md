# Futura Operator Helm Chart

Kubernetes operator for intelligent workload optimization with eBPF-based observability.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.0+
- kubectl configured to access your cluster

## Installation

### Install with default values

```bash
helm install futura-operator ./deploy/helm/futura-operator \
  --namespace futura-system \
  --create-namespace
```

### Install with custom values

```bash
helm install futura-operator ./deploy/helm/futura-operator \
  --namespace futura-system \
  --create-namespace \
  --set image.repository=your-registry/futura-operator \
  --set image.tag=v0.1.0 \
  --set operator.analytics.endpoint=analytics.example.com:50051
```

### Install from values file

```bash
helm install futura-operator ./deploy/helm/futura-operator \
  --namespace futura-system \
  --create-namespace \
  --values custom-values.yaml
```

## Configuration

The following table lists the configurable parameters of the Futura Operator chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Operator image repository | `futura-operator` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag | `latest` |
| `replicaCount` | Number of operator replicas | `1` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |
| `operator.logLevel` | Log level (debug, info, warn, error) | `info` |
| `operator.ebpf.enabled` | Enable eBPF metrics collection | `true` |
| `operator.ebpf.metricsInterval` | Metrics collection interval | `30s` |
| `operator.analytics.enabled` | Enable analytics backend | `true` |
| `operator.analytics.endpoint` | Analytics service endpoint | `analytics-service.futura-system.svc.cluster.local:50051` |
| `operator.clusterOptimization.syncPeriodSeconds` | Optimization sync period | `60` |
| `operator.clusterOptimization.clusterScalingMode` | Scaling mode (auto/recommend) | `recommend` |
| `namespace` | Namespace for operator | `futura-system` |

## Usage

### Create a ClusterOptimizationConfig

```bash
kubectl apply -f examples/clusteroptimizationconfig.yaml
```

### Create a ServiceLevelObjective

```bash
# Deploy a sample application
kubectl apply -f examples/slo-nginx-deployment.yaml

# Or use StatefulSet example
kubectl apply -f examples/slo-statefulset.yaml
```

### Verify Installation

```bash
# Check operator pod status
kubectl get pods -n futura-system

# Check CRDs
kubectl get crds | grep futura

# View operator logs
kubectl logs -n futura-system -l control-plane=controller-manager -c manager -f
```

### Monitor SLO Status

```bash
# List all SLOs
kubectl get servicelevelobjectives --all-namespaces

# Get detailed SLO status
kubectl describe servicelevelobjective nginx-app-slo -n default

# Check autoscaling decisions
kubectl get hpa -n default
kubectl get vpa -n default
```

## Uninstallation

```bash
# Delete all SLOs first
kubectl delete servicelevelobjectives --all --all-namespaces

# Delete ClusterOptimizationConfigs
kubectl delete clusteroptimizationconfigs --all --all-namespaces

# Uninstall the operator
helm uninstall futura-operator -n futura-system

# Delete CRDs (optional)
kubectl delete crd servicelevelobjectives.futura.opisvigilant.com
kubectl delete crd clusteroptimizationconfigs.futura.opisvigilant.com
```

## Troubleshooting

### Operator pod not starting

Check events and logs:
```bash
kubectl get events -n futura-system --sort-by='.lastTimestamp'
kubectl logs -n futura-system -l control-plane=controller-manager -c manager
```

### SLO not being reconciled

Check operator logs and SLO status:
```bash
kubectl logs -n futura-system -l control-plane=controller-manager -c manager | grep -i error
kubectl describe servicelevelobjective <slo-name> -n <namespace>
```

### eBPF metrics not collected

Ensure eBPF is enabled and check daemonset:
```bash
kubectl get ds -n futura-system
kubectl logs -n futura-system -l app=ebpf-collector
```

## Development

### Build and install from local image

```bash
# Build operator image
cd operator
make docker-build IMG=futura-operator:dev

# Load image into Kind cluster
kind load docker-image futura-operator:dev

# Install with local image
helm install futura-operator ./deploy/helm/futura-operator \
  --namespace futura-system \
  --create-namespace \
  --set image.tag=dev \
  --set image.pullPolicy=Never
```

## License

Apache License 2.0
