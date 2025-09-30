# Futura Engine Dockerfile
# Multi-stage build for production-ready container

# Build stage
FROM python:3.13-slim as builder

# Set environment variables
ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# Install system dependencies for building
RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# Set work directory
WORKDIR /app

# Copy dependency files
COPY pyproject.toml uv.lock ./

# Install UV for fast dependency management
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Install dependencies
RUN uv sync --frozen --no-dev

# Production stage
FROM python:3.13-slim as production

# Set environment variables
ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PATH="/app/.venv/bin:$PATH"

# Install runtime dependencies and uv
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Copy uv from official image
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Create app user
RUN useradd --create-home --shell /bin/bash app

# Set work directory
WORKDIR /app

# Copy virtual environment from builder
COPY --from=builder /app/.venv /app/.venv

# Copy application code
COPY --chown=app:app . .

# Ensure proper permissions
RUN chown -R app:app /app

# Switch to app user
USER app

# Create directories for logs and data
RUN mkdir -p /app/logs /app/data

# Expose default port
EXPOSE 8080

# Default command - start with config file
CMD ["uv", "run", "main.py", "--config", "config.toml"]

# Labels for metadata
LABEL \
    org.opencontainers.image.title="Futura Engine" \
    org.opencontainers.image.description="ML-powered Kubernetes optimization engine" \
    org.opencontainers.image.version="0.1.0" \
    org.opencontainers.image.vendor="Futura" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.source="https://github.com/futura/engine"