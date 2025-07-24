#!/usr/bin/env bash
set -euo pipefail

# 1. Create kind cluster if not exists
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

# 2. Wait for cluster to be ready (simple check)
echo "⏳ Waiting for Kubernetes API to be ready..."
for i in {1..30}; do
  if kubectl cluster-info &> /dev/null; then
    echo "✅ Kubernetes API is up"
    break
  fi
  sleep 1
done

# 3. Apply custom service manifests
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
