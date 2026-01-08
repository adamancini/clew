# Filesystem Reader is Now the Default

**Effective:** v0.2.0+

## Summary

The filesystem reader is now the default method for reading Claude Code configuration. Users no longer need to use the `--filesystem` flag.

## What Changed

### Before (v0.1.0)
```bash
# CLI reader was default (broken - issue #34)
clew export              # Used broken CLI reader
clew export --filesystem # Used working filesystem reader
```

### After (v0.2.0+)
```bash
# Filesystem reader is now default
clew export              # Uses working filesystem reader  ✅
clew export --cli        # Uses experimental CLI reader (broken)
```

## Flags

### New Flag
- `--cli` - Use claude CLI instead of filesystem (experimental, currently broken - see issue #34)

### Deprecated Flags
- `--filesystem` or `-f` - Still works for backward compatibility but no longer needed
- `--read-from-filesystem` - Still works for backward compatibility but no longer needed

## Usage

### Standard Usage (Recommended)
```bash
clew export
clew status
clew diff
clew sync
```

### Backward Compatible (Still Works)
```bash
# These still work but flag is unnecessary
clew export --filesystem
clew status -f
clew diff --read-from-filesystem
```

### CLI Reader (Not Recommended - Broken)
```bash
# Only use if testing CLI reader functionality
clew export --cli  # Will fail until issue #34 is fixed
```

## Why This Change?

### Problem
The CLI reader attempted to call `claude plugin list --json`, but this command doesn't exist:

```bash
$ claude plugin list --json
error: unknown command 'list'
```

### Impact Before Change
- Without `--filesystem`: Showed incorrect state (54 plugins to add when already installed)
- With `--filesystem`: Showed correct state
- Required users to remember `--filesystem` flag for every command

### Solution
Make filesystem reader the default since:
1. ✅ It works correctly
2. ✅ It's more reliable (reads directly from JSON files)
3. ✅ It doesn't depend on Claude CLI having specific commands
4. ✅ It's faster (no CLI subprocess overhead)
5. ⚠️ CLI reader may never work (unclear if `claude plugin list` will be added)

## Issue #34 Status

**Title:** CLI Reader fails: claude plugin list command doesn't exist
**Status:** Open
**URL:** https://github.com/adamancini/clew/issues/34

**Options:**
1. Wait for Claude CLI to add `plugin list` command (may never happen)
2. Remove CLI reader entirely (breaking change)
3. ✅ **Keep CLI reader opt-in via `--cli` flag** (current approach)

## Testing

### E2E Tests
- ✅ Filesystem reader tests run normally
- ⏭️ CLI reader tests skip automatically until issue #34 is fixed
- ✅ Tests will auto-enable when `claude plugin list` exists

### Backward Compatibility
All existing scripts using `--filesystem` flag will continue to work. The flag is deprecated but functional.

## Migration Guide

### For Users
No action required! Commands work better by default now.

### For Scripts
**Option 1 (Recommended):** Remove unnecessary `--filesystem` flags
```bash
# Before
clew export --filesystem > config.yaml

# After
clew export > config.yaml
```

**Option 2:** Keep flags for backward compatibility
```bash
# Still works, just deprecated
clew export --filesystem > config.yaml
```

### For CI/CD
No changes needed. Both approaches work:

```yaml
# Modern (recommended)
- run: clew status

# Legacy (still works)
- run: clew status --filesystem
```

## Documentation Updates

Updated documentation:
- ✅ `TEST_RESULTS.md` - Noted fix in v0.2.0
- ✅ `E2E_TESTS.md` - Updated CLI reader test strategy
- ✅ `CLI_READER_TESTS.md` - Documented auto-enable strategy
- ✅ Command help text - Shows deprecated flags
- ✅ This document - Complete migration guide

## Future Considerations

### If Issue #34 Gets Fixed
1. CLI reader tests will automatically start running
2. Users can opt into CLI reader with `--cli` flag if desired
3. Filesystem reader remains the default (more reliable)

### Potential Future Changes
- Remove CLI reader entirely if never fixed (v2.0.0?)
- Remove deprecated `--filesystem` flags (v2.0.0?)
- Add `--claude-dir` flag for test isolation

## Related Links

- [Issue #34](https://github.com/adamancini/clew/issues/34) - CLI reader bug
- [CLI_READER_TESTS.md](./test/e2e/CLI_READER_TESTS.md) - Auto-enable test strategy
- [E2E_TESTS.md](./E2E_TESTS.md) - Complete e2e test documentation
- [TEST_RESULTS.md](./TEST_RESULTS.md) - Manual functional test results
