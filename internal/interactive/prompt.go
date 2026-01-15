// Package interactive provides interactive prompts for user confirmation.
package interactive

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"golang.org/x/term"

	"github.com/adamancini/clew/internal/diff"
)

// titleCase capitalizes the first letter of a string.
// This is a simple replacement for strings.Title which is deprecated.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// Response represents the user's response to a prompt.
type Response int

const (
	ResponseYes Response = iota // Proceed with this change
	ResponseNo                  // Skip this change
	ResponseAll                 // Approve all remaining changes
	ResponseQuit                // Abort interactive mode
)

// Prompter handles interactive prompts for diff confirmation.
type Prompter struct {
	in       io.Reader
	out      io.Writer
	scanner  *bufio.Scanner
	approveAll bool
}

// Selection tracks which items were approved or skipped.
type Selection struct {
	Marketplaces map[string]bool // alias -> approved
	Plugins      map[string]bool // name -> approved
	MCPServers   map[string]bool // name -> approved
}

// NewSelection creates an empty selection.
func NewSelection() *Selection {
	return &Selection{
		Marketplaces: make(map[string]bool),
		Plugins:      make(map[string]bool),
		MCPServers:   make(map[string]bool),
	}
}

// NewPrompter creates a prompter with stdin/stdout.
func NewPrompter() *Prompter {
	return NewPrompterWithIO(os.Stdin, os.Stdout)
}

// NewPrompterWithIO creates a prompter with custom input/output (for testing).
func NewPrompterWithIO(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{
		in:      in,
		out:     out,
		scanner: bufio.NewScanner(in),
	}
}

// IsTerminal checks if stdin is a terminal (TTY).
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// prompt displays a question and reads the response.
func (p *Prompter) prompt(format string, args ...interface{}) Response {
	if p.approveAll {
		return ResponseYes
	}

	_, _ = fmt.Fprintf(p.out, format, args...)
	_, _ = fmt.Fprint(p.out, " [y/n/a/q] ")

	if !p.scanner.Scan() {
		return ResponseQuit
	}

	input := strings.ToLower(strings.TrimSpace(p.scanner.Text()))
	switch input {
	case "y", "yes":
		return ResponseYes
	case "n", "no":
		return ResponseNo
	case "a", "all":
		p.approveAll = true
		return ResponseYes
	case "q", "quit":
		return ResponseQuit
	default:
		// Default to no for invalid input
		_, _ = fmt.Fprintln(p.out, "Invalid response, skipping.")
		return ResponseNo
	}
}

// confirmFinal asks for final confirmation before executing.
func (p *Prompter) confirmFinal() bool {
	_, _ = fmt.Fprint(p.out, "\nProceed with sync? [y/n] ")
	if !p.scanner.Scan() {
		return false
	}
	input := strings.ToLower(strings.TrimSpace(p.scanner.Text()))
	return input == "y" || input == "yes"
}

// PromptForSelection interactively prompts for each item in the diff result.
// Returns a Selection indicating which items were approved, and whether to proceed.
func (p *Prompter) PromptForSelection(result *diff.Result) (*Selection, bool) {
	selection := NewSelection()

	// Track counts for summary
	willAdd := 0
	willUpdate := 0
	skipped := 0

	// Process marketplaces
	hasMarketplaces := false
	for _, m := range result.Marketplaces {
		if m.Action == diff.ActionNone || m.Action == diff.ActionRemove {
			continue
		}
		if !hasMarketplaces {
			_, _ = fmt.Fprintln(p.out, "\nMarketplaces:")
			hasMarketplaces = true
		}
		approved, quit := p.promptMarketplace(m)
		if quit {
			return nil, false
		}
		selection.Marketplaces[m.Alias] = approved
		if approved {
			if m.Action == diff.ActionAdd {
				willAdd++
			} else {
				willUpdate++
			}
		} else {
			skipped++
		}
	}

	// Process plugins
	hasPlugins := false
	for _, pl := range result.Plugins {
		if pl.Action == diff.ActionNone || pl.Action == diff.ActionRemove {
			continue
		}
		if !hasPlugins {
			_, _ = fmt.Fprintln(p.out, "\nPlugins:")
			hasPlugins = true
		}
		approved, quit := p.promptPlugin(pl)
		if quit {
			return nil, false
		}
		selection.Plugins[pl.Name] = approved
		if approved {
			if pl.Action == diff.ActionAdd {
				willAdd++
			} else {
				willUpdate++
			}
		} else {
			skipped++
		}
	}

	// Process MCP servers
	hasMCP := false
	for _, m := range result.MCPServers {
		if m.Action == diff.ActionNone || m.Action == diff.ActionRemove {
			continue
		}
		if !hasMCP {
			_, _ = fmt.Fprintln(p.out, "\nMCP Servers:")
			hasMCP = true
		}
		approved, quit := p.promptMCPServer(m)
		if quit {
			return nil, false
		}
		selection.MCPServers[m.Name] = approved
		if approved {
			if m.Action == diff.ActionAdd {
				willAdd++
			} else {
				willUpdate++
			}
		} else {
			skipped++
		}
	}

	// Show summary
	_, _ = fmt.Fprintln(p.out, "\nSummary:")
	_, _ = fmt.Fprintf(p.out, "  Will apply: %d changes\n", willAdd+willUpdate)
	if skipped > 0 {
		_, _ = fmt.Fprintf(p.out, "  Skipped: %d\n", skipped)
	}

	if willAdd+willUpdate == 0 {
		_, _ = fmt.Fprintln(p.out, "No changes selected.")
		return selection, false
	}

	// Final confirmation
	if !p.confirmFinal() {
		_, _ = fmt.Fprintln(p.out, "Aborted.")
		return selection, false
	}

	return selection, true
}

// promptMarketplace prompts for a single marketplace action.
func (p *Prompter) promptMarketplace(m diff.MarketplaceDiff) (approved bool, quit bool) {
	symbol, verb := actionSymbolVerb(m.Action)
	_, _ = fmt.Fprintf(p.out, "  %s %s (will %s)\n", symbol, m.Alias, verb)

	repo := ""
	if m.Desired != nil {
		repo = m.Desired.Repo
	}

	resp := p.prompt("    -> %s marketplace %s from %s?", titleCase(verb), m.Alias, repo)
	switch resp {
	case ResponseYes:
		return true, false
	case ResponseNo:
		_, _ = fmt.Fprintf(p.out, "    %s Skipped\n", skipSymbol)
		return false, false
	case ResponseQuit:
		_, _ = fmt.Fprintln(p.out, "\nAborted.")
		return false, true
	default:
		return true, false
	}
}

// promptPlugin prompts for a single plugin action.
func (p *Prompter) promptPlugin(pl diff.PluginDiff) (approved bool, quit bool) {
	symbol, verb := actionSymbolVerb(pl.Action)
	_, _ = fmt.Fprintf(p.out, "  %s %s (will %s)\n", symbol, pl.Name, verb)

	resp := p.prompt("    -> %s %s?", titleCase(verb), pl.Name)
	switch resp {
	case ResponseYes:
		return true, false
	case ResponseNo:
		_, _ = fmt.Fprintf(p.out, "    %s Skipped\n", skipSymbol)
		return false, false
	case ResponseQuit:
		_, _ = fmt.Fprintln(p.out, "\nAborted.")
		return false, true
	default:
		return true, false
	}
}

// promptMCPServer prompts for a single MCP server action.
func (p *Prompter) promptMCPServer(m diff.MCPServerDiff) (approved bool, quit bool) {
	symbol, verb := actionSymbolVerb(m.Action)

	extra := ""
	if m.RequiresOAuth {
		extra = " (requires OAuth - manual setup needed)"
	}
	_, _ = fmt.Fprintf(p.out, "  %s %s (will %s)%s\n", symbol, m.Name, verb, extra)

	resp := p.prompt("    -> %s MCP server %s?", titleCase(verb), m.Name)
	switch resp {
	case ResponseYes:
		return true, false
	case ResponseNo:
		_, _ = fmt.Fprintf(p.out, "    %s Skipped\n", skipSymbol)
		return false, false
	case ResponseQuit:
		_, _ = fmt.Fprintln(p.out, "\nAborted.")
		return false, true
	default:
		return true, false
	}
}

// Symbols for output
const (
	addSymbol    = "+"
	removeSymbol = "-"
	updateSymbol = "~"
	skipSymbol   = "-"
	okSymbol     = "ok"
)

// actionSymbolVerb returns the symbol and verb for a diff action.
func actionSymbolVerb(action diff.Action) (symbol, verb string) {
	switch action {
	case diff.ActionAdd:
		return addSymbol, "add"
	case diff.ActionRemove:
		return removeSymbol, "remove"
	case diff.ActionUpdate:
		return updateSymbol, "update"
	case diff.ActionEnable:
		return addSymbol, "enable"
	case diff.ActionDisable:
		return removeSymbol, "disable"
	default:
		return " ", ""
	}
}

// FilterDiffBySelection returns a new diff.Result containing only approved items.
func FilterDiffBySelection(result *diff.Result, selection *Selection) *diff.Result {
	filtered := &diff.Result{
		Marketplaces: make([]diff.MarketplaceDiff, 0),
		Plugins:      make([]diff.PluginDiff, 0),
		MCPServers:   make([]diff.MCPServerDiff, 0),
	}

	for _, m := range result.Marketplaces {
		// Keep ActionNone and ActionRemove (info only), filter actionable items by selection
		if m.Action == diff.ActionNone || m.Action == diff.ActionRemove {
			filtered.Marketplaces = append(filtered.Marketplaces, m)
		} else if selection.Marketplaces[m.Alias] {
			filtered.Marketplaces = append(filtered.Marketplaces, m)
		}
	}

	for _, p := range result.Plugins {
		if p.Action == diff.ActionNone || p.Action == diff.ActionRemove {
			filtered.Plugins = append(filtered.Plugins, p)
		} else if selection.Plugins[p.Name] {
			filtered.Plugins = append(filtered.Plugins, p)
		}
	}

	for _, m := range result.MCPServers {
		if m.Action == diff.ActionNone || m.Action == diff.ActionRemove {
			filtered.MCPServers = append(filtered.MCPServers, m)
		} else if selection.MCPServers[m.Name] {
			filtered.MCPServers = append(filtered.MCPServers, m)
		}
	}

	return filtered
}
