package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/interactive"
	"github.com/adamancini/clew/internal/output"
)

func newDiffCmd() *cobra.Command {
	var (
		interactiveMode bool
		showCommands    bool
	)

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show what would change (dry-run)",
		Long:  `Diff compares the Clewfile against current state and shows what sync would do.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff(interactiveMode, showCommands)
		},
	}

	cmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "Preview changes with prompts (dry-run)")
	cmd.Flags().BoolVar(&showCommands, "show-commands", false, "Output CLI commands to reconcile state")

	return cmd
}

// runDiff executes the diff workflow (dry-run mode).
func runDiff(interactiveMode bool, showCommands bool) error {
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
	reader := getStateReader()
	currentState, err := reader.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading current state: %v\n", err)
		os.Exit(1)
	}

	// 5. Compute diff
	diffResult := diff.Compute(clewfile, currentState)

	// 5a. Handle --show-commands flag
	if showCommands {
		commands := diffResult.GenerateCommands()
		if len(commands) == 0 {
			fmt.Println("# No commands needed - already in sync")
			return nil
		}

		// Output commands based on format
		format, err := output.ParseFormat(outputFormat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if format == output.FormatText {
			fmt.Println(diff.FormatCommands(commands, true))
		} else {
			writer := output.NewWriter(os.Stdout, format)
			if err := writer.Write(commands); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
				os.Exit(1)
			}
		}
		return nil
	}

	// 6. Handle interactive mode (preview with prompts)
	if interactiveMode {
		// Check if we're in a TTY
		if !interactive.IsTerminal() {
			fmt.Fprintln(os.Stderr, "Warning: Not running in a terminal. Falling back to non-interactive mode.")
			interactiveMode = false
		}
	}

	if interactiveMode {
		prompter := interactive.NewPrompter()
		selection, _ := prompter.PromptForSelection(diffResult)
		if selection != nil {
			// Show what would have been selected (dry-run only, no execution)
			filteredResult := interactive.FilterDiffBySelection(diffResult, selection)
			fmt.Println("\n--- Dry-run complete. No changes were made. ---")
			printDiffResultText(filteredResult)
		}
		return nil
	}

	// 7. Format and display output
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

	// Sources
	hasSourceChanges := false
	for _, src := range result.Sources {
		if src.Action == diff.ActionNone {
			continue
		}
		if !hasSourceChanges {
			fmt.Println("Sources:")
			hasSourceChanges = true
		}
		printDiffItem("source", src.Name, src.Action, src.Desired != nil, src.Current != nil)
	}

	// Plugins
	hasPluginChanges := false
	for _, p := range result.Plugins {
		if p.Action == diff.ActionNone {
			continue
		}
		if !hasPluginChanges {
			if hasSourceChanges {
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
			if hasSourceChanges || hasPluginChanges {
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
