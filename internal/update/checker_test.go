package update

import "testing"

func TestNewGitHubChecker(t *testing.T) {
	checker := NewGitHubChecker("0.8.2", "adamancini", "clew")

	if checker.currentVersion != "0.8.2" {
		t.Errorf("currentVersion = %s, want 0.8.2", checker.currentVersion)
	}

	if checker.owner != "adamancini" {
		t.Errorf("owner = %s, want adamancini", checker.owner)
	}

	if checker.repo != "clew" {
		t.Errorf("repo = %s, want clew", checker.repo)
	}
}

func TestGitHubCheckerWithToken(t *testing.T) {
	checker := NewGitHubChecker("0.8.2", "adamancini", "clew").
		WithToken("ghp_test123")

	if checker.githubToken != "ghp_test123" {
		t.Errorf("githubToken = %s, want ghp_test123", checker.githubToken)
	}
}

func TestGitHubCheckerCheckForUpdate(t *testing.T) {
	checker := NewGitHubChecker("0.8.2", "adamancini", "clew")

	// Currently returns not implemented
	_, err := checker.CheckForUpdate()
	if err == nil {
		t.Error("Expected not implemented error")
	}
}
