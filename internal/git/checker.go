package git

import (
	"fmt"

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

// CheckClewfile checks git status for all local path-based marketplaces in the Clewfile.
// Remote repo marketplaces are skipped since they are managed by the claude CLI.
func (c *Checker) CheckClewfile(clewfile *config.Clewfile) *CheckResult {
	result := NewCheckResult()

	// Check if git is available
	if !c.GitAvailable() {
		result.Info = append(result.Info, "git not available - skipping git status checks")
		return result
	}

	for alias, m := range clewfile.Marketplaces {
		if !m.IsLocal() {
			continue
		}
		status := c.CheckRepository(m.Path)
		result.Marketplaces[alias] = status
		switch status.Level {
		case LevelWarning:
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("marketplace %q has uncommitted changes (%s) - skipping git pull", alias, m.Path))
			result.SkipMarketplaces[alias] = true
		case LevelError:
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("marketplace %q git check failed: %v - skipping git pull", alias, status.Error))
			result.SkipMarketplaces[alias] = true
		case LevelInfo:
			result.Info = append(result.Info, fmt.Sprintf("marketplace %q: %s", alias, status.Message))
		}
	}

	return result
}
