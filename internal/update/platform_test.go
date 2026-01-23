package update

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	p := Detect()

	if p.OS == "" {
		t.Error("OS should not be empty")
	}

	if p.Arch == "" {
		t.Error("Arch should not be empty")
	}

	if p.OS != runtime.GOOS {
		t.Errorf("OS mismatch: got %s, want %s", p.OS, runtime.GOOS)
	}

	if p.Arch != runtime.GOARCH {
		t.Errorf("Arch mismatch: got %s, want %s", p.Arch, runtime.GOARCH)
	}
}

func TestPlatformBinaryName(t *testing.T) {
	tests := []struct {
		name string
		p    Platform
		want string
	}{
		{
			name: "darwin arm64",
			p:    Platform{OS: "darwin", Arch: "arm64"},
			want: "clew-darwin-arm64",
		},
		{
			name: "darwin amd64",
			p:    Platform{OS: "darwin", Arch: "amd64"},
			want: "clew-darwin-amd64",
		},
		{
			name: "linux amd64",
			p:    Platform{OS: "linux", Arch: "amd64"},
			want: "clew-linux-amd64",
		},
		{
			name: "linux arm64",
			p:    Platform{OS: "linux", Arch: "arm64"},
			want: "clew-linux-arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.BinaryName(); got != tt.want {
				t.Errorf("BinaryName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlatformIsSupported(t *testing.T) {
	tests := []struct {
		name string
		p    Platform
		want bool
	}{
		{
			name: "darwin arm64 supported",
			p:    Platform{OS: "darwin", Arch: "arm64"},
			want: true,
		},
		{
			name: "darwin amd64 supported",
			p:    Platform{OS: "darwin", Arch: "amd64"},
			want: true,
		},
		{
			name: "linux amd64 supported",
			p:    Platform{OS: "linux", Arch: "amd64"},
			want: true,
		},
		{
			name: "linux arm64 supported",
			p:    Platform{OS: "linux", Arch: "arm64"},
			want: true,
		},
		{
			name: "windows unsupported",
			p:    Platform{OS: "windows", Arch: "amd64"},
			want: false,
		},
		{
			name: "darwin 386 unsupported",
			p:    Platform{OS: "darwin", Arch: "386"},
			want: false,
		},
		{
			name: "freebsd unsupported",
			p:    Platform{OS: "freebsd", Arch: "amd64"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.IsSupported(); got != tt.want {
				t.Errorf("IsSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}
