package diff

import (
	"fmt"
	"strings"
)

// Command represents a CLI command to reconcile state.
type Command struct {
	Command     string `json:"command" yaml:"command"`
	Description string `json:"description" yaml:"description"`
}

// GenerateCommands generates CLI commands to reconcile the diff.
// Returns commands in the order they should be executed.
func (r *Result) GenerateCommands() []Command {
	var commands []Command

	// 1. Add marketplaces first (plugins depend on them)
	for _, m := range r.Marketplaces {
		if m.Action == ActionAdd && m.Desired != nil {
			cmd := fmt.Sprintf("claude plugin marketplace add %s", m.Desired.Repo)
			commands = append(commands, Command{
				Command:     cmd,
				Description: fmt.Sprintf("Add marketplace: %s", m.Alias),
			})
		}
	}

	// 2. Install plugins
	for _, p := range r.Plugins {
		switch p.Action {
		case ActionAdd:
			if p.Desired != nil {
				// All plugins are installed from github sources
				cmd := fmt.Sprintf("claude plugin install %s", p.Name)
				if p.Desired.Scope != "" && p.Desired.Scope != "user" {
					cmd += fmt.Sprintf(" --scope %s", p.Desired.Scope)
				}
				desc := fmt.Sprintf("Install plugin: %s", p.Name)
				commands = append(commands, Command{
					Command:     cmd,
					Description: desc,
				})
			}

		case ActionRemove:
			// Non-destructive by default, but show the command
			cmd := fmt.Sprintf("claude plugin uninstall %s", p.Name)
			commands = append(commands, Command{
				Command:     cmd,
				Description: fmt.Sprintf("Remove plugin not in Clewfile: %s", p.Name),
			})

		case ActionEnable:
			cmd := fmt.Sprintf("claude plugin enable %s", p.Name)
			if p.Current != nil && p.Current.Scope != "" && p.Current.Scope != "user" {
				cmd += fmt.Sprintf(" --scope %s", p.Current.Scope)
			}
			commands = append(commands, Command{
				Command:     cmd,
				Description: fmt.Sprintf("Enable plugin: %s", p.Name),
			})

		case ActionDisable:
			cmd := fmt.Sprintf("claude plugin disable %s", p.Name)
			if p.Current != nil && p.Current.Scope != "" && p.Current.Scope != "user" {
				cmd += fmt.Sprintf(" --scope %s", p.Current.Scope)
			}
			commands = append(commands, Command{
				Command:     cmd,
				Description: fmt.Sprintf("Disable plugin: %s", p.Name),
			})
		}
	}

	return commands
}

// FormatCommands formats commands for shell execution.
func FormatCommands(commands []Command, includeComments bool) string {
	var output strings.Builder

	for _, cmd := range commands {
		if includeComments {
			output.WriteString(fmt.Sprintf("# %s\n", cmd.Description))
		}
		output.WriteString(fmt.Sprintf("%s\n", cmd.Command))
		if includeComments {
			output.WriteString("\n")
		}
	}

	return output.String()
}
