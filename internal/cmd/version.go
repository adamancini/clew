package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/adamancini/clew/internal/update"
)

var (
	checkOnly bool
	doUpdate  bool
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information and check for updates",
		Long: `Display the current clew version and optionally check for or install updates.

Examples:
  clew version              # Show current version
  clew version --check      # Check if update is available
  clew version --update     # Download and install latest version`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion()
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for updates without installing")
	cmd.Flags().BoolVar(&doUpdate, "update", false, "Update to the latest version")

	return cmd
}

func runVersion() error {
	// If no flags, just show version
	if !checkOnly && !doUpdate {
		fmt.Printf("clew version %s\n", clewVersion)
		return nil
	}

	// Check for updates
	checker := update.NewGitHubChecker(clewVersion, "adamancini", "clew")

	// Use GITHUB_TOKEN if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		checker = checker.WithToken(token)
	}

	info, err := checker.CheckForUpdate()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Display current version
	fmt.Printf("Current version: %s\n", info.CurrentVersion)

	// Check if update is available
	if !info.Available {
		fmt.Println("Already running latest version")
		return nil
	}

	// Update is available
	fmt.Printf("Latest version: %s available\n", info.LatestVersion)

	if checkOnly {
		// Just checking, don't install
		fmt.Println("\nRelease notes:")
		fmt.Println(info.ReleaseNotes)
		fmt.Printf("\nRun 'clew version --update' to install\n")
		return nil
	}

	if !doUpdate {
		// Neither check nor update flag set, shouldn't happen
		return nil
	}

	// Perform the update
	return performUpdate(info)
}

func performUpdate(info *update.UpdateInfo) error {
	fmt.Println("\nDownloading update...")

	// Detect platform
	platform := update.Detect()
	if !platform.IsSupported() {
		return fmt.Errorf("unsupported platform: %s/%s", platform.OS, platform.Arch)
	}

	// Check if we have asset URLs
	if info.AssetURL == "" {
		return fmt.Errorf("no binary available for %s/%s", platform.OS, platform.Arch)
	}

	if info.ChecksumURL == "" {
		return fmt.Errorf("no checksums available for verification")
	}

	// Create temporary directory for download
	tmpDir, err := os.MkdirTemp("", "clew-update-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download binary
	downloader := update.NewHTTPDownloader()
	tmpBinary := filepath.Join(tmpDir, platform.BinaryName())

	fmt.Printf("Downloading %s...\n", platform.BinaryName())
	if err := downloader.Download(info.AssetURL, tmpBinary); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	fmt.Println("✓ Downloaded")

	// Verify checksum
	fmt.Println("Verifying checksum...")
	if err := downloader.VerifyChecksum(tmpBinary, info.ChecksumURL); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	fmt.Println("✓ Checksum verified")

	// Get current binary path
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current binary path: %w", err)
	}

	// Resolve symlinks if any
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}

	// Create backup and replace
	fmt.Printf("Installing to %s...\n", currentBinary)
	replacer := update.NewBinaryReplacer(currentBinary)

	if err := replacer.Replace(tmpBinary); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Println("✓ Installation complete")
	fmt.Printf("\nSuccessfully updated to v%s!\n", info.LatestVersion)

	return nil
}
