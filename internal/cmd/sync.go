package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var strict bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile system to match Clewfile",
		Long:  `Sync reads the Clewfile and ensures the system matches the declared state.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement sync logic
			fmt.Println("sync not yet implemented")
			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Exit non-zero on any failure")

	return cmd
}
