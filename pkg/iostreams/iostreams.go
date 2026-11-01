package iostreams

import (
	"io"
	"os"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// IOStreams represents standard input/output streams and terminal capabilities
type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer

	isInteractive bool
	colorEnabled  bool
	colorScheme   *ColorScheme
}

// System initializes IOStreams connected to standard OS streams
func System() *IOStreams {
	in := os.Stdin
	out := os.Stdout
	errOut := os.Stderr

	isTerm := term.IsTerminal(int(os.Stdout.Fd()))

	ios := &IOStreams{
		In:            in,
		Out:           out,
		ErrOut:        errOut,
		isInteractive: isTerm,
		colorEnabled:  isTerm,
	}

	if !isTerm {
		color.NoColor = true
	}

	ios.colorScheme = &ColorScheme{
		enabled: ios.colorEnabled,
	}

	return ios
}

// SetStdin overrides standard input
func (s *IOStreams) SetStdin(r io.ReadCloser) {
	s.In = r
}

// SetStdout overrides standard output
func (s *IOStreams) SetStdout(w io.Writer) {
	s.Out = w
}

// SetStderr overrides standard error
func (s *IOStreams) SetStderr(w io.Writer) {
	s.ErrOut = w
}

// IsStdoutTTY returns true if stdout is connected to a terminal
func (s *IOStreams) IsStdoutTTY() bool {
	return s.isInteractive
}

// IsStderrTTY returns true if stderr is connected to a terminal
func (s *IOStreams) IsStderrTTY() bool {
	return s.isInteractive
}

// ColorScheme returns the color scheme helper
func (s *IOStreams) ColorScheme() *ColorScheme {
	return s.colorScheme
}
