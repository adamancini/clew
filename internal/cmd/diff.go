package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Show what would change (dry-run)",
		Long:  `Diff compares the Clewfile against current state and shows what sync would do.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement diff logic
			fmt.Println("diff not yet implemented")
			return nil
		},
	}
}
