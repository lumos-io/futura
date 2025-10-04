# Futura Engine Test Suite

Comprehensive testing framework for the Futura ML-powered Kubernetes optimization engine.

## Test Structure

```
tests/
├── unit/                    # Unit tests for individual components
│   ├── test_scaling_algorithms.py
│   ├── test_pytorch_models.py
│   └── test_training_job_manager.py
├── integration/             # Integration tests with mocked dependencies
│   └── test_rl_server_integration.py
├── e2e/                     # End-to-end tests for complete workflows
│   └── test_grpc_services.py
├── conftest.py              # Global test configuration and fixtures
└── README.md                # This file
```

## Quick Start

### Prerequisites

1. **Install test dependencies:**

   ```bash
   cd engine
   pip install -e ".[dev]"
   ```

2. **Verify installation:**
   ```bash
   python -m pytest --version
   ```

### Running Tests

#### Run All Tests

```bash
# From engine directory
python -m pytest
```

#### Run by Category

```bash
# Unit tests only
python -m pytest tests/unit/ -v

# Integration tests only
python -m pytest tests/integration/ -v

# End-to-end tests only
python -m pytest tests/e2e/ -v
```

#### Run by Markers

```bash
# Fast unit tests
python -m pytest -m unit

# Integration tests with external deps
python -m pytest -m integration

# Slow end-to-end tests
python -m pytest -m "e2e and not slow"

# All tests except slow ones
python -m pytest -m "not slow"
```

#### Run Specific Test Files

```bash
# Test scaling algorithms
python -m pytest tests/unit/test_scaling_algorithms.py -v

# Test PyTorch models
python -m pytest tests/unit/test_pytorch_models.py -v

# Test gRPC services
python -m pytest tests/e2e/test_grpc_services.py -v
```

## Test Categories

### Unit Tests (`tests/unit/`)

Test individual components in isolation with minimal dependencies.

**Features:**

- ✅ Scaling algorithm logic
- ✅ PyTorch model architecture (mocked)
- ✅ Training job manager
- ✅ Resource state validation
- ✅ Edge case handling

**Run with:**

```bash
python -m pytest tests/unit/ -v
```

### Integration Tests (`tests/integration/`)

Test component interactions with mocked external dependencies.

**Features:**

- ✅ RLServer with mocked ClickHouse/Kubernetes
- ✅ Complete recommendation flow
- ✅ Safety policy enforcement
- ✅ Model management integration
- ✅ Error handling and fallbacks

**Run with:**

```bash
python -m pytest tests/integration/ -v
```

### End-to-End Tests (`tests/e2e/`)

Test complete workflows with full gRPC communication.

**Features:**

- ✅ gRPC service communication
- ✅ Full recommendation pipeline
- ✅ Training trigger to completion
- ✅ Error handling and retries
- ✅ System integration flows

**Run with:**

```bash
python -m pytest tests/e2e/ -v
```

## Test Configuration

### Pytest Configuration (`pytest.ini`)

```ini
[tool:pytest]
testpaths = tests
addopts =
    -v --tb=short --strict-markers
    --asyncio-mode=auto
    --cov=. --cov-report=term-missing
markers =
    unit: Unit tests
    integration: Integration tests
    e2e: End-to-end tests
    slow: Slow running tests
    requires_k8s: Tests requiring Kubernetes
    requires_clickhouse: Tests requiring ClickHouse
```

### Global Fixtures (`conftest.py`)

**Available Fixtures:**

- `mock_clickhouse_client` - Mocked ClickHouse client
- `mock_kubernetes_client` - Mocked Kubernetes client
- `mock_torch` - Mocked PyTorch for CPU testing
- `sample_resource_state` - Sample ResourceState data
- `sample_training_spec` - Sample TrainingJobSpec data
- `mock_grpc_server` - Mocked gRPC server
- `temp_model_dir` - Temporary model storage

**Auto-applied Mocks:**

- External dependencies are automatically mocked
- GPU detection disabled for testing
- Kubernetes config loading mocked

## Mock Strategy

### External Dependencies

All external dependencies are comprehensively mocked:

**ClickHouse:**

```python
@pytest.fixture
def mock_clickhouse_client():
    client = AsyncMock()
    client.execute = AsyncMock()
    client.fetch_rows = AsyncMock()
    client.insert_rows = AsyncMock()
    return client
```

**Kubernetes:**

```python
@pytest.fixture
def mock_kubernetes_client():
    client = Mock()
    client.BatchV1Api().create_namespaced_job = AsyncMock()
    client.AppsV1Api().patch_namespaced_deployment = AsyncMock()
    return client
```

**PyTorch:**

```python
@pytest.fixture
def mock_torch():
    with patch.dict('sys.modules', {
        'torch': Mock(),
        'torch.nn': Mock(),
        'torch.optim': Mock()
    }):
        yield
```

### Data Generation

**Sample Data:**

```python
# Generate metrics for testing
data = generate_metrics_data(num_points=100)

# Create resource states
state = ResourceState(
    num_replicas=3,
    cpu_util=0.75,
    memory_util=0.65,
    request_rate=150.0
)
```

## Test Markers

Use markers to control test execution:

```bash
# Only unit tests
pytest -m unit

# Only integration tests
pytest -m integration

# Skip slow tests
pytest -m "not slow"

# Only tests requiring Kubernetes
pytest -m requires_k8s

# Only tests requiring ClickHouse
pytest -m requires_clickhouse
```

## Coverage Reporting

### Generate Coverage Report

```bash
# Run tests with coverage
python -m pytest --cov=. --cov-report=html

# View HTML report
open htmlcov/index.html
```

### Coverage Targets

- **Unit Tests:** >90% line coverage
- **Integration Tests:** >80% branch coverage
- **Critical Paths:** 100% coverage

## Performance Testing

### Benchmark Tests

```bash
# Run performance benchmarks
python -m pytest tests/ -k benchmark --benchmark-only

# Compare performance
python -m pytest tests/ --benchmark-compare
```

### Load Testing

```bash
# Test with high concurrency
python -m pytest tests/e2e/ -k "load" --asyncio-mode=auto
```

## Debugging Tests

### Verbose Output

```bash
# Maximum verbosity
python -m pytest -vvv --tb=long

# Show print statements
python -m pytest -s

# Stop on first failure
python -m pytest -x
```

### Debug Specific Tests

```bash
# Debug single test
python -m pytest tests/unit/test_scaling_algorithms.py::TestScalingAlgorithms::test_slo_violation_detection -vvv -s

# Debug with pdb
python -m pytest --pdb
```

## Continuous Integration

### GitHub Actions

```yaml
# .github/workflows/test.yml
- name: Run Tests
  run: |
    python -m pytest tests/ --cov=. --cov-report=xml

- name: Upload Coverage
  uses: codecov/codecov-action@v3
```

### Pre-commit Hooks

```bash
# Install pre-commit
pip install pre-commit
pre-commit install

# Run manually
pre-commit run --all-files
```

## Common Test Patterns

### Testing Async Functions

```python
@pytest.mark.asyncio
async def test_async_function():
    result = await async_function()
    assert result is not None
```

### Testing Exceptions

```python
def test_exception_handling():
    with pytest.raises(ValueError, match="Invalid input"):
        function_that_raises()
```

### Parameterized Tests

```python
@pytest.mark.parametrize("input,expected", [
    (1, 2),
    (2, 4),
    (3, 6)
])
def test_multiply_by_two(input, expected):
    assert multiply_by_two(input) == expected
```

### Mock Patching

```python
@patch('module.external_function')
def test_with_mock(mock_func):
    mock_func.return_value = "mocked"
    result = function_using_external()
    assert result == "mocked"
```

## Troubleshooting

### Common Issues

**Import Errors:**

```bash
# Ensure engine is in Python path
export PYTHONPATH="/path/to/engine:$PYTHONPATH"

# Or use development install
pip install -e .
```

**Async Test Issues:**

```bash
# Use asyncio mode
pytest --asyncio-mode=auto

# Or mark individual tests
@pytest.mark.asyncio
```

**Mock Not Working:**

```python
# Use patch context manager
with patch('module.function') as mock:
    mock.return_value = "test"
    # test code
```

### Getting Help

1. **Check test output:** Use `-vvv` for maximum verbosity
2. **Review fixtures:** Check `conftest.py` for available fixtures
3. **Check markers:** Ensure proper test marking
4. **Verify mocks:** Confirm external dependencies are mocked

## Contributing

### Adding New Tests

1. **Choose appropriate category:** unit/integration/e2e
2. **Use existing fixtures:** Leverage `conftest.py` fixtures
3. **Mock external deps:** Always mock external dependencies
4. **Add proper markers:** Mark tests appropriately
5. **Update documentation:** Update this README if needed

### Test Naming

- Use descriptive names: `test_scaling_with_slo_violation`
- Include test category: `test_unit_`, `test_integration_`, `test_e2e_`
- Group related tests in classes: `class TestScalingAlgorithms`

### Best Practices

- ✅ Mock all external dependencies
- ✅ Use appropriate test markers
- ✅ Test both success and failure cases
- ✅ Keep tests focused and independent
- ✅ Use descriptive assertions
- ✅ Maintain test documentation
