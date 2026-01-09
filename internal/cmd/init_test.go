package cmd

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_DirectTemplate(t *testing.T) {
	// Create a temporary directory for output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "Clewfile")

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	err := runInit(stdin, &stdout, &stderr, "minimal", outputPath, false)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Clewfile was not created at %s", outputPath)
	}

	// Verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read Clewfile: %v", err)
	}

	if !strings.Contains(string(content), "version: 1") {
		t.Errorf("Clewfile missing version field")
	}

	if !strings.Contains(string(content), "sources:") {
		t.Errorf("Clewfile missing sources field")
	}

	// Verify output message
	if !strings.Contains(stdout.String(), "Created") {
		t.Errorf("stdout missing 'Created' message")
	}

	if !strings.Contains(stdout.String(), "Next steps:") {
		t.Errorf("stdout missing 'Next steps' guidance")
	}
}

func TestRunInit_AllTemplates(t *testing.T) {
	templates := []string{"minimal", "developer", "full"}

	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "Clewfile")

			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader("")

			err := runInit(stdin, &stdout, &stderr, tmpl, outputPath, false)
			if err != nil {
				t.Fatalf("runInit(%s) failed: %v", tmpl, err)
			}

			// Verify file was created
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("Clewfile was not created for template %s", tmpl)
			}

			// Verify content is valid YAML
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read Clewfile: %v", err)
			}

			if !strings.Contains(string(content), "version:") {
				t.Errorf("template %s: Clewfile missing version field", tmpl)
			}
		})
	}
}

func TestRunInit_ExistingFile_Abort(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "Clewfile")

	// Create existing file
	if err := os.WriteFile(outputPath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	// Simulate user pressing 'n' to abort
	stdin := strings.NewReader("n\n")

	err := runInit(stdin, &stdout, &stderr, "minimal", outputPath, false)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify file was NOT overwritten
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read Clewfile: %v", err)
	}

	if string(content) != "existing content" {
		t.Errorf("existing file was modified when user aborted")
	}

	// Verify abort message
	if !strings.Contains(stdout.String(), "Aborted") {
		t.Errorf("stdout missing 'Aborted' message")
	}
}

func TestRunInit_ExistingFile_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "Clewfile")

	// Create existing file
	if err := os.WriteFile(outputPath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	// Simulate user pressing 'y' to overwrite
	stdin := strings.NewReader("y\n")

	err := runInit(stdin, &stdout, &stderr, "minimal", outputPath, false)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify file WAS overwritten
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read Clewfile: %v", err)
	}

	if string(content) == "existing content" {
		t.Errorf("existing file was not overwritten when user confirmed")
	}

	if !strings.Contains(string(content), "version:") {
		t.Errorf("overwritten file does not contain valid Clewfile content")
	}
}

func TestRunInit_ForceFlag(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "Clewfile")

	// Create existing file
	if err := os.WriteFile(outputPath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	// Use force flag - should not prompt
	err := runInit(stdin, &stdout, &stderr, "minimal", outputPath, true)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify file WAS overwritten
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read Clewfile: %v", err)
	}

	if string(content) == "existing content" {
		t.Errorf("existing file was not overwritten with force flag")
	}
}

func TestRunInit_InvalidTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "Clewfile")

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	err := runInit(stdin, &stdout, &stderr, "nonexistent", outputPath, false)
	if err == nil {
		t.Errorf("expected error for nonexistent template, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should mention 'not found', got: %v", err)
	}
}

func TestRunInit_CreatesParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "nested", "dir", "Clewfile")

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	err := runInit(stdin, &stdout, &stderr, "minimal", outputPath, false)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify file was created with nested directories
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Clewfile was not created in nested directory")
	}
}

func TestExpandHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"~/.config/claude/Clewfile", filepath.Join(home, ".config/claude/Clewfile")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", "~"}, // Should not expand without trailing slash
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandHomePath(tt.input)
			if got != tt.want {
				t.Errorf("expandHomePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetDefaultClewfilePath(t *testing.T) {
	path := getDefaultClewfilePath()

	// Should contain claude directory
	if !strings.Contains(path, "claude") {
		t.Errorf("default path should contain 'claude': %s", path)
	}

	// Should end with Clewfile
	if !strings.HasSuffix(path, "Clewfile") {
		t.Errorf("default path should end with 'Clewfile': %s", path)
	}
}

func TestRunInit_EnvVarExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "Clewfile")

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	// Developer template uses ${HOME}
	err := runInit(stdin, &stdout, &stderr, "developer", outputPath, false)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read Clewfile: %v", err)
	}

	// Should NOT contain unexpanded ${HOME}
	if strings.Contains(string(content), "${HOME}") {
		t.Errorf("Clewfile contains unexpanded ${HOME}")
	}

	// Should contain actual home directory path
	home := os.Getenv("HOME")
	if !strings.Contains(string(content), home) {
		t.Errorf("Clewfile should contain expanded home path: %s", home)
	}
}

func TestSelectTemplateInteractive(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "select developer (1)",
			input: "1\n",
			want:  "developer", // First alphabetically: developer
		},
		{
			name:  "select full (2)",
			input: "2\n",
			want:  "full", // Second alphabetically: full
		},
		{
			name:  "select minimal (3)",
			input: "3\n",
			want:  "minimal", // Third alphabetically: minimal
		},
		{
			name:    "invalid selection",
			input:   "999\n",
			wantErr: true,
		},
		{
			name:    "non-numeric input",
			input:   "abc\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			reader := strings.NewReader(tt.input)

			// Use bufio.Reader for consistent behavior
			got, err := selectTemplateInteractive(bufio.NewReader(reader), &stdout)

			if tt.wantErr {
				if err == nil {
					t.Errorf("selectTemplateInteractive() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("selectTemplateInteractive() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("selectTemplateInteractive() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSelectTemplateInteractive_CustomURL(t *testing.T) {
	var stdout bytes.Buffer
	// Select custom option (4th), then provide URL
	stdin := strings.NewReader("4\nhttps://example.com/template.yaml\n")

	got, err := selectTemplateInteractive(bufio.NewReader(stdin), &stdout)
	if err != nil {
		t.Fatalf("selectTemplateInteractive() error: %v", err)
	}

	if got != "https://example.com/template.yaml" {
		t.Errorf("selectTemplateInteractive() = %q, want custom URL", got)
	}
}
