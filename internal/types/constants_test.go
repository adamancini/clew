package types

import (
	"testing"
)

func TestScopeValidate(t *testing.T) {
	tests := []struct {
		name    string
		s       Scope
		wantErr bool
	}{
		{"user valid", ScopeUser, false},
		{"project valid", ScopeProject, false},
		{"empty valid (defaults allowed)", "", false},
		{"invalid value", "global", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Scope.Validate() error = %v, wantErr %v", err, tt.wantErr)
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
		{ScopeProject, "project"},
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
	if ScopeUser.IsProject() {
		t.Error("user.IsProject() should be false")
	}
	if !ScopeProject.IsProject() {
		t.Error("project.IsProject() should be true")
	}
	if ScopeProject.IsUser() {
		t.Error("project.IsUser() should be false")
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
		{ScopeProject, ScopeProject},
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
		{"project", ScopeProject, false},
		{"PROJECT", ScopeProject, false},
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

func TestAllScopes(t *testing.T) {
	scopes := AllScopes()
	if len(scopes) != 2 {
		t.Errorf("AllScopes() returned %d scopes, want 2", len(scopes))
	}
}
