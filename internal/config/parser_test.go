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
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("EMPTY_VAR", "")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("EMPTY_VAR")

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
marketplaces:
  official:
    source: github
    repo: anthropics/claude-plugins
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

	if len(clewfile.Marketplaces) != 1 {
		t.Errorf("Marketplaces count = %d, want 1", len(clewfile.Marketplaces))
	}

	if m, ok := clewfile.Marketplaces["official"]; ok {
		if m.Source != "github" {
			t.Errorf("Marketplace source = %s, want github", m.Source)
		}
	} else {
		t.Error("Missing 'official' marketplace")
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

[marketplaces.official]
source = "github"
repo = "anthropics/claude-plugins"

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

	if len(clewfile.Marketplaces) != 1 {
		t.Errorf("Marketplaces count = %d, want 1", len(clewfile.Marketplaces))
	}

	if len(clewfile.Plugins) != 1 {
		t.Errorf("Plugins count = %d, want 1", len(clewfile.Plugins))
	}
}

func TestParseJSON(t *testing.T) {
	content := []byte(`{
  "version": 1,
  "marketplaces": {
    "official": {
      "source": "github",
      "repo": "anthropics/claude-plugins"
    }
  },
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
	os.Setenv("MCP_COMMAND", "/usr/local/bin/mcp-server")
	defer os.Unsetenv("MCP_COMMAND")

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
