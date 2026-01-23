package update

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// HTTPDownloader downloads binaries over HTTP
type HTTPDownloader struct {
	client *http.Client
}

// NewHTTPDownloader creates a new HTTP downloader
func NewHTTPDownloader() *HTTPDownloader {
	return &HTTPDownloader{
		client: &http.Client{},
	}
}

// Download downloads a file from url to dst
func (d *HTTPDownloader) Download(url string, dst string) error {
	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create the destination file
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Copy the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		// Clean up partial download
		_ = os.Remove(dst)
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// VerifyChecksum verifies the downloaded file's checksum against a checksums file
func (d *HTTPDownloader) VerifyChecksum(file, checksumURL string) error {
	// Calculate the file's SHA256 checksum
	actualChecksum, err := calculateSHA256(file)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Download the checksums file
	checksums, err := d.downloadChecksums(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	// Find the expected checksum for this file
	filename := getFilename(file)
	expectedChecksum, found := checksums[filename]
	if !found {
		return fmt.Errorf("checksum not found for %s", filename)
	}

	// Compare checksums
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// calculateSHA256 calculates the SHA256 checksum of a file
func calculateSHA256(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// downloadChecksums downloads and parses a checksums.txt file
// Expected format: <sha256>  <filename>
func (d *HTTPDownloader) downloadChecksums(url string) (map[string]string, error) {
	resp, err := d.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checksums download failed with status %d", resp.StatusCode)
	}

	// Parse the checksums file
	checksums := make(map[string]string)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue // Skip malformed lines
		}
		checksum := parts[0]
		filename := parts[1]
		checksums[filename] = checksum
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse checksums: %w", err)
	}

	return checksums, nil
}

// getFilename extracts the filename from a filepath
func getFilename(path string) string {
	// Simple basename extraction
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}
