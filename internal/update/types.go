package update

// UpdateInfo describes an available update
type UpdateInfo struct {
	Available      bool   // Whether an update is available
	CurrentVersion string // Currently installed version
	LatestVersion  string // Latest available version
	ReleaseURL     string // URL to the release page
	ReleaseNotes   string // Release notes/changelog
	AssetURL       string // Direct download URL for the binary
	ChecksumURL    string // URL to checksums file
}

// Platform describes the current system platform
type Platform struct {
	OS   string // Operating system (darwin, linux)
	Arch string // Architecture (amd64, arm64)
}

// Checker checks for available updates
type Checker interface {
	CheckForUpdate() (*UpdateInfo, error)
}

// Downloader downloads and verifies binaries
type Downloader interface {
	Download(url string, dst string) error
	VerifyChecksum(file, checksum string) error
}

// Replacer safely replaces the binary with rollback support
type Replacer interface {
	Replace(newBinary string) error
	Rollback() error
}
