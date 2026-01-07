package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add entries to Clewfile",
		Long:  `Add marketplaces, plugins, or MCP servers to the Clewfile.`,
	}

	cmd.AddCommand(newAddMarketplaceCmd())
	cmd.AddCommand(newAddPluginCmd())
	cmd.AddCommand(newAddMCPCmd())

	return cmd
}

func newAddMarketplaceCmd() *cobra.Command {
	var source, repo, path string

	cmd := &cobra.Command{
		Use:   "marketplace <name>",
		Short: "Add a marketplace to Clewfile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement add marketplace logic
			fmt.Printf("add marketplace %s not yet implemented\n", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "github", "Source type: github, local")
	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repo (owner/repo)")
	cmd.Flags().StringVar(&path, "path", "", "Local path (for source=local)")

	return cmd
}

func newAddPluginCmd() *cobra.Command {
	var enabled bool
	var scope string

	cmd := &cobra.Command{
		Use:   "plugin <plugin@marketplace>",
		Short: "Add a plugin to Clewfile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement add plugin logic
			fmt.Printf("add plugin %s not yet implemented\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable the plugin")
	cmd.Flags().StringVar(&scope, "scope", "", "Scope: user, project, local")

	return cmd
}

func newAddMCPCmd() *cobra.Command {
	var transport, command, url string
	var args []string

	cmd := &cobra.Command{
		Use:   "mcp <name>",
		Short: "Add an MCP server to Clewfile",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, cmdArgs []string) error {
			// TODO: Implement add mcp logic
			fmt.Printf("add mcp %s not yet implemented\n", cmdArgs[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport: stdio, http")
	cmd.Flags().StringVar(&command, "command", "", "Command to run (for stdio)")
	cmd.Flags().StringSliceVar(&args, "args", nil, "Arguments for command")
	cmd.Flags().StringVar(&url, "url", "", "URL (for http)")

	return cmd
}
