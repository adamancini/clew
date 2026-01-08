package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags
	outputFormat string
	configPath   string
	useFilesystem bool
	verbose      bool
	quiet        bool
)

func Execute(version, commit, date string) error {
	rootCmd := &cobra.Command{
		Use:   "clew",
		Short: "Declarative Claude Code configuration management",
		Long: `clew manages Claude Code plugins, marketplaces, and MCP servers declaratively.

Define your desired configuration in a Clewfile, sync it across machines with clew sync.`,
		Version: version,
		SilenceUsage: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text, json, yaml")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to Clewfile")
	rootCmd.PersistentFlags().BoolVarP(&useFilesystem, "filesystem", "f", false, "Read state from filesystem instead of claude CLI")
	rootCmd.PersistentFlags().BoolVar(&useFilesystem, "read-from-filesystem", false, "Read state from filesystem instead of claude CLI")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode (errors only)")

	// Add subcommands
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newDiffCmd())
	rootCmd.AddCommand(newExportCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newRemoveCmd())
	rootCmd.AddCommand(newCompletionCmd())

	// Register completion function for output flag
	_ = rootCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
	})

	return rootCmd.Execute()
}
