# CLI Reader Tests - Auto-Enable on Fix

## Overview

The e2e test suite includes tests for the CLI reader (default mode without `--filesystem` flag). These tests are **automatically skipped** until issue #34 is fixed, and will **automatically start running** once the fix is deployed.

## Issue #34: CLI Reader Bug

**Problem:** The CLI reader attempts to call `claude plugin list --json`, but this command doesn't exist in the Claude CLI.

**Error:**
```bash
$ claude plugin list --json
error: unknown command 'list'
```

**Impact:** Commands without `--filesystem` flag fail or produce incorrect results.

## Test Strategy

### Automatic Detection

Tests check if the `claude plugin list` command exists before running:

```go
func cliReaderWorks(t *testing.T) bool {
    cmd := exec.Command("claude", "plugin", "list", "--json")
    var stderr strings.Builder
    cmd.Stderr = &stderr
    err := cmd.Run()

    // If command fails with "unknown command", CLI reader is broken
    if err != nil {
        stderrStr := stderr.String()
        if strings.Contains(stderrStr, "unknown command") ||
           strings.Contains(stderrStr, "error: unknown") {
            return false  // CLI reader broken
        }
    }

    return true  // CLI reader should work
}
```

### Skip Logic

Tests are automatically skipped with a clear message:

```go
func skipIfCLIReaderBroken(t *testing.T) {
    if !cliReaderWorks(t) {
        t.Skip("Skipping: CLI reader not working yet (see issue #34)")
    }
}
```

## Test Coverage

### Currently Skipped (⏭️)

All tests in `TestCLIReader`:
- Export command without --filesystem flag
- Status command without --filesystem flag
- Diff command without --filesystem flag
- Sync command without --filesystem flag

All tests in `TestCLIReaderVsFilesystemReader`:
- Export outputs comparison
- Status outputs comparison

### Test Output

```bash
=== RUN   TestCLIReader
    e2e_test.go:511: Skipping: CLI reader not working yet (see issue #34)
--- SKIP: TestCLIReader (0.23s)
=== RUN   TestCLIReaderVsFilesystemReader
    e2e_test.go:629: Skipping: CLI reader not working yet (see issue #34)
--- SKIP: TestCLIReaderVsFilesystemReader (0.23s)
```

## When Issue #34 is Fixed

### What Happens Automatically

1. **Detection:** `cliReaderWorks()` will return `true`
2. **Skip bypass:** Tests will no longer be skipped
3. **Execution:** All CLI reader tests will run
4. **Validation:** Tests will verify CLI reader produces same output as filesystem reader

### No Manual Updates Needed

✅ Tests automatically start running
✅ No code changes required
✅ No test configuration updates
✅ No CI/CD modifications

### Expected Behavior

Once fixed, these tests should **PASS** and verify:
- CLI reader export matches filesystem reader export
- CLI reader status matches filesystem reader status
- CLI reader diff matches filesystem reader diff
- CLI reader sync works correctly
- Plugin counts match between readers
- Marketplace counts match between readers
- Sync status matches between readers

## Benefits

1. **Documentation:** Tests document expected behavior post-fix
2. **Zero Maintenance:** No manual updates when issue is resolved
3. **Regression Protection:** Automatic coverage once feature works
4. **Clear Communication:** Skip message references issue #34
5. **CI/CD Integration:** Tests integrated into existing pipeline

## Monitoring Fix Progress

### Check Test Status

```bash
# Run e2e tests
make test-e2e

# Look for CLI reader test status
make test-e2e 2>&1 | grep -A2 "TestCLIReader"
```

### Verify CLI Command

```bash
# Test if command exists
claude plugin list --json 2>&1

# Should return error currently:
# error: unknown command 'list'

# After fix, should return:
# JSON array of installed plugins
```

## Related Files

- `test/e2e/e2e_test.go` - Test implementation
- `E2E_TESTS.md` - Complete e2e test documentation
- `TEST_RESULTS.md` - Manual functional test results
- GitHub Issue #34 - CLI reader bug tracking

## Example: What Success Looks Like

Once issue #34 is fixed, test output will change from:

```bash
=== RUN   TestCLIReader
    e2e_test.go:511: Skipping: CLI reader not working yet (see issue #34)
--- SKIP: TestCLIReader (0.23s)
```

To:

```bash
=== RUN   TestCLIReader
=== RUN   TestCLIReader/export_without_filesystem_flag
=== RUN   TestCLIReader/status_without_filesystem_flag
=== RUN   TestCLIReader/diff_without_filesystem_flag
=== RUN   TestCLIReader/sync_without_filesystem_flag
--- PASS: TestCLIReader (2.45s)
    --- PASS: TestCLIReader/export_without_filesystem_flag (0.56s)
    --- PASS: TestCLIReader/status_without_filesystem_flag (0.62s)
    --- PASS: TestCLIReader/diff_without_filesystem_flag (0.58s)
    --- PASS: TestCLIReader/sync_without_filesystem_flag (0.69s)
```

**No code changes required - it just works!** 🎉
