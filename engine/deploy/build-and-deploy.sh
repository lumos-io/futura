#!/bin/bash
set -euo pipefail

# Futura Engine Build and Deploy Script
# Usage: ./build-and-deploy.sh [development|staging|production] [version]

ENVIRONMENT=${1:-development}
VERSION=${2:-latest}
REGISTRY=${DOCKER_REGISTRY:-futura}
IMAGE_NAME="futura/engine"
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${VERSION}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(development|staging|production)$ ]]; then
    error "Invalid environment. Use: development, staging, or production"
fi

log "🚀 Building and deploying Futura Engine"
log "Environment: $ENVIRONMENT"
log "Version: $VERSION"
log "Image: $FULL_IMAGE"

# Check prerequisites
log "📋 Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    error "Docker not found. Please install Docker."
fi

if ! command -v kubectl &> /dev/null; then
    error "kubectl not found. Please install kubectl."
fi

if ! kubectl cluster-info &> /dev/null; then
    error "Cannot connect to Kubernetes cluster. Check your kubeconfig."
fi

success "Prerequisites check passed"

# Build Docker image
log "🔨 Building Docker image..."
cd "$(dirname "$0")/.."  # Go to engine root directory

if ! docker build -t "$FULL_IMAGE" .; then
    error "Docker build failed"
fi

success "Docker image built: $FULL_IMAGE"

# Push image (optional)
if [[ "${PUSH_IMAGE:-true}" == "true" ]]; then
    log "📤 Pushing image to registry..."
    if ! docker push "$FULL_IMAGE"; then
        warning "Failed to push image. Continuing with local image..."
    else
        success "Image pushed to registry"
    fi
fi

# Prepare Kubernetes manifests
log "📝 Preparing Kubernetes manifests..."
cd deploy

# Create temporary kustomization with correct image
TEMP_DIR=$(mktemp -d)
cp -r overlays/"$ENVIRONMENT"/\* "$TEMP_DIR/"

# Update image tag in kustomization
if [[ -f "$TEMP_DIR/kustomization.yaml" ]]; then
    # Use yq if available, otherwise sed
    if command -v yq &> /dev/null; then
        yq eval ".images[0].newTag = \"$VERSION\"" -i "$TEMP_DIR/kustomization.yaml"
    else
        sed -i.bak "s/newTag:.*/newTag: $VERSION/" "$TEMP_DIR/kustomization.yaml"
    fi
fi

# Deploy to Kubernetes
log "🚀 Deploying to Kubernetes ($ENVIRONMENT)..."

# Apply base resources first
if ! kubectl apply -f base/namespace.yaml; then
    error "Failed to create namespaces"
fi

# Apply overlay with kustomize
if ! kubectl apply -k "$TEMP_DIR"; then
    error "Kubernetes deployment failed"
fi

# Clean up
rm -rf "$TEMP_DIR"

success "Deployed to Kubernetes"

# Wait for rollout
log "⏳ Waiting for rollout to complete..."

DEPLOYMENTS=(
    "futura-recommendation-service"
    "futura-rl-server"
    "futura-agent-coordinator"
)

# Check if using all-in-one deployment
if kubectl get deployment futura-engine-all -n futura-engine &> /dev/null; then
    DEPLOYMENTS=("futura-engine-all")
fi

for deployment in "${DEPLOYMENTS[@]}"; do
    if kubectl get deployment "$deployment" -n futura-engine &> /dev/null; then
        if ! kubectl rollout status deployment/"$deployment" -n futura-engine --timeout=300s; then
            error "Rollout failed for $deployment"
        fi
        success "Rollout completed for $deployment"
    fi
done

# Health check
log "🏥 Performing health check..."

# Wait a bit for services to be ready
sleep 10

# Find a pod to run health check from
POD_NAME=$(kubectl get pods -n futura-engine -l app.kubernetes.io/component=all-services -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || \
           kubectl get pods -n futura-engine -l app.kubernetes.io/component=recommendation-service -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || \
           kubectl get pods -n futura-engine -o jsonpath='{.items[0].metadata.name}')

if [[ -n "$POD_NAME" ]]; then
    if kubectl exec -n futura-engine "$POD_NAME" -- futura-engine health --endpoint localhost:8080; then
        success "Health check passed"
    else
        warning "Health check failed, but deployment completed"
    fi
else
    warning "No pods found for health check"
fi

# Display access information
log "📋 Deployment Summary"
echo ""
echo "Environment: $ENVIRONMENT"
echo "Version: $VERSION"
echo "Image: $FULL_IMAGE"
echo ""
echo "🔗 Service Access:"
echo "  Internal (within cluster):"
echo "    grpc://futura-recommendation-service.futura-engine.svc.cluster.local:8080"
echo "    grpc://futura-rl-server.futura-engine.svc.cluster.local:8081"
echo "    grpc://futura-agent-coordinator.futura-engine.svc.cluster.local:8082"
echo ""
echo "  External (port-forward):"
echo "    kubectl port-forward -n futura-engine svc/futura-engine-all 8080:8080"
echo "    kubectl port-forward -n futura-engine svc/futura-recommendation-service 8080:8080"
echo ""
echo "🔍 Useful Commands:"
echo "  View pods:    kubectl get pods -n futura-engine"
echo "  View logs:    kubectl logs -n futura-engine deployment/futura-engine-all -f"
echo "  Health check: kubectl exec -n futura-engine deployment/futura-engine-all -- futura-engine health"
echo "  Scale up:     kubectl scale deployment futura-recommendation-service -n futura-engine --replicas=3"
echo ""

success "🎉 Deployment completed successfully!"

# Show status
log "📊 Current Status:"
kubectl get pods,svc -n futura-engine