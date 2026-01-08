package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/adamancini/clew/internal/backup"
	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/output"
	"github.com/adamancini/clew/internal/state"
	"github.com/adamancini/clew/internal/sync"
)

// clewVersion is set during command initialization
var clewVersion = "dev"

// SetVersion sets the clew version for backup metadata.
func SetVersion(version string) {
	clewVersion = version
}

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup and restore Claude Code configuration",
		Long: `Backup manages snapshots of your Claude Code configuration.

Backups are stored in ~/.cache/clew/backups/ and include:
  - Installed marketplaces
  - Installed plugins and their enabled state
  - MCP server configurations

Use 'clew backup create' before making changes, and 'clew backup restore'
to recover a previous configuration.`,
	}

	cmd.AddCommand(newBackupCreateCmd())
	cmd.AddCommand(newBackupListCmd())
	cmd.AddCommand(newBackupRestoreCmd())
	cmd.AddCommand(newBackupDeleteCmd())
	cmd.AddCommand(newBackupPruneCmd())

	return cmd
}

func newBackupCreateCmd() *cobra.Command {
	var note string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new backup",
		Long:  `Create captures the current Claude Code configuration state and stores it as a backup.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupCreate(note)
		},
	}

	cmd.Flags().StringVar(&note, "note", "", "Add a note to describe this backup")

	return cmd
}

func newBackupListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all backups",
		Long:  `List displays all available backups with their creation time, notes, and size.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupList()
		},
	}
}

func newBackupRestoreCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "restore <id>",
		Short: "Restore from a backup",
		Long: `Restore applies a previous backup's configuration to the system.

Use 'latest' as the ID to restore the most recent backup.

This command shows the changes that will be made and prompts for confirmation
before applying them.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupRestore(args[0], yes)
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func newBackupDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a backup",
		Long:  `Delete removes a backup by its ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupDelete(args[0])
		},
	}
}

func newBackupPruneCmd() *cobra.Command {
	var keep int

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove old backups",
		Long: `Prune deletes old backups, keeping only the most recent N backups.

By default, keeps the 30 most recent backups.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupPrune(keep)
		},
	}

	cmd.Flags().IntVar(&keep, "keep", backup.DefaultKeepCount, "Number of backups to keep")

	return cmd
}

// runBackupCreate creates a new backup.
func runBackupCreate(note string) error {
	// Read current state
	var reader state.Reader
	if useFilesystem {
		reader = &state.FilesystemReader{}
	} else {
		reader = &state.CLIReader{}
	}

	currentState, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read current state: %w", err)
	}

	// Create backup
	manager, err := backup.NewManager(clewVersion)
	if err != nil {
		return err
	}

	bak, err := manager.Create(currentState, note)
	if err != nil {
		return err
	}

	// Output result
	format, err := output.ParseFormat(outputFormat)
	if err != nil {
		return err
	}

	if format == output.FormatText {
		fmt.Printf("Backup created: %s\n", bak.ID)
		if note != "" {
			fmt.Printf("Note: %s\n", note)
		}
		fmt.Printf("Location: %s/%s.json\n", manager.BackupDir(), bak.ID)
	} else {
		writer := output.NewWriter(os.Stdout, format)
		return writer.Write(bak)
	}

	return nil
}

// runBackupList lists all backups.
func runBackupList() error {
	manager, err := backup.NewManager(clewVersion)
	if err != nil {
		return err
	}

	backups, err := manager.List()
	if err != nil {
		return err
	}

	format, err := output.ParseFormat(outputFormat)
	if err != nil {
		return err
	}

	if format == output.FormatText {
		if len(backups) == 0 {
			fmt.Println("No backups found.")
			fmt.Printf("Backup directory: %s\n", manager.BackupDir())
			return nil
		}

		fmt.Printf("Backups stored in %s:\n\n", manager.BackupDir())

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tCreated\tNote\tSize")
		for _, b := range backups {
			sizeStr := formatSize(b.Size)
			note := b.Note
			if note == "" {
				note = "-"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				b.ID,
				b.CreatedAt.Format("2006-01-02 15:04:05"),
				note,
				sizeStr,
			)
		}
		_ = w.Flush()
	} else {
		writer := output.NewWriter(os.Stdout, format)
		return writer.Write(backups)
	}

	return nil
}

// runBackupRestore restores from a backup.
func runBackupRestore(id string, skipConfirm bool) error {
	manager, err := backup.NewManager(clewVersion)
	if err != nil {
		return err
	}

	// Get the backup
	bak, err := manager.Get(id)
	if err != nil {
		return err
	}

	fmt.Printf("Restoring from backup: %s\n", bak.ID)
	fmt.Printf("Created: %s\n", bak.CreatedAt.Format("2006-01-02 15:04:05"))
	if bak.Note != "" {
		fmt.Printf("Note: %s\n", bak.Note)
	}
	fmt.Println()

	// Read current state
	var reader state.Reader
	if useFilesystem {
		reader = &state.FilesystemReader{}
	} else {
		reader = &state.CLIReader{}
	}

	currentState, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read current state: %w", err)
	}

	// Create a Clewfile from the backup state for diff computation
	backupClewfile := backupToConfig(bak)

	// Compute diff between backup (desired) and current state
	diffResult := diff.Compute(backupClewfile, currentState)

	// Check if there's anything to restore
	add, update, remove, attention := diffResult.Summary()
	if add == 0 && update == 0 && remove == 0 && attention == 0 {
		fmt.Println("Current state already matches backup. Nothing to restore.")
		return nil
	}

	// Show what will change
	fmt.Println("Changes to apply:")
	printRestoreDiff(diffResult)
	fmt.Println()

	// Confirm
	if !skipConfirm {
		fmt.Print("Proceed? [y/n] ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Restore cancelled.")
			return nil
		}
	}

	// Execute sync to restore
	syncer := sync.NewSyncer()
	result, err := syncer.Execute(diffResult, sync.Options{
		Verbose: verbose,
		Quiet:   quiet,
	})
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	// Print result
	if result.Failed > 0 {
		fmt.Println()
		fmt.Printf("Restore completed with %d failures.\n", result.Failed)
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
		return fmt.Errorf("restore completed with errors")
	}

	fmt.Println()
	fmt.Println("Restored successfully")

	if len(result.Attention) > 0 {
		fmt.Println("\nItems needing manual attention:")
		for _, item := range result.Attention {
			fmt.Printf("  - %s\n", item)
		}
	}

	return nil
}

// runBackupDelete deletes a backup.
func runBackupDelete(id string) error {
	manager, err := backup.NewManager(clewVersion)
	if err != nil {
		return err
	}

	if err := manager.Delete(id); err != nil {
		return err
	}

	fmt.Printf("Backup deleted: %s\n", id)
	return nil
}

// runBackupPrune removes old backups.
func runBackupPrune(keep int) error {
	manager, err := backup.NewManager(clewVersion)
	if err != nil {
		return err
	}

	result, err := manager.Prune(keep)
	if err != nil {
		return err
	}

	format, err := output.ParseFormat(outputFormat)
	if err != nil {
		return err
	}

	if format == output.FormatText {
		if len(result.Deleted) == 0 {
			fmt.Printf("No backups to prune. Keeping %d backups.\n", result.Kept)
			return nil
		}

		fmt.Printf("Pruned %d backup(s), keeping %d:\n", len(result.Deleted), result.Kept)
		for _, b := range result.Deleted {
			fmt.Printf("  - %s (%s)\n", b.ID, b.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	} else {
		writer := output.NewWriter(os.Stdout, format)
		return writer.Write(result)
	}

	return nil
}

// backupToConfig converts a backup to a Clewfile for diff computation.
func backupToConfig(bak *backup.Backup) *config.Clewfile {
	clewfile := &config.Clewfile{
		Version:      1,
		Marketplaces: make(map[string]config.Marketplace),
		Plugins:      []config.Plugin{},
		MCPServers:   make(map[string]config.MCPServer),
	}

	// Convert marketplaces
	for name, m := range bak.State.Marketplaces {
		clewfile.Marketplaces[name] = config.Marketplace{
			Source: m.Source,
			Repo:   m.Repo,
			Path:   m.Path,
		}
	}

	// Convert plugins
	for _, p := range bak.State.Plugins {
		plugin := config.Plugin{
			Name:  p.Name,
			Scope: p.Scope,
		}
		if p.Marketplace != "" {
			plugin.Name = fmt.Sprintf("%s@%s", p.Name, p.Marketplace)
		}
		enabled := p.Enabled
		plugin.Enabled = &enabled
		clewfile.Plugins = append(clewfile.Plugins, plugin)
	}

	// Convert MCP servers
	for name, m := range bak.State.MCPServers {
		clewfile.MCPServers[name] = config.MCPServer{
			Transport: m.Transport,
			Command:   m.Command,
			Args:      m.Args,
			URL:       m.URL,
			Scope:     m.Scope,
		}
	}

	return clewfile
}

// printRestoreDiff prints a summary of restore changes.
func printRestoreDiff(result *diff.Result) {
	for _, m := range result.Marketplaces {
		if m.Action == diff.ActionNone {
			continue
		}
		printDiffLine(m.Action, "marketplace", m.Name)
	}

	for _, p := range result.Plugins {
		if p.Action == diff.ActionNone {
			continue
		}
		printDiffLine(p.Action, "plugin", p.Name)
	}

	for _, m := range result.MCPServers {
		if m.Action == diff.ActionNone {
			continue
		}
		printDiffLine(m.Action, "mcp", m.Name)
	}
}

// printDiffLine prints a single diff line with appropriate symbol.
func printDiffLine(action diff.Action, itemType, name string) {
	var symbol string
	switch action {
	case diff.ActionAdd:
		symbol = "+"
	case diff.ActionRemove:
		symbol = "-"
	case diff.ActionUpdate, diff.ActionEnable:
		symbol = "~"
	case diff.ActionDisable:
		symbol = "-"
	default:
		symbol = " "
	}
	fmt.Printf("  %s %s %s\n", symbol, itemType, name)
}

// formatSize formats a byte size as a human-readable string.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
