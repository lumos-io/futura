#!/usr/bin/env bash
set -euo pipefail

# 1. Update PATH
export PATH="$PWD/node_modules/.bin:$HOME/.bun/bin:$PATH"
echo "📦 PATH updated with node_modules/.bin and bun"

# 2. Ensure Vite is installed via Bun and ts-proto
if ! command -v vite &> /dev/null; then
  echo "⚙️ Installing vite via bun..."
  bun install -g vite  
else
  echo "✅ vite is already installed"
fi

if ! command -v protoc-gen-ts_proto &> /dev/null; then 
echo "⚙️ Installing ts-proto via bun..."
  bun install -g ts-proto
else
  echo "✅ ts-proto is already installed"
fi

# 3. Create kind cluster if not exists
KIND_CLUSTER_NAME="dev-cluster"

if ! kind get clusters | grep -q "^$KIND_CLUSTER_NAME$"; then
  echo "🌱 Creating kind cluster: $KIND_CLUSTER_NAME"
  if [ ! -f ./kind/kind.yaml ]; then
    echo "❌ Missing cluster config at ./kind/kind.yaml"
    exit 1
  fi

  kind create cluster --name "$KIND_CLUSTER_NAME" --config ./kind/kind.yaml
else
  echo "✅ kind cluster \"$KIND_CLUSTER_NAME\" already exists"
fi

# 4. Wait for cluster to be ready (simple check)
echo "⏳ Waiting for Kubernetes API to be ready..."
for i in {1..30}; do
  if kubectl cluster-info &> /dev/null; then
    echo "✅ Kubernetes API is up"
    break
  fi
  sleep 1
done

# 5. Apply custom service manifests
echo "📦 Deploying services..."

if [ -f ./kind/redis.yaml ]; then
  echo "🚀 Deploying Redis..."
  kubectl apply -f ./kind/redis.yaml
else
  echo "⚠️  redis.yaml not found"
fi

if [ -f ./kind/metrics-server.yaml ]; then
  echo "🚀 Deploying metrics-server..."
  kubectl apply -f ./kind/metrics-server.yaml
else
  echo "⚠️  metrics-server.yaml not found"
fi
