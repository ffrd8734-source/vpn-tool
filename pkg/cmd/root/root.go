package root

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"vpn-tool/pkg/cmd"
	"vpn-tool/pkg/cmdutil"
)

// NewCmdRoot creates and returns the root vpn-tool command with all subcommands wired
func NewCmdRoot(f *cmdutil.Factory, version, buildDate string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "vpn-tool <command> <subcommand> [flags]",
		Short: "VPN Gateway Manager CLI",
		Long: heredoc.Doc(`
			Work with VPN gateways, manage connections, test latencies, and interact with the API securely.
		`),
		Example: heredoc.Doc(`
			$ vpn-tool auth login
			$ vpn-tool countries
			$ vpn-tool create --country US
			$ vpn-tool connect --country US
			$ vpn-tool status --country US
		`),
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
	}

	// Set version info annotation for help/version rendering
	rootCmd.Annotations = map[string]string{
		"versionInfo": fmt.Sprintf("vpn-tool version %s (%s)\n", version, buildDate),
	}

	// Configure custom help and usage handlers mirroring GitHub CLI (gh) style
	rootCmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		rootHelpFunc(f, c, args)
	})
	rootCmd.SetUsageFunc(func(c *cobra.Command) error {
		_ = rootUsageFunc(f.IOStreams.Out, c)
		return nil
	})
	rootCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return rootFlagErrorFunc(c, err)
	})

	// Global / Persistent flags
	rootCmd.PersistentFlags().Bool("help", false, "Show help for command")
	rootCmd.Flags().BoolP("version", "v", false, "Show vpn-tool version")

	// Add all subcommands
	rootCmd.AddCommand(cmd.NewCmdAuth())
	rootCmd.AddCommand(cmd.NewCmdCountries())
	rootCmd.AddCommand(cmd.NewCmdVPNCreate())
	rootCmd.AddCommand(cmd.NewCmdVPNConnect())
	rootCmd.AddCommand(cmd.NewCmdVPNDisconnect())
	rootCmd.AddCommand(cmd.NewCmdStatus())
	rootCmd.AddCommand(cmd.NewCmdTest())
	rootCmd.AddCommand(cmd.NewCmdIP())
	rootCmd.AddCommand(cmd.NewCmdAPI())
	rootCmd.AddCommand(cmd.NewCmdCompletion())
	rootCmd.AddCommand(cmd.NewCmdHealthz())

	return rootCmd
}

// Execute parses command line arguments and runs the root command
func Execute(version, buildDate string) {
	f := cmdutil.NewFactory()
	rootCmd := NewCmdRoot(f, version, buildDate)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(f.IOStreams.ErrOut, f.IOStreams.ColorScheme().Error(fmt.Sprintf("error running vpn-tool: %v", err)))
		os.Exit(1)
	}

	if HasFailed() {
		os.Exit(1)
	}
}
