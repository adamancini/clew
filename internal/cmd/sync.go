package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/adamancini/clew/internal/backup"
	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/git"
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
		showCommands    bool
		skipGitCheck    bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile system to match Clewfile",
		Long: `Sync reads the Clewfile and ensures the system matches the declared state.

By default, a backup is created before making changes. Use --no-backup to disable.

For local marketplaces and plugins, git status is checked before sync:
- Uncommitted changes: Warning + skip that repository
- Behind remote: Info + suggest 'git pull'
- Ahead of remote: Info + suggest 'git push'

Use --skip-git-check to bypass git status checking.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --backup flag takes precedence, --no-backup disables
			createBackup := doBackup || !noBackup
			return runSync(strict, interactiveMode, createBackup, showCommands, skipGitCheck)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Exit non-zero on any failure")
	cmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "Prompt for confirmation of each change")
	cmd.Flags().BoolVar(&doBackup, "backup", false, "Create backup before sync (default behavior)")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip creating backup before sync")
	cmd.Flags().BoolVar(&showCommands, "show-commands", false, "Output CLI commands instead of executing")
	cmd.Flags().BoolVar(&skipGitCheck, "skip-git-check", false, "Skip git status checks for local repositories")

	return cmd
}

// runSync executes the sync workflow.
func runSync(strict bool, interactiveMode bool, createBackup bool, showCommands bool, skipGitCheck bool) error {
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

	// 6.6. Check git status for local repositories
	var gitResult *git.CheckResult
	if !skipGitCheck {
		gitChecker := git.NewChecker()
		gitResult = gitChecker.CheckClewfile(clewfile)

		// Display git warnings
		if gitResult.HasWarnings() {
			fmt.Fprintln(os.Stderr, "\nGit Status Warnings:")
			for _, warning := range gitResult.Warnings {
				fmt.Fprintf(os.Stderr, "  - %s\n", warning)
			}
		}

		// Display git info (if verbose)
		if gitResult.HasInfo() && verbose {
			fmt.Fprintln(os.Stderr, "\nGit Status Info:")
			for _, info := range gitResult.Info {
				fmt.Fprintf(os.Stderr, "  - %s\n", info)
			}
		}

		// Filter diff to skip items with git issues
		diffResult = filterDiffByGitStatus(diffResult, gitResult)
	}

	// 7. Execute sync
	syncer := sync.NewSyncer()
	result, err := syncer.Execute(diffResult, sync.Options{
		Strict:  strict,
		Verbose: verbose,
		Quiet:   quiet,
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
		printSyncResultText(result)
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
func printSyncResultText(result *sync.Result) {
	if result.Installed > 0 {
		fmt.Printf("Installed: %d\n", result.Installed)
	}
	if result.Updated > 0 {
		fmt.Printf("Updated: %d\n", result.Updated)
	}
	if result.Skipped > 0 {
		fmt.Printf("Skipped: %d\n", result.Skipped)
	}
	if result.Failed > 0 {
		fmt.Printf("Failed: %d\n", result.Failed)
	}

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

// filterDiffByGitStatus marks items as skipped if they have git issues.
// Items with uncommitted changes are marked to be skipped, and their status
// is added to the attention list during sync.
func filterDiffByGitStatus(d *diff.Result, gitResult *git.CheckResult) *diff.Result {
	if gitResult == nil {
		return d
	}

	filtered := &diff.Result{
		Marketplaces: make([]diff.MarketplaceDiff, 0, len(d.Marketplaces)),
		Plugins:      make([]diff.PluginDiff, 0, len(d.Plugins)),
		MCPServers:   d.MCPServers, // MCP servers are not affected by git status
	}

	// Filter marketplaces - skip those with git issues
	for _, m := range d.Marketplaces {
		if gitResult.ShouldSkipMarketplace(m.Name) {
			// Change action to indicate this needs attention
			m.Action = diff.ActionSkipGit
		}
		filtered.Marketplaces = append(filtered.Marketplaces, m)
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
