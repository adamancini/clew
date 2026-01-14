// Package types provides type-safe constants for the clew configuration system.
//
// This package centralizes all enumerated types used throughout the codebase,
// replacing magic strings with typed constants that provide compile-time safety
// and validation methods.
//
// SYNC REQUIREMENT: These types must stay in sync with:
//   - schema/clewfile.schema.json (JSON Schema validation)
//   - internal/config/validate.go (runtime validation)
package types

import (
	"fmt"
	"strings"
)

// SourceType represents how a source is accessed (github or local).
type SourceType string

const (
	// SourceTypeGitHub indicates a GitHub repository source.
	SourceTypeGitHub SourceType = "github"
	// SourceTypeLocal indicates a local filesystem path source.
	SourceTypeLocal SourceType = "local"
)

// AllSourceTypes returns all valid source types.
func AllSourceTypes() []SourceType {
	return []SourceType{SourceTypeGitHub, SourceTypeLocal}
}

// Validate checks if the SourceType is a valid value.
func (s SourceType) Validate() error {
	switch s {
	case SourceTypeGitHub, SourceTypeLocal:
		return nil
	case "":
		return fmt.Errorf("source type is required")
	default:
		return fmt.Errorf("invalid source type '%s' (must be github or local)", s)
	}
}

// String returns the string representation of the SourceType.
func (s SourceType) String() string {
	return string(s)
}

// IsGitHub returns true if the source type is GitHub.
func (s SourceType) IsGitHub() bool {
	return s == SourceTypeGitHub
}

// IsLocal returns true if the source type is local.
func (s SourceType) IsLocal() bool {
	return s == SourceTypeLocal
}

// ParseSourceType parses a string into a SourceType.
// Returns an error if the string is not a valid source type.
func ParseSourceType(s string) (SourceType, error) {
	st := SourceType(strings.ToLower(s))
	if err := st.Validate(); err != nil {
		return "", err
	}
	return st, nil
}

// SourceKind represents the type of source (marketplace, plugin, or local).
type SourceKind string

const (
	// SourceKindMarketplace indicates a marketplace source.
	SourceKindMarketplace SourceKind = "marketplace"
	// SourceKindPlugin indicates a plugin repository source.
	SourceKindPlugin SourceKind = "plugin"
	// SourceKindLocal indicates a local plugin source.
	SourceKindLocal SourceKind = "local"
)

// AllSourceKinds returns all valid source kinds.
func AllSourceKinds() []SourceKind {
	return []SourceKind{SourceKindMarketplace, SourceKindPlugin, SourceKindLocal}
}

// Validate checks if the SourceKind is a valid value.
func (k SourceKind) Validate() error {
	switch k {
	case SourceKindMarketplace, SourceKindPlugin, SourceKindLocal:
		return nil
	case "":
		return fmt.Errorf("source kind is required")
	default:
		return fmt.Errorf("invalid source kind '%s' (must be marketplace, plugin, or local)", k)
	}
}

// String returns the string representation of the SourceKind.
func (k SourceKind) String() string {
	return string(k)
}

// IsMarketplace returns true if the source kind is marketplace.
func (k SourceKind) IsMarketplace() bool {
	return k == SourceKindMarketplace
}

// IsPlugin returns true if the source kind is plugin.
func (k SourceKind) IsPlugin() bool {
	return k == SourceKindPlugin
}

// IsLocal returns true if the source kind is local.
func (k SourceKind) IsLocal() bool {
	return k == SourceKindLocal
}

// ParseSourceKind parses a string into a SourceKind.
// Returns an error if the string is not a valid source kind.
func ParseSourceKind(s string) (SourceKind, error) {
	sk := SourceKind(strings.ToLower(s))
	if err := sk.Validate(); err != nil {
		return "", err
	}
	return sk, nil
}

// Scope represents the installation scope (user or project).
type Scope string

const (
	// ScopeUser indicates user-level scope (applies to all projects).
	ScopeUser Scope = "user"
	// ScopeProject indicates project-level scope (applies only to current project).
	ScopeProject Scope = "project"
)

// AllScopes returns all valid scopes.
func AllScopes() []Scope {
	return []Scope{ScopeUser, ScopeProject}
}

// Validate checks if the Scope is a valid value.
// Empty scope is considered valid (defaults to user scope typically).
func (s Scope) Validate() error {
	switch s {
	case ScopeUser, ScopeProject, "":
		return nil
	default:
		return fmt.Errorf("invalid scope '%s' (must be user or project)", s)
	}
}

// String returns the string representation of the Scope.
func (s Scope) String() string {
	return string(s)
}

// IsUser returns true if the scope is user.
func (s Scope) IsUser() bool {
	return s == ScopeUser || s == ""
}

// IsProject returns true if the scope is project.
func (s Scope) IsProject() bool {
	return s == ScopeProject
}

// Default returns the default scope if empty, otherwise returns the current scope.
func (s Scope) Default() Scope {
	if s == "" {
		return ScopeUser
	}
	return s
}

// ParseScope parses a string into a Scope.
// Returns an error if the string is not a valid scope.
func ParseScope(s string) (Scope, error) {
	scope := Scope(strings.ToLower(s))
	if err := scope.Validate(); err != nil {
		return "", err
	}
	return scope, nil
}

// TransportType represents the MCP server transport protocol.
type TransportType string

const (
	// TransportStdio indicates stdio transport (command-based).
	TransportStdio TransportType = "stdio"
	// TransportHTTP indicates HTTP transport.
	TransportHTTP TransportType = "http"
	// TransportSSE indicates Server-Sent Events transport.
	TransportSSE TransportType = "sse"
)

// AllTransportTypes returns all valid transport types.
func AllTransportTypes() []TransportType {
	return []TransportType{TransportStdio, TransportHTTP, TransportSSE}
}

// Validate checks if the TransportType is a valid value.
func (t TransportType) Validate() error {
	switch t {
	case TransportStdio, TransportHTTP, TransportSSE:
		return nil
	case "":
		return fmt.Errorf("transport type is required")
	default:
		return fmt.Errorf("invalid transport type '%s' (must be stdio, http, or sse)", t)
	}
}

// String returns the string representation of the TransportType.
func (t TransportType) String() string {
	return string(t)
}

// IsStdio returns true if the transport is stdio.
func (t TransportType) IsStdio() bool {
	return t == TransportStdio
}

// IsHTTP returns true if the transport is HTTP.
func (t TransportType) IsHTTP() bool {
	return t == TransportHTTP
}

// IsSSE returns true if the transport is SSE.
func (t TransportType) IsSSE() bool {
	return t == TransportSSE
}

// IsHTTPBased returns true if the transport is HTTP-based (HTTP or SSE).
func (t TransportType) IsHTTPBased() bool {
	return t == TransportHTTP || t == TransportSSE
}

// RequiresCommand returns true if the transport requires a command.
func (t TransportType) RequiresCommand() bool {
	return t == TransportStdio
}

// RequiresURL returns true if the transport requires a URL.
func (t TransportType) RequiresURL() bool {
	return t == TransportHTTP || t == TransportSSE
}

// ParseTransportType parses a string into a TransportType.
// Returns an error if the string is not a valid transport type.
func ParseTransportType(s string) (TransportType, error) {
	tt := TransportType(strings.ToLower(s))
	if err := tt.Validate(); err != nil {
		return "", err
	}
	return tt, nil
}
