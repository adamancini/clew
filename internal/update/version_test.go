package update

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "0.8.2",
			want:  &Version{Major: 0, Minor: 8, Patch: 2},
		},
		{
			name:  "version with v prefix",
			input: "v0.8.2",
			want:  &Version{Major: 0, Minor: 8, Patch: 2},
		},
		{
			name:  "version with prerelease",
			input: "1.0.0-rc.1",
			want:  &Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "rc.1"},
		},
		{
			name:  "version with alpha",
			input: "v2.0.0-alpha",
			want:  &Version{Major: 2, Minor: 0, Patch: 0, Prerelease: "alpha"},
		},
		{
			name:  "version with beta",
			input: "0.9.0-beta.2",
			want:  &Version{Major: 0, Minor: 9, Patch: 0, Prerelease: "beta.2"},
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "missing patch",
			input:   "1.0",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor ||
				got.Patch != tt.want.Patch || got.Prerelease != tt.want.Prerelease {
				t.Errorf("ParseVersion() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		name    string
		version *Version
		want    string
	}{
		{
			name:    "simple version",
			version: &Version{Major: 0, Minor: 8, Patch: 2},
			want:    "0.8.2",
		},
		{
			name:    "version with prerelease",
			version: &Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "rc.1"},
			want:    "1.0.0-rc.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name  string
		v1    string
		v2    string
		want  int // 1 if v1 > v2, 0 if equal, -1 if v1 < v2
	}{
		// Equal versions
		{name: "equal versions", v1: "0.8.2", v2: "0.8.2", want: 0},
		{name: "equal with v prefix", v1: "v0.8.2", v2: "0.8.2", want: 0},
		{name: "equal prereleases", v1: "1.0.0-rc.1", v2: "1.0.0-rc.1", want: 0},

		// Major version differences
		{name: "major version greater", v1: "2.0.0", v2: "1.9.9", want: 1},
		{name: "major version less", v1: "1.0.0", v2: "2.0.0", want: -1},

		// Minor version differences
		{name: "minor version greater", v1: "1.9.0", v2: "1.8.5", want: 1},
		{name: "minor version less", v1: "1.8.0", v2: "1.9.0", want: -1},

		// Patch version differences
		{name: "patch version greater", v1: "1.0.3", v2: "1.0.2", want: 1},
		{name: "patch version less", v1: "1.0.1", v2: "1.0.2", want: -1},

		// Prerelease comparisons
		{name: "stable > prerelease", v1: "1.0.0", v2: "1.0.0-rc.1", want: 1},
		{name: "prerelease < stable", v1: "1.0.0-rc.1", v2: "1.0.0", want: -1},
		{name: "rc.2 > rc.1", v1: "1.0.0-rc.2", v2: "1.0.0-rc.1", want: 1},
		{name: "beta < rc", v1: "1.0.0-beta", v2: "1.0.0-rc.1", want: -1},
		{name: "alpha < beta", v1: "1.0.0-alpha", v2: "1.0.0-beta", want: -1},

		// Real world examples
		{name: "0.9.0 > 0.8.2", v1: "0.9.0", v2: "0.8.2", want: 1},
		{name: "0.8.2 < 0.9.0", v1: "0.8.2", v2: "0.9.0", want: -1},
		{name: "0.10.0 > 0.9.0", v1: "0.10.0", v2: "0.9.0", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver1, err := ParseVersion(tt.v1)
			if err != nil {
				t.Fatalf("Failed to parse v1: %v", err)
			}

			ver2, err := ParseVersion(tt.v2)
			if err != nil {
				t.Fatalf("Failed to parse v2: %v", err)
			}

			got := ver1.Compare(ver2)
			if got != tt.want {
				t.Errorf("Compare(%s, %s) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestVersionIsGreaterThan(t *testing.T) {
	v1, _ := ParseVersion("0.9.0")
	v2, _ := ParseVersion("0.8.2")

	if !v1.IsGreaterThan(v2) {
		t.Error("0.9.0 should be greater than 0.8.2")
	}

	if v2.IsGreaterThan(v1) {
		t.Error("0.8.2 should not be greater than 0.9.0")
	}
}

func TestVersionIsLessThan(t *testing.T) {
	v1, _ := ParseVersion("0.8.2")
	v2, _ := ParseVersion("0.9.0")

	if !v1.IsLessThan(v2) {
		t.Error("0.8.2 should be less than 0.9.0")
	}

	if v2.IsLessThan(v1) {
		t.Error("0.9.0 should not be less than 0.8.2")
	}
}

func TestVersionIsEqual(t *testing.T) {
	v1, _ := ParseVersion("0.8.2")
	v2, _ := ParseVersion("v0.8.2")
	v3, _ := ParseVersion("0.9.0")

	if !v1.IsEqual(v2) {
		t.Error("0.8.2 should equal v0.8.2")
	}

	if v1.IsEqual(v3) {
		t.Error("0.8.2 should not equal 0.9.0")
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		v1      string
		v2      string
		want    int
		wantErr bool
	}{
		{
			name: "0.9.0 > 0.8.2",
			v1:   "0.9.0",
			v2:   "0.8.2",
			want: 1,
		},
		{
			name: "0.8.2 < 0.9.0",
			v1:   "0.8.2",
			v2:   "0.9.0",
			want: -1,
		},
		{
			name: "equal versions",
			v1:   "0.8.2",
			v2:   "0.8.2",
			want: 0,
		},
		{
			name:    "invalid v1",
			v1:      "invalid",
			v2:      "0.8.2",
			wantErr: true,
		},
		{
			name:    "invalid v2",
			v1:      "0.8.2",
			v2:      "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareVersions(tt.v1, tt.v2)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CompareVersions(%s, %s) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with v prefix",
			input: "v0.8.2",
			want:  "0.8.2",
		},
		{
			name:  "without v prefix",
			input: "0.8.2",
			want:  "0.8.2",
		},
		{
			name:  "with prerelease",
			input: "v1.0.0-rc.1",
			want:  "1.0.0-rc.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeVersion(tt.input); got != tt.want {
				t.Errorf("NormalizeVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
