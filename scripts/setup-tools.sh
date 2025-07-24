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
