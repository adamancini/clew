// Package diff computes differences between desired and current state.
package diff

import (
	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/state"
)

// Compute calculates the diff between a Clewfile and current state.
func compute(clewfile *config.Clewfile, current *state.State) *Result {
	result := &Result{
		Marketplaces: computeMarketplaceDiffs(clewfile.Marketplaces, current.Marketplaces),
		Plugins:      computePluginDiffs(clewfile.Plugins, current.Plugins),
	}
	return result
}

func computeMarketplaceDiffs(desired map[string]config.Marketplace, current map[string]state.MarketplaceState) []MarketplaceDiff {
	var diffs []MarketplaceDiff
	seen := make(map[string]bool)

	// Check each desired marketplace
	for alias, d := range desired {
		seen[alias] = true
		desiredCopy := d

		if c, exists := current[alias]; exists {
			currentCopy := c
			// Check if update needed (repo or ref changed)
			if marketplaceNeedsUpdate(d, c) {
				diffs = append(diffs, MarketplaceDiff{
					Alias:   alias,
					Action:  ActionUpdate,
					Current: &currentCopy,
					Desired: &desiredCopy,
				})
			} else {
				diffs = append(diffs, MarketplaceDiff{
					Alias:   alias,
					Action:  ActionNone,
					Current: &currentCopy,
					Desired: &desiredCopy,
				})
			}
		} else {
			// Needs to be added
			diffs = append(diffs, MarketplaceDiff{
				Alias:   alias,
				Action:  ActionAdd,
				Desired: &desiredCopy,
			})
		}
	}

	// Check for extra marketplaces not in Clewfile
	for alias, c := range current {
		if !seen[alias] {
			currentCopy := c
			diffs = append(diffs, MarketplaceDiff{
				Alias:   alias,
				Action:  ActionRemove,
				Current: &currentCopy,
			})
		}
	}

	return diffs
}

func marketplaceNeedsUpdate(desired config.Marketplace, current state.MarketplaceState) bool {
	// Check if repo changed
	if desired.Repo != current.Repo {
		return true
	}
	// Check if ref changed
	if desired.Ref != current.Ref {
		return true
	}
	return false
}

func computePluginDiffs(desired []config.Plugin, current map[string]state.PluginState) []PluginDiff {
	var diffs []PluginDiff
	seen := make(map[string]bool)

	// Check each desired plugin
	for _, d := range desired {
		desiredCopy := d
		fullName := d.Name // Already includes @marketplace if specified

		seen[fullName] = true

		if c, exists := current[fullName]; exists {
			currentCopy := c
			action := ActionNone

			// Check enabled state
			desiredEnabled := d.Enabled == nil || *d.Enabled
			if desiredEnabled && !c.Enabled {
				action = ActionEnable
			} else if !desiredEnabled && c.Enabled {
				action = ActionDisable
			}

			// Check scope mismatch (would need reinstall)
			if d.Scope != "" && d.Scope != c.Scope {
				action = ActionUpdate
			}

			diffs = append(diffs, PluginDiff{
				Name:    fullName,
				Action:  action,
				Current: &currentCopy,
				Desired: &desiredCopy,
			})
		} else {
			// Needs to be installed
			diffs = append(diffs, PluginDiff{
				Name:    fullName,
				Action:  ActionAdd,
				Desired: &desiredCopy,
			})
		}
	}

	// Check for extra plugins not in Clewfile
	for name, c := range current {
		if !seen[name] {
			currentCopy := c
			diffs = append(diffs, PluginDiff{
				Name:    name,
				Action:  ActionRemove,
				Current: &currentCopy,
			})
		}
	}

	return diffs
}
