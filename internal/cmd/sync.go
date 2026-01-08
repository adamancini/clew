package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adamancini/clew/internal/backup"
	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/interactive"
	"github.com/adamancini/clew/internal/output"
	"github.com/adamancini/clew/internal/state"
	"github.com/adamancini/clew/internal/sync"
)

func newSyncCmd() *cobra.Command {
	var (
		strict          bool
		interactiveMode bool
		doBackup        bool
		noBackup        bool
		short           bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile system to match Clewfile",
		Long: `Sync reads the Clewfile and ensures the system matches the declared state.

By default, shows executed commands with descriptions and results.
Use --short for one-line-per-item output format.
Use --backup to create a backup before making changes (default behavior).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --backup flag takes precedence, --no-backup disables
			createBackup := doBackup || !noBackup
			return runSync(strict, interactiveMode, createBackup, short)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Exit non-zero on any failure")
	cmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "Prompt for confirmation of each change")
	cmd.Flags().BoolVar(&doBackup, "backup", false, "Create backup before sync (default behavior)")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip creating backup before sync")
	cmd.Flags().BoolVar(&short, "short", false, "One-line per item output format")

	return cmd
}

// runSync executes the sync workflow.
func runSync(strict bool, interactiveMode bool, createBackup bool, short bool) error {
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

	// 5. Compute diff
	diffResult := diff.Compute(clewfile, currentState)

	// Check if there's anything to do
	add, update, _, attention := diffResult.Summary()
	if add == 0 && update == 0 && attention == 0 {
		if !quiet {
			fmt.Println("Already in sync. Nothing to do.")
		}
		return nil
	}

	// 6. Handle interactive mode
	if interactiveMode {
		// Check if we're in a TTY
		if !interactive.IsTerminal() {
			fmt.Fprintln(os.Stderr, "Warning: Not running in a terminal. Falling back to non-interactive mode.")
			interactiveMode = false
		}
	}

	if interactiveMode {
		prompter := interactive.NewPrompter()
		selection, proceed := prompter.PromptForSelection(diffResult)
		if !proceed {
			return nil
		}
		// Filter diff to only include approved items
		diffResult = interactive.FilterDiffBySelection(diffResult, selection)
	}

	// 6.5. Create backup before making changes
	if createBackup {
		manager, err := backup.NewManager(clewVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize backup manager: %v\n", err)
		} else {
			bak, err := manager.Create(currentState, "Auto (sync)")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)
			} else if verbose {
				fmt.Fprintf(os.Stderr, "Backup created: %s\n", bak.ID)
			}
		}
	}

	// 7. Execute sync
	syncer := sync.NewSyncer()
	result, err := syncer.Execute(diffResult, sync.Options{
		Strict:  strict,
		Verbose: verbose,
		Quiet:   quiet,
		Short:   short,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during sync: %v\n", err)
		os.Exit(1)
	}

	// 8. Format and display output
	format, err := output.ParseFormat(outputFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if format == output.FormatText {
		printSyncResultText(result, sync.Options{
			Short:   short,
			Quiet:   quiet,
			Verbose: verbose,
		})
	} else {
		writer := output.NewWriter(os.Stdout, format)
		if err := writer.Write(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	}

	// Handle exit codes
	if result.Failed > 0 {
		if strict {
			os.Exit(2)
		}
		os.Exit(1)
	}

	return nil
}

// printSyncResultText outputs the sync result in human-readable format.
func printSyncResultText(result *sync.Result, opts sync.Options) {
	if opts.Short {
		printSyncResultShort(result)
	} else {
		printSyncResultVerbose(result)
	}
}

// printSyncResultVerbose outputs detailed sync results with commands and descriptions.
func printSyncResultVerbose(result *sync.Result) {
	// Print each operation with full details
	for i, op := range result.Operations {
		// Print action + description
		fmt.Printf("%s: %s\n", capitalizeAction(op.Action), op.Description)

		// Print the command executed
		if op.Command != "" {
			fmt.Printf("\u2192 %s\n", op.Command)
		}

		// Print success or failure
		if op.Success {
			fmt.Println("\u2713 Success")
		} else {
			if op.Error != "" {
				fmt.Printf("\u2717 Failed: %s\n", op.Error)
			} else {
				fmt.Println("\u2717 Failed")
			}
		}

		// Add blank line between operations (but not after the last one)
		if i < len(result.Operations)-1 {
			fmt.Println()
		}
	}

	// Add separator before summary if there were operations
	if len(result.Operations) > 0 {
		fmt.Println()
	}

	// Print summary
	fmt.Println("Summary:")
	fmt.Printf("  Installed: %d\n", result.Installed)
	fmt.Printf("  Updated: %d\n", result.Updated)
	fmt.Printf("  Failed: %d\n", result.Failed)

	if result.Skipped > 0 {
		fmt.Printf("  Skipped: %d\n", result.Skipped)
	}

	// TODO: Format git warnings from result.GitWarnings when issue #39 is implemented

	if len(result.Attention) > 0 {
		fmt.Println("\nItems needing attention:")
		for _, item := range result.Attention {
			fmt.Printf("  - %s\n", item)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}
}

// printSyncResultShort outputs a one-line-per-item summary of sync results.
func printSyncResultShort(result *sync.Result) {
	// Print each operation on one line
	for _, op := range result.Operations {
		if op.Success {
			fmt.Printf("\u2713 %s (%s %s)\n", op.Name, op.Type, op.Action)
		} else {
			fmt.Printf("\u2717 %s (%s %s)\n", op.Name, op.Type, op.Action)
			if op.Error != "" {
				fmt.Printf("  Error: %s\n", op.Error)
			}
		}
	}

	// Add blank line before summary if there were operations
	if len(result.Operations) > 0 {
		fmt.Println()
	}

	// Print summary
	parts := []string{}
	if result.Installed > 0 || result.Updated > 0 || result.Failed == 0 {
		parts = append(parts, fmt.Sprintf("%d installed", result.Installed))
		parts = append(parts, fmt.Sprintf("%d updated", result.Updated))
	}
	if result.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", result.Failed))
	}
	fmt.Printf("Summary: %s\n", strings.Join(parts, ", "))

	// TODO: Format git warnings from result.GitWarnings when issue #39 is implemented

	if len(result.Attention) > 0 {
		fmt.Println("\nItems needing attention:")
		for _, item := range result.Attention {
			fmt.Printf("  - %s\n", item)
		}
	}
}

// capitalizeAction capitalizes the first letter of an action string.
func capitalizeAction(action string) string {
	if len(action) == 0 {
		return action
	}
	return strings.ToUpper(action[:1]) + action[1:]
}
