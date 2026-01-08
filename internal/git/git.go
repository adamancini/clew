// Package git provides git repository status checking for local marketplaces and plugins.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Level represents the severity of a git status.
type Level string

const (
	LevelOK      Level = "ok"      // Clean and in sync
	LevelInfo    Level = "info"    // Ahead or behind remote
	LevelWarning Level = "warning" // Uncommitted changes
	LevelError   Level = "error"   // Git operation failed
)

// Status represents the git status of a repository.
type Status struct {
	Path           string // Absolute path to the repository
	IsGitRepo      bool   // Whether the path is a git repository
	IsClean        bool   // No uncommitted or unstaged changes
	HasUncommitted bool   // Has uncommitted changes (staged or unstaged)
	Ahead          int    // Number of commits ahead of remote
	Behind         int    // Number of commits behind remote
	CurrentBranch  string // Current branch name
	Remote         string // Remote tracking branch (e.g., "origin/main")
	Level          Level  // Overall severity level
	Message        string // Human-readable status message
	Error          error  // Non-fatal error if any
}

// CommandRunner is an interface for running external commands.
// This allows for mocking in tests.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
	RunInDir(dir, name string, args ...string) ([]byte, error)
}

// DefaultCommandRunner uses os/exec to run commands.
type DefaultCommandRunner struct{}

// Run executes a command in the current directory.
func (r *DefaultCommandRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// RunInDir executes a command in the specified directory.
func (r *DefaultCommandRunner) RunInDir(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// Checker checks git status for repositories.
type Checker struct {
	runner          CommandRunner
	skipPathCheck   bool // For testing: skip filesystem path existence check
}

// NewChecker creates a new Checker with the default command runner.
func NewChecker() *Checker {
	return &Checker{runner: &DefaultCommandRunner{}}
}

// NewCheckerWithRunner creates a Checker with a custom command runner (for testing).
func NewCheckerWithRunner(runner CommandRunner) *Checker {
	return &Checker{runner: runner}
}

// SetSkipPathCheck sets whether to skip filesystem path existence checks (for testing).
func (c *Checker) SetSkipPathCheck(skip bool) {
	c.skipPathCheck = skip
}

// CheckRepository checks the git status of a repository at the given path.
func (c *Checker) CheckRepository(path string) Status {
	return c.checkRepository(path, !c.skipPathCheck)
}

// checkRepositorySkipPathCheck checks the git status without verifying path exists (for testing).
func (c *Checker) checkRepositorySkipPathCheck(path string) Status {
	return c.checkRepository(path, false)
}

// checkRepository is the internal implementation.
func (c *Checker) checkRepository(path string, checkPathExists bool) Status {
	status := Status{Path: path}

	// Expand ~ to home directory
	expandedPath := expandPath(path)
	status.Path = expandedPath

	// Check if path exists (unless disabled for testing)
	if checkPathExists {
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			status.Level = LevelError
			status.Error = fmt.Errorf("path does not exist: %s", path)
			status.Message = fmt.Sprintf("path does not exist: %s", path)
			return status
		}
	}

	// Check if it's a git repository
	if !c.isGitRepo(expandedPath) {
		status.IsGitRepo = false
		status.Level = LevelInfo
		status.Message = "not a git repository"
		return status
	}
	status.IsGitRepo = true

	// Get current branch
	branch, err := c.getCurrentBranch(expandedPath)
	if err != nil {
		status.Level = LevelError
		status.Error = err
		status.Message = fmt.Sprintf("failed to get current branch: %v", err)
		return status
	}
	status.CurrentBranch = branch

	// Check for uncommitted changes using porcelain format
	hasChanges, err := c.hasUncommittedChanges(expandedPath)
	if err != nil {
		status.Level = LevelError
		status.Error = err
		status.Message = fmt.Sprintf("failed to check working tree: %v", err)
		return status
	}
	status.HasUncommitted = hasChanges
	status.IsClean = !hasChanges

	// If there are uncommitted changes, that's the highest priority warning
	if hasChanges {
		status.Level = LevelWarning
		status.Message = "uncommitted changes detected"
		return status
	}

	// Get remote tracking branch
	remote, err := c.getRemoteTrackingBranch(expandedPath)
	if err != nil {
		// No remote tracking branch is not an error, just info
		status.Level = LevelOK
		status.Message = "clean (no remote tracking branch)"
		return status
	}
	status.Remote = remote

	// Fetch from remote (best effort, continue if fails)
	_ = c.fetch(expandedPath)

	// Check ahead/behind
	ahead, behind, err := c.getAheadBehind(expandedPath, remote)
	if err != nil {
		// Couldn't check ahead/behind, but repo is clean
		status.Level = LevelOK
		status.Message = "clean"
		return status
	}
	status.Ahead = ahead
	status.Behind = behind

	// Determine level based on ahead/behind
	if behind > 0 && ahead > 0 {
		status.Level = LevelInfo
		status.Message = fmt.Sprintf("%d commits ahead, %d commits behind remote (consider: git pull --rebase && git push)", ahead, behind)
	} else if behind > 0 {
		status.Level = LevelInfo
		status.Message = fmt.Sprintf("%d commits behind remote (consider: git pull)", behind)
	} else if ahead > 0 {
		status.Level = LevelInfo
		status.Message = fmt.Sprintf("%d commits ahead of remote (consider: git push)", ahead)
	} else {
		status.Level = LevelOK
		status.Message = "clean and in sync"
	}

	return status
}

// isGitRepo checks if the path is a git repository.
func (c *Checker) isGitRepo(path string) bool {
	output, err := c.runner.RunInDir(path, "git", "rev-parse", "--git-dir")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// getCurrentBranch returns the current branch name.
func (c *Checker) getCurrentBranch(path string) (string, error) {
	output, err := c.runner.RunInDir(path, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// hasUncommittedChanges checks for uncommitted or unstaged changes.
func (c *Checker) hasUncommittedChanges(path string) (bool, error) {
	output, err := c.runner.RunInDir(path, "git", "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	// Any output means there are changes
	return strings.TrimSpace(string(output)) != "", nil
}

// getRemoteTrackingBranch returns the remote tracking branch (e.g., "origin/main").
func (c *Checker) getRemoteTrackingBranch(path string) (string, error) {
	output, err := c.runner.RunInDir(path, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return "", fmt.Errorf("no remote tracking branch")
	}
	return strings.TrimSpace(string(output)), nil
}

// fetch fetches from the remote (best effort).
func (c *Checker) fetch(path string) error {
	_, err := c.runner.RunInDir(path, "git", "fetch", "--quiet")
	return err
}

// getAheadBehind returns the number of commits ahead and behind the remote.
func (c *Checker) getAheadBehind(path, remote string) (ahead, behind int, err error) {
	output, err := c.runner.RunInDir(path, "git", "rev-list", "--left-right", "--count", "HEAD..."+remote)
	if err != nil {
		return 0, 0, fmt.Errorf("git rev-list failed: %w", err)
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected output format: %s", output)
	}

	ahead, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid ahead count: %w", err)
	}

	behind, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid behind count: %w", err)
	}

	return ahead, behind, nil
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// GitAvailable checks if git is available on the system.
func (c *Checker) GitAvailable() bool {
	_, err := c.runner.Run("git", "--version")
	return err == nil
}
