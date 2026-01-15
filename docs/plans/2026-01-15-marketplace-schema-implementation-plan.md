# Marketplace Schema Simplification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Simplify Clewfile schema from `sources:` array to `marketplaces:` map, removing SourceKind/SourceType complexity.

**Architecture:** Replace Sources (array with kind/type) with Marketplaces (map with simple repo field). Map keys serve as aliases. Supports GitHub short form (owner/repo) and full URLs (HTTPS/SSH).

**Tech Stack:** Go 1.21+, YAML/TOML/JSON parsing, git worktrees

**Design Doc:** `docs/plans/2026-01-15-marketplace-schema-simplification.md`

**GitHub Issues:** Epic #71, Phases #72-#78

---

## Phase 2: Update Config Structures (#73)

### Task 2.1: Update Marketplace struct in config.go

**Files:**
- Modify: `internal/config/config.go:35-92`

**Step 1: Replace Source/SourceConfig with Marketplace**

Remove old types (lines 13-92), replace with:

```go
// Type aliases for backward compatibility.
type (
	// Scope represents the installation scope.
	Scope = types.Scope
	// TransportType represents the MCP server transport protocol.
	TransportType = types.TransportType
)

// Scope constants - re-exported from types package.
const (
	ScopeUser    = types.ScopeUser
	ScopeProject = types.ScopeProject
)

// Transport type constants - re-exported from types package.
const (
	TransportStdio = types.TransportStdio
	TransportHTTP  = types.TransportHTTP
	TransportSSE   = types.TransportSSE
)

// Marketplace represents a Claude Code marketplace repository.
type Marketplace struct {
	Repo string `yaml:"repo" toml:"repo" json:"repo"` // Repository URL in any format
	Ref  string `yaml:"ref,omitempty" toml:"ref,omitempty" json:"ref,omitempty"` // Optional git ref
}
```

**Step 2: Update Clewfile struct**

Change (around line 76-82):

```go
// OLD
type Clewfile struct {
	Version    int                  `yaml:"version" toml:"version" json:"version"`
	Sources    []Source             `yaml:"sources,omitempty" toml:"sources,omitempty" json:"sources,omitempty"`
	Plugins    []Plugin             `yaml:"plugins" toml:"plugins" json:"plugins"`
	MCPServers map[string]MCPServer `yaml:"mcp_servers" toml:"mcp_servers" json:"mcp_servers"`
}

// NEW
type Clewfile struct {
	Version      int                    `yaml:"version" toml:"version" json:"version"`
	Marketplaces map[string]Marketplace `yaml:"marketplaces,omitempty" toml:"marketplaces,omitempty" json:"marketplaces,omitempty"`
	Plugins      []Plugin               `yaml:"plugins" toml:"plugins" json:"plugins"`
	MCPServers   map[string]MCPServer   `yaml:"mcp_servers" toml:"mcp_servers" json:"mcp_servers"`
}
```

**Step 3: Remove GetSourceByAliasOrName, add GetMarketplace**

Remove old method, add:

```go
// GetMarketplace returns the marketplace for the given name.
// Name is the map key, so this is just a simple lookup.
func (c *Clewfile) GetMarketplace(name string) (*Marketplace, bool) {
	m, ok := c.Marketplaces[name]
	if !ok {
		return nil, false
	}
	return &m, true
}
```

**Step 4: Update Plugin struct**

Remove Source field (around line 100):

```go
// OLD
type Plugin struct {
	Name    string        `yaml:"name" toml:"name" json:"name"`
	Source  *SourceConfig `yaml:"source,omitempty" toml:"source,omitempty" json:"source,omitempty"`
	Enabled *bool         `yaml:"enabled,omitempty" toml:"enabled,omitempty" json:"enabled,omitempty"`
	Scope   string        `yaml:"scope,omitempty" toml:"scope,omitempty" json:"scope,omitempty"`
}

// NEW
type Plugin struct {
	Name    string `yaml:"name" toml:"name" json:"name"`
	Enabled *bool  `yaml:"enabled,omitempty" toml:"enabled,omitempty" json:"enabled,omitempty"`
	Scope   string `yaml:"scope,omitempty" toml:"scope,omitempty" json:"scope,omitempty"`
}
```

**Step 5: Try to build to see what breaks**

```bash
go build ./internal/config
```

Expected: Compilation errors in parser.go, validate.go (references to removed types)

**Step 6: Commit config struct changes**

```bash
git add internal/config/config.go
git commit -m "refactor: Replace Sources with Marketplaces in config structs

- Change Sources []Source to Marketplaces map[string]Marketplace
- Remove SourceConfig, Source structs
- Add simple Marketplace struct with repo + optional ref
- Remove Plugin.Source field (plugins now only reference marketplaces)
- Remove GetSourceByAliasOrName (replaced with simple map lookup)

Part of #73"
```

### Task 2.2: Update parser.go for marketplace parsing

**Files:**
- Modify: `internal/config/parser.go`

**Step 1: Update parseRawClewfile function**

Find where Sources are parsed (around line 220-230), replace with:

```go
// Parse marketplaces
if rawClewfile.Marketplaces != nil {
	clewfile.Marketplaces = rawClewfile.Marketplaces
}
```

**Step 2: Remove source parsing logic**

Delete functions/code related to:
- Source array parsing
- SourceConfig parsing
- Kind/type inference

**Step 3: Test parsing**

```bash
go test ./internal/config -run TestParse -v
```

Expected: Some tests fail (need updating for marketplace format)

**Step 4: Commit parser changes**

```bash
git add internal/config/parser.go
git commit -m "refactor: Update parser for marketplaces map

- Parse marketplaces map instead of sources array
- Remove source parsing logic
- Simpler structure (no kind/type)

Part of #73"
```

### Task 2.3: Update validate.go for marketplace validation

**Files:**
- Modify: `internal/config/validate.go`

**Step 1: Replace validateSources with validateMarketplaces**

Remove `validateSources()` function, add:

```go
func validateMarketplaces(marketplaces map[string]Marketplace) error {
	if len(marketplaces) == 0 {
		return nil // Empty marketplaces is valid
	}

	for name, m := range marketplaces {
		if name == "" {
			return ValidationError{
				Field:   "marketplaces",
				Message: "marketplace name (map key) cannot be empty",
			}
		}

		if m.Repo == "" {
			return ValidationError{
				Field:   fmt.Sprintf("marketplaces.%s.repo", name),
				Message: "repo is required",
			}
		}
	}

	return nil
}
```

**Step 2: Update Validate() function**

Change call from `validateSources(c.Sources)` to `validateMarketplaces(c.Marketplaces)`

**Step 3: Simplify validatePlugin**

Remove inline source validation (lines ~161-178), since plugins no longer have Source field.

Plugin validation should just check:
- Name is non-empty
- Scope is valid (if specified)
- Name matches `plugin@marketplace` pattern
- Referenced marketplace exists

```go
func validatePlugin(index int, p Plugin, marketplaces map[string]Marketplace) error {
	if p.Name == "" {
		return ValidationError{
			Field:   fmt.Sprintf("plugins[%d].name", index),
			Message: "name is required",
		}
	}

	// Validate plugin@marketplace format
	parts := strings.Split(p.Name, "@")
	if len(parts) != 2 {
		return ValidationError{
			Field:   fmt.Sprintf("plugins[%d].name", index),
			Message: "plugin name must be in format 'plugin@marketplace'",
		}
	}

	// Validate marketplace exists
	marketplaceName := parts[1]
	if _, exists := marketplaces[marketplaceName]; !exists {
		return ValidationError{
			Field:   fmt.Sprintf("plugins[%d].name", index),
			Message: fmt.Sprintf("marketplace '%s' not found in marketplaces section", marketplaceName),
		}
	}

	// Validate scope
	if err := types.Scope(p.Scope).Validate(); err != nil {
		return ValidationError{
			Field:   fmt.Sprintf("plugins[%d].scope", index),
			Message: err.Error(),
		}
	}

	return nil
}
```

**Step 4: Update Validate to pass marketplaces to validatePlugin**

Change (around line 40-50):

```go
// Validate plugins
for i, p := range c.Plugins {
	if err := validatePlugin(i, p, c.Marketplaces); err != nil {
		return err
	}
}
```

**Step 5: Test validation**

```bash
go test ./internal/config -run TestValidate -v
```

Expected: Tests fail (fixtures need updating)

**Step 6: Commit validation changes**

```bash
git add internal/config/validate.go
git commit -m "refactor: Simplify validation for marketplaces

- Replace validateSources with validateMarketplaces
- Remove kind/type validation (no longer needed)
- Validate plugins reference existing marketplaces
- Check plugin@marketplace format

Part of #73"
```

**Step 7: Update GitHub issue #73**

```bash
gh issue comment 73 --body "Phase 2 complete:
- Config structs updated (Marketplace, Clewfile)
- Parser updated for map structure
- Validation simplified (no kind/type checks)
- Ready for diff/sync updates"

gh issue close 73 --comment "✓ Phase complete"
```

---

## Phase 3: Update Diff/Sync Logic (#74)

### Task 3.1: Update diff.go types

**Files:**
- Modify: `internal/diff/diff.go`

**Step 1: Rename SourceDiff to MarketplaceDiff**

```go
// OLD
type SourceDiff struct {
	Name    string
	Action  Action
	Current *state.SourceState
	Desired *config.Source
}

// NEW
type MarketplaceDiff struct {
	Name    string
	Action  Action
	Current *state.MarketplaceState
	Desired *config.Marketplace
}
```

**Step 2: Update Result struct**

```go
// OLD
type Result struct {
	Sources    []SourceDiff
	Plugins    []PluginDiff
	MCPServers []MCPServerDiff
}

// NEW
type Result struct {
	Marketplaces []MarketplaceDiff
	Plugins      []PluginDiff
	MCPServers   []MCPServerDiff
}
```

**Step 3: Update Summary() method**

Change references from `r.Sources` to `r.Marketplaces`

**Step 4: Remove PluginDiff.IsLocal() or simplify it**

Since local plugins are gone and we don't track plugin-kind sources differently anymore:

```go
// Remove this entire method - no longer needed
```

**Step 5: Commit diff types**

```bash
git add internal/diff/diff.go
git commit -m "refactor: Rename SourceDiff to MarketplaceDiff

Part of #74"
```

### Task 3.2: Update compute.go

**Files:**
- Modify: `internal/diff/compute.go`

**Step 1: Update Compute function signature**

```go
// Update to use Marketplaces
func Compute(clewfile *config.Clewfile, currentState *state.State) *Result {
	return &Result{
		Marketplaces: computeMarketplaceDiffs(clewfile.Marketplaces, currentState.Marketplaces),
		Plugins:      computePluginDiffs(clewfile.Plugins, currentState.Plugins),
		MCPServers:   computeMCPServerDiffs(clewfile.MCPServers, currentState.MCPServers),
	}
}
```

**Step 2: Rename computeSourceDiffs to computeMarketplaceDiffs**

```go
func computeMarketplaceDiffs(desired map[string]config.Marketplace, current map[string]state.MarketplaceState) []MarketplaceDiff {
	var diffs []MarketplaceDiff
	seen := make(map[string]bool)

	// Check desired marketplaces
	for name, m := range desired {
		seen[name] = true

		if curr, exists := current[name]; exists {
			// Marketplace exists - check if update needed
			if marketplaceChanged(m, curr) {
				diffs = append(diffs, MarketplaceDiff{
					Name:    name,
					Action:  ActionUpdate,
					Current: &curr,
					Desired: &m,
				})
			}
		} else {
			// New marketplace
			diffs = append(diffs, MarketplaceDiff{
				Name:    name,
				Action:  ActionAdd,
				Desired: &m,
			})
		}
	}

	// Check for removed marketplaces (attention only, non-destructive)
	for name, curr := range current {
		if !seen[name] {
			c := curr
			diffs = append(diffs, MarketplaceDiff{
				Name:    name,
				Action:  ActionAttention,
				Current: &c,
			})
		}
	}

	return diffs
}
```

**Step 3: Add marketplaceChanged helper**

```go
func marketplaceChanged(desired config.Marketplace, current state.MarketplaceState) bool {
	// Check if repo URL changed
	if desired.Repo != current.URL {
		return true
	}
	// Check if ref changed
	if desired.Ref != current.Ref {
		return true
	}
	return false
}
```

**Step 4: Simplify computePluginDiffs**

Remove source-related logic from plugin diff computation (plugins don't have inline sources anymore)

**Step 5: Test compilation**

```bash
go build ./internal/diff
```

Expected: May fail due to state.MarketplaceState not existing yet (that's Phase 4)

**Step 6: Commit compute changes**

```bash
git add internal/diff/compute.go
git commit -m "refactor: Update diff computation for marketplaces

- Rename computeSourceDiffs to computeMarketplaceDiffs
- Work with map instead of array
- Simplify plugin diff (no inline sources)
- Add marketplaceChanged helper

Part of #74"
```

### Task 3.3: Update commands.go

**Files:**
- Modify: `internal/diff/commands.go`

**Step 1: Update GenerateCommands for marketplaces**

```go
func (r *Result) GenerateCommands() []Command {
	var commands []Command

	// 1. Add marketplaces first (plugins depend on them)
	for _, m := range r.Marketplaces {
		if m.Action == ActionAdd && m.Desired != nil {
			cmd := fmt.Sprintf("claude plugin marketplace add %s", m.Desired.Repo)
			commands = append(commands, Command{
				Command:     cmd,
				Description: fmt.Sprintf("Add marketplace: %s", m.Name),
			})
		}
	}

	// 2. Install plugins (unchanged)
	for _, p := range r.Plugins {
		switch p.Action {
		case ActionAdd:
			if p.Desired != nil {
				cmd := fmt.Sprintf("claude plugin install %s", p.Name)
				if p.Desired.Scope != "" && p.Desired.Scope != "user" {
					cmd += fmt.Sprintf(" --scope %s", p.Desired.Scope)
				}
				desc := fmt.Sprintf("Install plugin: %s", p.Name)
				commands = append(commands, Command{
					Command:     cmd,
					Description: desc,
				})
			}
		// ... rest of plugin actions unchanged
		}
	}

	// 3. MCP servers (unchanged)
	// ...

	return commands
}
```

**Step 2: Update FormatCommands if needed**

Check if it references sources, update to marketplaces.

**Step 3: Commit commands changes**

```bash
git add internal/diff/commands.go
git commit -m "refactor: Update command generation for marketplaces

- Generate marketplace add commands from map
- Remove source kind checks
- Simpler logic (no type branching)

Part of #74"
```

### Task 3.4: Update executor.go

**Files:**
- Modify: `internal/sync/executor.go`

**Step 1: Rename addSource to addMarketplace**

```go
func (s *Syncer) addMarketplace(m MarketplaceDiff) (Operation, error) {
	op := Operation{
		Type:        "marketplace",
		Name:        m.Name,
		Action:      "add",
		Success:     false,
		Command:     "",
		Output:      "",
		Error:       "",
		Skipped:     false,
		Description: "",
	}

	if m.Desired == nil {
		op.Error = fmt.Sprintf("no desired state for marketplace %s", m.Name)
		return op, fmt.Errorf("no desired state for marketplace %s", m.Name)
	}

	repo := m.Desired.Repo
	op.Description = fmt.Sprintf("Add marketplace: %s", repo)
	op.Command = fmt.Sprintf("claude plugin marketplace add %s", repo)

	output, err := s.runner.Run("claude", "plugin", "marketplace", "add", repo)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to add marketplace %s: %v\nOutput: %s", m.Name, err, string(output))
		return op, fmt.Errorf("failed to add marketplace %s: %w", m.Name, err)
	}

	op.Success = true
	op.Output = string(output)
	return op, nil
}
```

**Step 2: Update Execute method**

```go
// Change loop from r.Sources to r.Marketplaces
for _, m := range r.Marketplaces {
	switch m.Action {
	case ActionAdd:
		op, err := s.addMarketplace(m)
		result.Operations = append(result.Operations, op)
		// ... error handling
	}
}
```

**Step 3: Remove updateSource (not needed for MVP)**

Marketplace updates are rare - can handle manually for now.

**Step 4: Commit executor changes**

```bash
git add internal/sync/executor.go
git commit -m "refactor: Update sync executor for marketplaces

- Rename addSource to addMarketplace
- Work with marketplace map
- Remove kind/type checks
- Simpler execution logic

Part of #74"
```

**Step 5: Update GitHub issue #74**

```bash
gh issue comment 74 --body "Phase 3 complete:
- Diff types updated (MarketplaceDiff)
- Compute logic updated for map structure
- Command generation simplified
- Executor updated for marketplaces"

gh issue close 74 --comment "✓ Phase complete"
```

---

## Phase 4: Update State Readers (#75)

### Task 4.1: Update state.go types

**Files:**
- Modify: `internal/state/state.go`

**Step 1: Rename SourceState to MarketplaceState**

```go
// OLD
type SourceState struct {
	Name            string
	Kind            string
	Type            string
	URL             string
	Path            string
	Ref             string
	InstallLocation string
	LastUpdated     string
}

// NEW
type MarketplaceState struct {
	Name            string
	URL             string  // Repo URL
	Ref             string
	InstallLocation string
	LastUpdated     string
}
```

**Step 2: Update State struct**

```go
// OLD
type State struct {
	Sources    map[string]SourceState
	Plugins    map[string]PluginState
	MCPServers map[string]MCPServerState
}

// NEW
type State struct {
	Marketplaces map[string]MarketplaceState
	Plugins      map[string]PluginState
	MCPServers   map[string]MCPServerState
}
```

**Step 3: Commit state types**

```bash
git add internal/state/state.go
git commit -m "refactor: Rename SourceState to MarketplaceState

- Simplify fields (remove Kind, Type, Path)
- Update State struct to use Marketplaces map

Part of #75"
```

### Task 4.2: Update fs_reader.go

**Files:**
- Modify: `internal/state/fs_reader.go`

**Step 1: Update Read() method**

```go
func (r *FilesystemReader) Read() (*State, error) {
	claudeDir := r.ClaudeDir
	state := &State{
		Marketplaces: make(map[string]MarketplaceState),
		Plugins:      make(map[string]PluginState),
		MCPServers:   make(map[string]MCPServerState),
	}

	if err := r.readMarketplaces(claudeDir, state); err != nil {
		return nil, fmt.Errorf("failed to read marketplaces: %w", err)
	}

	if err := r.readPlugins(claudeDir, state); err != nil {
		return nil, fmt.Errorf("failed to read plugins: %w", err)
	}

	if err := r.readMCPServers(claudeDir, state); err != nil {
		return nil, fmt.Errorf("failed to read MCP servers: %w", err)
	}

	return state, nil
}
```

**Step 2: Rename readSources to readMarketplaces**

```go
func (r *FilesystemReader) readMarketplaces(claudeDir string, state *State) error {
	path := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var marketplaces map[string]struct {
		Source struct {
			Source string `json:"source"`
			Repo   string `json:"repo"`
			Path   string `json:"path"`
		} `json:"source"`
		InstallLocation string `json:"installLocation"`
		LastUpdated     string `json:"lastUpdated"`
	}

	if err := json.Unmarshal(data, &marketplaces); err != nil {
		return err
	}

	for name, m := range marketplaces {
		marketplace := MarketplaceState{
			Name:            name,
			URL:             m.Source.Repo,
			InstallLocation: m.InstallLocation,
			LastUpdated:     m.LastUpdated,
		}

		state.Marketplaces[name] = marketplace
	}

	return nil
}
```

**Step 3: Remove readPluginRepos**

Delete this function entirely (no longer reading local plugin repos).

**Step 4: Commit fs_reader changes**

```bash
git add internal/state/fs_reader.go
git commit -m "refactor: Update filesystem reader for marketplaces

- Rename readSources to readMarketplaces
- Remove readPluginRepos (local plugins removed)
- Simplify marketplace state reading

Part of #75"
```

### Task 4.3: Update cli_reader.go (if needed)

**Files:**
- Modify: `internal/state/cli_reader.go`

**Step 1: Update parseSourceList to parseMarketplaceList**

Update function to work with marketplaces (similar changes to fs_reader).

**Step 2: Commit cli_reader changes**

```bash
git add internal/state/cli_reader.go
git commit -m "refactor: Update CLI reader for marketplaces

Part of #75"
```

**Step 3: Update GitHub issue #75**

```bash
gh issue comment 75 --body "Phase 4 complete:
- State types updated (MarketplaceState)
- Filesystem reader updated
- CLI reader updated
- Removed local plugin repo reading"

gh issue close 75 --comment "✓ Phase complete"
```

---

## Phase 5: Update Test Files (#76)

### Task 5.1: Update config test fixtures

**Files:**
- Modify: `internal/config/*_test.go`

**Step 1: Update validate_test.go**

Replace source test cases with marketplace test cases:

```go
func TestValidateMarketplaces(t *testing.T) {
	tests := []struct {
		name         string
		marketplaces map[string]config.Marketplace
		wantErr      bool
		errContains  string
	}{
		{
			name: "valid marketplaces",
			marketplaces: map[string]config.Marketplace{
				"official": {Repo: "anthropics/claude-plugins-official"},
				"superpowers": {Repo: "obra/superpowers-marketplace", Ref: "main"},
			},
			wantErr: false,
		},
		{
			name: "empty repo",
			marketplaces: map[string]config.Marketplace{
				"bad": {Repo: ""},
			},
			wantErr:     true,
			errContains: "repo is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMarketplaces(tt.marketplaces)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMarketplaces() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
			}
		})
	}
}
```

**Step 2: Update parser_test.go fixtures**

Change YAML fixtures from sources array to marketplaces map:

```go
content := []byte(`
version: 1
marketplaces:
  official:
    repo: anthropics/claude-plugins-official
  superpowers:
    repo: obra/superpowers-marketplace
plugins:
  - context7@official
`)
```

**Step 3: Run config tests**

```bash
go test ./internal/config -v
```

Expected: All passing after updates

**Step 4: Commit config tests**

```bash
git add internal/config/*_test.go
git commit -m "test: Update config tests for marketplaces

- Update test fixtures to use marketplaces map
- Remove source kind/type test cases
- Update validation tests

Part of #76"
```

### Task 5.2: Update diff/sync test fixtures

**Files:**
- Modify: `internal/diff/*_test.go`, `internal/sync/*_test.go`

**Step 1: Update compute_test.go**

Change test data from Sources to Marketplaces:

```go
clewfile := &config.Clewfile{
	Marketplaces: map[string]config.Marketplace{
		"official": {Repo: "anthropics/plugins"},
	},
	Plugins: []config.Plugin{
		{Name: "test@official"},
	},
}
```

**Step 2: Update executor_test.go**

Update test cases to use MarketplaceDiff instead of SourceDiff.

**Step 3: Run diff/sync tests**

```bash
go test ./internal/diff ./internal/sync -v
```

**Step 4: Commit diff/sync tests**

```bash
git add internal/diff/*_test.go internal/sync/*_test.go
git commit -m "test: Update diff/sync tests for marketplaces

Part of #76"
```

### Task 5.3: Update state test fixtures

**Files:**
- Modify: `internal/state/*_test.go`

**Step 1: Update fs_reader_test.go**

Update JSON fixtures to use marketplace structure:

```go
marketplacesJSON := `{
  "official": {
    "source": {
      "source": "github",
      "repo": "anthropics/claude-plugins-official"
    },
    "installLocation": "/path",
    "lastUpdated": "2025-01-01T00:00:00Z"
  }
}`
```

**Step 2: Run state tests**

```bash
go test ./internal/state -v
```

**Step 3: Commit state tests**

```bash
git add internal/state/*_test.go
git commit -m "test: Update state tests for marketplaces

Part of #76"
```

### Task 5.4: Update integration tests

**Files:**
- Modify: `internal/cmd/sync_integration_test.go`, `test/e2e/e2e_test.go`

**Step 1: Update all Clewfile fixtures**

Replace sources with marketplaces in test data.

**Step 2: Run all tests**

```bash
make test
```

Expected: All passing

**Step 3: Commit integration tests**

```bash
git add internal/cmd/*_test.go test/e2e/*_test.go
git commit -m "test: Update integration tests for marketplaces

Part of #76"
```

**Step 4: Update GitHub issue #76**

```bash
gh issue comment 76 --body "Phase 5 complete:
- All test files updated
- Fixtures use marketplaces map
- All tests passing
- ~600 lines of test code updated"

gh issue close 76 --comment "✓ Phase complete"
```

---

## Phase 6: Update Schema and Documentation (#77)

### Task 6.1: Update JSON Schema

**Files:**
- Modify: `schema/clewfile.schema.json`

**Step 1: Update marketplaces property**

```json
"marketplaces": {
  "type": "object",
  "description": "Plugin marketplace repositories",
  "additionalProperties": {
    "type": "object",
    "required": ["repo"],
    "properties": {
      "repo": {
        "type": "string",
        "minLength": 1,
        "description": "Repository URL - supports: short form (owner/repo), HTTPS URL, SSH URL",
        "examples": [
          "obra/superpowers-marketplace",
          "https://gitlab.com/company/plugins.git",
          "git@github.com:adamancini/devops-toolkit.git"
        ]
      },
      "ref": {
        "type": "string",
        "description": "Optional git ref (branch, tag, or SHA). Omit for repository's default branch.",
        "examples": ["main", "v1.2.3", "abc123def456"]
      }
    },
    "additionalProperties": false
  },
  "examples": [
    {
      "official": {
        "repo": "anthropics/claude-plugins-official"
      },
      "superpowers": {
        "repo": "obra/superpowers-marketplace",
        "ref": "v1.0.0"
      }
    }
  ]
}
```

**Step 2: Remove sources property**

Delete the entire "sources" property from the schema.

**Step 3: Update plugin schema**

Remove source/path properties from plugin object schema (only name, enabled, scope remain).

**Step 4: Validate schema**

```bash
python3 -m json.tool schema/clewfile.schema.json > /dev/null && echo "Schema valid"
```

**Step 5: Commit schema**

```bash
git add schema/clewfile.schema.json
git commit -m "refactor: Update schema for marketplaces map

- Change from sources array to marketplaces map
- Remove kind/type fields
- Simpler validation rules
- Update examples

Part of #77"
```

### Task 6.2: Update example files

**Files:**
- Modify: `schema/examples/*.yaml`

**Step 1: Update all example Clewfiles**

Change from sources to marketplaces in all example files.

**Step 2: Commit examples**

```bash
git add schema/examples/
git commit -m "docs: Update examples for marketplaces

Part of #77"
```

### Task 6.3: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Update Architecture section**

Change references from Sources to Marketplaces.

**Step 2: Update Data Flow section**

Update type mentions: SourceDiff → MarketplaceDiff

**Step 3: Update Implementation Status**

Remove mentions of source kind/type support.

**Step 4: Commit CLAUDE.md**

```bash
git add CLAUDE.md
git commit -m "docs: Update CLAUDE.md for marketplaces

Part of #77"
```

### Task 6.4: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Update all Clewfile examples**

Change from sources to marketplaces.

**Step 2: Commit README**

```bash
git add README.md
git commit -m "docs: Update README for marketplaces

Part of #77"
```

**Step 3: Update GitHub issue #77**

```bash
gh issue comment 77 --body "Phase 6 complete:
- JSON Schema updated
- Example files updated
- CLAUDE.md updated
- README.md updated
- All documentation reflects marketplaces model"

gh issue close 77 --comment "✓ Phase complete"
```

---

## Phase 7: Version Bump and Final Verification (#78)

### Task 7.1: Update CHANGELOG.md

**Files:**
- Modify: `CHANGELOG.md`

**Step 1: Add v0.8.0 entry**

```markdown
## [0.8.0] - 2026-01-15

### ⚠️ BREAKING CHANGES

- **Schema change**: `sources:` array replaced with `marketplaces:` map
  - Marketplaces are now a map where keys are aliases
  - Removed `kind` field (always marketplace)
  - Removed nested `source:` object - `repo` is at top level
  - Simpler structure: `marketplaces.<alias>.repo` + optional `.ref`
  - **Rationale**: Removed unnecessary complexity from unified source model

### Migration Guide

**v0.7.1 → v0.8.0:**

```yaml
# Before
sources:
  - name: superpowers-marketplace
    alias: superpowers
    kind: marketplace
    source:
      type: github
      url: obra/superpowers-marketplace

# After
marketplaces:
  superpowers:
    repo: obra/superpowers-marketplace
```

**Plugin repositories:** Single-plugin GitHub repos are no longer declared in Clewfile. Install manually:
```bash
claude plugin install git@github.com:you/your-plugin.git
```

Future v0.9.0 will add `repos:` section for declarative plugin repo management.

### Removed

- `SourceKind` type and validation
- `SourceType` type and validation
- `SourceConfig` nested struct
- `Source` struct
- Plugin inline source support
- ~200 lines of type/validation code

### Changed

- Config structure: `Sources []Source` → `Marketplaces map[string]Marketplace`
- Diff types: `SourceDiff` → `MarketplaceDiff`
- State types: `SourceState` → `MarketplaceState`
- Validation: Simpler marketplace validation (just check repo non-empty)
- Parser: Parse map instead of array

### Added

- Simple `Marketplace` struct with `repo` + optional `ref`
- Support for any git URL format (GitHub short form, HTTPS, SSH)
- Better error messages for missing marketplace references
```

**Step 2: Bump version in plugin.json**

```bash
jq '.version = "0.8.0"' .claude-plugin/plugin.json > tmp && mv tmp .claude-plugin/plugin.json
```

**Step 3: Commit version bump**

```bash
git add CHANGELOG.md .claude-plugin/plugin.json
git commit -m "chore: Bump version to 0.8.0

Breaking change: sources array → marketplaces map"
```

### Task 7.2: Run full test suite

**Step 1: Run all tests**

```bash
make test-all
```

Expected: All passing

**Step 2: Run lint**

```bash
make lint
```

Expected: Clean

**Step 3: Build binaries**

```bash
make build
```

Expected: Successful build

### Task 7.3: Final verification

**Step 1: Test with real Clewfile**

Create test Clewfile in worktree:

```yaml
version: 1
marketplaces:
  official:
    repo: anthropics/claude-plugins-official
plugins:
  - context7@official
```

```bash
./clew diff --config ./test-clewfile.yaml --filesystem
./clew sync --config ./test-clewfile.yaml --filesystem --show-commands
```

Expected: Commands generate correctly

**Step 2: Update GitHub issue #78**

```bash
gh issue comment 78 --body "Phase 7 complete:
- Version bumped to 0.8.0
- CHANGELOG updated with migration guide
- All tests passing
- Build successful
- Manual verification complete"

gh issue close 78 --comment "✓ Phase complete"
```

**Step 3: Update epic issue #71**

```bash
gh issue comment 71 --body "Implementation complete! All phases done:

✅ Phase 1: Types cleaned up
✅ Phase 2: Config structures updated
✅ Phase 3: Diff/sync logic updated
✅ Phase 4: State readers updated
✅ Phase 5: All tests updated
✅ Phase 6: Schema and docs updated
✅ Phase 7: Version bumped, verified

Ready for PR."

gh issue close 71 --comment "✓ Epic complete - ready for PR"
```

---

## Final Steps: Create PR

### Task: Push and create PR

**Step 1: Push branch**

```bash
git push -u origin refactor/marketplace-schema-v0.8.0
```

**Step 2: Create PR**

```bash
gh pr create --title "refactor: Simplify schema to marketplaces map (v0.8.0)" --body "$(cat <<'EOF'
## Summary

**Breaking Change**: Simplifies Clewfile schema by reverting from `sources:` array to `marketplaces:` map.

Closes #71

## Motivation

The unified `sources:` model (v0.5.0-v0.7.1) added complexity that's no longer needed:
- SourceKind enum (marketplace/plugin) - unnecessary for marketplace-only MVP
- SourceType enum (github) - can infer from repo format
- Nested source object - adds verbosity
- Dual name/alias - map key serves as alias

After removing local plugin support (v0.7.0), we're left with only GitHub marketplaces, making the unified model overcomplicated.

## Changes

### Structure Change

**Before (v0.7.1):**
```yaml
sources:
  - name: superpowers-marketplace
    alias: superpowers
    kind: marketplace
    source:
      type: github
      url: obra/superpowers-marketplace
```

**After (v0.8.0):**
```yaml
marketplaces:
  superpowers:
    repo: obra/superpowers-marketplace
```

### Code Changes

- Removed: SourceKind, SourceType types (~200 lines)
- Replaced: Sources []Source with Marketplaces map[string]Marketplace
- Simplified: Marketplace struct (just repo + optional ref)
- Updated: All diff/sync/state logic for map structure
- Updated: 36 files, ~700 lines changed (net -200 lines)

## Migration Guide

See CHANGELOG.md for complete migration guide.

**Key changes:**
1. `sources:` → `marketplaces:`
2. Array → Map (name becomes map key)
3. Remove `kind`, `alias`, nested `source`
4. `source.url` → `repo`

## Future Extensibility

v0.9.0 will add `repos:` for standalone plugin repositories without breaking v0.8.0 Clewfiles.

## Testing

- ✅ All unit tests passing
- ✅ All integration tests passing
- ✅ All e2e tests passing
- ✅ Schema validation working
- ✅ Manual verification with test Clewfiles

## Issues Closed

- Closes #71 (Epic)
- Closes #72 (Phase 1)
- Closes #73 (Phase 2)
- Closes #74 (Phase 3)
- Closes #75 (Phase 4)
- Closes #76 (Phase 5)
- Closes #77 (Phase 6)
- Closes #78 (Phase 7)
EOF
)"
```

---

## Implementation Notes

**Working Directory:** `.worktrees/refactor/marketplace-schema-v0.8.0`

**Commit Frequency:** After each task (every 2-5 minutes of work)

**Testing Strategy:**
- Run package tests after modifying each package
- Run full suite after each phase
- Fix compilation errors immediately
- Don't proceed to next phase until current phase compiles and tests pass

**GitHub Issue Updates:**
- Comment on issue after phase complete
- Close issue with "✓ Phase complete"
- Keep epic #71 open until all phases done

**Expected Timeline:**
- Phase 2: 30 min (config structures)
- Phase 3: 30 min (diff/sync logic)
- Phase 4: 20 min (state readers)
- Phase 5: 45 min (all tests)
- Phase 6: 20 min (schema/docs)
- Phase 7: 15 min (version/verification)
- **Total:** ~2.5 hours

**Verification Checklist:**
- [ ] All tests passing
- [ ] No compilation errors
- [ ] Schema validates
- [ ] clew diff works with new format
- [ ] clew sync generates correct commands
- [ ] Migration guide in CHANGELOG
- [ ] All GitHub issues closed
