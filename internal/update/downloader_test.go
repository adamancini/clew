package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewHTTPDownloader(t *testing.T) {
	downloader := NewHTTPDownloader()

	if downloader.client == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestHTTPDownloaderDownload_Success(t *testing.T) {
	// Create test content
	testContent := []byte("test binary content")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testContent)
	}))
	defer server.Close()

	// Create temporary directory
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, "test-binary")

	// Download
	downloader := NewHTTPDownloader()
	err := downloader.Download(server.URL, dstPath)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("Downloaded file does not exist")
	}

	// Verify content
	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Content mismatch: got %s, want %s", content, testContent)
	}
}

func TestHTTPDownloaderDownload_HTTPError(t *testing.T) {
	// Create mock server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, "test-binary")

	downloader := NewHTTPDownloader()
	err := downloader.Download(server.URL, dstPath)
	if err == nil {
		t.Error("Expected error for 404 response")
	}

	// Verify file was not created
	if _, err := os.Stat(dstPath); !os.IsNotExist(err) {
		t.Error("File should not exist after failed download")
	}
}

func TestHTTPDownloaderDownload_NetworkError(t *testing.T) {
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, "test-binary")

	downloader := NewHTTPDownloader()
	// Use invalid URL to trigger network error
	err := downloader.Download("http://invalid-url-that-does-not-exist.local", dstPath)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestHTTPDownloaderDownload_InvalidDestination(t *testing.T) {
	testContent := []byte("test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(testContent)
	}))
	defer server.Close()

	// Try to write to invalid path
	downloader := NewHTTPDownloader()
	err := downloader.Download(server.URL, "/invalid/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for invalid destination path")
	}
}

func TestCalculateSHA256(t *testing.T) {
	// Create temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("hello world")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate checksum
	checksum, err := calculateSHA256(testFile)
	if err != nil {
		t.Fatalf("calculateSHA256() error = %v", err)
	}

	// Calculate expected checksum
	hash := sha256.New()
	hash.Write(testContent)
	expected := hex.EncodeToString(hash.Sum(nil))

	if checksum != expected {
		t.Errorf("Checksum mismatch: got %s, want %s", checksum, expected)
	}
}

func TestCalculateSHA256_FileNotFound(t *testing.T) {
	_, err := calculateSHA256("/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestDownloadChecksums_Success(t *testing.T) {
	// Create mock checksums file
	checksums := `abc123  file1.bin
def456  file2.bin
789ghi  file3.bin`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(checksums))
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	result, err := downloader.downloadChecksums(server.URL)
	if err != nil {
		t.Fatalf("downloadChecksums() error = %v", err)
	}

	expected := map[string]string{
		"file1.bin": "abc123",
		"file2.bin": "def456",
		"file3.bin": "789ghi",
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d checksums, got %d", len(expected), len(result))
	}

	for filename, checksum := range expected {
		if result[filename] != checksum {
			t.Errorf("Checksum mismatch for %s: got %s, want %s", filename, result[filename], checksum)
		}
	}
}

func TestDownloadChecksums_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	_, err := downloader.downloadChecksums(server.URL)
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}

func TestDownloadChecksums_MalformedLines(t *testing.T) {
	// Checksums file with some malformed lines
	checksums := `abc123  file1.bin
malformed
def456  file2.bin
onlyonefield
789ghi  file3.bin`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksums))
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	result, err := downloader.downloadChecksums(server.URL)
	if err != nil {
		t.Fatalf("downloadChecksums() error = %v", err)
	}

	// Should only parse valid lines
	if len(result) != 3 {
		t.Errorf("Expected 3 valid checksums, got %d", len(result))
	}
}

func TestVerifyChecksum_Success(t *testing.T) {
	// Create test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "clew-darwin-arm64")
	testContent := []byte("binary content")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected checksum
	hash := sha256.New()
	hash.Write(testContent)
	expectedChecksum := hex.EncodeToString(hash.Sum(nil))

	// Create checksums file
	checksumsContent := fmt.Sprintf("%s  clew-darwin-arm64\n", expectedChecksum)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksumsContent))
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	err := downloader.VerifyChecksum(testFile, server.URL)
	if err != nil {
		t.Errorf("VerifyChecksum() error = %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	// Create test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "clew-darwin-arm64")
	testContent := []byte("binary content")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Use wrong checksum
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	checksumsContent := fmt.Sprintf("%s  clew-darwin-arm64\n", wrongChecksum)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksumsContent))
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	err := downloader.VerifyChecksum(testFile, server.URL)
	if err == nil {
		t.Error("Expected error for checksum mismatch")
	}

	if !contains(err.Error(), "mismatch") {
		t.Errorf("Error should mention mismatch, got: %v", err)
	}
}

func TestVerifyChecksum_FileNotInChecksums(t *testing.T) {
	// Create test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "clew-darwin-arm64")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Checksums file without our file
	checksumsContent := "abc123  other-file.bin\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksumsContent))
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	err := downloader.VerifyChecksum(testFile, server.URL)
	if err == nil {
		t.Error("Expected error for file not in checksums")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Error should mention not found, got: %v", err)
	}
}

func TestVerifyChecksum_ChecksumDownloadFails(t *testing.T) {
	// Create test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.bin")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	err := downloader.VerifyChecksum(testFile, server.URL)
	if err == nil {
		t.Error("Expected error when checksums download fails")
	}
}

func TestGetFilename(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		want  string
	}{
		{
			name: "unix path",
			path: "/tmp/test/file.bin",
			want: "file.bin",
		},
		{
			name: "simple filename",
			path: "file.bin",
			want: "file.bin",
		},
		{
			name: "nested path",
			path: "a/b/c/d/file.bin",
			want: "file.bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFilename(tt.path)
			if got != tt.want {
				t.Errorf("getFilename(%s) = %s, want %s", tt.path, got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
