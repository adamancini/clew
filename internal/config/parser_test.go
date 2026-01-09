package config

import (
	"os"
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		content  string
		expected Format
	}{
		{"yaml extension", "Clewfile.yaml", "", FormatYAML},
		{"yml extension", "Clewfile.yml", "", FormatYAML},
		{"toml extension", "Clewfile.toml", "", FormatTOML},
		{"json extension", "Clewfile.json", "", FormatJSON},
		{"json content", "Clewfile", `{"version": 1}`, FormatJSON},
		{"yaml content", "Clewfile", `version: 1`, FormatYAML},
		{"toml content", "Clewfile", `version = 1`, FormatTOML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFormat(tt.path, []byte(tt.content))
			if got != tt.expected {
				t.Errorf("detectFormat() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExpandEnvVars(t *testing.T) {
	_ = os.Setenv("TEST_VAR", "test_value")
	_ = os.Setenv("EMPTY_VAR", "")
	defer func() { _ = os.Unsetenv("TEST_VAR") }()
	defer func() { _ = os.Unsetenv("EMPTY_VAR") }()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple var", "${TEST_VAR}", "test_value"},
		{"var with default", "${MISSING_VAR:-default_value}", "default_value"},
		{"existing var ignores default", "${TEST_VAR:-default_value}", "test_value"},
		{"empty var uses default", "${EMPTY_VAR:-default_value}", "default_value"},
		{"no var", "plain text", "plain text"},
		{"mixed content", "prefix ${TEST_VAR} suffix", "prefix test_value suffix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(expandEnvVars([]byte(tt.input)))
			if got != tt.expected {
				t.Errorf("expandEnvVars() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseYAML(t *testing.T) {
	content := []byte(`
version: 1
sources:
  - name: official
    kind: marketplace
    source:
      type: github
      url: anthropics/claude-plugins
plugins:
  - name: superpowers@official
    enabled: true
    scope: user
  - simple-plugin@official
mcp_servers:
  filesystem:
    transport: stdio
    command: npx
    args: ["@modelcontextprotocol/server-filesystem", "/tmp"]
`)

	clewfile, err := parse(content, FormatYAML)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}

	if clewfile.Version != 1 {
		t.Errorf("Version = %d, want 1", clewfile.Version)
	}

	if len(clewfile.Sources) != 1 {
		t.Errorf("Sources count = %d, want 1", len(clewfile.Sources))
	}

	if clewfile.Sources[0].Name != "official" {
		t.Errorf("Source name = %s, want official", clewfile.Sources[0].Name)
	}
	if clewfile.Sources[0].Kind != SourceKindMarketplace {
		t.Errorf("Source kind = %s, want marketplace", clewfile.Sources[0].Kind)
	}
	if clewfile.Sources[0].Source.Type != SourceTypeGitHub {
		t.Errorf("Source type = %s, want github", clewfile.Sources[0].Source.Type)
	}

	if len(clewfile.Plugins) != 2 {
		t.Errorf("Plugins count = %d, want 2", len(clewfile.Plugins))
	}

	// Check structured plugin
	if clewfile.Plugins[0].Name != "superpowers@official" {
		t.Errorf("Plugin[0].Name = %s, want superpowers@official", clewfile.Plugins[0].Name)
	}
	if clewfile.Plugins[0].Enabled == nil || !*clewfile.Plugins[0].Enabled {
		t.Error("Plugin[0].Enabled should be true")
	}

	// Check simple string plugin
	if clewfile.Plugins[1].Name != "simple-plugin@official" {
		t.Errorf("Plugin[1].Name = %s, want simple-plugin@official", clewfile.Plugins[1].Name)
	}

	if len(clewfile.MCPServers) != 1 {
		t.Errorf("MCPServers count = %d, want 1", len(clewfile.MCPServers))
	}

	if s, ok := clewfile.MCPServers["filesystem"]; ok {
		if s.Transport != "stdio" {
			t.Errorf("MCP transport = %s, want stdio", s.Transport)
		}
		if s.Command != "npx" {
			t.Errorf("MCP command = %s, want npx", s.Command)
		}
	}
}

func TestParseTOML(t *testing.T) {
	content := []byte(`
version = 1

[[sources]]
name = "official"
kind = "marketplace"

[sources.source]
type = "github"
url = "anthropics/claude-plugins"

[[plugins]]
name = "superpowers@official"
enabled = true
scope = "user"

[mcp_servers.filesystem]
transport = "stdio"
command = "npx"
args = ["@modelcontextprotocol/server-filesystem", "/tmp"]
`)

	clewfile, err := parse(content, FormatTOML)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}

	if clewfile.Version != 1 {
		t.Errorf("Version = %d, want 1", clewfile.Version)
	}

	if len(clewfile.Sources) != 1 {
		t.Errorf("Sources count = %d, want 1", len(clewfile.Sources))
	}

	if len(clewfile.Plugins) != 1 {
		t.Errorf("Plugins count = %d, want 1", len(clewfile.Plugins))
	}
}

func TestParseJSON(t *testing.T) {
	content := []byte(`{
  "version": 1,
  "sources": [
    {
      "name": "official",
      "kind": "marketplace",
      "source": {
        "type": "github",
        "url": "anthropics/claude-plugins"
      }
    }
  ],
  "plugins": [
    {"name": "superpowers@official", "enabled": true, "scope": "user"},
    "simple-plugin@official"
  ],
  "mcp_servers": {
    "filesystem": {
      "transport": "stdio",
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "/tmp"]
    }
  }
}`)

	clewfile, err := parse(content, FormatJSON)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}

	if clewfile.Version != 1 {
		t.Errorf("Version = %d, want 1", clewfile.Version)
	}

	if len(clewfile.Plugins) != 2 {
		t.Errorf("Plugins count = %d, want 2", len(clewfile.Plugins))
	}
}

func TestParseEnvVarExpansion(t *testing.T) {
	_ = os.Setenv("MCP_COMMAND", "/usr/local/bin/mcp-server")
	defer func() { _ = os.Unsetenv("MCP_COMMAND") }()

	content := []byte(`
version: 1
mcp_servers:
  custom:
    transport: stdio
    command: ${MCP_COMMAND}
`)

	clewfile, err := parse(content, FormatYAML)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}

	if s, ok := clewfile.MCPServers["custom"]; ok {
		if s.Command != "/usr/local/bin/mcp-server" {
			t.Errorf("MCP command = %s, want /usr/local/bin/mcp-server", s.Command)
		}
	}
}

func TestParseLocalPlugin(t *testing.T) {
	content := []byte(`
version: 1
plugins:
  # Marketplace plugin (simple string format)
  - superpowers@superpowers-marketplace

  # Local plugin (simplified format as per issue #65)
  - name: devops-toolkit
    source: local
    path: ~/.claude/plugins/repos/devops-toolkit
    scope: user
    enabled: true

  # Local plugin without enabled (defaults to true)
  - name: my-other-plugin
    source: local
    path: /path/to/plugin
`)

	clewfile, err := parse(content, FormatYAML)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}

	if len(clewfile.Plugins) != 3 {
		t.Fatalf("Plugins count = %d, want 3", len(clewfile.Plugins))
	}

	// Check marketplace plugin
	if clewfile.Plugins[0].Name != "superpowers@superpowers-marketplace" {
		t.Errorf("Plugin[0].Name = %s, want superpowers@superpowers-marketplace", clewfile.Plugins[0].Name)
	}
	if clewfile.Plugins[0].Source != nil {
		t.Errorf("Plugin[0].Source should be nil for marketplace plugins")
	}

	// Check first local plugin
	if clewfile.Plugins[1].Name != "devops-toolkit" {
		t.Errorf("Plugin[1].Name = %s, want devops-toolkit", clewfile.Plugins[1].Name)
	}
	if clewfile.Plugins[1].Source == nil {
		t.Fatal("Plugin[1].Source should not be nil for local plugins")
	}
	if clewfile.Plugins[1].Source.Type != SourceTypeLocal {
		t.Errorf("Plugin[1].Source.Type = %s, want local", clewfile.Plugins[1].Source.Type)
	}
	if clewfile.Plugins[1].Source.Path != "~/.claude/plugins/repos/devops-toolkit" {
		t.Errorf("Plugin[1].Source.Path = %s, want ~/.claude/plugins/repos/devops-toolkit", clewfile.Plugins[1].Source.Path)
	}
	if clewfile.Plugins[1].Scope != "user" {
		t.Errorf("Plugin[1].Scope = %s, want user", clewfile.Plugins[1].Scope)
	}
	if clewfile.Plugins[1].Enabled == nil || !*clewfile.Plugins[1].Enabled {
		t.Error("Plugin[1].Enabled should be true")
	}

	// Check second local plugin (without enabled)
	if clewfile.Plugins[2].Name != "my-other-plugin" {
		t.Errorf("Plugin[2].Name = %s, want my-other-plugin", clewfile.Plugins[2].Name)
	}
	if clewfile.Plugins[2].Source == nil {
		t.Fatal("Plugin[2].Source should not be nil for local plugins")
	}
	if clewfile.Plugins[2].Source.Type != SourceTypeLocal {
		t.Errorf("Plugin[2].Source.Type = %s, want local", clewfile.Plugins[2].Source.Type)
	}
	if clewfile.Plugins[2].Source.Path != "/path/to/plugin" {
		t.Errorf("Plugin[2].Source.Path = %s, want /path/to/plugin", clewfile.Plugins[2].Source.Path)
	}
}

func TestParseLocalPluginJSON(t *testing.T) {
	content := []byte(`{
  "version": 1,
  "plugins": [
    "marketplace-plugin@marketplace",
    {
      "name": "local-plugin",
      "source": "local",
      "path": "~/.claude/plugins/repos/local-plugin",
      "scope": "user"
    }
  ]
}`)

	clewfile, err := parse(content, FormatJSON)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}

	if len(clewfile.Plugins) != 2 {
		t.Fatalf("Plugins count = %d, want 2", len(clewfile.Plugins))
	}

	// Check local plugin parsed correctly from JSON
	if clewfile.Plugins[1].Name != "local-plugin" {
		t.Errorf("Plugin[1].Name = %s, want local-plugin", clewfile.Plugins[1].Name)
	}
	if clewfile.Plugins[1].Source == nil {
		t.Fatal("Plugin[1].Source should not be nil")
	}
	if clewfile.Plugins[1].Source.Type != SourceTypeLocal {
		t.Errorf("Plugin[1].Source.Type = %s, want local", clewfile.Plugins[1].Source.Type)
	}
	if clewfile.Plugins[1].Source.Path != "~/.claude/plugins/repos/local-plugin" {
		t.Errorf("Plugin[1].Source.Path = %s, want ~/.claude/plugins/repos/local-plugin", clewfile.Plugins[1].Source.Path)
	}
}
