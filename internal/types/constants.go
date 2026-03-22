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

// Scope represents the installation scope (user or project).
type Scope string

const (
	// ScopeUser indicates user-level scope (applies to all projects).
	ScopeUser Scope = "user"
)

// AllScopes returns all valid scopes.
func AllScopes() []Scope {
	return []Scope{ScopeUser}
}

// Validate checks if the Scope is a valid value.
// Empty scope is considered valid (defaults to user scope).
func (s Scope) Validate() error {
	switch s {
	case ScopeUser, "":
		return nil
	default:
		return fmt.Errorf("invalid scope '%s': clew 1.0 only supports user scope", s)
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
