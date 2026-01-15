// Package state handles detection of current Claude Code configuration state.
package state

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Read implements Reader using claude CLI commands.
func (r *CLIReader) Read() (*State, error) {
	state := &State{
		Marketplaces: make(map[string]MarketplaceState),
		Plugins:      make(map[string]PluginState),
		MCPServers:   make(map[string]MCPServerState),
	}

	// Read marketplaces from marketplace list command
	if err := r.readMarketplaces(state); err != nil {
		return nil, fmt.Errorf("failed to read marketplaces: %w", err)
	}

	// Read MCP servers
	if err := r.readMCPServers(state); err != nil {
		return nil, fmt.Errorf("failed to read MCP servers: %w", err)
	}

	return state, nil
}

// readMarketplaces parses output from `claude plugin marketplace list`.
func (r *CLIReader) readMarketplaces(state *State) error {
	cmd := exec.Command("claude", "plugin", "marketplace", "list")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run claude plugin marketplace list: %w", err)
	}

	return parseMarketplaceList(output, state)
}

// parseMarketplaceList parses the human-readable marketplace list output.
// Format:
//
//	❯ marketplace-name
//	  Source: GitHub (owner/repo)
func parseMarketplaceList(output []byte, state *State) error {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var currentName string

	// Regex to match marketplace name line: ❯ marketplace-name
	nameRegex := regexp.MustCompile(`^\s*❯\s+(\S+)\s*$`)
	// Regex to match source line: Source: GitHub (owner/repo)
	sourceRegex := regexp.MustCompile(`^\s*Source:\s+GitHub\s+\(([^)]+)\)\s*$`)

	for scanner.Scan() {
		line := scanner.Text()

		if matches := nameRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentName = matches[1]
			continue
		}

		if matches := sourceRegex.FindStringSubmatch(line); len(matches) > 1 && currentName != "" {
			repo := matches[1]

			state.Marketplaces[currentName] = MarketplaceState{
				Alias: currentName,
				Repo:  repo,
			}
			currentName = ""
		}
	}

	return scanner.Err()
}

// readMCPServers parses output from `claude mcp list`.
func (r *CLIReader) readMCPServers(state *State) error {
	cmd := exec.Command("claude", "mcp", "list")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run claude mcp list: %w", err)
	}

	return parseMCPList(output, state)
}

// parseMCPList parses the human-readable MCP server list output.
// Format:
//
//	server-name: command args - status
//	server-name: https://url (HTTP) - status
func parseMCPList(output []byte, state *State) error {
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Skip the "Checking MCP server health..." line
	// Regex to match MCP server lines:
	// name: command args - status
	// name: url (HTTP/SSE) - status
	serverRegex := regexp.MustCompile(`^([^:]+):\s+(.+)\s+-\s+.+$`)
	httpRegex := regexp.MustCompile(`^(https?://[^\s]+)\s+\((HTTP|SSE)\)$`)

	for scanner.Scan() {
		line := scanner.Text()

		matches := serverRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		name := strings.TrimSpace(matches[1])
		details := strings.TrimSpace(matches[2])

		// Skip plugin-provided MCP servers (they have : in the name)
		if strings.Contains(name, ":") {
			continue
		}

		server := MCPServerState{
			Name:  name,
			Scope: "user", // Default scope
		}

		// Check if it's an HTTP/SSE server
		if httpMatches := httpRegex.FindStringSubmatch(details); len(httpMatches) > 2 {
			server.Transport = strings.ToLower(httpMatches[2])
			server.URL = httpMatches[1]
		} else {
			// It's a stdio server with command and args
			server.Transport = "stdio"
			parts := strings.Fields(details)
			if len(parts) > 0 {
				server.Command = parts[0]
				if len(parts) > 1 {
					server.Args = parts[1:]
				}
			}
		}

		state.MCPServers[name] = server
	}

	return scanner.Err()
}
