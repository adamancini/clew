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

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show sync status summary",
		Long:  `Status shows a quick summary of the sync state between Clewfile and system.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

// StatusSummary represents a summary of the sync status.
type StatusSummary struct {
	InSync    bool `json:"in_sync" yaml:"in_sync"`
	Add       int  `json:"add" yaml:"add"`
	Update    int  `json:"update" yaml:"update"`
	Remove    int  `json:"remove" yaml:"remove"`
	Attention int  `json:"attention" yaml:"attention"`
}

// String implements fmt.Stringer for text output.
func (s StatusSummary) String() string {
	if s.InSync {
		return "In sync"
	}
	return fmt.Sprintf("Add: %d, Update: %d, Remove: %d, Attention: %d",
		s.Add, s.Update, s.Remove, s.Attention)
}

// runStatus executes the status workflow.
func runStatus() error {
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

	// 6. Get summary counts
	add, update, remove, attention := diffResult.Summary()

	summary := StatusSummary{
		InSync:    add == 0 && update == 0 && remove == 0 && attention == 0,
		Add:       add,
		Update:    update,
		Remove:    remove,
		Attention: attention,
	}

	// 7. Format and display output
	format, err := output.ParseFormat(outputFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if format == output.FormatText {
		printStatusText(summary)
	} else {
		writer := output.NewWriter(os.Stdout, format)
		if err := writer.Write(summary); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	}

	return nil
}

// printStatusText outputs the status summary in human-readable format.
func printStatusText(summary StatusSummary) {
	if summary.InSync {
		fmt.Println("Status: In sync")
		return
	}

	fmt.Println("Status: Out of sync")
	fmt.Println()

	if summary.Add > 0 {
		fmt.Printf("  To add:       %d\n", summary.Add)
	}
	if summary.Update > 0 {
		fmt.Printf("  To update:    %d\n", summary.Update)
	}
	if summary.Remove > 0 {
		fmt.Printf("  To remove:    %d\n", summary.Remove)
	}
	if summary.Attention > 0 {
		fmt.Printf("  Need attention: %d\n", summary.Attention)
	}

	fmt.Println()
	fmt.Println("Run 'clew diff' for details or 'clew sync' to apply changes.")
}
