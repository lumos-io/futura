#!/usr/bin/env bash
set -euo pipefail

NATS_SERVICE_PORT=4222
LOCAL_PORT=14222
BUCKET_NAME="api_keys"
KEY_NAME="df9166bbacd761c74aecc50bb7a902342dd61a1de84551e253f7133154947d88"
KEY_VALUE='{"customer_id":"1", "status":"active", "cluster_id":"1", "cloud_provider_id":"1"}'

# Give it a second to come up
sleep 5

# Port-forward NATS port (runs in background)
kubectl port-forward svc/nats "$LOCAL_PORT:$NATS_SERVICE_PORT" > /tmp/nats-portforward.log 2>&1 &
PF_PID=$!
echo "✅ Port-forward started (PID $PF_PID), waiting for connection..."

# Give it a second to connect
sleep 2

# Set NATS CLI context to local forwarded port
export NATS_URL="nats://localhost:$LOCAL_PORT"

# Create bucket (ignore error if it exists)
nats kv add "$BUCKET_NAME" || echo "✅ Bucket '$BUCKET_NAME' may already exist, continuing..."

# Add key with JSON value
nats kv put "$BUCKET_NAME" "$KEY_NAME" "$KEY_VALUE"

echo "✅ Added key '$KEY_NAME' to bucket '$BUCKET_NAME'"

# Cleanup port-forward
kill "$PF_PID"