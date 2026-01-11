# Shippy Test Suite

Comprehensive Bats test suite for testing all Shippy commands and functionality.

## Prerequisites

- [Bats](https://github.com/bats-core/bats-core) >= 1.5.0
- Bats libraries: bats-assert, bats-file, bats-support
- Docker (for integration tests)
- SSH keys in `tests/ssh_keys/` (for integration tests)

### Installing Bats

**macOS:**
```bash
brew install bats-core
brew install bats-assert bats-file bats-support
```

**Linux:**
```bash
# Install bats-core
git clone https://github.com/bats-core/bats-core.git
cd bats-core
./install.sh /usr/local

# Install bats libraries
mkdir -p /usr/lib/bats
git clone https://github.com/bats-core/bats-assert /usr/lib/bats/bats-assert
git clone https://github.com/bats-core/bats-file /usr/lib/bats/bats-file
git clone https://github.com/bats-core/bats-support /usr/lib/bats/bats-support
```

## Test Structure

```
tests/
├── test_helpers.bash          # Shared test helper functions
├── test_root.bats             # Root command tests (7 tests)
├── test_env.bats              # Env command tests (6 tests)
├── test_config.bats           # Config validation/show tests (13 tests)
├── test_init.bats             # Init command tests (12 tests)
├── test_deploy.bats           # Deploy command tests (11 tests)
├── test_unlock.bats           # Unlock command tests (6 tests)
├── test_integration.bats      # Full integration tests (2 tests)
└── config-test/
    ├── minimal.yaml           # Minimal valid config
    ├── full.yaml              # Full config with all options
    ├── multi-host.yaml        # Multiple hosts config
    ├── invalid-syntax.yaml    # Invalid YAML for error testing
    └── missing-fields.yaml    # Missing required fields
```

## Running Tests

### **IMPORTANT: Do NOT run tests in parallel**

Tests share resources (Docker containers, temporary directories) and **must run serially**.

### All Tests (Serial Execution)

```bash
# From project root
bats tests/

# Or using make
make test
```

**⚠️ WARNING: Never use `--jobs` flag - it will cause test conflicts!**

```bash
# ❌ DON'T DO THIS - will cause failures
bats --jobs 4 tests/

# ✅ DO THIS - serial execution
bats tests/
```

### Quick Tests Only (No Docker Required)

```bash
# Fast unit-style tests only
bats --filter-tags quick tests/

# Or using make
make test-quick
```

### Integration Tests Only (Requires Docker)

```bash
# Full deployment tests with Docker
bats --filter-tags integration tests/

# Or using make
make test-integration
```

### Single Test File

```bash
bats tests/test_env.bats
bats tests/test_root.bats
bats tests/test_config.bats
```

### With Verbose Output

```bash
# Show all output (useful for debugging)
bats tests/ --show-output-of-passing-tests --verbose-run --print-output-on-failure

# Or using make
make test-verbose
```

## Test Tags

Tests are organized using tags for selective execution:

- **`quick`** - Fast unit-style tests (no Docker/SSH required)
- **`integration`** - Full deployment tests (requires Docker/SSH)
- **`slow`** - Tests that take significant time
- **`bats:serial`** - Tests that must run serially (not in parallel)

### Filter by Tags

```bash
# Run only quick tests
bats --filter-tags quick tests/

# Exclude integration tests
bats --filter-tags '!integration' tests/

# Run only integration tests
bats --filter-tags integration tests/
```

## Test Coverage

### Commands Tested

- ✅ `shippy` - Root command and help
- ✅ `shippy env` - Environment variables
- ✅ `shippy config validate` - Configuration validation
- ✅ `shippy config show` - Configuration display
- ✅ `shippy init` - Configuration initialization
- ✅ `shippy deploy` - Deployment (unit and integration)
- ✅ `shippy unlock` - Lock removal

### Scenarios Covered

- ✅ Help output for all commands
- ✅ All command-line flags and arguments
- ✅ Configuration file validation
- ✅ Error handling (missing files, invalid configs, etc.)
- ✅ Default values application
- ✅ Configuration generation
- ✅ Full deployment workflow (integration tests)

## Continuous Integration

For CI environments, use serial execution without the `--jobs` flag:

```yaml
# GitHub Actions example
- name: Run tests
  run: bats tests/

# GitLab CI example
test:
  script:
    - bats tests/
```

## Troubleshooting

### Tests Failing with "No such file or directory: dist/tinnie"

Build the binary first:
```bash
make build
# or
mkdir -p dist && go build -o dist/tinnie
```

### Integration Tests Failing

1. Ensure Docker is running:
   ```bash
   docker info
   ```

2. Check SSH keys exist:
   ```bash
   ls -la tests/ssh_keys/tinnie_key
   ```

3. Run Docker Compose manually:
   ```bash
   cd tests
   docker compose -f docker-compose.yaml up -d
   ```

### Tests Conflicting with Each Other

**Always run tests serially** - never use the `--jobs` flag:

```bash
# ✅ Correct
bats tests/

# ❌ Wrong - will cause conflicts
bats --jobs 4 tests/
```

## Writing New Tests

1. Add tests to appropriate `test_*.bats` file or create a new one
2. Use shared helpers from `test_helpers.bash`
3. Tag tests appropriately (`quick`, `integration`, `slow`)
4. Add `bats:serial` tag if test shares resources
5. Ensure tests clean up after themselves in `teardown()`

Example:
```bash
#!/usr/bin/env bats

# bats file_tags=quick

bats_require_minimum_version 1.5.0

setup() {
  load test_helpers
  common_setup
}

@test "My test description" {
  run -0 ${BIN} some-command
  assert_success
  assert_output --partial "expected output"
}
```

## Additional Resources

- [Bats Documentation](https://bats-core.readthedocs.io/)
- [bats-assert Documentation](https://github.com/bats-core/bats-assert)
- [bats-file Documentation](https://github.com/bats-core/bats-file)
