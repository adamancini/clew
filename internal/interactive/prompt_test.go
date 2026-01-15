package interactive

import (
	"bytes"
	"strings"
	"testing"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/state"
)

func TestPrompterYesResponse(t *testing.T) {
	input := strings.NewReader("y\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	resp := p.prompt("Test prompt?")

	if resp != ResponseYes {
		t.Errorf("expected ResponseYes, got %v", resp)
	}
}

func TestPrompterNoResponse(t *testing.T) {
	input := strings.NewReader("n\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	resp := p.prompt("Test prompt?")

	if resp != ResponseNo {
		t.Errorf("expected ResponseNo, got %v", resp)
	}
}

func TestPrompterAllResponse(t *testing.T) {
	input := strings.NewReader("a\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	resp := p.prompt("First prompt?")
	if resp != ResponseYes {
		t.Errorf("expected ResponseYes after 'a', got %v", resp)
	}

	// Subsequent prompts should auto-approve
	resp = p.prompt("Second prompt?")
	if resp != ResponseYes {
		t.Errorf("expected ResponseYes (auto-approve), got %v", resp)
	}
}

func TestPrompterQuitResponse(t *testing.T) {
	input := strings.NewReader("q\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	resp := p.prompt("Test prompt?")

	if resp != ResponseQuit {
		t.Errorf("expected ResponseQuit, got %v", resp)
	}
}

func TestPrompterInvalidResponse(t *testing.T) {
	input := strings.NewReader("invalid\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	resp := p.prompt("Test prompt?")

	if resp != ResponseNo {
		t.Errorf("expected ResponseNo for invalid input, got %v", resp)
	}
	if !strings.Contains(output.String(), "Invalid response") {
		t.Errorf("expected 'Invalid response' message in output")
	}
}

func TestPrompterEOFResponse(t *testing.T) {
	input := strings.NewReader("")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	resp := p.prompt("Test prompt?")

	if resp != ResponseQuit {
		t.Errorf("expected ResponseQuit on EOF, got %v", resp)
	}
}

func TestConfirmFinalYes(t *testing.T) {
	input := strings.NewReader("y\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	result := p.confirmFinal()

	if !result {
		t.Error("expected true for 'y' confirmation")
	}
}

func TestConfirmFinalNo(t *testing.T) {
	input := strings.NewReader("n\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	result := p.confirmFinal()

	if result {
		t.Error("expected false for 'n' confirmation")
	}
}

func TestNewSelection(t *testing.T) {
	sel := NewSelection()

	if sel.Marketplaces == nil {
		t.Error("expected non-nil Marketplaces map")
	}
	if sel.Plugins == nil {
		t.Error("expected non-nil Plugins map")
	}
	if sel.MCPServers == nil {
		t.Error("expected non-nil MCPServers map")
	}
}

func TestActionSymbolVerb(t *testing.T) {
	tests := []struct {
		action       diff.Action
		wantSymbol   string
		wantVerb     string
	}{
		{diff.ActionAdd, "+", "add"},
		{diff.ActionRemove, "-", "remove"},
		{diff.ActionUpdate, "~", "update"},
		{diff.ActionEnable, "+", "enable"},
		{diff.ActionDisable, "-", "disable"},
		{diff.ActionNone, " ", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			symbol, verb := actionSymbolVerb(tt.action)
			if symbol != tt.wantSymbol {
				t.Errorf("symbol = %q, want %q", symbol, tt.wantSymbol)
			}
			if verb != tt.wantVerb {
				t.Errorf("verb = %q, want %q", verb, tt.wantVerb)
			}
		})
	}
}

func TestFilterDiffBySelection(t *testing.T) {
	enabled := true
	result := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{Alias: "approved-marketplace", Action: diff.ActionAdd, Desired: &config.Marketplace{Repo: "test/repo"}},
			{Alias: "skipped-marketplace", Action: diff.ActionAdd, Desired: &config.Marketplace{Repo: "test/repo2"}},
			{Alias: "info-marketplace", Action: diff.ActionRemove, Current: &state.MarketplaceState{}},
		},
		Plugins: []diff.PluginDiff{
			{Name: "approved-plugin", Action: diff.ActionAdd, Desired: &config.Plugin{Name: "test", Enabled: &enabled}},
			{Name: "skipped-plugin", Action: diff.ActionEnable, Desired: &config.Plugin{Name: "test2", Enabled: &enabled}},
		},
		MCPServers: []diff.MCPServerDiff{
			{Name: "approved-mcp", Action: diff.ActionAdd, Desired: &config.MCPServer{Transport: "stdio", Command: "test"}},
		},
	}

	selection := &Selection{
		Marketplaces: map[string]bool{
			"approved-marketplace": true,
			"skipped-marketplace":  false,
		},
		Plugins: map[string]bool{
			"approved-plugin": true,
			"skipped-plugin":  false,
		},
		MCPServers: map[string]bool{
			"approved-mcp": true,
		},
	}

	filtered := FilterDiffBySelection(result, selection)

	// Check marketplaces
	if len(filtered.Marketplaces) != 2 { // approved + info (remove)
		t.Errorf("expected 2 marketplaces, got %d", len(filtered.Marketplaces))
	}
	foundApproved := false
	foundInfo := false
	for _, m := range filtered.Marketplaces {
		if m.Alias == "approved-marketplace" {
			foundApproved = true
		}
		if m.Alias == "info-marketplace" && m.Action == diff.ActionRemove {
			foundInfo = true
		}
	}
	if !foundApproved {
		t.Error("approved-marketplace should be in filtered result")
	}
	if !foundInfo {
		t.Error("info-marketplace (ActionRemove) should be kept")
	}

	// Check plugins
	if len(filtered.Plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(filtered.Plugins))
	}
	if filtered.Plugins[0].Name != "approved-plugin" {
		t.Errorf("expected approved-plugin, got %s", filtered.Plugins[0].Name)
	}

	// Check MCP servers
	if len(filtered.MCPServers) != 1 {
		t.Errorf("expected 1 MCP server, got %d", len(filtered.MCPServers))
	}
	if filtered.MCPServers[0].Name != "approved-mcp" {
		t.Errorf("expected approved-mcp, got %s", filtered.MCPServers[0].Name)
	}
}

func TestPromptForSelectionQuit(t *testing.T) {
	enabled := true
	result := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{Alias: "test-marketplace", Action: diff.ActionAdd, Desired: &config.Marketplace{Repo: "test/repo"}},
		},
		Plugins: []diff.PluginDiff{
			{Name: "test-plugin", Action: diff.ActionAdd, Desired: &config.Plugin{Name: "test", Enabled: &enabled}},
		},
		MCPServers: []diff.MCPServerDiff{},
	}

	// User quits at first prompt
	input := strings.NewReader("q\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	selection, proceed := p.PromptForSelection(result)

	if selection != nil {
		t.Error("expected nil selection on quit")
	}
	if proceed {
		t.Error("expected proceed=false on quit")
	}
}

func TestPromptForSelectionApproveAll(t *testing.T) {
	enabled := true
	result := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{Alias: "m1", Action: diff.ActionAdd, Desired: &config.Marketplace{Repo: "test/repo1"}},
			{Alias: "m2", Action: diff.ActionAdd, Desired: &config.Marketplace{Repo: "test/repo2"}},
		},
		Plugins: []diff.PluginDiff{
			{Name: "p1", Action: diff.ActionAdd, Desired: &config.Plugin{Name: "test1", Enabled: &enabled}},
		},
		MCPServers: []diff.MCPServerDiff{
			{Name: "mcp1", Action: diff.ActionAdd, Desired: &config.MCPServer{Transport: "stdio", Command: "test"}},
		},
	}

	// User approves all at first prompt, then confirms final
	input := strings.NewReader("a\ny\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	selection, proceed := p.PromptForSelection(result)

	if selection == nil {
		t.Fatal("expected non-nil selection")
		return
	}
	if !proceed {
		t.Error("expected proceed=true")
	}

	// All items should be approved
	if !selection.Marketplaces["m1"] {
		t.Error("m1 should be approved")
	}
	if !selection.Marketplaces["m2"] {
		t.Error("m2 should be approved")
	}
	if !selection.Plugins["p1"] {
		t.Error("p1 should be approved")
	}
	if !selection.MCPServers["mcp1"] {
		t.Error("mcp1 should be approved")
	}
}

func TestPromptForSelectionPartialApproval(t *testing.T) {
	enabled := true
	result := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{Alias: "m1", Action: diff.ActionAdd, Desired: &config.Marketplace{Repo: "test/repo1"}},
		},
		Plugins: []diff.PluginDiff{
			{Name: "p1", Action: diff.ActionAdd, Desired: &config.Plugin{Name: "test1", Enabled: &enabled}},
			{Name: "p2", Action: diff.ActionAdd, Desired: &config.Plugin{Name: "test2", Enabled: &enabled}},
		},
		MCPServers: []diff.MCPServerDiff{},
	}

	// Approve m1, approve p1, skip p2, confirm final
	input := strings.NewReader("y\ny\nn\ny\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	selection, proceed := p.PromptForSelection(result)

	if selection == nil {
		t.Fatal("expected non-nil selection")
		return
	}
	if !proceed {
		t.Error("expected proceed=true")
	}

	if !selection.Marketplaces["m1"] {
		t.Error("m1 should be approved")
	}
	if !selection.Plugins["p1"] {
		t.Error("p1 should be approved")
	}
	if selection.Plugins["p2"] {
		t.Error("p2 should NOT be approved")
	}
}

func TestPromptForSelectionNoChanges(t *testing.T) {
	result := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{Alias: "existing", Action: diff.ActionNone},
		},
		Plugins:    []diff.PluginDiff{},
		MCPServers: []diff.MCPServerDiff{},
	}

	input := strings.NewReader("")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	selection, proceed := p.PromptForSelection(result)

	if selection == nil {
		t.Fatal("expected non-nil selection")
	}
	if proceed {
		t.Error("expected proceed=false when no changes selected")
	}
	if !strings.Contains(output.String(), "No changes selected") {
		t.Error("expected 'No changes selected' message")
	}
}

func TestPromptForSelectionSkipsRemoveActions(t *testing.T) {
	result := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{Alias: "remove-only", Action: diff.ActionRemove, Current: &state.MarketplaceState{}},
		},
		Plugins:    []diff.PluginDiff{},
		MCPServers: []diff.MCPServerDiff{},
	}

	input := strings.NewReader("")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	selection, proceed := p.PromptForSelection(result)

	// ActionRemove items are info-only, no prompts for them
	if selection == nil {
		t.Fatal("expected non-nil selection")
	}
	if proceed {
		t.Error("expected proceed=false when only remove actions")
	}
}

func TestPromptForSelectionCancelFinal(t *testing.T) {
	enabled := true
	result := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{},
		Plugins: []diff.PluginDiff{
			{Name: "p1", Action: diff.ActionAdd, Desired: &config.Plugin{Name: "test", Enabled: &enabled}},
		},
		MCPServers: []diff.MCPServerDiff{},
	}

	// Approve plugin, but decline final confirmation
	input := strings.NewReader("y\nn\n")
	output := &bytes.Buffer{}
	p := NewPrompterWithIO(input, output)

	selection, proceed := p.PromptForSelection(result)

	if proceed {
		t.Error("expected proceed=false when final confirmation declined")
	}
	if selection == nil {
		t.Fatal("expected non-nil selection even when cancelled")
	}
	if !strings.Contains(output.String(), "Aborted") {
		t.Error("expected 'Aborted' message")
	}
}
