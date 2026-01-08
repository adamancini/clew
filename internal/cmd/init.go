package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/templates"
)

func newInitCmd() *cobra.Command {
	var templateName string
	var outputPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new Clewfile from a template",
		Long: `Create a new Clewfile from a built-in or custom template.

Available templates:
  minimal    - Basic starter (2 plugins)
  developer  - Development tools (3 plugins + MCP)
  full       - Comprehensive setup with all options

Examples:
  clew init                              # Interactive mode
  clew init --template=minimal           # Direct template selection
  clew init --template=developer
  clew init --template=https://...       # Custom template URL
  clew init --config ~/path/Clewfile     # Custom output location`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), templateName, outputPath, force)
		},
	}

	cmd.Flags().StringVarP(&templateName, "template", "t", "", "Template name or URL")
	cmd.Flags().StringVar(&outputPath, "config", "", "Output path for Clewfile")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing Clewfile")

	// Register completion for template flag
	_ = cmd.RegisterFlagCompletionFunc("template", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		templateList := templates.List()
		var completions []string
		for _, name := range templateList {
			desc := templates.GetDescription(name)
			completions = append(completions, fmt.Sprintf("%s\t%s", name, desc))
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// runInit executes the init workflow.
func runInit(stdin io.Reader, stdout, stderr io.Writer, templateName, outputPath string, force bool) error {
	reader := bufio.NewReader(stdin)

	// Determine output path
	if outputPath == "" {
		outputPath = getDefaultClewfilePath()
	}

	// Expand ~ in path
	outputPath = expandHomePath(outputPath)

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil && !force {
		_, _ = fmt.Fprintf(stderr, "Clewfile already exists at %s\n", outputPath)
		_, _ = fmt.Fprintf(stdout, "Overwrite? [y/N]: ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			_, _ = fmt.Fprintln(stdout, "Aborted.")
			return nil
		}
	}

	// Get template content
	var content []byte
	var selectedTemplate string

	if templateName == "" {
		// Interactive mode
		selected, err := selectTemplateInteractive(reader, stdout)
		if err != nil {
			return err
		}
		templateName = selected
	}

	// Check if it's a URL
	if strings.HasPrefix(templateName, "http://") || strings.HasPrefix(templateName, "https://") {
		var err error
		content, err = fetchRemoteTemplate(templateName)
		if err != nil {
			return fmt.Errorf("failed to fetch template: %w", err)
		}
		selectedTemplate = "custom"
	} else {
		// Built-in template
		tmpl, err := templates.GetExpanded(templateName)
		if err != nil {
			return fmt.Errorf("failed to load template: %w", err)
		}
		content = tmpl.Content
		selectedTemplate = templateName
	}

	// Validate the template content before writing
	if err := validateTemplateContent(content); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	// Show preview in interactive mode
	if selectedTemplate != "custom" && !quiet {
		_, _ = fmt.Fprintf(stdout, "\nPreview of '%s' template:\n", selectedTemplate)
		_, _ = fmt.Fprintln(stdout, strings.Repeat("-", 40))
		// Show first 20 lines or full content if shorter
		lines := strings.Split(string(content), "\n")
		maxLines := 20
		if len(lines) <= maxLines {
			_, _ = fmt.Fprintln(stdout, string(content))
		} else {
			for i := 0; i < maxLines; i++ {
				_, _ = fmt.Fprintln(stdout, lines[i])
			}
			_, _ = fmt.Fprintf(stdout, "... (%d more lines)\n", len(lines)-maxLines)
		}
		_, _ = fmt.Fprintln(stdout, strings.Repeat("-", 40))
	}

	// Ask for output location in interactive mode if not specified via flag
	if configPath == "" && outputPath == getDefaultClewfilePath() && !quiet {
		_, _ = fmt.Fprintf(stdout, "\nWhere should I create the Clewfile? [%s]: ", outputPath)
		answer, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read input: %w", err)
		}
		answer = strings.TrimSpace(answer)
		if answer != "" {
			outputPath = expandHomePath(answer)
		}
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
	}

	// Write the file
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write Clewfile: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "\nCreated %s\n", outputPath)
	_, _ = fmt.Fprintln(stdout, "\nNext steps:")
	_, _ = fmt.Fprintln(stdout, "  1. Edit the Clewfile to customize")
	_, _ = fmt.Fprintln(stdout, "  2. Run 'clew diff' to preview changes")
	_, _ = fmt.Fprintln(stdout, "  3. Run 'clew sync' to apply")

	return nil
}

// selectTemplateInteractive shows an interactive menu for template selection.
func selectTemplateInteractive(reader *bufio.Reader, stdout io.Writer) (string, error) {
	templateList := templates.List()

	_, _ = fmt.Fprintln(stdout, "\nSelect a Clewfile template:")
	for i, name := range templateList {
		desc := templates.GetDescription(name)
		_, _ = fmt.Fprintf(stdout, "  %d. %-12s - %s\n", i+1, name, desc)
	}
	_, _ = fmt.Fprintf(stdout, "  %d. %-12s - Provide custom template URL\n", len(templateList)+1, "custom")

	_, _ = fmt.Fprintf(stdout, "\nSelect [1-%d]: ", len(templateList)+1)

	answer, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	answer = strings.TrimSpace(answer)

	// Parse selection
	num, err := strconv.Atoi(answer)
	if err != nil || num < 1 || num > len(templateList)+1 {
		return "", fmt.Errorf("invalid selection: %s", answer)
	}

	if num == len(templateList)+1 {
		// Custom template URL
		_, _ = fmt.Fprint(stdout, "Enter template URL: ")
		url, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read URL: %w", err)
		}
		return strings.TrimSpace(url), nil
	}

	return templateList[num-1], nil
}

// fetchRemoteTemplate downloads a template from a URL.
func fetchRemoteTemplate(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return content, nil
}

// validateTemplateContent validates that the content is a valid Clewfile.
func validateTemplateContent(content []byte) error {
	// Use the config parser to validate
	// We create a temporary file to leverage the existing Load function
	tmpFile, err := os.CreateTemp("", "clewfile-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmpFile.Write(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	_, err = config.Load(tmpName)
	return err
}

// getDefaultClewfilePath returns the default Clewfile location.
func getDefaultClewfilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "Clewfile"
	}

	// Prefer XDG_CONFIG_HOME if set
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig != "" {
		return filepath.Join(xdgConfig, "claude", "Clewfile")
	}

	return filepath.Join(home, ".config", "claude", "Clewfile")
}

// expandHomePath expands ~ to the user's home directory.
func expandHomePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
