#!/usr/bin/env bash
set -euo pipefail

REDIS_SERVICE_PORT=6379
LOCAL_PORT=16379
KEY_NAME="df9166bbacd761c74aecc50bb7a902342dd61a1de84551e253f7133154947d88"
KEY_VALUE='{"customer_id":"1", "status":"active", "cluster_id":"1", "cloud_provider_id":"1"}'

MAX_RETRIES=5
RETRY_DELAY=3
PF_PID=""

# Function to attempt port-forward with retry
port_forward() {
  for attempt in $(seq 1 "$MAX_RETRIES"); do
    echo "🔄 Attempt $attempt to port-forward Redis..."

    kubectl port-forward svc/redis "$LOCAL_PORT:$REDIS_SERVICE_PORT" > /tmp/redis-portforward.log 2>&1 &
    PF_PID=$!

    sleep 2

    # Check if port-forward is working by probing the port
    if nc -z localhost "$LOCAL_PORT"; then
      echo "✅ Port-forward successful (PID $PF_PID)"
      return 0
    else
      echo "⚠️  Port-forward attempt $attempt failed, retrying in $RETRY_DELAY seconds..."
      kill "$PF_PID" >/dev/null 2>&1 || true
      sleep "$RETRY_DELAY"
    fi
  done

  echo "❌ Failed to port-forward Redis after $MAX_RETRIES attempts."
  exit 1
}

# Try to start port-forward
port_forward

# Add key with JSON value using redis-cli
echo "SET $KEY_NAME '$KEY_VALUE'" | redis-cli -p "$LOCAL_PORT"

echo "✅ Added key '$KEY_NAME' to Redis"

# Cleanup port-forward
kill "$PF_PID" >/dev/null 2>&1 || true
