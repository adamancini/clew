package update

import "testing"

func TestNewHTTPDownloader(t *testing.T) {
	downloader := NewHTTPDownloader()

	if downloader.client == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestHTTPDownloaderDownload(t *testing.T) {
	downloader := NewHTTPDownloader()

	// Currently returns not implemented
	err := downloader.Download("https://example.com/binary", "/tmp/binary")
	if err == nil {
		t.Error("Expected not implemented error")
	}
}

func TestHTTPDownloaderVerifyChecksum(t *testing.T) {
	downloader := NewHTTPDownloader()

	// Currently returns not implemented
	err := downloader.VerifyChecksum("/tmp/binary", "abc123")
	if err == nil {
		t.Error("Expected not implemented error")
	}
}
