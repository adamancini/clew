// Package templates provides embedded Clewfile templates for clew init.
package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"
)

//go:embed *.yaml
var templatesFS embed.FS

// Template represents a Clewfile template with metadata.
type Template struct {
	Name        string
	Description string
	Content     []byte
}

// Available templates with their descriptions.
var templateDescriptions = map[string]string{
	"minimal":   "Basic starter (2 plugins)",
	"developer": "Development tools (3 plugins + MCP)",
	"full":      "Comprehensive setup with all options",
}

// List returns all available template names sorted alphabetically.
func List() []string {
	entries, err := templatesFS.ReadDir(".")
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// Get returns a template by name.
func Get(name string) (*Template, error) {
	filename := name + ".yaml"
	content, err := templatesFS.ReadFile(filename)
	if err != nil {
		if pathErr, ok := err.(*fs.PathError); ok {
			return nil, fmt.Errorf("template '%s' not found: %w", name, pathErr)
		}
		return nil, fmt.Errorf("failed to read template '%s': %w", name, err)
	}

	return &Template{
		Name:        name,
		Description: templateDescriptions[name],
		Content:     content,
	}, nil
}

// GetDescription returns the description for a template.
func GetDescription(name string) string {
	if desc, ok := templateDescriptions[name]; ok {
		return desc
	}
	return "Custom template"
}

// envVarPattern matches ${VAR} and ${VAR:-default} patterns.
var envVarPattern = regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)

// ExpandEnvVars replaces ${VAR} and ${VAR:-default} patterns in content.
func ExpandEnvVars(content []byte) []byte {
	result := envVarPattern.ReplaceAllFunc(content, func(match []byte) []byte {
		parts := envVarPattern.FindSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := string(parts[1])
		value := os.Getenv(varName)

		if value == "" && len(parts) >= 3 && len(parts[2]) > 0 {
			// Use default value
			value = string(parts[2])
		}

		return []byte(value)
	})

	return result
}

// GetExpanded returns a template with environment variables expanded.
func GetExpanded(name string) (*Template, error) {
	tmpl, err := Get(name)
	if err != nil {
		return nil, err
	}

	tmpl.Content = ExpandEnvVars(tmpl.Content)
	return tmpl, nil
}
