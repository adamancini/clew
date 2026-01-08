# clew v0.2.0 Functional Test Results

**Date:** 2026-01-08
**Version:** v0.1.0-15-g9ae18d0 (pre-0.2.0)
**Platform:** darwin (macOS)
**Tester:** ada

## Executive Summary

Core functionality is working well with the filesystem reader. The CLI reader has a critical bug that requires the `--filesystem` flag for all operations. Several features are not yet implemented.

## Test Environment

- **Current Configuration:**
  - 4 marketplaces
  - 56 plugins (mix of user and project scope)
  - 0 standalone MCP servers (plugins provide MCP internally)
- **Test Clewfile:** `~/.claude/Clewfile.yaml` (exported from current state)

## ✅ Working Features

### 1. Export Command
**Status:** ✅ Fully Working

```bash
clew export --filesystem
```

**Results:**
- ✅ Exports all 4 marketplaces correctly
- ✅ Exports all 56 plugins with:
  - Correct enabled/disabled state
  - Correct scope (user vs project)
  - Marketplace source correctly identified
- ✅ Output formats work: text (YAML), JSON, YAML

**Example:**
```bash
clew export --filesystem --output json > config.json
clew export --filesystem --output yaml > config.yaml
clew export --filesystem > Clewfile.yaml  # default YAML
```

### 2. Status Command
**Status:** ✅ Working (with --filesystem flag)

```bash
clew status --filesystem
```

**Results:**
- ✅ Correctly reports "In sync" when Clewfile matches current state
- ✅ Correctly reports "Out of sync" with counts when differences exist
- ✅ Output formats work: text, JSON, YAML

**Example Output:**
```
Status: In sync
```

```json
{
  "in_sync": true,
  "add": 0,
  "update": 0,
  "remove": 0,
  "attention": 0
}
```

### 3. Diff Command
**Status:** ✅ Working (with --filesystem flag)

```bash
clew diff --filesystem
```

**Results:**
- ✅ Correctly identifies items to add
- ✅ Correctly identifies items to update
- ✅ Correctly identifies items not in Clewfile as "need attention" (non-destructive)
- ✅ Shows detailed changes for marketplaces and plugins
- ✅ Output formats work: text, JSON, YAML

**Example:**
```
Changes that would be made:

Marketplaces:
  - claude-plugins-official: remove (not in Clewfile)
  - claude-code-workflows: remove (not in Clewfile)

Plugins:
  - pr-review-toolkit@claude-code-plugins: remove (not in Clewfile)
  ...

Summary: 0 to add, 0 to update, 0 to remove, 52 need attention
```

### 4. Sync Command
**Status:** ✅ Working (with --filesystem flag)

```bash
clew sync --filesystem
```

**Results:**
- ✅ Correctly detects "already in sync" state
- ✅ Verbose mode shows Clewfile location and inferred scope
- ✅ `--strict` flag available for non-zero exit on failures

**Example:**
```bash
$ clew sync --filesystem --verbose
Already in sync. Nothing to do.
Using Clewfile: /Users/ada/.claude/Clewfile.yaml
Inferred scope: user
```

### 5. Validation
**Status:** ✅ Working

**Results:**
- ✅ Validates marketplace source types (github, local)
- ✅ Clear error messages for invalid configurations
- ✅ Detects missing Clewfile

**Example:**
```bash
$ clew status --config /tmp/invalid.yaml --filesystem
Error: validation errors:
  - marketplaces.invalid-marketplace.source: invalid source 'invalid_source' (must be github or local)
```

### 6. Output Formats
**Status:** ✅ Working

**Supported Formats:**
- ✅ `text` (default) - Human-readable YAML
- ✅ `json` - Machine-readable JSON
- ✅ `yaml` - YAML format

All commands support `-o/--output` flag with all three formats.

### 7. Scope Inference
**Status:** ✅ Working

**Results:**
- ✅ Correctly infers `user` scope for `~/.claude/Clewfile.yaml`
- ✅ Would infer `project` scope for project directories (not tested in detail)

## ❌ Issues Found

### 1. CLI Reader Broken (Critical)
**Issue:** https://github.com/adamancini/clew/issues/34

**Problem:**
The CLI reader attempts to call `claude plugin list --json` but this command doesn't exist in the Claude CLI.

**Impact:**
- Without `--filesystem` flag: All commands report incorrect state
- Status shows "Out of sync" when actually in sync
- Diff shows all plugins need to be added even when installed

**Workaround:**
Always use `--filesystem` flag:
```bash
clew status --filesystem
clew diff --filesystem
clew sync --filesystem
clew export --filesystem
```

**Evidence:**
```bash
$ claude plugin list --json
error: unknown command 'list'

$ clew status  # Without --filesystem
Status: Out of sync
  To add:       54  # INCORRECT - plugins already installed

$ clew status --filesystem  # With --filesystem
Status: In sync  # CORRECT
```

**Root Cause:**
The Claude CLI doesn't provide programmatic plugin listing. Available commands:
- `claude plugin install`
- `claude plugin uninstall`
- `claude plugin enable`
- `claude plugin disable`
- `claude plugin update`
- `claude plugin marketplace`

No `list` command exists.

**Recommendation:**
Remove CLI reader and make filesystem reader the default. The Claude CLI doesn't support the required operations.

## 🚧 Not Yet Implemented

### 1. Add Commands
**Status:** ❌ Not Implemented

```bash
$ clew add plugin superpowers@superpowers-marketplace
add plugin superpowers@superpowers-marketplace not yet implemented
```

**Affects:**
- `clew add marketplace`
- `clew add plugin`
- `clew add mcp`

### 2. Remove Commands
**Status:** ❌ Not Implemented

```bash
$ clew remove plugin superpowers@superpowers-marketplace
remove plugin superpowers@superpowers-marketplace not yet implemented
```

**Affects:**
- `clew remove plugin`
- `clew remove mcp`

### 3. Init Command
**Status:** ❌ Not Implemented

```bash
$ clew init
Error: unknown command "init" for "clew"
```

The `init` command mentioned in CLAUDE.md doesn't exist yet.

### 4. Sync with Dry-Run
**Status:** ⚠️ No explicit --dry-run flag

The `sync` command doesn't have a `--dry-run` flag. Users must use `clew diff` to preview changes before running `clew sync`.

**Workaround:**
```bash
clew diff --filesystem  # Preview changes
clew sync --filesystem  # Apply changes
```

## 📊 Test Coverage

| Feature | Status | Notes |
|---------|--------|-------|
| Export current state | ✅ | Works with --filesystem |
| Status check | ✅ | Works with --filesystem |
| Diff/preview changes | ✅ | Works with --filesystem |
| Sync to Clewfile | ✅ | Works with --filesystem |
| CLI reader | ❌ | Broken - requires --filesystem |
| Filesystem reader | ✅ | Fully working |
| Output formats | ✅ | text, json, yaml all work |
| Validation | ✅ | Catches invalid configs |
| Scope inference | ✅ | user/project detection works |
| Add commands | ❌ | Not implemented |
| Remove commands | ❌ | Not implemented |
| Init command | ❌ | Not implemented |
| Version reporting | ✅ | Shows v0.1.0-15-g9ae18d0 |

## 🔍 Additional Observations

### Version Reporting
Current binary reports `v0.1.0-15-g9ae18d0` (15 commits past v0.1.0 tag). The v0.2.0 release tag hasn't been created yet.

### Non-Destructive Behavior
✅ Confirmed: Items not in Clewfile are reported as "need attention" but NOT automatically removed. This matches the design decision documented in CLAUDE.md.

### Error Handling
✅ Good error messages for:
- Missing Clewfile
- Invalid YAML syntax
- Invalid configuration values
- Missing required fields

## 📝 Recommendations

### High Priority
1. **Fix CLI reader bug** (#34)
   - Remove CLI reader OR
   - Make filesystem reader the default OR
   - Document that CLI approach won't work

2. **Update version** to v0.2.0
   - Create git tag: `git tag v0.2.0`
   - Rebuild: `make build`

### Medium Priority
3. **Implement add/remove commands**
   - These are scaffolded but not implemented
   - Would improve UX significantly

4. **Add --dry-run to sync**
   - Or document that users should use `diff` before `sync`

### Low Priority
5. **Implement init command**
   - Would help new users bootstrap Clewfiles
   - Templates for common configurations

## ✅ Overall Assessment

**Core functionality is solid.** The filesystem reader works well for all read operations (export, status, diff, sync). The CLI reader bug is critical but has a simple workaround.

**Recommendation:** This is ready for a v0.2.0 release with:
- Clear documentation about using `--filesystem` flag
- Issue #34 tracked for CLI reader fix
- Documentation noting add/remove/init are planned features

The export → edit → diff → sync workflow is functional and useful.
