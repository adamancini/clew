# E2E Testing for clew

## Overview

End-to-end tests validate the clew CLI commands in real-world scenarios. These tests build the binary and execute CLI commands, verifying outputs and exit codes.

## Running E2E Tests

```bash
# Run all e2e tests
make test-e2e

# Run with verbose output
make test-e2e-verbose

# Run all tests (unit + e2e)
make test-all
```

## Test Structure

```
test/e2e/
├── e2e_test.go          # Main e2e test suite
└── fixtures/             # Test data
    ├── installed_plugins.json
    ├── known_marketplaces.json
    ├── complete-clewfile.yaml
    ├── minimal-clewfile.yaml
    ├── invalid-source-clewfile.yaml
    └── empty-clewfile.yaml
```

## Test Coverage

### Export Command
- ✅ Export with text (YAML) output (filesystem reader)
- ✅ Export with JSON output (filesystem reader)
- ✅ Export with YAML output explicit (filesystem reader)
- ✅ Validates output structure
- ⏭️ Export without --filesystem flag (skipped until issue #34 fixed)

### Status Command
- ✅ Status with in-sync state (filesystem reader)
- ✅ Status with out-of-sync state (filesystem reader)
- ✅ JSON output format (filesystem reader)
- ✅ YAML output format (filesystem reader)
- ⏭️ Status without --filesystem flag (skipped until issue #34 fixed)

### Diff Command
- ✅ Diff with matching state (filesystem reader)
- ✅ Diff with differences (filesystem reader)
- ✅ JSON output format (filesystem reader)
- ✅ Shows "not in Clewfile" items
- ⏭️ Diff without --filesystem flag (skipped until issue #34 fixed)

### Sync Command
- ✅ Sync when already in sync (filesystem reader)
- ✅ Sync with verbose output (filesystem reader)
- ✅ Shows Clewfile path and scope
- ⏭️ Sync without --filesystem flag (skipped until issue #34 fixed)

### CLI Reader Tests (Issue #34)
- ⏭️ Export command without --filesystem flag
- ⏭️ Status command without --filesystem flag
- ⏭️ Diff command without --filesystem flag
- ⏭️ Sync command without --filesystem flag
- ⏭️ CLI reader vs filesystem reader output comparison

**Note:** These tests are currently skipped with message: "Skipping: CLI reader not working yet (see issue #34)"
They will **automatically start running** once issue #34 is fixed!

### Validation
- ✅ Invalid marketplace source detection
- ✅ Missing Clewfile error handling
- ✅ Clear error messages

### Version Command
- ✅ Version output format

### Output Formats
- ✅ Text (default) format
- ✅ JSON format validation
- ✅ YAML format validation
- ✅ Format flag on all commands

## Current Limitations

### CLI Reader Status (Issue #34)
The CLI reader attempts to call `claude plugin list --json`, but this command doesn't exist in the Claude CLI.

**Test Strategy:**
- ✅ **Filesystem reader tests** run normally with `--filesystem` flag
- ⏭️ **CLI reader tests** are automatically skipped until issue #34 is resolved
- 🎯 **Automatic detection:** Tests check if `claude plugin list` works before running
- 🚀 **Zero maintenance:** When issue #34 is fixed, tests will automatically start running

**Skip Detection Logic:**
```go
func cliReaderWorks(t *testing.T) bool {
    cmd := exec.Command("claude", "plugin", "list", "--json")
    // Returns false if command outputs "unknown command"
    // Returns true otherwise (CLI reader should work)
}
```

**Benefits of This Approach:**
1. Tests document expected behavior once CLI reader is fixed
2. No manual test updates needed when issue is resolved
3. Clear indication of what needs to work (via skipped tests)
4. Regression protection once feature is working

### Test Isolation
The current e2e tests have limited isolation:
- Tests use fixtures but commands read from `~/.claude` by default
- Some tests may be affected by the actual system state
- Future improvement: Add `--claude-dir` flag to clew for full test isolation

## Test Patterns

### Basic Command Test
```go
func TestMyCommand(t *testing.T) {
    stdout, stderr, err := runClew(t, "command", "--flag", "value")
    if err != nil {
        t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
    }

    if !strings.Contains(stdout, "expected") {
        t.Errorf("expected output, got: %s", stdout)
    }
}
```

### JSON Output Validation
```go
var result map[string]interface{}
if err := json.Unmarshal([]byte(stdout), &result); err != nil {
    t.Fatalf("output is not valid JSON: %v", err)
}

if _, ok := result["field"]; !ok {
    t.Error("expected field in JSON output")
}
```

### Error Handling Test
```go
_, stderr, err := runClew(t, "command", "--invalid")
if err == nil {
    t.Fatal("expected command to fail")
}

if !strings.Contains(stderr, "error message") {
    t.Errorf("expected error message, got: %s", stderr)
}
```

## CI/CD Integration

The e2e tests run in GitHub Actions CI on:
- Multiple Go versions (1.21, 1.22, 1.23)
- Multiple platforms (Ubuntu, macOS)
- Every push to main
- Every pull request

## Adding New Tests

1. **Add fixtures** if needed:
   ```bash
   echo "..." > test/e2e/fixtures/my-test-clewfile.yaml
   ```

2. **Write test function** in `e2e_test.go`:
   ```go
   func TestNewFeature(t *testing.T) {
       t.Run("description", func(t *testing.T) {
           stdout, stderr, err := runClew(t, "command", "args...")
           // assertions
       })
   }
   ```

3. **Run tests locally**:
   ```bash
   make test-e2e
   ```

4. **Verify CI passes** after pushing

## Future Improvements

### High Priority
1. **Add `--claude-dir` flag** to clew
   - Allow specifying Claude directory location
   - Enable full test isolation
   - Remove dependency on ~/.claude for tests

2. **Fix CLI reader** (issue #34)
   - Add tests for CLI reader once fixed
   - Compare CLI reader vs filesystem reader outputs

### Medium Priority
3. **Add integration tests**
   - Test actual sync operations (requires mock or isolated environment)
   - Test add/remove commands once implemented
   - Test with real marketplace cloning

4. **Add performance benchmarks**
   - Measure export/diff/sync performance
   - Track performance over time
   - Identify regressions

### Low Priority
5. **Add test coverage reporting**
   - Track e2e test coverage separately from unit tests
   - Generate coverage reports for e2e paths

6. **Add chaos testing**
   - Test with malformed JSON files
   - Test with corrupt Git repos
   - Test with network issues (for marketplace operations)

## Debugging Failed Tests

### View Full Output
```bash
go test -v ./test/e2e/... 2>&1 | less
```

### Run Single Test
```bash
go test -v -run TestExportCommand ./test/e2e/...
```

### Debug with Print Statements
```go
t.Logf("stdout: %s", stdout)
t.Logf("stderr: %s", stderr)
```

### Check Built Binary
```bash
./clew --version
./clew export --filesystem --output json | jq .
```

## Test Maintenance

- **Update fixtures** when plugin structure changes
- **Update expected outputs** when command formats change
- **Add new tests** for new commands and flags
- **Remove tests** for deprecated functionality
- **Keep test data minimal** - only what's needed to verify behavior

## Related Documentation

- [TEST_RESULTS.md](./TEST_RESULTS.md) - Manual functional test results
- [CLI_READER_TESTS.md](./test/e2e/CLI_READER_TESTS.md) - CLI reader auto-enable strategy
- [CLAUDE.md](./CLAUDE.md) - Project overview and build commands
- [GitHub Issue #34](https://github.com/adamancini/clew/issues/34) - CLI reader bug
