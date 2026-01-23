package update

import (
	"fmt"
	"net/http"
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
// Implementation coming in next phase
func (d *HTTPDownloader) Download(url string, dst string) error {
	// TODO: Implement download with progress reporting
	return fmt.Errorf("not implemented yet")
}

// VerifyChecksum verifies the downloaded file's checksum
// Implementation coming in next phase
func (d *HTTPDownloader) VerifyChecksum(file, checksum string) error {
	// TODO: Implement SHA256 checksum verification
	return fmt.Errorf("not implemented yet")
}
