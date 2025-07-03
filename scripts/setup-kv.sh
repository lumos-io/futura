#!/usr/bin/env bash
set -euo pipefail

NATS_POD=$(kubectl get pods -l app=nats -o jsonpath='{.items[0].metadata.name}')
BUCKET_NAME="api_keys"
KEY_NAME="df9166bbacd761c74aecc50bb7a902342dd61a1de84551e253f7133154947d88"
KEY_VALUE='{"customer_id":"1", "status":"active", "cluster_id":"1", "cloud_provider_id":"1"}'

echo "Using NATS pod: $NATS_POD"

# Create the KV bucket (ignore error if already exists)
kubectl exec "$NATS_POD" -- nats kv add "$BUCKET_NAME" || echo "Bucket '$BUCKET_NAME' may already exist, continuing..."

# Add API Key
kubectl exec "$NATS_POD" -- nats kv put "$BUCKET_NAME" "$KEY_NAME" "$KEY_VALUE"

echo "Added key '$KEY_NAME' to bucket '$BUCKET_NAME'"