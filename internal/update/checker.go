package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GitHubChecker checks for updates via GitHub API
type GitHubChecker struct {
	currentVersion string
	githubToken    string      // Optional, for rate limiting
	owner          string      // Repository owner
	repo           string      // Repository name
	client         *http.Client
	baseURL        string      // Base URL for GitHub API (for testing)
}

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// NewGitHubChecker creates a new GitHub checker
func NewGitHubChecker(currentVersion, owner, repo string) *GitHubChecker {
	return &GitHubChecker{
		currentVersion: currentVersion,
		owner:          owner,
		repo:           repo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.github.com",
	}
}

// WithToken sets an optional GitHub token for authentication
func (c *GitHubChecker) WithToken(token string) *GitHubChecker {
	c.githubToken = token
	return c
}

// CheckForUpdate checks if an update is available
func (c *GitHubChecker) CheckForUpdate() (*UpdateInfo, error) {
	// Get latest release from GitHub
	release, err := c.getLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	// Parse versions
	currentVer, err := ParseVersion(c.currentVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid current version: %w", err)
	}

	latestVer, err := ParseVersion(release.TagName)
	if err != nil {
		return nil, fmt.Errorf("invalid latest version: %w", err)
	}

	// Determine if update is available
	available := latestVer.IsGreaterThan(currentVer)

	// Get asset URLs for current platform
	platform := Detect()
	assetURL, checksumURL := c.findAssetURLs(release, platform)

	info := &UpdateInfo{
		Available:      available,
		CurrentVersion: NormalizeVersion(c.currentVersion),
		LatestVersion:  NormalizeVersion(release.TagName),
		ReleaseURL:     release.HTMLURL,
		ReleaseNotes:   release.Body,
		AssetURL:       assetURL,
		ChecksumURL:    checksumURL,
	}

	return info, nil
}

// getLatestRelease fetches the latest release from GitHub API
func (c *GitHubChecker) getLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, c.owner, c.repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.githubToken)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// findAssetURLs finds the binary and checksum URLs for the current platform
func (c *GitHubChecker) findAssetURLs(release *GitHubRelease, platform Platform) (string, string) {
	binaryName := platform.BinaryName()
	var assetURL, checksumURL string

	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			assetURL = asset.BrowserDownloadURL
		}
		if asset.Name == "checksums.txt" {
			checksumURL = asset.BrowserDownloadURL
		}
	}

	return assetURL, checksumURL
}
