package update

import (
	"fmt"
	"runtime"
)

// Detect returns the current platform (OS and architecture)
func Detect() Platform {
	return Platform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// BinaryName returns the binary name for this platform
// e.g., "clew-darwin-arm64"
func (p Platform) BinaryName() string {
	return fmt.Sprintf("clew-%s-%s", p.OS, p.Arch)
}

// IsSupported returns true if this platform is supported
func (p Platform) IsSupported() bool {
	supportedPlatforms := map[string][]string{
		"darwin": {"amd64", "arm64"},
		"linux":  {"amd64", "arm64"},
	}

	archs, ok := supportedPlatforms[p.OS]
	if !ok {
		return false
	}

	for _, arch := range archs {
		if p.Arch == arch {
			return true
		}
	}

	return false
}
