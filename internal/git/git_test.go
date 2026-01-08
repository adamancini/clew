package git

import (
	"errors"
	"testing"

	"github.com/adamancini/clew/internal/config"
)

// MockCommandRunner mocks command execution for testing.
type MockCommandRunner struct {
	// Commands maps "dir:command args..." to output
	Commands map[string]struct {
		Output []byte
		Error  error
	}
}

// NewMockCommandRunner creates a new MockCommandRunner.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		Commands: make(map[string]struct {
			Output []byte
			Error  error
		}),
	}
}

// AddCommand adds a command response.
func (m *MockCommandRunner) AddCommand(dir, cmd string, output []byte, err error) {
	key := dir + ":" + cmd
	m.Commands[key] = struct {
		Output []byte
		Error  error
	}{Output: output, Error: err}
}

// Run executes a command (not in a specific directory).
func (m *MockCommandRunner) Run(name string, args ...string) ([]byte, error) {
	return m.RunInDir("", name, args...)
}

// RunInDir executes a command in a directory.
func (m *MockCommandRunner) RunInDir(dir, name string, args ...string) ([]byte, error) {
	cmd := name
	for _, arg := range args {
		cmd += " " + arg
	}
	key := dir + ":" + cmd
	if resp, ok := m.Commands[key]; ok {
		return resp.Output, resp.Error
	}
	// Also try without dir for global commands
	key = ":" + cmd
	if resp, ok := m.Commands[key]; ok {
		return resp.Output, resp.Error
	}
	return nil, errors.New("command not mocked: " + key)
}

func TestCheckerGitAvailable(t *testing.T) {
	tests := []struct {
		name      string
		gitOutput []byte
		gitError  error
		want      bool
	}{
		{
			name:      "git available",
			gitOutput: []byte("git version 2.40.0"),
			gitError:  nil,
			want:      true,
		},
		{
			name:      "git not available",
			gitOutput: nil,
			gitError:  errors.New("git not found"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockCommandRunner()
			mock.AddCommand("", "git --version", tt.gitOutput, tt.gitError)

			checker := NewCheckerWithRunner(mock)
			got := checker.GitAvailable()
			if got != tt.want {
				t.Errorf("GitAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckRepositoryClean(t *testing.T) {
	mock := NewMockCommandRunner()
	path := "/tmp/testrepo"

	// Git repo check
	mock.AddCommand(path, "git rev-parse --git-dir", []byte(".git\n"), nil)
	// Current branch
	mock.AddCommand(path, "git rev-parse --abbrev-ref HEAD", []byte("main\n"), nil)
	// Status (clean)
	mock.AddCommand(path, "git status --porcelain", []byte(""), nil)
	// Remote tracking
	mock.AddCommand(path, "git rev-parse --abbrev-ref --symbolic-full-name @{u}", []byte("origin/main\n"), nil)
	// Fetch
	mock.AddCommand(path, "git fetch --quiet", []byte(""), nil)
	// Ahead/behind (in sync)
	mock.AddCommand(path, "git rev-list --left-right --count HEAD...origin/main", []byte("0\t0\n"), nil)

	checker := NewCheckerWithRunner(mock)
	status := checker.checkRepositorySkipPathCheck(path)

	if status.Level != LevelOK {
		t.Errorf("Level = %v, want %v", status.Level, LevelOK)
	}
	if status.HasUncommitted {
		t.Error("HasUncommitted = true, want false")
	}
	if !status.IsClean {
		t.Error("IsClean = false, want true")
	}
	if status.CurrentBranch != "main" {
		t.Errorf("CurrentBranch = %v, want main", status.CurrentBranch)
	}
}

func TestCheckRepositoryUncommittedChanges(t *testing.T) {
	mock := NewMockCommandRunner()
	path := "/tmp/testrepo"

	// Git repo check
	mock.AddCommand(path, "git rev-parse --git-dir", []byte(".git\n"), nil)
	// Current branch
	mock.AddCommand(path, "git rev-parse --abbrev-ref HEAD", []byte("main\n"), nil)
	// Status (has changes)
	mock.AddCommand(path, "git status --porcelain", []byte(" M README.md\n?? newfile.txt\n"), nil)

	checker := NewCheckerWithRunner(mock)
	status := checker.checkRepositorySkipPathCheck(path)

	if status.Level != LevelWarning {
		t.Errorf("Level = %v, want %v", status.Level, LevelWarning)
	}
	if !status.HasUncommitted {
		t.Error("HasUncommitted = false, want true")
	}
	if status.IsClean {
		t.Error("IsClean = true, want false")
	}
	if status.Message != "uncommitted changes detected" {
		t.Errorf("Message = %v, want 'uncommitted changes detected'", status.Message)
	}
}

func TestCheckRepositoryBehindRemote(t *testing.T) {
	mock := NewMockCommandRunner()
	path := "/tmp/testrepo"

	// Git repo check
	mock.AddCommand(path, "git rev-parse --git-dir", []byte(".git\n"), nil)
	// Current branch
	mock.AddCommand(path, "git rev-parse --abbrev-ref HEAD", []byte("main\n"), nil)
	// Status (clean)
	mock.AddCommand(path, "git status --porcelain", []byte(""), nil)
	// Remote tracking
	mock.AddCommand(path, "git rev-parse --abbrev-ref --symbolic-full-name @{u}", []byte("origin/main\n"), nil)
	// Fetch
	mock.AddCommand(path, "git fetch --quiet", []byte(""), nil)
	// Ahead/behind (3 behind)
	mock.AddCommand(path, "git rev-list --left-right --count HEAD...origin/main", []byte("0\t3\n"), nil)

	checker := NewCheckerWithRunner(mock)
	status := checker.checkRepositorySkipPathCheck(path)

	if status.Level != LevelInfo {
		t.Errorf("Level = %v, want %v", status.Level, LevelInfo)
	}
	if status.Behind != 3 {
		t.Errorf("Behind = %v, want 3", status.Behind)
	}
	if status.Ahead != 0 {
		t.Errorf("Ahead = %v, want 0", status.Ahead)
	}
}

func TestCheckRepositoryAheadOfRemote(t *testing.T) {
	mock := NewMockCommandRunner()
	path := "/tmp/testrepo"

	// Git repo check
	mock.AddCommand(path, "git rev-parse --git-dir", []byte(".git\n"), nil)
	// Current branch
	mock.AddCommand(path, "git rev-parse --abbrev-ref HEAD", []byte("feature\n"), nil)
	// Status (clean)
	mock.AddCommand(path, "git status --porcelain", []byte(""), nil)
	// Remote tracking
	mock.AddCommand(path, "git rev-parse --abbrev-ref --symbolic-full-name @{u}", []byte("origin/feature\n"), nil)
	// Fetch
	mock.AddCommand(path, "git fetch --quiet", []byte(""), nil)
	// Ahead/behind (2 ahead)
	mock.AddCommand(path, "git rev-list --left-right --count HEAD...origin/feature", []byte("2\t0\n"), nil)

	checker := NewCheckerWithRunner(mock)
	status := checker.checkRepositorySkipPathCheck(path)

	if status.Level != LevelInfo {
		t.Errorf("Level = %v, want %v", status.Level, LevelInfo)
	}
	if status.Ahead != 2 {
		t.Errorf("Ahead = %v, want 2", status.Ahead)
	}
	if status.Behind != 0 {
		t.Errorf("Behind = %v, want 0", status.Behind)
	}
}

func TestCheckRepositoryNotGitRepo(t *testing.T) {
	mock := NewMockCommandRunner()
	path := "/tmp/notarepo"

	// Git repo check fails
	mock.AddCommand(path, "git rev-parse --git-dir", nil, errors.New("not a git repository"))

	checker := NewCheckerWithRunner(mock)
	status := checker.checkRepositorySkipPathCheck(path)

	if status.IsGitRepo {
		t.Error("IsGitRepo = true, want false")
	}
	if status.Level != LevelInfo {
		t.Errorf("Level = %v, want %v", status.Level, LevelInfo)
	}
	if status.Message != "not a git repository" {
		t.Errorf("Message = %v, want 'not a git repository'", status.Message)
	}
}

func TestCheckRepositoryNoRemoteTracking(t *testing.T) {
	mock := NewMockCommandRunner()
	path := "/tmp/testrepo"

	// Git repo check
	mock.AddCommand(path, "git rev-parse --git-dir", []byte(".git\n"), nil)
	// Current branch
	mock.AddCommand(path, "git rev-parse --abbrev-ref HEAD", []byte("main\n"), nil)
	// Status (clean)
	mock.AddCommand(path, "git status --porcelain", []byte(""), nil)
	// Remote tracking fails (no upstream)
	mock.AddCommand(path, "git rev-parse --abbrev-ref --symbolic-full-name @{u}", nil, errors.New("no upstream"))

	checker := NewCheckerWithRunner(mock)
	status := checker.checkRepositorySkipPathCheck(path)

	if status.Level != LevelOK {
		t.Errorf("Level = %v, want %v", status.Level, LevelOK)
	}
	if status.Message != "clean (no remote tracking branch)" {
		t.Errorf("Message = %v, want 'clean (no remote tracking branch)'", status.Message)
	}
}

func TestCheckClewfile(t *testing.T) {
	mock := NewMockCommandRunner()

	// Git is available
	mock.AddCommand("", "git --version", []byte("git version 2.40.0"), nil)

	// Local marketplace with uncommitted changes
	marketplacePath := "/home/user/.claude/plugins/repos/local-marketplace"
	mock.AddCommand(marketplacePath, "git rev-parse --git-dir", []byte(".git\n"), nil)
	mock.AddCommand(marketplacePath, "git rev-parse --abbrev-ref HEAD", []byte("main\n"), nil)
	mock.AddCommand(marketplacePath, "git status --porcelain", []byte(" M plugin.json\n"), nil)

	// Local plugin that is clean
	pluginPath := "/home/user/.claude/plugins/repos/my-plugin"
	mock.AddCommand(pluginPath, "git rev-parse --git-dir", []byte(".git\n"), nil)
	mock.AddCommand(pluginPath, "git rev-parse --abbrev-ref HEAD", []byte("main\n"), nil)
	mock.AddCommand(pluginPath, "git status --porcelain", []byte(""), nil)
	mock.AddCommand(pluginPath, "git rev-parse --abbrev-ref --symbolic-full-name @{u}", []byte("origin/main\n"), nil)
	mock.AddCommand(pluginPath, "git fetch --quiet", []byte(""), nil)
	mock.AddCommand(pluginPath, "git rev-list --left-right --count HEAD...origin/main", []byte("0\t0\n"), nil)

	clewfile := &config.Clewfile{
		Marketplaces: map[string]config.Marketplace{
			"local-marketplace": {
				Source: "local",
				Path:   marketplacePath,
			},
			"github-marketplace": {
				Source: "github",
				Repo:   "anthropics/claude-plugins-official",
			},
		},
		Plugins: []config.Plugin{
			{
				Name:   "my-plugin@local-marketplace",
				Source: "local",
				Path:   pluginPath,
			},
			{
				Name: "other-plugin@github-marketplace",
			},
		},
	}

	checker := NewCheckerWithRunner(mock)
	checker.SetSkipPathCheck(true) // Skip path existence checks for testing
	result := checker.CheckClewfile(clewfile)

	// Should have warning for marketplace with uncommitted changes
	if !result.ShouldSkipMarketplace("local-marketplace") {
		t.Error("Expected local-marketplace to be skipped")
	}

	// Should not skip plugin (it's clean)
	if result.ShouldSkipPlugin("my-plugin@local-marketplace") {
		t.Error("Expected my-plugin to NOT be skipped")
	}

	// Should have 1 warning
	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestCheckClewfileGitNotAvailable(t *testing.T) {
	mock := NewMockCommandRunner()

	// Git is NOT available
	mock.AddCommand("", "git --version", nil, errors.New("git not found"))

	clewfile := &config.Clewfile{
		Marketplaces: map[string]config.Marketplace{
			"local-marketplace": {
				Source: "local",
				Path:   "/tmp/marketplace",
			},
		},
	}

	checker := NewCheckerWithRunner(mock)
	result := checker.CheckClewfile(clewfile)

	// Should have info message about git not available
	if len(result.Info) != 1 {
		t.Errorf("Expected 1 info message, got %d", len(result.Info))
	}

	// Should not skip anything
	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(result.Warnings))
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string // Empty means it should start with home dir
	}{
		{
			name: "absolute path unchanged",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "relative path unchanged",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "tilde expands",
			path: "~/test",
			want: "", // Will be checked to end with /test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPath(tt.path)
			if tt.want != "" {
				if got != tt.want {
					t.Errorf("expandPath(%q) = %q, want %q", tt.path, got, tt.want)
				}
			} else {
				// For tilde expansion, just check it doesn't start with ~
				if got[0] == '~' {
					t.Errorf("expandPath(%q) = %q, expected tilde to be expanded", tt.path, got)
				}
			}
		})
	}
}
