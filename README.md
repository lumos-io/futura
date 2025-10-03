# Futura Monorepo

This monorepo contains all the code to make the OpisVigilant platform working with the except of the infrastructure

## Nix Development setup

This project uses `nix` to make it more portable. To install nix, you can run the following command `curl -fsSL https://install.determinate.systems/nix | sh -s -- install --determinate`.

### .env file

You need to create a `.env.local` file where you add secrets that shouldn't be part of the git commit. Nix will try to read the file and stop in case it cannot find it. A `.env.tmp` file is committed with the variables that need to be used.

### Run Nix

Assuming `nix` has been installed correctly and the `.env.local` is available, you can proceed with `nix develop` to enter the environment. By default, `kind` is not created (also metric-server and redis are skipped). To enable Kind deployment, run nix as follow `SKIP_KIND=false nix develop`.

#### Accessing Redis

Redis is deployed in Kubernetes using kind. The nix shell has the CLI installed. When the Nix shell starts, it runs a few commands against Redis which are described
in the `/scripts/setup-kv.sh`. Specifically, it will create the `api_keys` bucket and add an api key for testing.
To access the server run the following commands in two separate shells

```bash
# Shell 1
kubectl port-forward svc/redis 16379:6379
```

and then to access redis, in the second terminal you can use the `redis-cli` to connect and run commands.

## Docker Compose Profile

Now all the containers need to be created for local development. I have attached a `profile` flag to the containers so that they will be created if and only if the profile is specified. For example

```bash
docker compose --profile dev-tools up -d
```

will also start the containers with that profile (like `redisinsight`).

## Bazel Build Guide

This document provides quick reference for building Futura services with Bazel 8.4.

## Prerequisites

- **Bazel 8.4.0+**: Install via [Bazelisk](https://github.com/bazelbuild/bazelisk) (recommended)
- **Docker**: Required for watcher and engine builds
- **Git**: For version stamping

```bash
# Install Bazelisk (manages Bazel versions)
# macOS
brew install bazelisk

# Linux
curl -L https://github.com/bazelbuild/bazelisk/releases/latest/download/bazelisk-linux-amd64 -o /usr/local/bin/bazel
chmod +x /usr/local/bin/bazel
```

## Quick Start

### Gazelle - Auto-generate BUILD files

**Gazelle** automatically generates and updates BUILD.bazel files for Go code, eliminating manual dependency management.

```bash
# Generate/update BUILD files for all Go packages
bazel run //:gazelle

# Update external Go dependencies from go.work
bazel run //:gazelle-update-repos

# Check if BUILD files are up-to-date (use in CI)
bazel run //:gazelle -- --mode=diff
```

**When to run Gazelle:**

- After adding new `.go` files
- After adding new imports
- After creating new Go packages
- After modifying `go.mod` or `go.work`

### Build All Services

```bash
# Build everything (images for all services)
bazel build //...

# Build only changed targets (incremental)
bazel build //...
```

### Build Individual Services

```bash
# Frontend (bundle only, no image)
bazel build //frontend:frontend_bundle

# APIs (with embedded frontend)
bazel build //apis:apis_image

# Analytics
bazel build //analytics:analytics_image

# Pipeline
bazel build //pipeline:pipeline_image

# Operator
bazel build //operator:operator_image

# Watcher (requires Docker)
bazel build //watcher:watcher_image

# Engine (requires Docker)
bazel build //engine:engine_image
```

### Push Images to Registry

```bash
# Push single service
bazel run //apis:push

# Push all services (requires Docker Hub authentication)
bazel run //apis:push //analytics:push //pipeline:push //operator:push //watcher:push //engine:push
```

**Note:** Configure Docker Hub credentials:

```bash
docker login docker.io
```

## Build Configurations

### Development Build

```bash
bazel build --config=dev //apis:apis_image
```

### Multi-Architecture Build

```bash
bazel build --config=multiarch //apis:apis_image
```

### CI Build (with caching)

```bash
export BAZEL_REMOTE_CACHE_URL="https://your-cache-url"
bazel build --config=ci //...
```

## Common Commands

### Clean Build Cache

```bash
# Clean build outputs
bazel clean

# Deep clean (removes all cached artifacts)
bazel clean --expunge
```

### Query Targets

```bash
# List all buildable targets
bazel query //...

# List all images
bazel query 'kind("oci_image", //...)'

# Show dependencies for a target
bazel query 'deps(//apis:apis_image)'
```

### Test

```bash
# Run all tests
bazel test //...

# Run tests for specific service
bazel test //apis/...

# Run with race detector (Go only)
bazel test --config=race //apis/...
```

## Service-Specific Notes

### Frontend

- **Target**: `//frontend:frontend_bundle`
- **Output**: Tarball containing Vite build output
- **Dependencies**: Bun, Node.js, TypeScript
- **Build metadata**: Injects git SHA and build timestamp

```bash
# Build frontend bundle
bazel build //frontend:frontend_bundle

# Extract to local directory
tar -xzf bazel-bin/frontend/frontend_bundle.tar.gz -C ./frontend/dist
```

### APIs

- **Target**: `//apis:apis_image`
- **Dependencies**: Frontend bundle (auto-built)
- **Embedded files**: Frontend static assets in `public/`

```bash
# Build APIs (includes frontend build)
bazel build //apis:apis_image

# Push to registry
bazel run //apis:push
```

### Watcher

- **Target**: `//watcher:watcher_image`
- **Special requirements**: Docker (for eBPF compilation)
- **Build time**: ~5-10 minutes (compiles eBPF programs)

```bash
# Build watcher (orchestrates Dockerfile)
bazel build //watcher:watcher_image

# Note: Requires Docker daemon running
```

### Engine

- **Target**: `//engine:engine_image`
- **Special requirements**: Docker (for UV package manager)
- **Python version**: 3.13

```bash
# Build engine (orchestrates Dockerfile)
bazel build //engine:engine_image
```

### Operator

- **Target**: `//operator:operator_image`
- **Base image**: distroless/static (minimal)
- **User**: non-root (65532:65532)

```bash
# Build operator
bazel build //operator:operator_image

# Run locally (against current kubeconfig)
bazel run //operator:manager
```

## Gazelle Workflow

### What is Gazelle?

Gazelle is a build file generator that automatically creates and updates `BUILD.bazel` files for Go projects. It analyzes your Go source code and:

- Discovers `.go` files in each package
- Detects import statements
- Generates `go_library`, `go_binary`, and `go_test` rules
- Manages internal and external dependencies
- Keeps BUILD files in sync with code changes

### Common Gazelle Commands

```bash
# Generate/update all BUILD files
bazel run //:gazelle

# Update external Go dependencies from go.work
# Run this after modifying any go.mod file
bazel run //:gazelle-update-repos

# Verify BUILD files are up-to-date (CI check)
bazel run //:gazelle -- --mode=diff

# Fix a specific directory only
bazel run //:gazelle -- fix //apis/controllers

# See what Gazelle would change without modifying files
bazel run //:gazelle -- -mode=diff
```

### Developer Workflow with Gazelle

**Scenario 1: Adding a new Go file**

```bash
# 1. Create new file
touch apis/controllers/new_controller.go

# 2. Write code with imports
# (no need to update BUILD.bazel manually!)

# 3. Run Gazelle
bazel run //:gazelle

# Gazelle automatically:
# - Adds new_controller.go to srcs
# - Adds any new dependencies to deps
# - Creates BUILD files for new packages
```

**Scenario 2: Adding a new package**

```bash
# 1. Create new package directory
mkdir -p apis/services/billing

# 2. Add Go files
touch apis/services/billing/billing.go

# 3. Run Gazelle
bazel run //:gazelle

# Gazelle creates apis/services/billing/BUILD.bazel automatically
```

**Scenario 3: Updating go.mod**

```bash
# 1. Add dependency to go.mod
cd apis && go get github.com/new/package@v1.0.0

# 2. Update go.work
cd .. && go work sync

# 3. Update Bazel dependencies
bazel run //:gazelle-update-repos

# 4. Regenerate BUILD files
bazel run //:gazelle
```

### Gazelle Directives

Gazelle behavior is customized via `# gazelle:` comments in BUILD files:

**Common Directives:**

```python
# Set import prefix for a module (overrides global)
# gazelle:prefix io.lumos/futura

# Exclude directories from scanning
# gazelle:exclude public
# gazelle:exclude node_modules

# Custom proto resolution
# gazelle:resolve proto go github.com/opisvigilant/futura/proto //proto:futura_go_proto

# Disable proto rule generation (we use Makefile)
# gazelle:proto disable

# Map Go package to Bazel label
# gazelle:resolve go github.com/custom/pkg //third_party:custom_pkg
```

**Applied in Futura:**

- **Root BUILD.bazel**: Global prefix + proto disabled
- **apis/BUILD.bazel**: Exclude `public/` (embedded frontend)
- **watcher/BUILD.bazel**: Exclude eBPF C files (`.c`, `.h`)
- **operator/BUILD.bazel**: Custom prefix `io.lumos/futura`

### Hybrid Approach: Gazelle + Manual Rules

We use a **hybrid approach** where Gazelle manages Go rules, but custom rules (OCI images, genrules) are hand-written:

```python
# apis/BUILD.bazel

# Gazelle-managed section (auto-updated)
load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "apis",
    srcs = ["main.go"],  # Auto-updated by Gazelle
    deps = [             # Auto-updated by Gazelle
        "//apis/controllers",
        "//apis/models",
        # ...
    ],
)

# Manual section (never touched by Gazelle)
load("@rules_oci//oci:defs.bzl", "oci_image")

oci_image(
    name = "apis_image",
    base = "@distroless_base",
    entrypoint = ["/app/apis"],
)
```

### Troubleshooting Gazelle

**Gazelle not finding imports:**

```bash
# Make sure go.work references all modules
cat go.work

# Resync workspace
go work sync

# Regenerate
bazel run //:gazelle
```

**Gazelle overwrites custom rules:**

Add directives to protect sections:

```python
# gazelle:ignore
oci_image(
    name = "custom_image",
    # Won't be modified by Gazelle
)
```

**External dependency not found:**

```bash
# Update MODULE.bazel with new dependencies
bazel run //:gazelle-update-repos

# Then regenerate BUILD files
bazel run //:gazelle
```

## Versioning

Version is managed via `VERSION` file at repository root:

```bash
# Current version
cat VERSION

# Update version (triggers rebuild of all images)
echo "0.2.0" > VERSION
```

Version is embedded in images via build stamping:

```bash
bazel build --stamp //apis:apis_image
```

## Troubleshooting

### "MODULE.bazel.lock out of sync"

```bash
bazel sync --configure
```

### "Cannot find external dependency"

```bash
# Re-fetch all external dependencies
bazel sync
bazel build //...
```

### "Docker build failed" (watcher/engine)

Ensure Docker daemon is running:

```bash
docker ps
```

### Slow builds

Enable disk cache in `~/.bazelrc`:

```starlark
build --disk_cache=~/.cache/bazel
```

### Remote cache issues (CI)

Verify `BAZEL_REMOTE_CACHE_URL` is set:

```bash
echo $BAZEL_REMOTE_CACHE_URL
```

## CI/CD Integration

GitHub Actions workflow (`.github/workflows/build-and-push.yaml`):

- **Triggers**: Push to `main`, Pull Requests
- **Change detection**: Only builds modified services
- **Caching**: Uses GitHub Actions cache for Bazel
- **Multi-arch**: Builds linux/amd64 and linux/arm64
- **Registry**: Pushes to Docker Hub (`docker.io/davideberdin/futura-*`)

### Required Secrets

Set in GitHub repository settings:

- `DOCKER_HUB_TOKEN`: Docker Hub access token

### Workflow Jobs

1. **detect-changes**: Detects which services changed via `paths-filter`
2. **build-push** (main only): Builds and pushes changed images
3. **verify-build** (PRs only): Verifies builds without pushing

## Performance Tips

1. **Use Bazelisk**: Automatically uses correct Bazel version from `.bazelversion`
2. **Enable disk cache**: Add to `~/.bazelrc` (see above)
3. **Parallel builds**: Bazel auto-detects CPU count (`--jobs=auto`)
4. **Incremental builds**: Only rebuilds changed targets
5. **Remote caching**: Use in CI for cross-build caching

## Advanced Usage

### Custom Image Tags

```bash
bazel run //apis:push -- --tag docker.io/davideberdin/futura-apis:v1.2.3
```

### Build with Debug Info

```bash
bazel build --compilation_mode=dbg //apis:apis
```

### Profile Build Performance

```bash
bazel build --profile=profile.json //...
bazel analyze-profile profile.json
```

## Resources

- [Bazel Documentation](https://bazel.build)
- [rules_go](https://github.com/bazelbuild/rules_go)
- [rules_oci](https://github.com/bazel-contrib/rules_oci)
- [aspect_rules_js](https://github.com/aspect-build/rules_js)
- [rules_python](https://github.com/bazelbuild/rules_python)
