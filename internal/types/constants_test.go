package types

import (
	"strings"
	"testing"
)

func TestScopeValidate(t *testing.T) {
	tests := []struct {
		name        string
		s           Scope
		wantErr     bool
		errContains string
	}{
		{"user valid", ScopeUser, false, ""},
		{"project rejected", Scope("project"), true, "clew 1.0 only supports user scope"},
		{"empty valid (defaults allowed)", "", false, ""},
		{"invalid value", "global", true, "clew 1.0 only supports user scope"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Scope.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Scope.Validate() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestScopeString(t *testing.T) {
	tests := []struct {
		s    Scope
		want string
	}{
		{ScopeUser, "user"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("Scope.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScopeHelpers(t *testing.T) {
	if !ScopeUser.IsUser() {
		t.Error("user.IsUser() should be true")
	}
	// Empty scope defaults to user
	var empty Scope
	if !empty.IsUser() {
		t.Error("empty.IsUser() should be true (defaults to user)")
	}
}

func TestScopeDefault(t *testing.T) {
	tests := []struct {
		s    Scope
		want Scope
	}{
		{ScopeUser, ScopeUser},
		{"", ScopeUser},
	}

	for _, tt := range tests {
		t.Run(string(tt.s), func(t *testing.T) {
			if got := tt.s.Default(); got != tt.want {
				t.Errorf("Scope.Default() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseScope(t *testing.T) {
	tests := []struct {
		input   string
		want    Scope
		wantErr bool
	}{
		{"user", ScopeUser, false},
		{"USER", ScopeUser, false},
		{"project", "", true},
		{"PROJECT", "", true},
		{"", "", false}, // Empty is valid
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseScope(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseScope() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseScopeProjectErrorMessage(t *testing.T) {
	_, err := ParseScope("project")
	if err == nil {
		t.Fatal("ParseScope(\"project\") should return error")
	}
	if !strings.Contains(err.Error(), "clew 1.0 only supports user scope") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "clew 1.0 only supports user scope")
	}
}

func TestAllScopes(t *testing.T) {
	scopes := AllScopes()
	if len(scopes) != 1 {
		t.Errorf("AllScopes() returned %d scopes, want 1", len(scopes))
	}
	if scopes[0] != ScopeUser {
		t.Errorf("AllScopes()[0] = %v, want %v", scopes[0], ScopeUser)
	}
}
