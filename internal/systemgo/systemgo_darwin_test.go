//go:build darwin

package systemgo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePkgInfo(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want map[string]string
	}{
		{
			name: "typical pkgutil output",
			in:   "package-id: org.golang.go\nversion: go1.26.5\nvolume: /\nlocation: /\ninstall-time: 1784453018\n",
			want: map[string]string{
				"package-id":   "org.golang.go",
				"version":      "go1.26.5",
				"volume":       "/",
				"location":     "/",
				"install-time": "1784453018",
			},
		},
		{
			name: "empty output",
			in:   "",
			want: map[string]string{},
		},
		{
			name: "line without colon is ignored",
			in:   "no receipt for 'org.golang.go' found at '/'.\n",
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePkgInfo(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("parsePkgInfo() = %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parsePkgInfo()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestGoRootFromPathsD(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"trailing newline", "/usr/local/go/bin\n", "/usr/local/go"},
		{"no trailing newline", "/usr/local/go/bin", "/usr/local/go"},
		{"empty file", "", ""},
		{"whitespace only", "   \n", ""},
		{"multiple lines uses first", "/usr/local/go/bin\n/opt/other/bin\n", "/usr/local/go"},
		{"custom volume", "/Volumes/Data/usr/local/go/bin\n", "/Volumes/Data/usr/local/go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := goRootFromPathsD(tt.in); got != tt.want {
				t.Errorf("goRootFromPathsD(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsSafeGoRoot(t *testing.T) {
	t.Run("valid GOROOT with bin/go", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "bin", "go"), []byte("#!/bin/sh"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := isSafeGoRoot(dir); err != nil {
			t.Errorf("isSafeGoRoot() = %v, want nil", err)
		}
	})

	t.Run("valid GOROOT with VERSION file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("go1.26.5"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := isSafeGoRoot(dir); err != nil {
			t.Errorf("isSafeGoRoot() = %v, want nil", err)
		}
	})

	t.Run("empty directory is rejected", func(t *testing.T) {
		dir := t.TempDir()
		if err := isSafeGoRoot(dir); err == nil {
			t.Error("isSafeGoRoot() = nil, want error for directory with no Go markers")
		}
	})

	t.Run("symlink is rejected", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "real")
		if err := os.Mkdir(target, 0o755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if err := isSafeGoRoot(link); err == nil {
			t.Error("isSafeGoRoot() = nil, want error for symlink")
		}
	})

	for _, protected := range []string{"/", "/usr", "/usr/local", "/etc", "/opt"} {
		t.Run("protected path "+protected, func(t *testing.T) {
			if err := isSafeGoRoot(protected); err == nil {
				t.Errorf("isSafeGoRoot(%q) = nil, want error", protected)
			}
		})
	}

	t.Run("nonexistent path", func(t *testing.T) {
		if err := isSafeGoRoot(filepath.Join(t.TempDir(), "missing")); err == nil {
			t.Error("isSafeGoRoot() = nil, want error for missing path")
		}
	})
}

// withStubs overrides all of Detect's external seams for the duration of a
// test and restores them afterwards.
func withStubs(t *testing.T, pkgsOut, pkgInfoOut string, pathsD string, hasPathsD bool, goRootDir string, makeGoRoot bool, homebrewDir string, makeHomebrew bool) {
	t.Helper()

	origRunCommand := runCommand
	origPathsDGoFile := pathsDGoFile
	origDefaultGoRoot := defaultGoRoot
	origHomebrewPrefixes := homebrewPrefixes
	t.Cleanup(func() {
		runCommand = origRunCommand
		pathsDGoFile = origPathsDGoFile
		defaultGoRoot = origDefaultGoRoot
		homebrewPrefixes = origHomebrewPrefixes
	})

	runCommand = func(name string, args ...string) (string, error) {
		if name == "pkgutil" && len(args) > 0 && args[0] == "--pkgs" {
			return pkgsOut, nil
		}
		if name == "pkgutil" && len(args) > 0 && args[0] == "--pkg-info" {
			return pkgInfoOut, nil
		}
		return "", nil
	}

	pathsDGoFile = pathsD
	if hasPathsD {
		if err := os.WriteFile(pathsD, []byte(goRootDir+"/bin\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	defaultGoRoot = goRootDir
	if makeGoRoot {
		if err := os.MkdirAll(filepath.Join(goRootDir, "bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(goRootDir, "bin", "go"), []byte("#!/bin/sh"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	homebrewPrefixes = []string{homebrewDir}
	if makeHomebrew {
		if err := os.MkdirAll(filepath.Join(homebrewDir, "opt", "go"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDetectNothingFound(t *testing.T) {
	dir := t.TempDir()
	withStubs(t, "", "", filepath.Join(dir, "paths.d-go"), false, filepath.Join(dir, "usr-local-go"), false, filepath.Join(dir, "brew"), false)

	got, err := Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Detect() = %+v, want empty", got)
	}
}

func TestDetectReceiptOnlyGoRootAlreadyRemoved(t *testing.T) {
	dir := t.TempDir()
	pathsD := filepath.Join(dir, "paths.d-go")
	goRoot := filepath.Join(dir, "usr-local-go")
	brew := filepath.Join(dir, "brew")

	// volume/location point at the tempdir sandbox, not the real "/", so this
	// test never depends on (or collides with) an actual /usr/local/go.
	withStubs(t,
		"org.golang.go\n",
		"package-id: org.golang.go\nversion: go1.26.5\nvolume: "+dir+"\nlocation: /\n",
		pathsD, true,
		goRoot, false,
		brew, false,
	)

	got, err := Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	var sawReceipt, sawPathsD, sawDirMissing bool
	for _, item := range got {
		switch item.Artifact.Kind {
		case KindReceipt:
			sawReceipt = item.Exists && item.Artifact.Path == "org.golang.go"
		case KindFile:
			sawPathsD = item.Exists
		case KindDir:
			sawDirMissing = !item.Exists
		}
	}
	if !sawReceipt {
		t.Error("expected a present KindReceipt artifact for org.golang.go")
	}
	if !sawPathsD {
		t.Error("expected a present KindFile artifact for paths.d/go")
	}
	if !sawDirMissing {
		t.Error("expected the GOROOT candidate to be reported as not existing")
	}
}

func TestDetectFullInstallation(t *testing.T) {
	dir := t.TempDir()
	pathsD := filepath.Join(dir, "paths.d-go")
	goRoot := filepath.Join(dir, "usr-local-go")
	brew := filepath.Join(dir, "brew")

	withStubs(t,
		"org.golang.go\n",
		"package-id: org.golang.go\nversion: go1.26.5\nvolume: "+dir+"\nlocation: /\n",
		pathsD, true,
		goRoot, true,
		brew, false,
	)

	got, err := Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	plan := Plan(got)
	if len(plan) != 3 {
		t.Fatalf("Plan() returned %d steps, want 3 (dir, file, receipt); got %+v", len(plan), plan)
	}
}

func TestDetectHomebrewOnly(t *testing.T) {
	dir := t.TempDir()
	pathsD := filepath.Join(dir, "paths.d-go")
	goRoot := filepath.Join(dir, "usr-local-go")
	brew := filepath.Join(dir, "brew")

	withStubs(t, "", "", pathsD, false, goRoot, false, brew, true)

	got, err := Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(got) != 1 || got[0].Artifact.Kind != KindHomebrew {
		t.Fatalf("Detect() = %+v, want a single Homebrew artifact", got)
	}
	if got[0].Artifact.Managed {
		t.Error("Homebrew artifact must not be Managed")
	}

	if plan := Plan(got); len(plan) != 0 {
		t.Errorf("Plan() = %+v, want no steps for an unmanaged Homebrew install", plan)
	}
}
