package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show sync status summary",
		Long:  `Status shows a quick summary of the sync state between Clewfile and system.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement status logic
			fmt.Println("status not yet implemented")
			return nil
		},
	}
}
