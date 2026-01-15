package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

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
	Version      int                           `json:"version" yaml:"version"`
	Marketplaces map[string]ExportedMarketplace `json:"marketplaces,omitempty" yaml:"marketplaces,omitempty"`
	Plugins      []ExportedPlugin              `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	MCPServers   map[string]ExportedMCPServer  `json:"mcp_servers,omitempty" yaml:"mcp_servers,omitempty"`
}

// ExportedMarketplace represents a marketplace for export.
type ExportedMarketplace struct {
	Repo string `json:"repo" yaml:"repo"`
	Ref  string `json:"ref,omitempty" yaml:"ref,omitempty"`
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
	reader := getStateReader()
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
		Version:      1,
		Marketplaces: make(map[string]ExportedMarketplace),
		Plugins:      make([]ExportedPlugin, 0),
		MCPServers:   make(map[string]ExportedMCPServer),
	}

	// Convert marketplaces
	for alias, m := range s.Marketplaces {
		em := ExportedMarketplace{
			Repo: m.Repo,
		}
		if m.Ref != "" {
			em.Ref = m.Ref
		}
		exported.Marketplaces[alias] = em
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

	// Sort plugins by marketplace name, then by plugin name for readability
	sort.Slice(exported.Plugins, func(i, j int) bool {
		iMarketplace := ""
		jMarketplace := ""

		// Extract marketplace name from plugin name (part after @)
		if strings.Contains(exported.Plugins[i].Name, "@") {
			parts := strings.SplitN(exported.Plugins[i].Name, "@", 2)
			iMarketplace = parts[1]
		}
		if strings.Contains(exported.Plugins[j].Name, "@") {
			parts := strings.SplitN(exported.Plugins[j].Name, "@", 2)
			jMarketplace = parts[1]
		}

		// Sort by marketplace first
		if iMarketplace != jMarketplace {
			return iMarketplace < jMarketplace
		}

		// Then by plugin name
		return exported.Plugins[i].Name < exported.Plugins[j].Name
	})

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
	if len(exported.Marketplaces) == 0 {
		exported.Marketplaces = nil
	}
	if len(exported.Plugins) == 0 {
		exported.Plugins = nil
	}
	if len(exported.MCPServers) == 0 {
		exported.MCPServers = nil
	}

	return exported
}
