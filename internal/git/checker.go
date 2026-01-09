package git

import (
	"fmt"

	"github.com/adamancini/clew/internal/config"
)

// CheckResult holds git status results for all local items in a Clewfile.
type CheckResult struct {
	Marketplaces map[string]Status // Key is marketplace name
	Plugins      map[string]Status // Key is plugin name
	Warnings     []string          // Items that should be skipped (uncommitted changes)
	Info         []string          // Informational messages (ahead/behind)
	SkipMarkets  map[string]bool   // Marketplaces to skip due to git issues
	SkipPlugins  map[string]bool   // Plugins to skip due to git issues
}

// NewCheckResult creates an empty CheckResult.
func NewCheckResult() *CheckResult {
	return &CheckResult{
		Marketplaces: make(map[string]Status),
		Plugins:      make(map[string]Status),
		SkipMarkets:  make(map[string]bool),
		SkipPlugins:  make(map[string]bool),
	}
}

// ShouldSkipMarketplace returns true if the marketplace should be skipped due to git issues.
func (r *CheckResult) ShouldSkipMarketplace(name string) bool {
	return r.SkipMarkets[name]
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

	// Check local sources
	for _, source := range clewfile.Sources {
		if source.Source.Type == config.SourceTypeLocal && source.Source.Path != "" {
			status := c.CheckRepository(source.Source.Path)
			result.Marketplaces[source.Name] = status

			switch status.Level {
			case LevelWarning:
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("source %q at %s: %s (skipping)", source.Name, source.Source.Path, status.Message))
				result.SkipMarkets[source.Name] = true
			case LevelInfo:
				result.Info = append(result.Info,
					fmt.Sprintf("source %q at %s: %s", source.Name, source.Source.Path, status.Message))
			case LevelError:
				if status.Error != nil {
					result.Info = append(result.Info,
						fmt.Sprintf("source %q at %s: %s", source.Name, source.Source.Path, status.Message))
				}
			}
		}
	}

	// Check plugins with inline local sources
	for _, plugin := range clewfile.Plugins {
		if plugin.Source != nil && plugin.Source.Type == config.SourceTypeLocal && plugin.Source.Path != "" {
			status := c.CheckRepository(plugin.Source.Path)
			result.Plugins[plugin.Name] = status

			switch status.Level {
			case LevelWarning:
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("plugin %q at %s: %s (skipping)", plugin.Name, plugin.Source.Path, status.Message))
				result.SkipPlugins[plugin.Name] = true
			case LevelInfo:
				result.Info = append(result.Info,
					fmt.Sprintf("plugin %q at %s: %s", plugin.Name, plugin.Source.Path, status.Message))
			case LevelError:
				if status.Error != nil {
					result.Info = append(result.Info,
						fmt.Sprintf("plugin %q at %s: %s", plugin.Name, plugin.Source.Path, status.Message))
				}
			}
		}
	}

	return result
}
