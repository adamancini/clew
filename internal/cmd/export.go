package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/adamancini/clew/internal/output"
	"github.com/adamancini/clew/internal/state"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export current state as Clewfile",
		Long:  `Export reads the current Claude Code configuration and outputs it as a Clewfile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport()
		},
	}
}

// ExportedClewfile represents the exported configuration in Clewfile format.
// This is separate from config.Clewfile to allow for cleaner serialization.
type ExportedClewfile struct {
	Version    int                          `json:"version" yaml:"version"`
	Sources    []ExportedSource             `json:"sources,omitempty" yaml:"sources,omitempty"`
	Plugins    []ExportedPlugin             `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	MCPServers map[string]ExportedMCPServer `json:"mcp_servers,omitempty" yaml:"mcp_servers,omitempty"`
}

// ExportedSource represents a source for export.
type ExportedSource struct {
	Name   string              `json:"name" yaml:"name"`
	Alias  string              `json:"alias,omitempty" yaml:"alias,omitempty"`
	Kind   string              `json:"kind" yaml:"kind"`
	Source ExportedSourceConfig `json:"source" yaml:"source"`
}

// ExportedSourceConfig represents source configuration for export.
type ExportedSourceConfig struct {
	Type string `json:"type" yaml:"type"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
	Ref  string `json:"ref,omitempty" yaml:"ref,omitempty"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

// ExportedPlugin represents a plugin for export.
type ExportedPlugin struct {
	Name    string `json:"name" yaml:"name"`
	Enabled *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Scope   string `json:"scope,omitempty" yaml:"scope,omitempty"`
}

// ExportedMCPServer represents an MCP server for export.
type ExportedMCPServer struct {
	Transport string   `json:"transport" yaml:"transport"`
	Command   string   `json:"command,omitempty" yaml:"command,omitempty"`
	Args      []string `json:"args,omitempty" yaml:"args,omitempty"`
	URL       string   `json:"url,omitempty" yaml:"url,omitempty"`
	Scope     string   `json:"scope,omitempty" yaml:"scope,omitempty"`
}

// runExport executes the export workflow.
func runExport() error {
	// 1. Read current state
	var reader state.Reader
	if useCLI {
		// CLI reader is experimental and currently broken (issue #34)
		reader = &state.CLIReader{}
	} else {
		// Filesystem reader is the default
		reader = &state.FilesystemReader{}
	}

	currentState, err := reader.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading current state: %v\n", err)
		os.Exit(1)
	}

	// 2. Convert state to Clewfile structure
	exported := convertStateToClewfile(currentState)

	// 3. Output in the specified format
	format, err := output.ParseFormat(outputFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Default to YAML for export text format (most readable)
	if format == output.FormatText {
		format = output.FormatYAML
	}

	writer := output.NewWriter(os.Stdout, format)
	if err := writer.Write(exported); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	return nil
}

// convertStateToClewfile converts the current state to a Clewfile structure.
func convertStateToClewfile(s *state.State) *ExportedClewfile {
	exported := &ExportedClewfile{
		Version:    1,
		Sources:    make([]ExportedSource, 0),
		Plugins:    make([]ExportedPlugin, 0),
		MCPServers: make(map[string]ExportedMCPServer),
	}

	// Convert sources
	for name, src := range s.Sources {
		es := ExportedSource{
			Name: name,
			Kind: src.Kind,
			Source: ExportedSourceConfig{
				Type: src.Type,
				URL:  src.URL,
				Ref:  src.Ref,
				Path: src.Path,
			},
		}
		exported.Sources = append(exported.Sources, es)
	}

	// Convert plugins
	for fullName, p := range s.Plugins {
		ep := ExportedPlugin{
			Name: fullName,
		}
		// Only include enabled if false (default is true)
		if !p.Enabled {
			enabled := false
			ep.Enabled = &enabled
		}
		// Only include scope if not empty
		if p.Scope != "" && p.Scope != "user" {
			ep.Scope = p.Scope
		}
		exported.Plugins = append(exported.Plugins, ep)
	}

	// Convert MCP servers
	for name, m := range s.MCPServers {
		em := ExportedMCPServer{
			Transport: m.Transport,
		}
		switch m.Transport {
		case "stdio":
			em.Command = m.Command
			if len(m.Args) > 0 {
				em.Args = m.Args
			}
		case "http", "sse":
			em.URL = m.URL
		}
		// Only include scope if not user (default)
		if m.Scope != "" && m.Scope != "user" {
			em.Scope = m.Scope
		}
		exported.MCPServers[name] = em
	}

	// Clean up empty slices/maps for nicer output
	if len(exported.Sources) == 0 {
		exported.Sources = nil
	}
	if len(exported.Plugins) == 0 {
		exported.Plugins = nil
	}
	if len(exported.MCPServers) == 0 {
		exported.MCPServers = nil
	}

	return exported
}
