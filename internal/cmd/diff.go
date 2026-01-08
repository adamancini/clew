package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/output"
	"github.com/adamancini/clew/internal/state"
)

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Show what would change (dry-run)",
		Long:  `Diff compares the Clewfile against current state and shows what sync would do.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff()
		},
	}
}

// runDiff executes the diff workflow (dry-run mode).
func runDiff() error {
	// 1. Find Clewfile
	clewfilePath, err := config.FindClewfile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Using Clewfile: %s\n", clewfilePath)
	}

	// 2. Load Clewfile
	clewfile, err := config.Load(clewfilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 3. Infer scope
	scope := config.InferScope(clewfilePath)
	if verbose {
		fmt.Fprintf(os.Stderr, "Inferred scope: %s\n", scope)
	}

	// 4. Read current state
	var reader state.Reader
	if useFilesystem {
		reader = &state.FilesystemReader{}
	} else {
		reader = &state.CLIReader{}
	}

	currentState, err := reader.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading current state: %v\n", err)
		os.Exit(1)
	}

	// 5. Compute diff
	diffResult := diff.Compute(clewfile, currentState)

	// 6. Format and display output
	format, err := output.ParseFormat(outputFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if format == output.FormatText {
		printDiffResultText(diffResult)
	} else {
		writer := output.NewWriter(os.Stdout, format)
		if err := writer.Write(diffResult); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	}

	return nil
}

// printDiffResultText outputs the diff result in human-readable format.
func printDiffResultText(result *diff.Result) {
	add, update, remove, attention := result.Summary()

	// Print summary first
	if add == 0 && update == 0 && remove == 0 && attention == 0 {
		fmt.Println("Already in sync. Nothing would change.")
		return
	}

	fmt.Println("Changes that would be made:")
	fmt.Println()

	// Marketplaces
	hasMarketplaceChanges := false
	for _, m := range result.Marketplaces {
		if m.Action == diff.ActionNone {
			continue
		}
		if !hasMarketplaceChanges {
			fmt.Println("Marketplaces:")
			hasMarketplaceChanges = true
		}
		printDiffItem("marketplace", m.Name, m.Action, m.Desired != nil, m.Current != nil)
	}

	// Plugins
	hasPluginChanges := false
	for _, p := range result.Plugins {
		if p.Action == diff.ActionNone {
			continue
		}
		if !hasPluginChanges {
			if hasMarketplaceChanges {
				fmt.Println()
			}
			fmt.Println("Plugins:")
			hasPluginChanges = true
		}
		printDiffItem("plugin", p.Name, p.Action, p.Desired != nil, p.Current != nil)
	}

	// MCP Servers
	hasMCPChanges := false
	for _, m := range result.MCPServers {
		if m.Action == diff.ActionNone {
			continue
		}
		if !hasMCPChanges {
			if hasMarketplaceChanges || hasPluginChanges {
				fmt.Println()
			}
			fmt.Println("MCP Servers:")
			hasMCPChanges = true
		}
		extra := ""
		if m.RequiresOAuth {
			extra = " (requires OAuth - manual setup needed)"
		}
		printDiffItem("mcp", m.Name, m.Action, m.Desired != nil, m.Current != nil)
		if extra != "" {
			fmt.Printf("    %s\n", extra)
		}
	}

	// Summary
	fmt.Println()
	fmt.Printf("Summary: %d to add, %d to update, %d to remove, %d need attention\n",
		add, update, remove, attention)
}

// printDiffItem prints a single diff item with appropriate formatting.
func printDiffItem(itemType, name string, action diff.Action, hasDesired, hasCurrent bool) {
	var symbol, verb string
	switch action {
	case diff.ActionAdd:
		symbol = "+"
		verb = "add"
	case diff.ActionRemove:
		symbol = "-"
		verb = "remove (not in Clewfile)"
	case diff.ActionUpdate:
		symbol = "~"
		verb = "update"
	case diff.ActionEnable:
		symbol = "+"
		verb = "enable"
	case diff.ActionDisable:
		symbol = "-"
		verb = "disable"
	default:
		symbol = " "
		verb = ""
	}

	fmt.Printf("  %s %s: %s\n", symbol, name, verb)
}
