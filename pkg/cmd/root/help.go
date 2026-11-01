package root

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"vpn-tool/pkg/cmdutil"
	"vpn-tool/pkg/iostreams"
	"vpn-tool/pkg/text"
)

var hasFailed bool

// HasFailed signals that the main process should exit with a non-zero status code
func HasFailed() bool {
	return hasFailed
}

func rootUsageFunc(w io.Writer, command *cobra.Command) error {
	fmt.Fprintf(w, "Usage: %s", command.UseLine())
	var subcommands []*cobra.Command
	for _, c := range command.Commands() {
		if !c.IsAvailableCommand() {
			continue
		}
		subcommands = append(subcommands, c)
	}
	if len(subcommands) > 0 {
		fmt.Fprint(w, "\n\nAvailable commands:\n")
		for _, c := range subcommands {
			fmt.Fprintf(w, " %s\n", c.Name())
		}
		return nil
	}
	flagUsages := command.LocalFlags().FlagUsages()
	if flagUsages != "" {
		fmt.Fprintln(w, "\n\nFlags:")
		fmt.Fprint(w, text.Indent(text.Dedent(flagUsages), " "))
	}
	return nil
}

func rootFlagErrorFunc(cmd *cobra.Command, err error) error {
	if err == pflag.ErrHelp {
		return err
	}
	return cmdutil.FlagErrorWrap(err)
}

// nestedSuggestFunc displays helpful suggestions if a subcommand name was mistyped
func nestedSuggestFunc(w io.Writer, command *cobra.Command, arg string) {
	fmt.Fprintf(w, "unknown command %q for %q\n", arg, command.CommandPath())
	var candidates []string
	if arg == "help" {
		candidates = []string{"--help"}
	} else {
		if command.SuggestionsMinimumDistance <= 0 {
			command.SuggestionsMinimumDistance = 2
		}
		candidates = command.SuggestionsFor(arg)
	}
	if len(candidates) > 0 {
		fmt.Fprint(w, "\nDid you mean this?\n")
		for _, c := range candidates {
			fmt.Fprintf(w, "\t%s\n", c)
		}
	}
	fmt.Fprint(w, "\n")
	_ = rootUsageFunc(w, command)
}

func isRootCmd(command *cobra.Command) bool {
	return command != nil && !command.HasParent()
}

func rootHelpFunc(f *cmdutil.Factory, command *cobra.Command, _ []string) {
	flags := command.Flags()
	if isRootCmd(command) {
		if versionVal, err := flags.GetBool("version"); err == nil && versionVal {
			if versionInfo, ok := command.Annotations["versionInfo"]; ok {
				fmt.Fprint(f.IOStreams.Out, versionInfo)
			} else {
				fmt.Fprint(f.IOStreams.Out, command.Version)
			}
			return
		} else if err != nil {
			fmt.Fprintln(f.IOStreams.ErrOut, err)
			hasFailed = true
			return
		}
	}
	cs := f.IOStreams.ColorScheme()
	if help, _ := flags.GetBool("help"); !help && !command.Runnable() && len(flags.Args()) > 0 {
		nestedSuggestFunc(f.IOStreams.ErrOut, command, flags.Args()[0])
		hasFailed = true
		return
	}
	WriteHelp(f.IOStreams.Out, cs, command)
}

// WriteHelp renders the full help text for command to w in GitHub CLI style,
// placing key operations like auth, test, and api under CORE COMMANDS.
func WriteHelp(w io.Writer, cs *iostreams.ColorScheme, command *cobra.Command) {
	type helpEntry struct {
		Title string
		Body  string
	}
	longText := command.Long
	if longText == "" {
		longText = command.Short
	}
	if longText != "" && command.LocalFlags().Lookup("jq") != nil {
		longText = strings.TrimRight(longText, "\n") +
			"\n\nFor more information about output formatting flags, see `vpn-tool help formatting`."
	}
	helpEntries := []helpEntry{}
	if longText != "" {
		helpEntries = append(helpEntries, helpEntry{"", longText})
	}
	helpEntries = append(helpEntries, helpEntry{"USAGE", command.UseLine()})
	if len(command.Aliases) > 0 {
		helpEntries = append(helpEntries, helpEntry{"ALIASES", strings.Join(command.Aliases, ", ") + "\n"})
	}

	namePadding := 14

	// Define core commands for the root help display, including "test" and "api"
	coreCommandNames := map[string]bool{
		"auth":       true,
		"countries":  true,
		"create":     true,
		"connect":    true,
		"disconnect": true,
		"status":     true,
		"ip":         true,
		"test":       true,
		"api":        true,
	}

	if len(command.Commands()) > 0 {
		var coreNames []string
		var otherNames []string

		for _, c := range command.Commands() {
			if !c.IsAvailableCommand() {
				continue
			}
			line := rpad(c.Name()+":", namePadding) + c.Short
			if coreCommandNames[c.Name()] {
				coreNames = append(coreNames, line)
			} else {
				otherNames = append(otherNames, line)
			}
		}

		if len(coreNames) > 0 {
			sort.Strings(coreNames)
			helpEntries = append(helpEntries, helpEntry{
				Title: "CORE COMMANDS",
				Body:  strings.Join(coreNames, "\n"),
			})
		}

		if len(otherNames) > 0 {
			sort.Strings(otherNames)
			helpEntries = append(helpEntries, helpEntry{
				Title: "ADDITIONAL COMMANDS",
				Body:  strings.Join(otherNames, "\n"),
			})
		}
	}

	flagUsages := command.LocalFlags().FlagUsages()
	if flagUsages != "" {
		helpEntries = append(helpEntries, helpEntry{"FLAGS", text.Dedent(flagUsages)})
	}
	inheritedFlagUsages := command.InheritedFlags().FlagUsages()
	if inheritedFlagUsages != "" {
		helpEntries = append(helpEntries, helpEntry{"INHERITED FLAGS", text.Dedent(inheritedFlagUsages)})
	}
	if val, ok := command.Annotations["help:arguments"]; ok {
		helpEntries = append(helpEntries, helpEntry{"ARGUMENTS", val})
	}
	if command.Example != "" {
		helpEntries = append(helpEntries, helpEntry{"EXAMPLES", command.Example})
	}
	if val, ok := command.Annotations["help:environment"]; ok {
		helpEntries = append(helpEntries, helpEntry{"ENVIRONMENT", val})
	}

	helpEntries = append(helpEntries, helpEntry{"LEARN MORE", heredoc.Docf(`
		Use %[1]s%[1]s <command> <subcommand> --help%[1]s for more information about a command.
	`, "`")})

	for _, e := range helpEntries {
		if e.Title != "" {
			fmt.Fprintln(w, cs.Bold(e.Title))
			fmt.Fprintln(w, text.Indent(strings.Trim(e.Body, "\r\n"), " "))
		} else {
			fmt.Fprintln(w, e.Body)
		}
		fmt.Fprintln(w)
	}
}

func rpad(s string, padding int) string {
	tmpl := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(tmpl, s)
}
