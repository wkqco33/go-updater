package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withCleanTestState(t *testing.T, home string) *bytes.Buffer {
	t.Helper()
	oldHome := userHomeDir
	oldAll, oldUnused, oldSystem := cleanAll, cleanUnused, cleanSystem
	userHomeDir = func() (string, error) { return home, nil }
	cleanAll, cleanUnused, cleanSystem = false, false, false
	t.Cleanup(func() {
		userHomeDir = oldHome
		cleanAll, cleanUnused, cleanSystem = oldAll, oldUnused, oldSystem
	})
	return &bytes.Buffer{}
}

func TestCleanCommandRemovesUnusedVersions(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".go")
	for _, version := range []string{"go1.22.0", "go1.23.0"} {
		if err := os.MkdirAll(filepath.Join(root, "versions", version), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "versions", "go1.23.0"), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}

	out := withCleanTestState(t, home)
	cleanUnused = true
	cmd := *cleanCmd
	cmd.SetOut(out)
	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatalf("clean RunE() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "go1.22.0")); !os.IsNotExist(err) {
		t.Fatal("unused version still exists")
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "go1.23.0")); err != nil {
		t.Fatal("active version was removed")
	}
	if !strings.Contains(out.String(), "총 1개의") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCleanCommandAllRemovesCurrentAndVersions(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".go")
	if err := os.MkdirAll(filepath.Join(root, "versions", "go1.23.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "versions", "go1.23.0"), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}

	out := withCleanTestState(t, home)
	cleanAll = true
	cmd := *cleanCmd
	cmd.SetOut(out)
	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatalf("clean RunE() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "versions")); !os.IsNotExist(err) {
		t.Fatal("versions directory still exists")
	}
	if _, err := os.Lstat(filepath.Join(root, "current")); !os.IsNotExist(err) {
		t.Fatal("current link still exists")
	}
}
