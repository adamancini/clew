package config

import (
	"strings"
	"testing"
)

func TestValidateMarketplaces(t *testing.T) {
	tests := []struct {
		name         string
		marketplaces map[string]Marketplace
		wantErr      bool
		errContains  string
	}{
		{
			name: "valid marketplace with short repo",
			marketplaces: map[string]Marketplace{
				"official": {Repo: "org/repo"},
			},
			wantErr: false,
		},
		{
			name: "valid marketplace with HTTPS URL",
			marketplaces: map[string]Marketplace{
				"gitlab": {Repo: "https://gitlab.com/company/plugins.git"},
			},
			wantErr: false,
		},
		{
			name: "valid marketplace with SSH URL",
			marketplaces: map[string]Marketplace{
				"private": {Repo: "git@github.com:org/repo.git"},
			},
			wantErr: false,
		},
		{
			name: "valid marketplace with ref",
			marketplaces: map[string]Marketplace{
				"official": {Repo: "org/repo", Ref: "v1.0.0"},
			},
			wantErr: false,
		},
		{
			name: "missing repo",
			marketplaces: map[string]Marketplace{
				"bad": {},
			},
			wantErr:     true,
			errContains: "repo is required",
		},
		{
			name:         "empty marketplaces map is valid",
			marketplaces: map[string]Marketplace{},
			wantErr:      false,
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

func TestValidatePlugin(t *testing.T) {
	tests := []struct {
		name        string
		plugin      Plugin
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid plugin@marketplace format",
			plugin:  Plugin{Name: "test@marketplace"},
			wantErr: false,
		},
		{
			name:    "valid with scope user",
			plugin:  Plugin{Name: "test@marketplace", Scope: "user"},
			wantErr: false,
		},
		{
			name:    "valid with scope project",
			plugin:  Plugin{Name: "test@marketplace", Scope: "project"},
			wantErr: false,
		},
		{
			name:        "missing name",
			plugin:      Plugin{},
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "invalid format - missing @",
			plugin:      Plugin{Name: "test"},
			wantErr:     true,
			errContains: "must be plugin@marketplace format",
		},
		{
			name:        "invalid format - just @",
			plugin:      Plugin{Name: "@"},
			wantErr:     true,
			errContains: "must be plugin@marketplace format",
		},
		{
			name:        "invalid format - missing marketplace",
			plugin:      Plugin{Name: "test@"},
			wantErr:     true,
			errContains: "must be plugin@marketplace format",
		},
		{
			name:        "invalid format - missing plugin name",
			plugin:      Plugin{Name: "@marketplace"},
			wantErr:     true,
			errContains: "must be plugin@marketplace format",
		},
		{
			name:        "invalid scope",
			plugin:      Plugin{Name: "test@marketplace", Scope: "invalid"},
			wantErr:     true,
			errContains: "invalid scope",
		},
		{
			name:    "valid plugin name with dashes",
			plugin:  Plugin{Name: "my-plugin@my-marketplace"},
			wantErr: false,
		},
		{
			name:    "valid plugin name with underscores",
			plugin:  Plugin{Name: "my_plugin@my_marketplace"},
			wantErr: false,
		},
		{
			name:    "valid plugin name with numbers",
			plugin:  Plugin{Name: "plugin123@marketplace456"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlugin(0, tt.plugin)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePlugin() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateMCPServer(t *testing.T) {
	tests := []struct {
		name        string
		server      MCPServer
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid stdio",
			server:  MCPServer{Transport: "stdio", Command: "/usr/bin/server"},
			wantErr: false,
		},
		{
			name:    "valid http",
			server:  MCPServer{Transport: "http", URL: "http://localhost:8080"},
			wantErr: false,
		},
		{
			name:    "valid sse",
			server:  MCPServer{Transport: "sse", URL: "http://localhost:8080/sse"},
			wantErr: false,
		},
		{
			name:        "missing transport",
			server:      MCPServer{},
			wantErr:     true,
			errContains: "transport type is required",
		},
		{
			name:        "stdio missing command",
			server:      MCPServer{Transport: "stdio"},
			wantErr:     true,
			errContains: "command is required",
		},
		{
			name:        "http missing url",
			server:      MCPServer{Transport: "http"},
			wantErr:     true,
			errContains: "url is required",
		},
		{
			name:        "invalid transport",
			server:      MCPServer{Transport: "websocket"},
			wantErr:     true,
			errContains: "invalid transport",
		},
		{
			name:        "invalid scope",
			server:      MCPServer{Transport: "stdio", Command: "cmd", Scope: "global"},
			wantErr:     true,
			errContains: "invalid scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPServer("test", tt.server)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMCPServer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidatePluginReferences(t *testing.T) {
	tests := []struct {
		name        string
		clewfile    *Clewfile
		wantErr     bool
		errContains string
	}{
		{
			name: "valid reference to existing marketplace",
			clewfile: &Clewfile{
				Marketplaces: map[string]Marketplace{
					"official": {Repo: "org/repo"},
				},
				Plugins: []Plugin{
					{Name: "test@official"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid reference to non-existent marketplace",
			clewfile: &Clewfile{
				Marketplaces: map[string]Marketplace{
					"official": {Repo: "org/repo"},
				},
				Plugins: []Plugin{
					{Name: "test@nonexistent"},
				},
			},
			wantErr:     true,
			errContains: "references unknown marketplace 'nonexistent'",
		},
		{
			name: "multiple valid plugins",
			clewfile: &Clewfile{
				Marketplaces: map[string]Marketplace{
					"official":    {Repo: "org/repo"},
					"superpowers": {Repo: "obra/superpowers"},
				},
				Plugins: []Plugin{
					{Name: "context7@official"},
					{Name: "brainstorming@superpowers"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePluginReferences(tt.clewfile)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePluginReferences() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestValidateFull(t *testing.T) {
	valid := &Clewfile{
		Version: 1,
		Marketplaces: map[string]Marketplace{
			"official": {Repo: "anthropics/plugins"},
		},
		Plugins: []Plugin{
			{Name: "test@official"},
		},
		MCPServers: map[string]MCPServer{
			"fs": {Transport: "stdio", Command: "npx"},
		},
	}

	if err := Validate(valid); err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}

	invalid := &Clewfile{
		Version: 1,
		Marketplaces: map[string]Marketplace{
			"bad": {}, // Missing repo
		},
		Plugins: []Plugin{
			{Name: ""}, // Missing name
		},
		MCPServers: map[string]MCPServer{
			"bad": {Transport: ""}, // Missing transport
		},
	}

	if err := Validate(invalid); err == nil {
		t.Error("Validate() should return error for invalid config")
	} else if !strings.Contains(err.Error(), "validation errors") {
		t.Errorf("error should mention validation errors, got: %v", err)
	}
}
