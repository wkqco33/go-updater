package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommandWritesToConfiguredOutput(t *testing.T) {
	var out bytes.Buffer
	cmd := *versionCmd
	cmd.SetOut(&out)
	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "gu version 0.1.0") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestUseCommandUsesInjectedHomeDirectory(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".go")
	target := filepath.Join(root, "versions", "go1.23.0")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	oldHome := useHomeDir
	useHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { useHomeDir = oldHome })

	var out bytes.Buffer
	cmd := *useCmd
	cmd.SetOut(&out)
	if err := cmd.RunE(&cmd, []string{"1.23"}); err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(filepath.Join(root, "current"))
	if err != nil {
		t.Fatal(err)
	}
	if link != target {
		t.Fatalf("current link = %q, want %q", link, target)
	}
	if !strings.Contains(out.String(), "go1.23.0") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestListCommandUsesInjectedHomeDirectoryAndOutput(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".go")
	if err := os.MkdirAll(filepath.Join(root, "versions", "go1.23.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldHome := listHomeDir
	listHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { listHomeDir = oldHome })

	var out bytes.Buffer
	cmd := *listCmd
	cmd.SetOut(&out)
	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "go1.23.0") {
		t.Fatalf("output = %q", out.String())
	}
}
