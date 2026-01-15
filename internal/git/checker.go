package git

import (
	"github.com/adamancini/clew/internal/config"
)

// CheckResult holds git status results for all local items in a Clewfile.
type CheckResult struct {
	Marketplaces    map[string]Status // Key is marketplace alias
	Plugins         map[string]Status // Key is plugin name
	Warnings        []string          // Items that should be skipped (uncommitted changes)
	Info            []string          // Informational messages (ahead/behind)
	SkipMarketplaces map[string]bool  // Marketplaces to skip due to git issues
	SkipPlugins     map[string]bool   // Plugins to skip due to git issues
}

// NewCheckResult creates an empty CheckResult.
func NewCheckResult() *CheckResult {
	return &CheckResult{
		Marketplaces:     make(map[string]Status),
		Plugins:          make(map[string]Status),
		SkipMarketplaces: make(map[string]bool),
		SkipPlugins:      make(map[string]bool),
	}
}

// ShouldSkipMarketplace returns true if the marketplace should be skipped due to git issues.
func (r *CheckResult) ShouldSkipMarketplace(alias string) bool {
	return r.SkipMarketplaces[alias]
}

// ShouldSkipPlugin returns true if the plugin should be skipped due to git issues.
func (r *CheckResult) ShouldSkipPlugin(name string) bool {
	return r.SkipPlugins[name]
}

// HasWarnings returns true if there are any warnings (items to skip).
func (r *CheckResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// HasInfo returns true if there are any informational messages.
func (r *CheckResult) HasInfo() bool {
	return len(r.Info) > 0
}

// CheckClewfile checks git status for all local marketplaces and plugins in the Clewfile.
func (c *Checker) CheckClewfile(clewfile *config.Clewfile) *CheckResult {
	result := NewCheckResult()

	// Check if git is available
	if !c.GitAvailable() {
		result.Info = append(result.Info, "git not available - skipping git status checks")
		return result
	}

	// Local sources are no longer supported (removed in v0.7.0)
	// Git status checking only applies to github sources, which are
	// not stored locally and thus don't need git status checks.

	return result
}
