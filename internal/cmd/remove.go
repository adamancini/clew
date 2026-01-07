package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Short:   "Remove entries from Clewfile",
		Long:    `Remove plugins or MCP servers from the Clewfile.`,
	}

	cmd.AddCommand(newRemovePluginCmd())
	cmd.AddCommand(newRemoveMCPCmd())

	return cmd
}

func newRemovePluginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plugin <plugin@marketplace>",
		Short: "Remove a plugin from Clewfile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement remove plugin logic
			fmt.Printf("remove plugin %s not yet implemented\n", args[0])
			return nil
		},
	}
}

func newRemoveMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp <name>",
		Short: "Remove an MCP server from Clewfile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement remove mcp logic
			fmt.Printf("remove mcp %s not yet implemented\n", args[0])
			return nil
		},
	}
}
