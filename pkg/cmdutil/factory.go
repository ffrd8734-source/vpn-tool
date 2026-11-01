package cmdutil

import (
	"vpn-tool/pkg/iostreams"
)

// Factory provides shared dependencies for command execution
type Factory struct {
	IOStreams *iostreams.IOStreams
}

// NewFactory initializes a new command factory using default IO streams
func NewFactory() *Factory {
	return &Factory{
		IOStreams: iostreams.System(),
	}
}
