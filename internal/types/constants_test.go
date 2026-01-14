package types

import (
	"testing"
)

func TestSourceTypeValidate(t *testing.T) {
	tests := []struct {
		name    string
		st      SourceType
		wantErr bool
	}{
		{"github valid", SourceTypeGitHub, false},
		{"empty invalid", "", true},
		{"invalid value", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.st.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SourceType.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSourceTypeString(t *testing.T) {
	tests := []struct {
		st   SourceType
		want string
	}{
		{SourceTypeGitHub, "github"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.st.String(); got != tt.want {
				t.Errorf("SourceType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSourceTypeHelpers(t *testing.T) {
	if !SourceTypeGitHub.IsGitHub() {
		t.Error("github.IsGitHub() should be true")
	}
}

func TestParseSourceType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SourceType
		wantErr bool
	}{
		{"github lowercase", "github", SourceTypeGitHub, false},
		{"github uppercase", "GITHUB", SourceTypeGitHub, false},
		{"empty", "", "", true},
		{"invalid", "unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSourceType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSourceType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSourceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllSourceTypes(t *testing.T) {
	types := AllSourceTypes()
	if len(types) != 1 {
		t.Errorf("AllSourceTypes() returned %d types, want 1", len(types))
	}
}

func TestSourceKindValidate(t *testing.T) {
	tests := []struct {
		name    string
		sk      SourceKind
		wantErr bool
	}{
		{"marketplace valid", SourceKindMarketplace, false},
		{"plugin valid", SourceKindPlugin, false},
		{"empty invalid", "", true},
		{"invalid value", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sk.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SourceKind.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSourceKindString(t *testing.T) {
	tests := []struct {
		sk   SourceKind
		want string
	}{
		{SourceKindMarketplace, "marketplace"},
		{SourceKindPlugin, "plugin"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.sk.String(); got != tt.want {
				t.Errorf("SourceKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSourceKindHelpers(t *testing.T) {
	if !SourceKindMarketplace.IsMarketplace() {
		t.Error("marketplace.IsMarketplace() should be true")
	}
	if !SourceKindPlugin.IsPlugin() {
		t.Error("plugin.IsPlugin() should be true")
	}
}

func TestParseSourceKind(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SourceKind
		wantErr bool
	}{
		{"marketplace lowercase", "marketplace", SourceKindMarketplace, false},
		{"marketplace uppercase", "MARKETPLACE", SourceKindMarketplace, false},
		{"plugin lowercase", "plugin", SourceKindPlugin, false},
		{"plugin uppercase", "PLUGIN", SourceKindPlugin, false},
		{"empty", "", "", true},
		{"invalid", "unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSourceKind(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSourceKind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSourceKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllSourceKinds(t *testing.T) {
	kinds := AllSourceKinds()
	if len(kinds) != 2 {
		t.Errorf("AllSourceKinds() returned %d kinds, want 2", len(kinds))
	}
}

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

func TestTransportTypeValidate(t *testing.T) {
	tests := []struct {
		name    string
		tt      TransportType
		wantErr bool
	}{
		{"stdio valid", TransportStdio, false},
		{"http valid", TransportHTTP, false},
		{"sse valid", TransportSSE, false},
		{"empty invalid", "", true},
		{"invalid value", "websocket", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.tt.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("TransportType.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestTransportTypeString(t *testing.T) {
	tests := []struct {
		tt   TransportType
		want string
	}{
		{TransportStdio, "stdio"},
		{TransportHTTP, "http"},
		{TransportSSE, "sse"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.tt.String(); got != tc.want {
				t.Errorf("TransportType.String() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTransportTypeHelpers(t *testing.T) {
	if !TransportStdio.IsStdio() {
		t.Error("stdio.IsStdio() should be true")
	}
	if TransportStdio.IsHTTP() {
		t.Error("stdio.IsHTTP() should be false")
	}
	if !TransportHTTP.IsHTTP() {
		t.Error("http.IsHTTP() should be true")
	}
	if !TransportSSE.IsSSE() {
		t.Error("sse.IsSSE() should be true")
	}
	if !TransportHTTP.IsHTTPBased() {
		t.Error("http.IsHTTPBased() should be true")
	}
	if !TransportSSE.IsHTTPBased() {
		t.Error("sse.IsHTTPBased() should be true")
	}
	if TransportStdio.IsHTTPBased() {
		t.Error("stdio.IsHTTPBased() should be false")
	}
}

func TestTransportTypeRequirements(t *testing.T) {
	if !TransportStdio.RequiresCommand() {
		t.Error("stdio.RequiresCommand() should be true")
	}
	if TransportHTTP.RequiresCommand() {
		t.Error("http.RequiresCommand() should be false")
	}
	if !TransportHTTP.RequiresURL() {
		t.Error("http.RequiresURL() should be true")
	}
	if !TransportSSE.RequiresURL() {
		t.Error("sse.RequiresURL() should be true")
	}
	if TransportStdio.RequiresURL() {
		t.Error("stdio.RequiresURL() should be false")
	}
}

func TestParseTransportType(t *testing.T) {
	tests := []struct {
		input   string
		want    TransportType
		wantErr bool
	}{
		{"stdio", TransportStdio, false},
		{"STDIO", TransportStdio, false},
		{"http", TransportHTTP, false},
		{"HTTP", TransportHTTP, false},
		{"sse", TransportSSE, false},
		{"SSE", TransportSSE, false},
		{"", "", true},
		{"websocket", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseTransportType(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseTransportType() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("ParseTransportType() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAllTransportTypes(t *testing.T) {
	types := AllTransportTypes()
	if len(types) != 3 {
		t.Errorf("AllTransportTypes() returned %d types, want 3", len(types))
	}
}
