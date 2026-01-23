package update

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.-]+))?$`)

// Version represents a semantic version
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

// ParseVersion parses a semantic version string
// Supports formats like "0.8.2", "v0.8.2", "0.9.0-rc.1"
func ParseVersion(s string) (*Version, error) {
	matches := versionRegex.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("invalid version format: %s", s)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	prerelease := matches[4]

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}, nil
}

// String returns the string representation
func (v *Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	return s
}

// Compare compares two versions
// Returns:
//   - 1 if v > other
//   - 0 if v == other
//   - -1 if v < other
func (v *Version) Compare(other *Version) int {
	// Compare major version
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}

	// Compare minor version
	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}

	// Compare patch version
	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}

	// Compare prerelease
	// Stable versions (no prerelease) are greater than prereleases
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}

	// Both have prereleases, compare lexicographically
	if v.Prerelease != other.Prerelease {
		if v.Prerelease > other.Prerelease {
			return 1
		}
		return -1
	}

	return 0
}

// IsGreaterThan returns true if v > other
func (v *Version) IsGreaterThan(other *Version) bool {
	return v.Compare(other) > 0
}

// IsLessThan returns true if v < other
func (v *Version) IsLessThan(other *Version) bool {
	return v.Compare(other) < 0
}

// IsEqual returns true if v == other
func (v *Version) IsEqual(other *Version) bool {
	return v.Compare(other) == 0
}

// CompareVersions compares two version strings
// Returns:
//   - 1 if v1 > v2
//   - 0 if v1 == v2
//   - -1 if v1 < v2
//   - error if either version is invalid
func CompareVersions(v1, v2 string) (int, error) {
	ver1, err := ParseVersion(v1)
	if err != nil {
		return 0, fmt.Errorf("invalid version v1: %w", err)
	}

	ver2, err := ParseVersion(v2)
	if err != nil {
		return 0, fmt.Errorf("invalid version v2: %w", err)
	}

	return ver1.Compare(ver2), nil
}

// NormalizeVersion removes the 'v' prefix if present
func NormalizeVersion(s string) string {
	return strings.TrimPrefix(s, "v")
}
