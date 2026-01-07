package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export current state as Clewfile",
		Long:  `Export reads the current Claude Code configuration and outputs it as a Clewfile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement export logic
			fmt.Println("export not yet implemented")
			return nil
		},
	}
}
