# Futura Engine Makefile
# Development and testing utilities

.PHONY: help install test test-unit test-integration test-e2e test-coverage clean lint format check-format

# Default target
help:
	@echo "Futura Engine Development Commands"
	@echo "=================================="
	@echo ""
	@echo "Setup:"
	@echo "  install          Install package and dev dependencies"
	@echo "  install-dev      Install in development mode"
	@echo ""
	@echo "Testing:"
	@echo "  test             Run all tests"
	@echo "  test-unit        Run unit tests only"
	@echo "  test-integration Run integration tests only"
	@echo "  test-e2e         Run end-to-end tests only"
	@echo "  test-coverage    Run tests with coverage report"
	@echo "  test-fast        Run only fast tests (skip slow)"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint             Run linting checks"
	@echo "  format           Format code with black and isort"
	@echo "  check-format     Check code formatting"
	@echo ""
	@echo "Cleanup:"
	@echo "  clean            Clean generated files and caches"
	@echo ""

# Installation
install:
	pip install -e ".[dev]"

install-dev: install
	@echo "Development environment ready!"

# Testing
test:
	python3 -m pytest tests/ -v

test-unit:
	python3 -m pytest tests/unit/ -v

test-integration:
	python3 -m pytest tests/integration/ -v

test-e2e:
	python3 -m pytest tests/e2e/ -v

test-coverage:
	python3 -m pytest tests/ --cov=. --cov-report=term-missing --cov-report=html

test-fast:
	python3 -m pytest tests/ -m "not slow" -v

# Code quality
lint:
	@echo "Running linting checks..."
	python3 -m flake8 . --count --select=E9,F63,F7,F82 --show-source --statistics
	python3 -m flake8 . --count --exit-zero --max-complexity=10 --max-line-length=127 --statistics

format:
	@echo "Formatting code..."
	python3 -m black . --line-length=100
	python3 -m isort . --profile=black

check-format:
	@echo "Checking code formatting..."
	python3 -m black . --check --line-length=100
	python3 -m isort . --check-only --profile=black

# Cleanup
clean:
	@echo "Cleaning up..."
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} +
	rm -rf build/
	rm -rf dist/
	rm -rf .coverage
	rm -rf htmlcov/
	rm -rf .pytest_cache/

# Development shortcuts
dev-setup: install-dev
	@echo "Installing additional dev tools..."
	pip install black isort flake8

# Test specific components
test-scaling:
	python3 -m pytest tests/unit/test_scaling_algorithms.py -v

test-models:
	python3 -m pytest tests/unit/test_pytorch_models.py -v

test-training:
	python3 -m pytest tests/unit/test_training_job_manager.py -v

test-grpc:
	python3 -m pytest tests/e2e/test_grpc_services.py -v

# CI/CD helpers
ci-test: test-coverage lint

# Docker testing (if needed)
test-docker:
	docker build -t futura-engine-test .
	docker run --rm futura-engine-test python3 -m pytest tests/ -v

# Performance testing
test-benchmark:
	python3 -m pytest tests/ --benchmark-only

# Debug testing
test-debug:
	python3 -m pytest tests/ -vvv --tb=long -s

# Watch tests (requires pytest-watch)
test-watch:
	ptw tests/ -- -v