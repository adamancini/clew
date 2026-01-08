# E2E Tests

End-to-end tests for the clew CLI.

## Quick Start

```bash
# From project root
make test-e2e
```

## Test Files

- `e2e_test.go` - Main test suite with all e2e tests
- `fixtures/` - Test data and sample Clewfiles

## What's Tested

- Export command (text, JSON, YAML outputs)
- Status command (sync state detection)
- Diff command (change preview)
- Sync command (reconciliation)
- Validation (error handling)
- Version command
- Output format flags

## Current Status

✅ **Working Tests:**
- Export command (all output formats)
- Validation (error handling)
- Version command
- Output format flags

⏭️ **Skipped Tests (will auto-run when issue #34 is fixed):**
- CLI reader tests (export, status, diff, sync without --filesystem)
- CLI reader vs filesystem reader comparison tests

⚠️ **Known Issues:**
- Some filesystem reader tests fail due to fixture/state mismatch
- CLI reader tests properly skipped until `claude plugin list` command exists

## Documentation

See [E2E_TESTS.md](../../E2E_TESTS.md) for:
- Detailed test coverage
- Test patterns
- Adding new tests
- CI/CD integration
- Debugging tips
- Future improvements
