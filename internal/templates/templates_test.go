package templates

import (
	"os"
	"strings"
	"testing"
)

func TestList(t *testing.T) {
	names := List()

	// Should have at least 3 built-in templates
	if len(names) < 3 {
		t.Errorf("expected at least 3 templates, got %d", len(names))
	}

	// Check for expected templates
	expected := []string{"developer", "full", "minimal"}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected template '%s' not found in list", exp)
		}
	}

	// Should be sorted
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("templates not sorted: %v", names)
			break
		}
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"minimal", false},
		{"developer", false},
		{"full", false},
		{"nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := Get(tt.name)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Get(%s) expected error, got nil", tt.name)
				}
				return
			}

			if err != nil {
				t.Errorf("Get(%s) unexpected error: %v", tt.name, err)
				return
			}

			if tmpl == nil {
				t.Errorf("Get(%s) returned nil template", tt.name)
				return
			}

			if tmpl.Name != tt.name {
				t.Errorf("Get(%s) name = %s, want %s", tt.name, tmpl.Name, tt.name)
			}

			if len(tmpl.Content) == 0 {
				t.Errorf("Get(%s) returned empty content", tt.name)
			}

			// Verify it's valid YAML with required fields
			content := string(tmpl.Content)
			if !strings.Contains(content, "version:") {
				t.Errorf("Get(%s) content missing 'version:' field", tt.name)
			}
		})
	}
}

func TestGetDescription(t *testing.T) {
	tests := []struct {
		name     string
		wantDesc string
	}{
		{"minimal", "Basic starter (2 plugins)"},
		{"developer", "Development tools (3 plugins + MCP)"},
		{"full", "Comprehensive setup with all options"},
		{"unknown", "Custom template"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := GetDescription(tt.name)
			if desc != tt.wantDesc {
				t.Errorf("GetDescription(%s) = %q, want %q", tt.name, desc, tt.wantDesc)
			}
		})
	}
}

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variable
	_ = os.Setenv("TEST_VAR", "test_value")
	defer func() { _ = os.Unsetenv("TEST_VAR") }()

	tests := []struct {
		name    string
		input   string
		want    string
	}{
		{
			name:  "simple variable",
			input: "path: ${TEST_VAR}/subdir",
			want:  "path: test_value/subdir",
		},
		{
			name:  "variable with default, var set",
			input: "path: ${TEST_VAR:-default}/subdir",
			want:  "path: test_value/subdir",
		},
		{
			name:  "variable with default, var unset",
			input: "path: ${UNSET_VAR:-default_value}/subdir",
			want:  "path: default_value/subdir",
		},
		{
			name:  "unset variable without default",
			input: "path: ${UNSET_VAR}/subdir",
			want:  "path: /subdir",
		},
		{
			name:  "multiple variables",
			input: "path: ${TEST_VAR}/${TEST_VAR}",
			want:  "path: test_value/test_value",
		},
		{
			name:  "HOME variable",
			input: "path: ${HOME}/projects",
			want:  "path: " + os.Getenv("HOME") + "/projects",
		},
		{
			name:  "no variables",
			input: "path: /some/static/path",
			want:  "path: /some/static/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(ExpandEnvVars([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("ExpandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetExpanded(t *testing.T) {
	// The developer template uses ${HOME}
	tmpl, err := GetExpanded("developer")
	if err != nil {
		t.Fatalf("GetExpanded(developer) error: %v", err)
	}

	content := string(tmpl.Content)
	home := os.Getenv("HOME")

	// Should have expanded ${HOME}
	if strings.Contains(content, "${HOME}") {
		t.Errorf("GetExpanded(developer) did not expand ${HOME}")
	}

	// Should contain the expanded home path
	if !strings.Contains(content, home) {
		t.Errorf("GetExpanded(developer) content does not contain expanded HOME path")
	}
}

func TestTemplateContentValidity(t *testing.T) {
	// Verify all templates have valid YAML structure
	for _, name := range List() {
		t.Run(name, func(t *testing.T) {
			tmpl, err := Get(name)
			if err != nil {
				t.Fatalf("Get(%s) error: %v", name, err)
			}

			content := string(tmpl.Content)

			// Check required fields
			requiredFields := []string{
				"version:",
				"sources:",
				"plugins:",
			}

			for _, field := range requiredFields {
				if !strings.Contains(content, field) {
					t.Errorf("template %s missing required field: %s", name, field)
				}
			}
		})
	}
}
