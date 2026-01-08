package config

import (
	"strings"
	"testing"
)

func TestValidateMarketplace(t *testing.T) {
	tests := []struct {
		name        string
		marketplace Marketplace
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid github",
			marketplace: Marketplace{Source: "github", Repo: "org/repo"},
			wantErr:     false,
		},
		{
			name:        "valid local",
			marketplace: Marketplace{Source: "local", Path: "/path/to/plugin"},
			wantErr:     false,
		},
		{
			name:        "missing source",
			marketplace: Marketplace{},
			wantErr:     true,
			errContains: "source is required",
		},
		{
			name:        "github missing repo",
			marketplace: Marketplace{Source: "github"},
			wantErr:     true,
			errContains: "repo is required",
		},
		{
			name:        "local missing path",
			marketplace: Marketplace{Source: "local"},
			wantErr:     true,
			errContains: "path is required",
		},
		{
			name:        "invalid source",
			marketplace: Marketplace{Source: "invalid"},
			wantErr:     true,
			errContains: "invalid source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMarketplace("test", tt.marketplace)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMarketplace() error = %v, wantErr %v", err, tt.wantErr)
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
			name:    "valid simple",
			plugin:  Plugin{Name: "test@marketplace"},
			wantErr: false,
		},
		{
			name:    "valid with scope",
			plugin:  Plugin{Name: "test@marketplace", Scope: "user"},
			wantErr: false,
		},
		{
			name:        "missing name",
			plugin:      Plugin{},
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "invalid scope",
			plugin:      Plugin{Name: "test", Scope: "invalid"},
			wantErr:     true,
			errContains: "invalid scope",
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
			errContains: "transport is required",
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

func TestValidateFull(t *testing.T) {
	valid := &Clewfile{
		Version: 1,
		Marketplaces: map[string]Marketplace{
			"official": {Source: "github", Repo: "anthropics/plugins"},
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
			"bad": {Source: "invalid"},
		},
		Plugins: []Plugin{
			{Name: ""},
		},
		MCPServers: map[string]MCPServer{
			"bad": {Transport: ""},
		},
	}

	if err := Validate(invalid); err == nil {
		t.Error("Validate() should return error for invalid config")
	} else if !strings.Contains(err.Error(), "validation errors") {
		t.Errorf("error should mention validation errors, got: %v", err)
	}
}
