package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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

// runExport executes the export workflow.
func runExport() error {
	// 1. Read current state
	reader := &state.FilesystemReader{}
	currentState, err := reader.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading current state: %v\n", err)
		os.Exit(1)
	}

	// 2. Resolve marketplaces directory for orphan detection
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	marketplacesDir := filepath.Join(home, ".claude", "plugins", "marketplaces")

	// 3. Convert state to Clewfile structure
	exported := convertStateToClewfile(currentState, marketplacesDir)

	// 4. Output in the specified format
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
// marketplacesDir is the path to the marketplaces directory (e.g., ~/.claude/plugins/marketplaces)
// used to verify that plugins still exist in their marketplace before exporting.
func convertStateToClewfile(s *state.State, marketplacesDir string) *ExportedClewfile {
	exported := &ExportedClewfile{
		Version:      1,
		Marketplaces: make(map[string]ExportedMarketplace),
		Plugins:      make([]ExportedPlugin, 0),
	}

	// Convert marketplaces and track valid marketplace names.
	// Skip local/directory-sourced marketplaces (empty Repo) since they
	// cannot be represented in a portable Clewfile.
	validMarketplaces := make(map[string]bool)
	var skippedMarketplaces []string
	for alias, m := range s.Marketplaces {
		if m.Repo == "" {
			skippedMarketplaces = append(skippedMarketplaces, alias)
			continue
		}
		em := ExportedMarketplace{
			Repo: m.Repo,
		}
		if m.Ref != "" {
			em.Ref = m.Ref
		}
		exported.Marketplaces[alias] = em
		validMarketplaces[alias] = true
	}
	if len(skippedMarketplaces) > 0 {
		sort.Strings(skippedMarketplaces)
		fmt.Fprintf(os.Stderr, "Note: Skipped %d local marketplace(s) (no repo): %v\n",
			len(skippedMarketplaces), skippedMarketplaces)
	}

	// Convert plugins, skipping those that reference non-existent marketplaces
	// or whose directory no longer exists in the marketplace.
	var skippedNoMarketplace []string // plugin references a marketplace not in exported state
	var skippedOrphaned []string      // plugin's marketplace exists but plugin directory doesn't
	for fullName, p := range s.Plugins {
		// Parse plugin@marketplace format and check if marketplace exists
		if parts := strings.SplitN(fullName, "@", 2); len(parts) == 2 {
			pluginName := parts[0]
			marketplace := parts[1]
			if !validMarketplaces[marketplace] {
				skippedNoMarketplace = append(skippedNoMarketplace, fullName)
				continue // Skip this plugin - marketplace not in exported state
			}

			// Check if the plugin directory actually exists in the marketplace
			pluginDir := filepath.Join(marketplacesDir, marketplace, "plugins", pluginName)
			if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
				skippedOrphaned = append(skippedOrphaned, fullName)
				continue // Skip this plugin - not found in marketplace directory
			}
		}

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

	// Log skipped plugins to stderr
	if len(skippedNoMarketplace) > 0 {
		sort.Strings(skippedNoMarketplace)
		fmt.Fprintf(os.Stderr, "Note: Skipped %d plugin(s) referencing non-marketplace sources: %v\n",
			len(skippedNoMarketplace), skippedNoMarketplace)
	}
	if len(skippedOrphaned) > 0 {
		sort.Strings(skippedOrphaned)
		fmt.Fprintf(os.Stderr, "Note: Skipped %d plugin(s) not found in marketplace directory: %v\n",
			len(skippedOrphaned), skippedOrphaned)
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

	// Clean up empty slices/maps for nicer output
	if len(exported.Marketplaces) == 0 {
		exported.Marketplaces = nil
	}
	if len(exported.Plugins) == 0 {
		exported.Plugins = nil
	}

	return exported
}
