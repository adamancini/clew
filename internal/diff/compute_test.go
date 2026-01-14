package diff

import (
	"testing"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/state"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestComputeSources(t *testing.T) {
	clewfile := &config.Clewfile{
		Sources: []config.Source{
			{Name: "existing", Kind: config.SourceKindMarketplace, Source: config.SourceConfig{Type: config.SourceTypeGitHub, URL: "owner/existing"}},
			{Name: "new", Kind: config.SourceKindMarketplace, Source: config.SourceConfig{Type: config.SourceTypeGitHub, URL: "owner/new"}},
			{Name: "updated", Kind: config.SourceKindMarketplace, Source: config.SourceConfig{Type: config.SourceTypeGitHub, URL: "owner/updated-new"}},
		},
	}

	current := &state.State{
		Sources: map[string]state.SourceState{
			"existing": {Name: "existing", Kind: "marketplace", Type: "github", URL: "owner/existing"},
			"updated":  {Name: "updated", Kind: "marketplace", Type: "github", URL: "owner/updated-old"},
			"extra":    {Name: "extra", Kind: "marketplace", Type: "github", URL: "owner/extra"},
		},
		Plugins:    make(map[string]state.PluginState),
		MCPServers: make(map[string]state.MCPServerState),
	}

	result := Compute(clewfile, current)

	// Should have 4 source diffs
	if len(result.Sources) != 4 {
		t.Errorf("Sources count = %d, want 4", len(result.Sources))
	}

	actionCounts := make(map[Action]int)
	for _, src := range result.Sources {
		actionCounts[src.Action]++
	}

	if actionCounts[ActionNone] != 1 {
		t.Errorf("ActionNone count = %d, want 1", actionCounts[ActionNone])
	}
	if actionCounts[ActionAdd] != 1 {
		t.Errorf("ActionAdd count = %d, want 1", actionCounts[ActionAdd])
	}
	if actionCounts[ActionUpdate] != 1 {
		t.Errorf("ActionUpdate count = %d, want 1", actionCounts[ActionUpdate])
	}
	if actionCounts[ActionRemove] != 1 {
		t.Errorf("ActionRemove count = %d, want 1", actionCounts[ActionRemove])
	}
}

func TestComputePlugins(t *testing.T) {
	clewfile := &config.Clewfile{
		Plugins: []config.Plugin{
			{Name: "installed@marketplace", Enabled: boolPtr(true)},
			{Name: "new@marketplace"},
			{Name: "to-enable@marketplace", Enabled: boolPtr(true)},
			{Name: "to-disable@marketplace", Enabled: boolPtr(false)},
		},
		Sources:    []config.Source{},
		MCPServers: make(map[string]config.MCPServer),
	}

	current := &state.State{
		Plugins: map[string]state.PluginState{
			"installed@marketplace":  {Name: "installed", Marketplace: "marketplace", Enabled: true},
			"to-enable@marketplace":  {Name: "to-enable", Marketplace: "marketplace", Enabled: false},
			"to-disable@marketplace": {Name: "to-disable", Marketplace: "marketplace", Enabled: true},
			"extra@marketplace":      {Name: "extra", Marketplace: "marketplace", Enabled: true},
		},
		Sources:    make(map[string]state.SourceState),
		MCPServers: make(map[string]state.MCPServerState),
	}

	result := Compute(clewfile, current)

	actionCounts := make(map[Action]int)
	for _, p := range result.Plugins {
		actionCounts[p.Action]++
	}

	if actionCounts[ActionNone] != 1 {
		t.Errorf("ActionNone count = %d, want 1 (installed)", actionCounts[ActionNone])
	}
	if actionCounts[ActionAdd] != 1 {
		t.Errorf("ActionAdd count = %d, want 1 (new)", actionCounts[ActionAdd])
	}
	if actionCounts[ActionEnable] != 1 {
		t.Errorf("ActionEnable count = %d, want 1", actionCounts[ActionEnable])
	}
	if actionCounts[ActionDisable] != 1 {
		t.Errorf("ActionDisable count = %d, want 1", actionCounts[ActionDisable])
	}
	if actionCounts[ActionRemove] != 1 {
		t.Errorf("ActionRemove count = %d, want 1 (extra)", actionCounts[ActionRemove])
	}
}

func TestComputeMCPServers(t *testing.T) {
	clewfile := &config.Clewfile{
		MCPServers: map[string]config.MCPServer{
			"existing":   {Transport: "stdio", Command: "npx"},
			"new-stdio":  {Transport: "stdio", Command: "node"},
			"new-oauth":  {Transport: "http", URL: "https://api.example.com/mcp"},
			"new-authed": {Transport: "http", URL: "https://api.example.com/mcp", Env: map[string]string{"API_KEY": "secret"}},
		},
		Sources: []config.Source{},
		Plugins: []config.Plugin{},
	}

	current := &state.State{
		MCPServers: map[string]state.MCPServerState{
			"existing": {Name: "existing", Transport: "stdio", Command: "npx"},
			"extra":    {Name: "extra", Transport: "stdio", Command: "extra-cmd"},
		},
		Sources: make(map[string]state.SourceState),
		Plugins: make(map[string]state.PluginState),
	}

	result := Compute(clewfile, current)

	// Check OAuth detection
	var oauthCount, nonOAuthCount int
	for _, m := range result.MCPServers {
		if m.Action == ActionAdd {
			if m.RequiresOAuth {
				oauthCount++
			} else {
				nonOAuthCount++
			}
		}
	}

	if oauthCount != 1 {
		t.Errorf("OAuth requiring servers = %d, want 1", oauthCount)
	}
	if nonOAuthCount != 2 {
		t.Errorf("Non-OAuth servers = %d, want 2", nonOAuthCount)
	}
}

func TestServerRequiresOAuth(t *testing.T) {
	tests := []struct {
		name     string
		server   config.MCPServer
		expected bool
	}{
		{
			name:     "stdio never needs OAuth",
			server:   config.MCPServer{Transport: "stdio", Command: "npx"},
			expected: false,
		},
		{
			name:     "http without auth needs OAuth",
			server:   config.MCPServer{Transport: "http", URL: "https://api.example.com"},
			expected: true,
		},
		{
			name:     "http with API_KEY env",
			server:   config.MCPServer{Transport: "http", URL: "https://api.example.com", Env: map[string]string{"API_KEY": "secret"}},
			expected: false,
		},
		{
			name:     "http with token env",
			server:   config.MCPServer{Transport: "http", URL: "https://api.example.com", Env: map[string]string{"AUTH_TOKEN": "secret"}},
			expected: false,
		},
		{
			name:     "http with Authorization header",
			server:   config.MCPServer{Transport: "http", URL: "https://api.example.com", Headers: map[string]string{"Authorization": "Bearer xyz"}},
			expected: false,
		},
		{
			name:     "sse without auth needs OAuth",
			server:   config.MCPServer{Transport: "sse", URL: "https://api.example.com/sse"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serverRequiresOAuth(tt.server)
			if got != tt.expected {
				t.Errorf("serverRequiresOAuth() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	result := &Result{
		Sources: []SourceDiff{
			{Name: "src1", Action: ActionAdd},
			{Name: "src2", Action: ActionRemove},
		},
		Plugins: []PluginDiff{
			{Name: "p1", Action: ActionAdd},
			{Name: "p2", Action: ActionEnable},
			{Name: "p3", Action: ActionDisable},
			{Name: "p4", Action: ActionRemove},
		},
		MCPServers: []MCPServerDiff{
			{Name: "s1", Action: ActionAdd, RequiresOAuth: false},
			{Name: "s2", Action: ActionAdd, RequiresOAuth: true},
			{Name: "s3", Action: ActionUpdate},
			{Name: "s4", Action: ActionRemove},
		},
	}

	add, update, remove, attention := result.Summary()

	if add != 3 { // src1 + p1 + s1
		t.Errorf("add = %d, want 3", add)
	}
	if update != 3 { // p2 + p3 + s3
		t.Errorf("update = %d, want 3", update)
	}
	if remove != 0 { // Non-destructive, removals count as attention
		t.Errorf("remove = %d, want 0", remove)
	}
	if attention != 4 { // src2 + p4 + s2 + s4
		t.Errorf("attention = %d, want 4", attention)
	}
}

func TestPluginDiffIsLocal(t *testing.T) {
	// NOTE: As of v0.7.0, IsLocal() always returns false since local plugins are no longer supported
	tests := []struct {
		name     string
		diff     PluginDiff
		expected bool
	}{
		{
			name: "marketplace plugin (no source)",
			diff: PluginDiff{
				Name: "test@marketplace",
				Desired: &config.Plugin{
					Name: "test@marketplace",
				},
			},
			expected: false,
		},
		{
			name: "marketplace plugin (github source)",
			diff: PluginDiff{
				Name: "test@marketplace",
				Desired: &config.Plugin{
					Name: "test@marketplace",
					Source: &config.SourceConfig{
						Type: config.SourceTypeGitHub,
						URL:  "owner/repo",
					},
				},
			},
			expected: false,
		},
		{
			name: "plugin from current state",
			diff: PluginDiff{
				Name: "existing-marketplace",
				Current: &state.PluginState{
					Name:        "existing-marketplace",
					Marketplace: "marketplace",
					IsLocal:     false,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.diff.IsLocal()
			if got != tt.expected {
				t.Errorf("IsLocal() = %v, want %v", got, tt.expected)
			}
		})
	}
}
