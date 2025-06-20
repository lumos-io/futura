#!/usr/bin/env bash
set -a
if [ -f .env.local ]; then
  echo "📄 Loading environment from .env.local"
  . .env.local
else
  echo "⚠️  .env.local not found"
fi
set +a
