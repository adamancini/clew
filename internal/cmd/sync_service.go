// Package cmd contains the CLI command implementations.
package cmd

import (
	"fmt"
	"os"

	"github.com/adamancini/clew/internal/backup"
	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/git"
	"github.com/adamancini/clew/internal/interactive"
	"github.com/adamancini/clew/internal/output"
	"github.com/adamancini/clew/internal/state"
	"github.com/adamancini/clew/internal/sync"
)

// SyncOptions configures sync behavior.
type SyncOptions struct {
	Strict       bool   // Exit non-zero on any failure
	Interactive  bool   // Prompt for confirmation of each change
	CreateBackup bool   // Create backup before sync
	Short        bool   // One-line per item output format
	ShowCommands bool   // Output CLI commands instead of executing
	SkipGitCheck bool   // Skip git status checks for local repositories
	OutputFormat string // Output format (text, json, yaml)
	Verbose      bool   // Verbose output
	Quiet        bool   // Quiet mode (errors only)
}

// SyncService orchestrates the sync workflow with proper separation of concerns.
// It breaks down the complex sync operation into smaller, testable steps.
type SyncService struct {
	configPath  string
	stateReader state.Reader
	syncer      *sync.Syncer
	backupMgr   *backup.Manager
	gitChecker  *git.Checker
	prompter    *interactive.Prompter
	version     string
}

// NewSyncService creates a new sync service with default dependencies.
func NewSyncService(configPath, version string) *SyncService {
	return &SyncService{
		configPath:  configPath,
		stateReader: getStateReader(),
		syncer:      sync.NewSyncer(),
		gitChecker:  git.NewChecker(),
		version:     version,
	}
}

// NewSyncServiceWithDeps creates a sync service with custom dependencies (for testing).
func NewSyncServiceWithDeps(
	configPath string,
	stateReader state.Reader,
	syncer *sync.Syncer,
	backupMgr *backup.Manager,
	gitChecker *git.Checker,
	version string,
) *SyncService {
	return &SyncService{
		configPath:  configPath,
		stateReader: stateReader,
		syncer:      syncer,
		backupMgr:   backupMgr,
		gitChecker:  gitChecker,
		version:     version,
	}
}

// LoadConfiguration finds and loads the Clewfile.
func (s *SyncService) LoadConfiguration() (*config.Clewfile, string, error) {
	clewfilePath, err := config.FindClewfile(s.configPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to find Clewfile: %w", err)
	}

	clewfile, err := config.Load(clewfilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load Clewfile: %w", err)
	}

	return clewfile, clewfilePath, nil
}

// ReadCurrentState reads the current Claude Code configuration state.
func (s *SyncService) ReadCurrentState() (*state.State, error) {
	currentState, err := s.stateReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read current state: %w", err)
	}
	return currentState, nil
}

// ComputeDiff computes the differences between desired and current state.
func (s *SyncService) ComputeDiff(clewfile *config.Clewfile, currentState *state.State) *diff.Result {
	return diff.Compute(clewfile, currentState)
}

// IsInSync checks if the system is already in sync with the Clewfile.
func (s *SyncService) IsInSync(diffResult *diff.Result) bool {
	add, update, _, attention := diffResult.Summary()
	return add == 0 && update == 0 && attention == 0
}

// ValidateGitStatus checks git status for local repositories and returns warnings.
func (s *SyncService) ValidateGitStatus(clewfile *config.Clewfile) *git.CheckResult {
	if s.gitChecker == nil {
		return nil
	}
	return s.gitChecker.CheckClewfile(clewfile)
}

// FilterDiffByGitStatus filters the diff to skip items with git issues.
func (s *SyncService) FilterDiffByGitStatus(diffResult *diff.Result, gitResult *git.CheckResult) *diff.Result {
	if gitResult == nil {
		return diffResult
	}
	return filterDiffByGitStatus(diffResult, gitResult)
}

// CreateBackup creates a backup of the current state.
func (s *SyncService) CreateBackup(currentState *state.State) (*backup.Backup, error) {
	if s.backupMgr == nil {
		var err error
		s.backupMgr, err = backup.NewManager(s.version)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize backup manager: %w", err)
		}
	}
	return s.backupMgr.Create(currentState, "Auto (sync)")
}

// GetUserApproval prompts the user for confirmation in interactive mode.
// Returns the filtered diff and whether to proceed.
func (s *SyncService) GetUserApproval(diffResult *diff.Result) (*diff.Result, bool, error) {
	if s.prompter == nil {
		s.prompter = interactive.NewPrompter()
	}

	if !interactive.IsTerminal() {
		return diffResult, true, fmt.Errorf("not running in a terminal, falling back to non-interactive mode")
	}

	selection, proceed := s.prompter.PromptForSelection(diffResult)
	if !proceed {
		return nil, false, nil
	}

	filtered := interactive.FilterDiffBySelection(diffResult, selection)
	return filtered, true, nil
}

// ExecuteSync applies the diff to bring the system in line with the Clewfile.
func (s *SyncService) ExecuteSync(diffResult *diff.Result, opts SyncOptions) (*sync.Result, error) {
	return s.syncer.Execute(diffResult, sync.Options{
		Strict:  opts.Strict,
		Verbose: opts.Verbose,
		Quiet:   opts.Quiet,
		Short:   opts.Short,
	})
}

// GenerateCommands generates CLI commands without executing them.
func (s *SyncService) GenerateCommands(diffResult *diff.Result) []diff.Command {
	return diffResult.GenerateCommands()
}

// FormatOutput formats the output according to the specified format.
func (s *SyncService) FormatOutput(format output.Format, data interface{}) error {
	writer := output.NewWriter(os.Stdout, format)
	return writer.Write(data)
}

// Run executes the complete sync workflow.
// This is the main entry point that orchestrates all the steps.
func (s *SyncService) Run(opts SyncOptions) error {
	// 1. Load configuration
	clewfile, clewfilePath, err := s.LoadConfiguration()
	if err != nil {
		return err
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Using Clewfile: %s\n", clewfilePath)
		fmt.Fprintf(os.Stderr, "Inferred scope: %s\n", config.InferScope(clewfilePath))
	}

	// 2. Read current state
	currentState, err := s.ReadCurrentState()
	if err != nil {
		return err
	}

	// 3. Compute diff
	diffResult := s.ComputeDiff(clewfile, currentState)

	// 4. Check if already in sync
	if s.IsInSync(diffResult) {
		if !opts.Quiet {
			fmt.Println("Already in sync. Nothing to do.")
		}
		return nil
	}

	// 5. Handle --show-commands flag
	if opts.ShowCommands {
		return s.handleShowCommands(diffResult, opts)
	}

	// 6. Handle interactive mode
	if opts.Interactive {
		diffResult, err = s.handleInteractiveMode(diffResult)
		if err != nil {
			if !opts.Quiet {
				fmt.Fprintln(os.Stderr, "Warning: "+err.Error())
			}
			// Continue in non-interactive mode
		}
		if diffResult == nil {
			return nil // User cancelled
		}
	}

	// 7. Create backup
	if opts.CreateBackup {
		s.handleBackup(currentState, opts.Verbose)
	}

	// 8. Check git status
	if !opts.SkipGitCheck {
		diffResult = s.handleGitCheck(clewfile, diffResult, opts.Verbose)
	}

	// 9. Execute sync
	result, err := s.ExecuteSync(diffResult, opts)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// 10. Format and display output
	return s.handleOutput(result, opts)
}

// handleShowCommands handles the --show-commands flag.
func (s *SyncService) handleShowCommands(diffResult *diff.Result, opts SyncOptions) error {
	commands := s.GenerateCommands(diffResult)
	if len(commands) == 0 {
		fmt.Println("# No commands needed - already in sync")
		return nil
	}

	format, err := output.ParseFormat(opts.OutputFormat)
	if err != nil {
		return err
	}

	if format == output.FormatText {
		fmt.Println(diff.FormatCommands(commands, true))
	} else {
		return s.FormatOutput(format, commands)
	}
	return nil
}

// handleInteractiveMode handles the interactive mode workflow.
func (s *SyncService) handleInteractiveMode(diffResult *diff.Result) (*diff.Result, error) {
	filtered, proceed, err := s.GetUserApproval(diffResult)
	if err != nil {
		return diffResult, err // Return original diff with warning
	}
	if !proceed {
		return nil, nil
	}
	return filtered, nil
}

// handleBackup creates a backup before sync.
func (s *SyncService) handleBackup(currentState *state.State, verbose bool) {
	bak, err := s.CreateBackup(currentState)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)
		return
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Backup created: %s\n", bak.ID)
	}
}

// handleGitCheck performs git status checking for local repositories.
func (s *SyncService) handleGitCheck(clewfile *config.Clewfile, diffResult *diff.Result, verbose bool) *diff.Result {
	gitResult := s.ValidateGitStatus(clewfile)
	if gitResult == nil {
		return diffResult
	}

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

	return s.FilterDiffByGitStatus(diffResult, gitResult)
}

// handleOutput formats and displays the sync result.
func (s *SyncService) handleOutput(result *sync.Result, opts SyncOptions) error {
	format, err := output.ParseFormat(opts.OutputFormat)
	if err != nil {
		return err
	}

	if format == output.FormatText {
		printSyncResultText(result, sync.Options{
			Short:   opts.Short,
			Quiet:   opts.Quiet,
			Verbose: opts.Verbose,
		})
	} else {
		if err := s.FormatOutput(format, result); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	// Handle exit codes
	if result.Failed > 0 {
		if opts.Strict {
			return fmt.Errorf("sync completed with %d failures (strict mode)", result.Failed)
		}
		return fmt.Errorf("sync completed with %d failures", result.Failed)
	}

	return nil
}
