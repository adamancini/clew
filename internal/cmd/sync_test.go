package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/adamancini/clew/internal/sync"
)

// captureStdout captures stdout output during function execution.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stdout = w

	f()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe: %v", err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	return buf.String()
}

func TestCapitalizeAction(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   string
	}{
		{
			name:   "lowercase action",
			action: "add",
			want:   "Add",
		},
		{
			name:   "already capitalized",
			action: "Add",
			want:   "Add",
		},
		{
			name:   "empty string",
			action: "",
			want:   "",
		},
		{
			name:   "single character",
			action: "a",
			want:   "A",
		},
		{
			name:   "multi-word action",
			action: "enable",
			want:   "Enable",
		},
		{
			name:   "disable action",
			action: "disable",
			want:   "Disable",
		},
		{
			name:   "update action",
			action: "update",
			want:   "Update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := capitalizeAction(tt.action)
			if got != tt.want {
				t.Errorf("capitalizeAction(%q) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

func TestPrintSyncResultVerbose_SuccessfulOperations(t *testing.T) {
	tests := []struct {
		name           string
		result         *sync.Result
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "single successful add operation",
			result: &sync.Result{
				Installed: 1,
				Updated:   0,
				Failed:    0,
				Operations: []sync.Operation{
					{
						Type:        "plugin",
						Name:        "context7",
						Action:      "add",
						Command:     "claude plugin install context7",
						Description: "Installing plugin context7",
						Success:     true,
					},
				},
			},
			wantContains: []string{
				"Add: Installing plugin context7",
				"→ claude plugin install context7",
				"✓ Success",
				"Summary:",
				"Installed: 1",
				"Updated: 0",
				"Failed: 0",
			},
			wantNotContain: []string{
				"✗ Failed",
				"Errors:",
			},
		},
		{
			name: "multiple successful operations",
			result: &sync.Result{
				Installed: 2,
				Updated:   1,
				Failed:    0,
				Operations: []sync.Operation{
					{
						Type:        "marketplace",
						Name:        "official",
						Action:      "add",
						Command:     "claude marketplace add official",
						Description: "Adding marketplace official",
						Success:     true,
					},
					{
						Type:        "plugin",
						Name:        "test-plugin",
						Action:      "add",
						Command:     "claude plugin install test-plugin",
						Description: "Installing plugin test-plugin",
						Success:     true,
					},
					{
						Type:        "plugin",
						Name:        "other-plugin",
						Action:      "enable",
						Command:     "claude plugin enable other-plugin",
						Description: "Enabling plugin other-plugin",
						Success:     true,
					},
				},
			},
			wantContains: []string{
				"Add: Adding marketplace official",
				"Add: Installing plugin test-plugin",
				"Enable: Enabling plugin other-plugin",
				"Installed: 2",
				"Updated: 1",
			},
		},
		{
			name: "no operations - empty result",
			result: &sync.Result{
				Installed:  0,
				Updated:    0,
				Failed:     0,
				Operations: []sync.Operation{},
			},
			wantContains: []string{
				"Summary:",
				"Installed: 0",
				"Updated: 0",
				"Failed: 0",
			},
		},
		{
			name: "operations with skipped count",
			result: &sync.Result{
				Installed: 1,
				Updated:   0,
				Skipped:   2,
				Failed:    0,
				Operations: []sync.Operation{
					{
						Type:        "plugin",
						Name:        "plugin1",
						Action:      "add",
						Command:     "claude plugin install plugin1",
						Description: "Installing plugin1",
						Success:     true,
					},
				},
			},
			wantContains: []string{
				"Installed: 1",
				"Skipped: 2",
			},
		},
		{
			name: "operations with attention items",
			result: &sync.Result{
				Installed: 1,
				Updated:   0,
				Failed:    0,
				Operations: []sync.Operation{
					{
						Type:        "plugin",
						Name:        "plugin1",
						Action:      "add",
						Command:     "claude plugin install plugin1",
						Description: "Installing plugin1",
						Success:     true,
					},
				},
				Attention: []string{
					"mcp (oauth): notion",
					"plugin: deprecated-plugin",
				},
			},
			wantContains: []string{
				"Items needing attention:",
				"mcp (oauth): notion",
				"plugin: deprecated-plugin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				printSyncResultVerbose(tt.result)
			})

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("output should not contain %q\nGot:\n%s", notWant, output)
				}
			}
		})
	}
}

func TestPrintSyncResultVerbose_FailedOperations(t *testing.T) {
	tests := []struct {
		name         string
		result       *sync.Result
		wantContains []string
	}{
		{
			name: "single failed operation with error message",
			result: &sync.Result{
				Installed: 0,
				Updated:   0,
				Failed:    1,
				Operations: []sync.Operation{
					{
						Type:        "plugin",
						Name:        "bad-plugin",
						Action:      "add",
						Command:     "claude plugin install bad-plugin",
						Description: "Installing plugin bad-plugin",
						Success:     false,
						Error:       "plugin not found in marketplace",
					},
				},
				Errors: []error{
					&testError{msg: "plugin not found in marketplace"},
				},
			},
			wantContains: []string{
				"Add: Installing plugin bad-plugin",
				"✗ Failed: plugin not found in marketplace",
				"Failed: 1",
				"Errors:",
				"plugin not found in marketplace",
			},
		},
		{
			name: "failed operation without error message",
			result: &sync.Result{
				Installed: 0,
				Updated:   0,
				Failed:    1,
				Operations: []sync.Operation{
					{
						Type:        "mcp",
						Name:        "server1",
						Action:      "add",
						Command:     "claude mcp add server1",
						Description: "Adding MCP server server1",
						Success:     false,
						Error:       "",
					},
				},
			},
			wantContains: []string{
				"Add: Adding MCP server server1",
				"✗ Failed",
				"Failed: 1",
			},
		},
		{
			name: "mixed success and failure",
			result: &sync.Result{
				Installed: 1,
				Updated:   0,
				Failed:    1,
				Operations: []sync.Operation{
					{
						Type:        "plugin",
						Name:        "good-plugin",
						Action:      "add",
						Command:     "claude plugin install good-plugin",
						Description: "Installing plugin good-plugin",
						Success:     true,
					},
					{
						Type:        "plugin",
						Name:        "bad-plugin",
						Action:      "add",
						Command:     "claude plugin install bad-plugin",
						Description: "Installing plugin bad-plugin",
						Success:     false,
						Error:       "network timeout",
					},
				},
			},
			wantContains: []string{
				"✓ Success",
				"✗ Failed: network timeout",
				"Installed: 1",
				"Failed: 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				printSyncResultVerbose(tt.result)
			})

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestPrintSyncResultShort_SuccessfulOperations(t *testing.T) {
	tests := []struct {
		name         string
		result       *sync.Result
		wantContains []string
	}{
		{
			name: "single successful operation",
			result: &sync.Result{
				Installed: 1,
				Updated:   0,
				Failed:    0,
				Operations: []sync.Operation{
					{
						Type:    "plugin",
						Name:    "context7",
						Action:  "add",
						Success: true,
					},
				},
			},
			wantContains: []string{
				"✓ context7 (plugin add)",
				"Summary: 1 installed, 0 updated",
			},
		},
		{
			name: "multiple successful operations",
			result: &sync.Result{
				Installed: 2,
				Updated:   1,
				Failed:    0,
				Operations: []sync.Operation{
					{
						Type:    "marketplace",
						Name:    "official",
						Action:  "add",
						Success: true,
					},
					{
						Type:    "plugin",
						Name:    "test-plugin",
						Action:  "add",
						Success: true,
					},
					{
						Type:    "plugin",
						Name:    "other-plugin",
						Action:  "enable",
						Success: true,
					},
				},
			},
			wantContains: []string{
				"✓ official (marketplace add)",
				"✓ test-plugin (plugin add)",
				"✓ other-plugin (plugin enable)",
				"Summary: 2 installed, 1 updated",
			},
		},
		{
			name: "operations with attention items",
			result: &sync.Result{
				Installed: 1,
				Updated:   0,
				Failed:    0,
				Operations: []sync.Operation{
					{
						Type:    "plugin",
						Name:    "plugin1",
						Action:  "add",
						Success: true,
					},
				},
				Attention: []string{
					"mcp (oauth): notion",
				},
			},
			wantContains: []string{
				"✓ plugin1 (plugin add)",
				"Items needing attention:",
				"mcp (oauth): notion",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				printSyncResultShort(tt.result)
			})

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestPrintSyncResultShort_FailedOperations(t *testing.T) {
	tests := []struct {
		name         string
		result       *sync.Result
		wantContains []string
	}{
		{
			name: "single failed operation with error",
			result: &sync.Result{
				Installed: 0,
				Updated:   0,
				Failed:    1,
				Operations: []sync.Operation{
					{
						Type:    "plugin",
						Name:    "bad-plugin",
						Action:  "add",
						Success: false,
						Error:   "plugin not found",
					},
				},
			},
			wantContains: []string{
				"✗ bad-plugin (plugin add)",
				"Error: plugin not found",
				"1 failed",
			},
		},
		{
			name: "failed operation without error message",
			result: &sync.Result{
				Installed: 0,
				Updated:   0,
				Failed:    1,
				Operations: []sync.Operation{
					{
						Type:    "mcp",
						Name:    "server1",
						Action:  "add",
						Success: false,
						Error:   "",
					},
				},
			},
			wantContains: []string{
				"✗ server1 (mcp add)",
				"1 failed",
			},
		},
		{
			name: "mixed success and failure",
			result: &sync.Result{
				Installed: 1,
				Updated:   0,
				Failed:    1,
				Operations: []sync.Operation{
					{
						Type:    "plugin",
						Name:    "good-plugin",
						Action:  "add",
						Success: true,
					},
					{
						Type:    "plugin",
						Name:    "bad-plugin",
						Action:  "add",
						Success: false,
						Error:   "timeout",
					},
				},
			},
			wantContains: []string{
				"✓ good-plugin (plugin add)",
				"✗ bad-plugin (plugin add)",
				"Error: timeout",
				"1 installed",
				"1 failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				printSyncResultShort(tt.result)
			})

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestPrintSyncResultText_SelectsCorrectFormatter(t *testing.T) {
	result := &sync.Result{
		Installed: 1,
		Updated:   0,
		Failed:    0,
		Operations: []sync.Operation{
			{
				Type:        "plugin",
				Name:        "test-plugin",
				Action:      "add",
				Command:     "claude plugin install test-plugin",
				Description: "Installing test-plugin",
				Success:     true,
			},
		},
	}

	t.Run("short mode uses short formatter", func(t *testing.T) {
		opts := sync.Options{Short: true}
		output := captureStdout(t, func() {
			printSyncResultText(result, opts)
		})

		// Short format shows "✓ name (type action)"
		if !strings.Contains(output, "✓ test-plugin (plugin add)") {
			t.Errorf("short mode should use short formatter\nGot:\n%s", output)
		}

		// Short format should NOT show full command
		if strings.Contains(output, "→ claude plugin install") {
			t.Errorf("short mode should not show commands\nGot:\n%s", output)
		}
	})

	t.Run("non-short mode uses verbose formatter", func(t *testing.T) {
		opts := sync.Options{Short: false}
		output := captureStdout(t, func() {
			printSyncResultText(result, opts)
		})

		// Verbose format shows command
		if !strings.Contains(output, "→ claude plugin install test-plugin") {
			t.Errorf("verbose mode should show commands\nGot:\n%s", output)
		}

		// Verbose format shows description
		if !strings.Contains(output, "Add: Installing test-plugin") {
			t.Errorf("verbose mode should show description\nGot:\n%s", output)
		}
	})
}

func TestPrintSyncResultVerbose_OperationWithoutCommand(t *testing.T) {
	result := &sync.Result{
		Installed: 1,
		Updated:   0,
		Failed:    0,
		Operations: []sync.Operation{
			{
				Type:        "plugin",
				Name:        "test-plugin",
				Action:      "add",
				Command:     "", // Empty command
				Description: "Installing test-plugin",
				Success:     true,
			},
		},
	}

	output := captureStdout(t, func() {
		printSyncResultVerbose(result)
	})

	// Should still show description and success
	if !strings.Contains(output, "Add: Installing test-plugin") {
		t.Errorf("should show description even without command\nGot:\n%s", output)
	}
	if !strings.Contains(output, "✓ Success") {
		t.Errorf("should show success status\nGot:\n%s", output)
	}

	// Should NOT show arrow with empty command
	if strings.Contains(output, "→ \n") {
		t.Errorf("should not show empty command line\nGot:\n%s", output)
	}
}

func TestPrintSyncResultShort_EmptyResult(t *testing.T) {
	result := &sync.Result{
		Installed:  0,
		Updated:    0,
		Failed:     0,
		Operations: []sync.Operation{},
	}

	output := captureStdout(t, func() {
		printSyncResultShort(result)
	})

	// Should show summary even with no operations
	if !strings.Contains(output, "Summary:") {
		t.Errorf("should show summary\nGot:\n%s", output)
	}
	if !strings.Contains(output, "0 installed") {
		t.Errorf("should show 0 installed\nGot:\n%s", output)
	}
}

// testError is a simple error implementation for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
