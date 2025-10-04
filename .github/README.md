# GitHub Actions CI/CD Setup

This directory contains the GitHub Actions workflows for the Futura monorepo.

## Workflows

### 1. `test.yml` - Pull Request Testing & Build

**Trigger:** On all pull requests

**Phase 1 - Testing (run in parallel):**

- ✅ **test-apis**: Go tests for APIs service (with frontend build)
- ✅ **test-analytics**: Go tests for Analytics service
- ✅ **test-pipeline**: Go tests for Pipeline service
- ✅ **test-watcher**: Go tests for Watcher service (with eBPF generation)
- ✅ **test-operator**: Operator build verification
- ✅ **test-engine**: Python tests for Engine service
- ✅ **test-frontend**: Bun lint + TypeScript type-checking

**Phase 2 - Version & Build (runs only if all tests pass):**

- 🔢 **version-pr**: Increments **patch version** (3rd digit: `v0.1.0` → `v0.1.1`)
- 🏗️ **build-frontend-pr**: Builds frontend assets
- 🐳 **build-and-push-pr**: Builds and pushes Docker images with tags:
  - `davideberdin/futura-{service}:{version}` (e.g., `v0.1.1`)
  - `davideberdin/futura-{service}:pr-{number}` (e.g., `pr-123`)

**Purpose:** Test all services, then build and push versioned Docker images for successful PRs.

### 2. `build-push.yml` - Production Build on Main

**Trigger:** On push to `main` branch (e.g., when PRs are merged)

**Process:**

1. **Version Management**:

   - Reads current version from `.version` file
   - Auto-increments **minor version** (2nd digit: `v0.1.5` → `v0.2.0`)
   - Resets patch version to 0
   - Commits new version back to repository with `[skip ci]` flag

2. **Frontend Build**:

   - Builds frontend assets
   - Uploads as artifact for APIs image

3. **Multi-Architecture Image Build**:

   - Builds Docker images for all services (apis, analytics, operator, watcher, pipeline, engine)
   - Pushes to Docker Hub with tags:
     - `davideberdin/futura-{service}:{version}` (e.g., `v0.1.1`)
     - `davideberdin/futura-{service}:latest`
   - Supports `linux/amd64` and `linux/arm64` platforms

4. **Summary**:
   - Generates GitHub Actions summary with version and pushed images

## Required GitHub Secrets

To enable Docker Hub image pushing, you need to configure the following secret in your GitHub repository:

### `DOCKERHUB_TOKEN`

**Setup Instructions:**

1. **Create a Docker Hub Access Token:**

   - Go to [Docker Hub](https://hub.docker.com)
   - Navigate to Account Settings → Security → Access Tokens
   - Click "New Access Token"
   - Give it a description (e.g., "GitHub Actions CI/CD")
   - Select "Read, Write" permissions
   - Click "Generate" and copy the token (you won't be able to see it again!)

2. **Add Secret to GitHub Repository:**
   - Go to your GitHub repository
   - Navigate to Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `DOCKERHUB_TOKEN`
   - Value: Paste your Docker Hub access token
   - Click "Add secret"

## Version Management

The project uses semantic versioning stored in the `.version` file at the repository root.

**Current format:** `v{major}.{minor}.{patch}`

**Auto-increment behavior:**

- **Pull Requests**: Increment **patch** version (3rd digit)
  - Example: `v0.1.0` → `v0.1.1` → `v0.1.2`
- **Main branch** (merged PRs): Increment **minor** version (2nd digit) and reset patch to 0
  - Example: `v0.1.5` → `v0.2.0`
- The version commit includes `[skip ci]` to prevent infinite CI loops

**Version Flow Example:**
```
PR #1: v0.1.0 → v0.1.1 (patch bump)
PR #2: v0.1.1 → v0.1.2 (patch bump)
Merge to main: v0.1.2 → v0.2.0 (minor bump, patch reset)
PR #3: v0.2.0 → v0.2.1 (patch bump)
Merge to main: v0.2.1 → v0.3.0 (minor bump, patch reset)
```

**Manual version bumps:**
If you need to bump major or minor versions manually:

```bash
# Bump to v1.0.0 (major release)
echo "v1.0.0" > .version
git add .version
git commit -m "chore: bump to v1.0.0"
git push

# Bump to v0.2.0 (minor release)
echo "v0.2.0" > .version
git add .version
git commit -m "chore: bump to v0.2.0"
git push
```

The next automatic push to `main` will increment from your new version.

## Dependencies Handling

The CI workflows handle monorepo dependencies correctly:

### Go Services (apis, analytics, operator, watcher, pipeline)

- Uses Go workspace (`go work`) to resolve local module dependencies
- Includes `proto/` and `go-lib/` modules automatically
- Dockerfiles use multi-stage builds with proper COPY directives

### Python Service (engine)

- Uses `uv` for fast dependency installation
- Runs pytest with full verbosity

### Frontend

- Uses Bun for package management and building
- Build artifacts are copied to APIs Docker image

## Troubleshooting

### Test failures in PRs

- Check the specific job that failed in the GitHub Actions UI
- Each service tests independently, making it easy to identify the failing component
- Run tests locally: `cd {service} && make test` (or equivalent command)

### Docker Hub push failures

- Verify `DOCKERHUB_TOKEN` secret is set correctly
- Check Docker Hub for rate limits or authentication issues
- Ensure your Docker Hub account has permissions for the `davideberdin` namespace

### Version conflicts

- If the version commit fails, check for concurrent pushes to `main`
- Pull latest changes and try again
- Verify `.version` file exists and is properly formatted

### Frontend build issues

- Ensure frontend builds successfully locally: `cd frontend && bun run build`
- Check that `apis/public/` directory is created with build artifacts
- Verify the upload/download artifact steps in the workflow

## Local Testing

### Run All Tests Locally

You can run all tests locally using the root Makefile:

```bash
# Run all tests (Go, Python, Frontend)
make test

# Run only Go service tests
make test-go

# Run only Python tests
make test-python

# Run only Frontend tests
make test-frontend
```

### Test with Act (GitHub Actions locally)

You can test workflows locally using [act](https://github.com/nektos/act):

```bash
# Test PR workflow
act pull_request

# Test build workflow (requires secrets)
act push -s DOCKERHUB_TOKEN=your-token-here
```

## CI/CD Flow Diagram

```bash
Pull Request Created
    ↓
test.yml → Phase 1: Run all tests in parallel
    ↓
All tests pass? ─────┐
    ↓                │
    Yes              No → ❌ PR blocked
    ↓
Phase 2: Version & Build
    1. Bump patch version (v0.1.0 → v0.1.1)
    2. Build frontend
    3. Build & push Docker images
       - Tags: v0.1.1, pr-123
    ↓
✅ PR ready to merge
    ↓
Merge to main
    ↓
build-push.yml
    1. Bump minor version (v0.1.1 → v0.2.0)
    2. Build frontend
    3. Build & push Docker images
       - Tags: v0.2.0, latest
    ↓
🚀 Production images on Docker Hub
```
