package state

import (
	"testing"
)

func TestParseSourceList(t *testing.T) {
	output := []byte(`Configured marketplaces:

  ❯ superpowers-marketplace
    Source: GitHub (obra/superpowers-marketplace)

  ❯ local-plugins
    Source: Local (/Users/test/plugins)

  ❯ claude-code-plugins
    Source: GitHub (anthropics/claude-code)
`)

	state := &State{
		Sources: make(map[string]SourceState),
	}

	err := parseSourceList(output, state)
	if err != nil {
		t.Fatalf("parseSourceList() error = %v", err)
	}

	if len(state.Sources) != 3 {
		t.Errorf("Sources count = %d, want 3", len(state.Sources))
	}

	// Test GitHub source
	if src, ok := state.Sources["superpowers-marketplace"]; ok {
		if src.Type != "github" {
			t.Errorf("Type = %s, want github", src.Type)
		}
		if src.URL != "obra/superpowers-marketplace" {
			t.Errorf("URL = %s, want obra/superpowers-marketplace", src.URL)
		}
		if src.Kind != "marketplace" {
			t.Errorf("Kind = %s, want marketplace", src.Kind)
		}
	} else {
		t.Error("Missing superpowers-marketplace")
	}

	// Test local source
	if src, ok := state.Sources["local-plugins"]; ok {
		if src.Type != "local" {
			t.Errorf("Type = %s, want local", src.Type)
		}
		if src.Path != "/Users/test/plugins" {
			t.Errorf("Path = %s, want /Users/test/plugins", src.Path)
		}
	} else {
		t.Error("Missing local-plugins")
	}
}

func TestParseMCPList(t *testing.T) {
	output := []byte(`Checking MCP server health...

filesystem: npx -y @modelcontextprotocol/server-filesystem /tmp - ✓ Connected
sentry: https://mcp.sentry.dev/mcp (HTTP) - ✓ Connected
asana: https://mcp.asana.com/sse (SSE) - ✓ Connected
custom: /usr/local/bin/custom-server --flag value - ✓ Connected
plugin:context7:context7: npx -y @upstash/context7-mcp - ✓ Connected
`)

	state := &State{
		MCPServers: make(map[string]MCPServerState),
	}

	err := parseMCPList(output, state)
	if err != nil {
		t.Fatalf("parseMCPList() error = %v", err)
	}

	// Should have 4 servers (plugin servers are filtered out)
	if len(state.MCPServers) != 4 {
		t.Errorf("MCPServers count = %d, want 4", len(state.MCPServers))
	}

	// Test stdio server
	if s, ok := state.MCPServers["filesystem"]; ok {
		if s.Transport != "stdio" {
			t.Errorf("Transport = %s, want stdio", s.Transport)
		}
		if s.Command != "npx" {
			t.Errorf("Command = %s, want npx", s.Command)
		}
		expectedArgs := []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"}
		if len(s.Args) != len(expectedArgs) {
			t.Errorf("Args count = %d, want %d", len(s.Args), len(expectedArgs))
		}
	} else {
		t.Error("Missing filesystem server")
	}

	// Test HTTP server
	if s, ok := state.MCPServers["sentry"]; ok {
		if s.Transport != "http" {
			t.Errorf("Transport = %s, want http", s.Transport)
		}
		if s.URL != "https://mcp.sentry.dev/mcp" {
			t.Errorf("URL = %s, want https://mcp.sentry.dev/mcp", s.URL)
		}
	} else {
		t.Error("Missing sentry server")
	}

	// Test SSE server
	if s, ok := state.MCPServers["asana"]; ok {
		if s.Transport != "sse" {
			t.Errorf("Transport = %s, want sse", s.Transport)
		}
		if s.URL != "https://mcp.asana.com/sse" {
			t.Errorf("URL = %s, want https://mcp.asana.com/sse", s.URL)
		}
	} else {
		t.Error("Missing asana server")
	}

	// Test that plugin servers are filtered
	if _, ok := state.MCPServers["plugin:context7:context7"]; ok {
		t.Error("Plugin servers should be filtered out")
	}
}

func TestParseEmptyOutput(t *testing.T) {
	state := &State{
		Sources:    make(map[string]SourceState),
		MCPServers: make(map[string]MCPServerState),
	}

	err := parseSourceList([]byte(""), state)
	if err != nil {
		t.Errorf("parseSourceList() empty should not error: %v", err)
	}
	if len(state.Sources) != 0 {
		t.Errorf("Sources should be empty")
	}

	err = parseMCPList([]byte(""), state)
	if err != nil {
		t.Errorf("parseMCPList() empty should not error: %v", err)
	}
	if len(state.MCPServers) != 0 {
		t.Errorf("MCPServers should be empty")
	}
}
