package diff

import (
	"testing"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/state"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestComputeMarketplaces(t *testing.T) {
	clewfile := &config.Clewfile{
		Marketplaces: map[string]config.Marketplace{
			"existing": {Repo: "owner/existing"},
			"new":      {Repo: "owner/new"},
			"updated":  {Repo: "owner/updated-new"},
		},
	}

	current := &state.State{
		Marketplaces: map[string]state.MarketplaceState{
			"existing": {Alias: "existing", Repo: "owner/existing"},
			"updated":  {Alias: "updated", Repo: "owner/updated-old"},
			"extra":    {Alias: "extra", Repo: "owner/extra"},
		},
		Plugins: make(map[string]state.PluginState),
	}

	result := Compute(clewfile, current)

	// Should have 4 marketplace diffs
	if len(result.Marketplaces) != 4 {
		t.Errorf("Marketplaces count = %d, want 4", len(result.Marketplaces))
	}

	actionCounts := make(map[Action]int)
	for _, m := range result.Marketplaces {
		actionCounts[m.Action]++
	}

	if actionCounts[ActionNone] != 1 {
		t.Errorf("ActionNone count = %d, want 1", actionCounts[ActionNone])
	}
	if actionCounts[ActionAdd] != 1 {
		t.Errorf("ActionAdd count = %d, want 1", actionCounts[ActionAdd])
	}
	if actionCounts[ActionUpdate] != 1 {
		t.Errorf("ActionUpdate count = %d, want 1", actionCounts[ActionUpdate])
	}
	if actionCounts[ActionRemove] != 1 {
		t.Errorf("ActionRemove count = %d, want 1", actionCounts[ActionRemove])
	}
}

func TestComputePlugins(t *testing.T) {
	clewfile := &config.Clewfile{
		Plugins: []config.Plugin{
			{Name: "installed@marketplace", Enabled: boolPtr(true)},
			{Name: "new@marketplace"},
			{Name: "to-enable@marketplace", Enabled: boolPtr(true)},
			{Name: "to-disable@marketplace", Enabled: boolPtr(false)},
		},
		Marketplaces: make(map[string]config.Marketplace),
	}

	current := &state.State{
		Plugins: map[string]state.PluginState{
			"installed@marketplace":  {Name: "installed", Marketplace: "marketplace", Enabled: true},
			"to-enable@marketplace":  {Name: "to-enable", Marketplace: "marketplace", Enabled: false},
			"to-disable@marketplace": {Name: "to-disable", Marketplace: "marketplace", Enabled: true},
			"extra@marketplace":      {Name: "extra", Marketplace: "marketplace", Enabled: true},
		},
		Marketplaces: make(map[string]state.MarketplaceState),
	}

	result := Compute(clewfile, current)

	actionCounts := make(map[Action]int)
	for _, p := range result.Plugins {
		actionCounts[p.Action]++
	}

	if actionCounts[ActionNone] != 1 {
		t.Errorf("ActionNone count = %d, want 1 (installed)", actionCounts[ActionNone])
	}
	if actionCounts[ActionAdd] != 1 {
		t.Errorf("ActionAdd count = %d, want 1 (new)", actionCounts[ActionAdd])
	}
	if actionCounts[ActionEnable] != 1 {
		t.Errorf("ActionEnable count = %d, want 1", actionCounts[ActionEnable])
	}
	if actionCounts[ActionDisable] != 1 {
		t.Errorf("ActionDisable count = %d, want 1", actionCounts[ActionDisable])
	}
	if actionCounts[ActionRemove] != 1 {
		t.Errorf("ActionRemove count = %d, want 1 (extra)", actionCounts[ActionRemove])
	}
}

func TestSummary(t *testing.T) {
	result := &Result{
		Marketplaces: []MarketplaceDiff{
			{Alias: "m1", Action: ActionAdd},
			{Alias: "m2", Action: ActionRemove},
		},
		Plugins: []PluginDiff{
			{Name: "p1", Action: ActionAdd},
			{Name: "p2", Action: ActionEnable},
			{Name: "p3", Action: ActionDisable},
			{Name: "p4", Action: ActionRemove},
		},
	}

	add, update, remove, attention := result.Summary()

	if add != 2 { // m1 + p1
		t.Errorf("add = %d, want 2", add)
	}
	if update != 2 { // p2 + p3
		t.Errorf("update = %d, want 2", update)
	}
	if remove != 0 { // Non-destructive, removals count as attention
		t.Errorf("remove = %d, want 0", remove)
	}
	if attention != 2 { // m2 + p4
		t.Errorf("attention = %d, want 2", attention)
	}
}
