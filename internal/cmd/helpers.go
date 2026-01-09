package cmd

import (
	"github.com/adamancini/clew/internal/state"
)

// getStateReader returns the appropriate state reader based on global flags.
func getStateReader() state.Reader {
	if useCLI {
		// CLI reader is experimental and currently broken (issue #34)
		return &state.CLIReader{}
	}
	// Filesystem reader is the default
	return &state.FilesystemReader{}
}
