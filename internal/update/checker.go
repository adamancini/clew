package update

import "fmt"

// GitHubChecker checks for updates via GitHub API
type GitHubChecker struct {
	currentVersion string
	githubToken    string // Optional, for rate limiting
	owner          string // Repository owner
	repo           string // Repository name
}

// NewGitHubChecker creates a new GitHub checker
func NewGitHubChecker(currentVersion, owner, repo string) *GitHubChecker {
	return &GitHubChecker{
		currentVersion: currentVersion,
		owner:          owner,
		repo:           repo,
	}
}

// WithToken sets an optional GitHub token for authentication
func (c *GitHubChecker) WithToken(token string) *GitHubChecker {
	c.githubToken = token
	return c
}

// CheckForUpdate checks if an update is available
// Implementation coming in next phase
func (c *GitHubChecker) CheckForUpdate() (*UpdateInfo, error) {
	// TODO: Implement GitHub API query
	return nil, fmt.Errorf("not implemented yet")
}
