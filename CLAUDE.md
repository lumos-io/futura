# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Futura** is a Kubernetes optimization platform with eBPF-based observability. It's a monorepo containing multiple services that work together to provide intelligent workload scaling, resource optimization, and SLO management.

### Architecture Components

The system follows a **distributed microservices architecture** with the following key components:

1. **Frontend** (React/TypeScript) - User interface with real-time SSE streaming
2. **APIs** (Go/Gin) - REST API backend serving frontend and managing cluster metadata
3. **Analytics** (Go/gRPC) - Analytics service that queries ClickHouse for metrics aggregation
4. **Watcher** (Go + eBPF) - Kubernetes event watcher deployed as DaemonSet to collect cluster state
5. **Operator** (Go/Kubebuilder) - Kubernetes operator managing SLOs and cluster optimization
6. **Engine** (Python) - Optimization engine for recommendations and autoscaling decisions
7. **Pipeline** (Go/Kafka) - Data ingestion pipeline processing Kafka streams into ClickHouse

### Data Flow

```
Kubernetes Cluster
    ↓ (events/metrics)
Watcher (eBPF + K8s API)
    ↓ (Kafka)
Pipeline
    ↓ (writes)
ClickHouse
    ↑ (gRPC queries)
Analytics Service
    ↑ (gRPC)
APIs Backend
    ↑ (SSE/REST)
Frontend
```

### Key Technologies

- **Go 1.24**: APIs, Analytics, Watcher, Pipeline, Operator
- **Python 3.12+**: Engine (optimization algorithms)
- **React 18 + TypeScript**: Frontend with shadcn/ui
- **eBPF**: Low-level kernel tracing in Watcher
- **ClickHouse**: Time-series metrics storage
- **Kafka**: Event streaming
- **PostgreSQL**: Application data (users, clusters, configs)
- **Redis**: API key storage and caching
- **Protocol Buffers**: Cross-service communication

## Development Setup

### Nix Development Environment

This project uses Nix for reproducible development environments.

**Prerequisites:**

```bash
# Install Nix
curl -fsSL https://install.determinate.systems/nix | sh -s -- install --determinate
```

**Setup:**

1. Copy `.env.tmp` to `.env.local` and fill in secrets
2. Enter Nix shell: `nix develop`
   - To skip Kind setup: `SKIP_KIND=false nix develop`

The Nix shell automatically:

- Sets up Go, Node.js, Bun, protobuf tools
- Runs Docker login and tool setup scripts
- Creates Kind cluster (unless `SKIP_KIND` is set)
- Configures Redis with API keys

### Docker Compose Services

```bash
# Start all services
docker compose up -d

# Start with dev tools (RedisInsight, Kafka UI)
docker compose --profile dev-tools up -d
```

**Services:**

- PostgreSQL (5432): Application database
- Redis (6379): API key store
- ClickHouse (8123 HTTP, 9000 native): Metrics database
- Kafka (9092 internal, 9094 host): Event streaming
- Unleash (4242): Feature flags
- RedisInsight (5540): Redis GUI [dev-tools profile]
- Kafka UI (9080): Kafka GUI [dev-tools profile]

**Network:** All services connect to external `kind` network for local K8s integration.

## Common Commands

### Protocol Buffers

Proto files are organized by target:

- `proto/` - Go services (Analytics, APIs)
- `proto/backend/` - TypeScript (Frontend)
- `proto/engine/` - Python (Engine)

```bash
# Generate all proto files
make proto-files

# Individual targets
make proto-go      # Go + gRPC
make proto-ts      # TypeScript
make proto-py      # Python + gRPC

# Clean generated files
make proto-clean
```

### Frontend

```bash
make frontend-dev      # Start dev server (port 3000)
make frontend-build    # Production build
make frontend-lint     # ESLint check
make frontend-preview  # Preview production build
```

### APIs Backend

Located in `apis/` directory:

```bash
make apis-run          # Run APIs + Analytics (builds frontend first)
cd apis && make run    # Run APIs only
cd apis && make test   # Run tests
cd apis && make build  # Build binary
```

**Important:** APIs serve the built frontend from `frontend/dist`, so frontend must be built first.

### Analytics Service

Located in `analytics/` directory:

```bash
make analytics-run     # Run analytics server (gRPC on :50051)
cd analytics && make run
```

Requires ClickHouse running (see docker-compose.yaml).

### Watcher (eBPF)

Located in `watcher/` directory. Requires Linux environment for eBPF compilation.

```bash
cd watcher && make build          # Build with Docker (eBPF + Go binary)
cd watcher && make docker-image   # Build final image
cd watcher && make deploy         # Deploy to Kind cluster
```

**Architecture:** Multi-stage Docker build:

1. Builder stage: Compiles eBPF programs and generates Go bindings
2. Runtime stage: Minimal image with compiled binary

### Operator

Located in `operator/` directory. Uses Kubebuilder framework.

```bash
# CRD/manifest management
make operator-manifests   # Generate CRDs
make operator-generate    # Generate deepcopy code

# Build and test
make operator-build       # Build binary
make operator-test        # Run tests
make operator-run         # Run locally (against current kubeconfig)

# Docker
make operator-docker-build

# Deploy to cluster
make operator-install     # Install CRDs
make operator-deploy      # Deploy operator
make operator-undeploy    # Remove operator
make operator-uninstall   # Remove CRDs
```

**Helm Chart:** `operator/deploy/helm/futura-operator/`

```bash
helm install futura-operator ./operator/deploy/helm/futura-operator \
  --namespace futura-system --create-namespace
```

**Examples:** `operator/examples/`

- `clusteroptimizationconfig.yaml` - Cluster-wide optimization config
- `slo-nginx-deployment.yaml` - SLO with HPA/VPA for Deployment
- `slo-statefulset.yaml` - SLO for StatefulSet

### Engine (Python)

Located in `engine/` directory:

```bash
cd engine && make install        # Install with pip
cd engine && make test           # Run all tests
cd engine && make test-unit      # Unit tests only
cd engine && make lint           # Linting
cd engine && make format         # Auto-format code
```

### Pipeline

Located in `pipeline/` directory:

```bash
cd pipeline && make run          # Run Kafka consumer → ClickHouse writer
cd pipeline && make deploy       # Deploy to Kind
```

## Code Structure

### APIs (`apis/`)

```
apis/
├── controllers/     # HTTP handlers (cluster, auth, connect)
├── models/          # GORM models for PostgreSQL
├── pkg/analytics/   # Analytics gRPC client wrapper
├── routes/          # Router setup
├── utils/           # Auth, audit logging, response helpers
└── internal/config/ # Configuration management
```

**Key Patterns:**

- All controllers use `utils.RespondOK()` and `utils.RespondError()` for consistent responses
- Audit logging via `utils.LogAuditEvent()` for security events
- SSE endpoints use `c.SSEvent()` with 5-second polling intervals

### Analytics (`analytics/`)

```
analytics/
├── internal/server/    # gRPC server implementation
├── internal/config/    # Configuration
└── db/migrations/      # ClickHouse migrations
```

**ClickHouse Query Patterns:**

- Use `argMax(field, timestamp)` for latest values
- `countIf()` for conditional aggregations
- CTEs for complex queries
- Always scan into `uint64` for count results, cast to `int32` for proto

**Migrations:**

```bash
# Run migrations
migrate -path analytics/db/migrations \
  -database "clickhouse://localhost:9000?username=user&password=password&database=events&x-multi-statement=true" \
  up
```

### Frontend (`frontend/src/`)

```
frontend/src/
├── app/              # Pages (clusters/, connect/, settings/)
├── components/       # Reusable UI components
├── hooks/            # Custom hooks (auth-provider, sse-handler)
├── lib/              # Utilities
└── models/           # TypeScript types
```

**Important Patterns:**

- SSE hooks: `useSSE<T>()` in `hooks/sse-handler.tsx`
- Auth context: `useAuth()` provides user session
- Backend proto types: uint64 comes as string in JSON (must parse)

### Operator (`operator/`)

```
operator/
├── api/v1/                    # CRD types
├── internal/controller/       # Reconciliation logic
└── deploy/helm/futura-operator/  # Helm chart
```

**CRDs:**

- `ServiceLevelObjective` - Manages HPA + VPA for workloads
- `ClusterOptimizationConfig` - Cluster-wide optimization settings

## Testing

### Go Services

```bash
# APIs
cd apis && make test
cd apis && make test-coverage

# Analytics
cd analytics && go test ./...

# Operator
make operator-test
```

### Frontend

```bash
cd frontend && bun test          # Run tests
cd frontend && bun run type-check  # TypeScript check
```

### Python Engine

```bash
cd engine && make test
cd engine && make test-coverage
```

## Lima VM for Local Testing

For testing eBPF and Kind on Mac, use the provided Lima VM:

```bash
# Start VM
limactl start lima-debian-vm.yaml

# Access VM
limactl shell <vm-name>

# Inside VM:
kind create cluster --name futura-test
docker ps  # Should work without sudo
```

The VM includes: Docker, Kind, kubectl, Helm, and Go.

## Important Notes

### Proto Generation Dependencies

- **Go:** Requires `protoc-gen-go` and `protoc-gen-go-grpc` (installed via Nix)
- **TypeScript:** Uses `protoc-gen-ts_proto` from `proto/node_modules`
- **Python:** Uses `uv` with `grpcio-tools` (auto-installed by Makefile)

### Analytics Service Type Handling

When implementing analytics queries:

1. ClickHouse `countIf()` returns `UInt64`
2. Scan into `uint64` variables in Go
3. Cast to `int32` when populating proto messages
4. Frontend receives `int32` as numbers in JSON

### SSE Implementation Pattern

Backend SSE endpoints follow this pattern:

```go
ticker := time.NewTicker(5 * time.Second)
defer ticker.Stop()

sendData := func() {
    // Query analytics
    resp, err := client.GetMetrics(...)
    // Marshal and send
    c.SSEvent("event-name", string(data))
    flusher.Flush()
}

sendData() // Initial send

for {
    select {
    case <-ticker.C:
        sendData()
    case <-c.Request.Context().Done():
        return
    }
}
```

Frontend SSE consumption:

```typescript
const { latest: sseData } = useSSE<DataType>(url, {
  event: "event-name",
  withCredentials: false,
});

useEffect(() => {
  if (!sseData) return;
  setLoading(false);
  // Handle data
}, [sseData]);
```

### Go Workspace

This project uses Go workspaces (`go.work`). Run `make dev-env` to initialize:

- Creates `go.work` file
- Adds all Go modules recursively
- Syncs dependencies

### Operator Development

When modifying CRDs:

1. Edit types in `operator/api/v1/*_types.go`
2. Run `make operator-manifests` to regenerate CRDs
3. Run `make operator-generate` for deepcopy methods
4. Update Helm chart CRDs: copy from `operator/config/crd/bases/` to `operator/deploy/helm/futura-operator/crds/`
