package cmd

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

// NewCmdCompletion creates the completion command for generating shell autocompletions
func NewCmdCompletion() *cobra.Command {
	var shell string

	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: heredoc.Doc(`
			Generate shell completion scripts for vpn-tool.

			To load completions:

			Bash:
			  $ source <(vpn-tool completion --shell bash)

			  # To load completions for each session, execute once:
			  # Linux:
			  $ vpn-tool completion --shell bash > /etc/bash_completion.d/vpn-tool
			  # macOS:
			  $ vpn-tool completion --shell bash > $(brew --prefix)/etc/bash_completion.d/vpn-tool

			Zsh:
			  # If shell completion is not already enabled in your environment,
			  # you will need to enable it. Execute the following:
			  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

			  # To load completions for each session, execute once:
			  $ vpn-tool completion --shell zsh > "${fpath[1]}/_vpn-tool"

			  # You will need to start a new shell for this setup to take effect.

			Fish:
			  $ vpn-tool completion --shell fish | source

			  # To load completions for each session, execute once:
			  $ vpn-tool completion --shell fish > ~/.config/fish/completions/vpn-tool.fish
		`),
		Example: heredoc.Doc(`
			$ vpn-tool completion -s zsh
			$ vpn-tool completion --shell bash
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch shell {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout)
			case "powershell":
				return cmd.Root().GenPowerShellCompletion(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell %q. Choose from: bash, zsh, fish, powershell", shell)
			}
		},
	}

	// Add --shell / -s flag
	cmd.Flags().StringVarP(&shell, "shell", "s", "bash", "Shell type (bash, zsh, fish, powershell)")
	
	// Mark shell flag as required if desired, or let it default to bash
	// _ = cmd.MarkFlagRequired("shell")

	return cmd
}
