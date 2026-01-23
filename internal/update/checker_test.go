package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	if checker.client == nil {
		t.Error("HTTP client should not be nil")
	}

	if checker.baseURL != "https://api.github.com" {
		t.Errorf("baseURL = %s, want https://api.github.com", checker.baseURL)
	}
}

func TestGitHubCheckerWithToken(t *testing.T) {
	checker := NewGitHubChecker("0.8.2", "adamancini", "clew").
		WithToken("ghp_test123")

	if checker.githubToken != "ghp_test123" {
		t.Errorf("githubToken = %s, want ghp_test123", checker.githubToken)
	}
}

func TestGitHubCheckerCheckForUpdate_UpdateAvailable(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Error("Expected Accept header")
		}

		// Return mock release
		release := GitHubRelease{
			TagName: "v0.9.0",
			Name:    "v0.9.0",
			Body:    "Release notes for 0.9.0",
			HTMLURL: "https://github.com/adamancini/clew/releases/tag/v0.9.0",
			Assets: []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			}{
				{
					Name:               "clew-darwin-arm64",
					BrowserDownloadURL: "https://github.com/.../clew-darwin-arm64",
				},
				{
					Name:               "checksums.txt",
					BrowserDownloadURL: "https://github.com/.../checksums.txt",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	// Create checker with mock server
	checker := NewGitHubChecker("0.8.2", "adamancini", "clew")
	checker.baseURL = server.URL

	info, err := checker.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}

	if !info.Available {
		t.Error("Update should be available")
	}

	if info.CurrentVersion != "0.8.2" {
		t.Errorf("CurrentVersion = %s, want 0.8.2", info.CurrentVersion)
	}

	if info.LatestVersion != "0.9.0" {
		t.Errorf("LatestVersion = %s, want 0.9.0", info.LatestVersion)
	}

	if info.ReleaseNotes != "Release notes for 0.9.0" {
		t.Errorf("ReleaseNotes = %s", info.ReleaseNotes)
	}

	if info.AssetURL != "https://github.com/.../clew-darwin-arm64" {
		t.Errorf("AssetURL = %s", info.AssetURL)
	}

	if info.ChecksumURL != "https://github.com/.../checksums.txt" {
		t.Errorf("ChecksumURL = %s", info.ChecksumURL)
	}
}

func TestGitHubCheckerCheckForUpdate_NoUpdateAvailable(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := GitHubRelease{
			TagName: "v0.8.2",
			Name:    "v0.8.2",
			Body:    "Current version",
			HTMLURL: "https://github.com/adamancini/clew/releases/tag/v0.8.2",
			Assets:  []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			}{},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	checker := NewGitHubChecker("0.8.2", "adamancini", "clew")
	checker.baseURL = server.URL

	info, err := checker.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}

	if info.Available {
		t.Error("Update should not be available (same version)")
	}

	if info.CurrentVersion != "0.8.2" {
		t.Errorf("CurrentVersion = %s, want 0.8.2", info.CurrentVersion)
	}

	if info.LatestVersion != "0.8.2" {
		t.Errorf("LatestVersion = %s, want 0.8.2", info.LatestVersion)
	}
}

func TestGitHubCheckerCheckForUpdate_OlderVersion(t *testing.T) {
	// Create mock server returning 0.7.0 (older than current 0.8.2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := GitHubRelease{
			TagName: "v0.7.0",
			Name:    "v0.7.0",
			Body:    "Older version",
			HTMLURL: "https://github.com/adamancini/clew/releases/tag/v0.7.0",
			Assets:  []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			}{},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	checker := NewGitHubChecker("0.8.2", "adamancini", "clew")
	checker.baseURL = server.URL

	info, err := checker.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}

	if info.Available {
		t.Error("Update should not be available (current version is newer)")
	}

	if info.CurrentVersion != "0.8.2" {
		t.Errorf("CurrentVersion = %s, want 0.8.2", info.CurrentVersion)
	}

	if info.LatestVersion != "0.7.0" {
		t.Errorf("LatestVersion = %s, want 0.7.0", info.LatestVersion)
	}
}

func TestGitHubCheckerCheckForUpdate_APIError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewGitHubChecker("0.8.2", "adamancini", "clew")
	checker.baseURL = server.URL

	_, err := checker.CheckForUpdate()
	if err == nil {
		t.Error("Expected error from API")
	}
}

func TestGitHubCheckerCheckForUpdate_WithToken(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer ghp_test123" {
			t.Errorf("Expected Authorization header, got %s", auth)
		}

		release := GitHubRelease{
			TagName: "v0.9.0",
			Name:    "v0.9.0",
			Body:    "Release",
			HTMLURL: "https://github.com/adamancini/clew/releases/tag/v0.9.0",
			Assets:  []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			}{},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	checker := NewGitHubChecker("0.8.2", "adamancini", "clew").
		WithToken("ghp_test123")
	checker.baseURL = server.URL

	_, err := checker.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}
}

func TestFindAssetURLs(t *testing.T) {
	checker := NewGitHubChecker("0.8.2", "adamancini", "clew")

	release := &GitHubRelease{
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{
				Name:               "clew-darwin-arm64",
				BrowserDownloadURL: "https://example.com/clew-darwin-arm64",
			},
			{
				Name:               "clew-linux-amd64",
				BrowserDownloadURL: "https://example.com/clew-linux-amd64",
			},
			{
				Name:               "checksums.txt",
				BrowserDownloadURL: "https://example.com/checksums.txt",
			},
		},
	}

	platform := Platform{OS: "darwin", Arch: "arm64"}
	assetURL, checksumURL := checker.findAssetURLs(release, platform)

	if assetURL != "https://example.com/clew-darwin-arm64" {
		t.Errorf("assetURL = %s", assetURL)
	}

	if checksumURL != "https://example.com/checksums.txt" {
		t.Errorf("checksumURL = %s", checksumURL)
	}
}

func TestFindAssetURLs_NotFound(t *testing.T) {
	checker := NewGitHubChecker("0.8.2", "adamancini", "clew")

	release := &GitHubRelease{
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{},
	}

	platform := Platform{OS: "darwin", Arch: "arm64"}
	assetURL, checksumURL := checker.findAssetURLs(release, platform)

	if assetURL != "" {
		t.Errorf("assetURL should be empty, got %s", assetURL)
	}

	if checksumURL != "" {
		t.Errorf("checksumURL should be empty, got %s", checksumURL)
	}
}
