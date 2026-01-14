package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/git"
	"github.com/adamancini/clew/internal/sync"
)

func newSyncCmd() *cobra.Command {
	var (
		strict          bool
		interactiveMode bool
		doBackup        bool
		noBackup        bool
		short           bool
		showCommands    bool
		skipGitCheck    bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile system to match Clewfile",
		Long: `Sync reads the Clewfile and ensures the system matches the declared state.

By default, shows executed commands with descriptions and results.
Use --short for one-line-per-item output format.
Use --backup to create a backup before making changes (default behavior).

For local marketplaces and plugins, git status is checked before sync:
- Uncommitted changes: Warning + skip that repository
- Behind remote: Info + suggest 'git pull'
- Ahead of remote: Info + suggest 'git push'

Use --skip-git-check to bypass git status checking.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --backup flag takes precedence, --no-backup disables
			createBackup := doBackup || !noBackup
			return runSync(strict, interactiveMode, createBackup, short, showCommands, skipGitCheck)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Exit non-zero on any failure")
	cmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "Prompt for confirmation of each change")
	cmd.Flags().BoolVar(&doBackup, "backup", false, "Create backup before sync (default behavior)")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip creating backup before sync")
	cmd.Flags().BoolVar(&short, "short", false, "One-line per item output format")
	cmd.Flags().BoolVar(&showCommands, "show-commands", false, "Output CLI commands instead of executing")
	cmd.Flags().BoolVar(&skipGitCheck, "skip-git-check", false, "Skip git status checks for local repositories")

	return cmd
}

// runSync executes the sync workflow using the SyncService.
func runSync(strict bool, interactiveMode bool, createBackup bool, short bool, showCommands bool, skipGitCheck bool) error {
	service := NewSyncService(configPath, clewVersion)

	opts := SyncOptions{
		Strict:       strict,
		Interactive:  interactiveMode,
		CreateBackup: createBackup,
		Short:        short,
		ShowCommands: showCommands,
		SkipGitCheck: skipGitCheck,
		OutputFormat: outputFormat,
		Verbose:      verbose,
		Quiet:        quiet,
	}

	err := service.Run(opts)
	if err != nil {
		// Check if this is a failure exit code error
		if err.Error() == "sync completed with failures (strict mode)" {
			os.Exit(2)
		}
		// For other failures, exit with code 1
		if strings.Contains(err.Error(), "sync completed with") {
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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

// filterDiffByGitStatus marks items as skipped if they have git issues.
// Items with uncommitted changes are marked to be skipped, and their status
// is added to the attention list during sync.
func filterDiffByGitStatus(d *diff.Result, gitResult *git.CheckResult) *diff.Result {
	if gitResult == nil {
		return d
	}

	filtered := &diff.Result{
		Sources:    make([]diff.SourceDiff, 0, len(d.Sources)),
		Plugins:    make([]diff.PluginDiff, 0, len(d.Plugins)),
		MCPServers: d.MCPServers, // MCP servers are not affected by git status
	}

	// Filter sources - skip those with git issues
	for _, src := range d.Sources {
		if gitResult.ShouldSkipSource(src.Name) {
			// Change action to indicate this needs attention
			src.Action = diff.ActionSkipGit
		}
		filtered.Sources = append(filtered.Sources, src)
	}

	// Filter plugins - skip those with git issues
	for _, p := range d.Plugins {
		if gitResult.ShouldSkipPlugin(p.Name) {
			// Change action to indicate this needs attention
			p.Action = diff.ActionSkipGit
		}
		filtered.Plugins = append(filtered.Plugins, p)
	}

	return filtered
}
