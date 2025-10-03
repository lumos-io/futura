# Futura APIs Helm Chart

REST API backend for Futura with embedded frontend.

## Overview

The APIs service provides the main REST API backend for Futura, including:
- User authentication (OAuth with Google/GitHub)
- Cluster management
- SSE streaming endpoints for real-time updates
- Embedded React frontend serving

**Service Port:** 8080 (HTTP)

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- PostgreSQL database
- Redis server
- Analytics service (gRPC)
- (Optional) Unleash feature flag service

## Installation

### Basic Installation

```bash
helm install futura-apis ./deployments/futura-apis \
  --namespace futura-system \
  --create-namespace
```

### With Custom Values

```bash
helm install futura-apis ./deployments/futura-apis \
  --namespace futura-system \
  --create-namespace \
  --set database.host=postgres-service \
  --set database.password=<your-password> \
  --set redis.servers={redis-service:6379}
```

### Production Deployment with Secrets

```bash
# Create secrets for sensitive data
kubectl create secret generic futura-apis-db \
  --from-literal=postgres-password=<db-password> \
  -n futura-system

kubectl create secret generic futura-apis-secrets \
  --from-literal=jwt-secret=<random-64-char-string> \
  --from-literal=cookie-store-secret=<random-32-char-string> \
  --from-literal=csrf-secret=<random-32-char-string> \
  -n futura-system

kubectl create secret generic futura-apis-oauth \
  --from-literal=google-client-id=<google-client-id> \
  --from-literal=google-client-secret=<google-client-secret> \
  --from-literal=github-client-id=<github-client-id> \
  --from-literal=github-client-secret=<github-client-secret> \
  -n futura-system

# Install with secret references
helm install futura-apis ./deployments/futura-apis \
  --namespace futura-system \
  --set database.existingSecret=futura-apis-db \
  --set secrets.existingSecret=futura-apis-secrets \
  --set oauth.google.existingSecret=futura-apis-oauth \
  --set oauth.github.existingSecret=futura-apis-oauth \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=futura.example.com
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `2` |
| `image.repository` | APIs image repository | `davideberdin/futura-apis` |
| `image.tag` | Image tag | `latest` |
| `service.port` | HTTP service port | `8080` |
| `environment` | Environment (development, production) | `production` |
| `analytics.endpoint` | Analytics gRPC endpoint | `futura-analytics:50061` |
| `database.host` | PostgreSQL host | `postgres` |
| `database.port` | PostgreSQL port | `5432` |
| `database.name` | Database name | `futura` |
| `redis.servers` | Redis server addresses | `["redis:6379"]` |
| `oauth.google.clientId` | Google OAuth client ID | `""` |
| `oauth.github.clientId` | GitHub OAuth client ID | `""` |
| `ingress.enabled` | Enable ingress | `false` |
| `autoscaling.enabled` | Enable HPA | `false` |

### Example Production Values

Create `production-values.yaml`:

```yaml
replicaCount: 3

image:
  repository: myregistry/futura-apis
  tag: "v0.2.0"

environment: production

analytics:
  endpoint: futura-analytics:50061

database:
  host: postgres-primary.database.svc.cluster.local
  port: 5432
  user: futura_app
  name: futura_prod
  sslmode: require
  existingSecret: postgres-credentials
  existingSecretPasswordKey: password

redis:
  servers:
    - redis-master:6379
  namespace: api_keys
  existingSecret: redis-credentials
  existingSecretPasswordKey: password

oauth:
  google:
    clientId: ""  # Set via secret
    callbackUrl: https://futura.example.com/auth/google/callback
    existingSecret: oauth-secrets
  github:
    clientId: ""  # Set via secret
    callbackUrl: https://futura.example.com/auth/github/callback
    existingSecret: oauth-secrets

secrets:
  existingSecret: app-secrets

frontend:
  url: https://futura.example.com/

unleash:
  enabled: true
  url: http://unleash:4242/api/
  existingSecret: unleash-credentials
  existingSecretTokenKey: api-token

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: futura.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: futura-tls
      hosts:
        - futura.example.com

resources:
  limits:
    cpu: 2000m
    memory: 2Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
  targetCPUUtilizationPercentage: 70

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchExpressions:
        - key: app.kubernetes.io/name
          operator: In
          values:
          - futura-apis
      topologyKey: kubernetes.io/hostname
```

Install with production values:

```bash
helm install futura-apis ./deployments/futura-apis \
  -f production-values.yaml \
  --namespace futura-system
```

## Upgrading

```bash
helm upgrade futura-apis ./deployments/futura-apis \
  --namespace futura-system \
  -f production-values.yaml
```

## Uninstalling

```bash
helm uninstall futura-apis --namespace futura-system
```

## Health Checks

The chart includes HTTP health probes:

- **Liveness Probe:** `GET /healthz` (initial delay 30s, period 10s)
- **Readiness Probe:** `GET /readyz` (initial delay 10s, period 5s)

## OAuth Configuration

### Google OAuth Setup

1. Create OAuth credentials at [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Set authorized redirect URI: `https://your-domain.com/auth/google/callback`
3. Store credentials in secret:
```bash
kubectl create secret generic oauth-secrets \
  --from-literal=google-client-id=<client-id> \
  --from-literal=google-client-secret=<client-secret> \
  -n futura-system
```

### GitHub OAuth Setup

1. Create OAuth app at [GitHub Developer Settings](https://github.com/settings/developers)
2. Set callback URL: `https://your-domain.com/auth/github/callback`
3. Store credentials in secret:
```bash
kubectl create secret generic oauth-secrets \
  --from-literal=github-client-id=<client-id> \
  --from-literal=github-client-secret=<client-secret> \
  -n futura-system
```

## Testing the Deployment

After installation, verify the service:

```bash
# Check pods
kubectl get pods -n futura-system -l app.kubernetes.io/name=futura-apis

# Check service
kubectl get svc -n futura-system -l app.kubernetes.io/name=futura-apis

# View logs
kubectl logs -n futura-system -l app.kubernetes.io/name=futura-apis -f

# Port forward for local testing
kubectl port-forward -n futura-system svc/futura-apis 8080:8080

# Test API
curl http://localhost:8080/healthz
```

## Architecture

The APIs service:
- Serves embedded React frontend from `/public` directory
- Listens on port 8080 for HTTP requests
- Connects to Analytics service via gRPC
- Uses PostgreSQL for user/cluster data
- Uses Redis for API key storage
- Supports SSE for real-time updates
- Integrates with Unleash for feature flags

## Troubleshooting

### Pods not starting

Check logs:
```bash
kubectl logs -n futura-system <pod-name>
```

Common issues:
- Database connection failure
- Missing secrets
- Analytics service not available
- Invalid configuration

### Database connection errors

Verify PostgreSQL connectivity:
```bash
kubectl exec -n futura-system <apis-pod> -- nc -zv <postgres-host> 5432
```

### OAuth not working

Check callback URLs match your ingress configuration and OAuth provider settings.

### SSE endpoints timing out

Ensure ingress has proper timeout settings:
```yaml
ingress:
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "300"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "300"
```

## Development

For local development with port forwarding:

```bash
# Forward APIs service
kubectl port-forward -n futura-system svc/futura-apis 8080:8080

# Access at http://localhost:8080
```

## License

Copyright © 2025 Futura Team
