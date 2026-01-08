package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for clew.

To load completions:

Bash:
  $ source <(clew completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ clew completion bash > /etc/bash_completion.d/clew
  # macOS:
  $ clew completion bash > $(brew --prefix)/etc/bash_completion.d/clew

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ clew completion zsh > "${fpath[1]}/_clew"

  # You will need to start a new shell for this setup to take effect.

  # Oh My Zsh:
  $ mkdir -p ~/.oh-my-zsh/completions
  $ clew completion zsh > ~/.oh-my-zsh/completions/_clew

Fish:
  $ clew completion fish > ~/.config/fish/completions/clew.fish
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			}
			return nil
		},
	}
}
