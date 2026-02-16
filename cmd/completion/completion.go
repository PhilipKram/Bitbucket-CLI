package completion

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCmdCompletion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for the Bitbucket CLI.

The completion script should be sourced to enable autocompletion.
See each sub-command's help for details on how to use the generated script.`,
	}

	cmd.AddCommand(newCmdBash())
	cmd.AddCommand(newCmdZsh())
	cmd.AddCommand(newCmdFish())
	cmd.AddCommand(newCmdPowershell())

	return cmd
}

func newCmdBash() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long: `Generate bash completion script for the Bitbucket CLI.

To load completions in your current shell session:
  source <(bb completion bash)

To load completions for every new session, execute once:
  # Linux:
  bb completion bash > /etc/bash_completion.d/bb

  # macOS:
  bb completion bash > /usr/local/etc/bash_completion.d/bb`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
	}
	return cmd
}

func newCmdZsh() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: `Generate zsh completion script for the Bitbucket CLI.

To load completions in your current shell session:
  source <(bb completion zsh)

To load completions for every new session, execute once:
  bb completion zsh > "${fpath[1]}/_bb"

You will need to start a new shell for this setup to take effect.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	}
	return cmd
}

func newCmdFish() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long: `Generate fish completion script for the Bitbucket CLI.

To load completions in your current shell session:
  bb completion fish | source

To load completions for every new session, execute once:
  bb completion fish > ~/.config/fish/completions/bb.fish`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		},
	}
	return cmd
}

func newCmdPowershell() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generate powershell completion script",
		Long: `Generate powershell completion script for the Bitbucket CLI.

To load completions in your current shell session:
  bb completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output to your powershell profile:
  bb completion powershell > bb.ps1
  # and source this file from your powershell profile`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		},
	}
	return cmd
}
