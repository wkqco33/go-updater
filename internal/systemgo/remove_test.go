package systemgo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveDeletesManagedArtifactsAndKeepsUnmanaged(t *testing.T) {
	dir := t.TempDir()
	managedFile := filepath.Join(dir, "paths")
	managedDir := filepath.Join(dir, "go")
	unmanaged := filepath.Join(dir, "brew")
	if err := os.WriteFile(managedFile, []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(managedDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(managedDir, "bin", "go"), []byte("go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unmanaged, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	items := []Present{
		{Artifact: Artifact{Kind: KindFile, Path: managedFile, Managed: true}, Exists: true},
		{Artifact: Artifact{Kind: KindDir, Path: managedDir, Managed: true}, Exists: true},
		{Artifact: Artifact{Kind: KindHomebrew, Path: unmanaged, Managed: false}, Exists: true},
	}
	err := Remove(Plan(items))
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	for _, path := range []string{managedFile, managedDir} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists", path)
		}
	}
	if _, err := os.Stat(unmanaged); err != nil {
		t.Fatalf("unmanaged artifact was removed: %v", err)
	}
}

func TestRemoveUsesPrivilegedCommandForReceiptAndContinuesAfterFailure(t *testing.T) {
	old := runCommandPrivileged
	var commands []string
	runCommandPrivileged = func(name string, args ...string) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		if len(commands) == 1 {
			return errors.New("permission denied")
		}
		return nil
	}
	t.Cleanup(func() { runCommandPrivileged = old })

	plan := []PlannedCommand{
		{Artifact: Artifact{Kind: KindReceipt, Path: "org.golang.go", Managed: true}},
		{Artifact: Artifact{Kind: KindReceipt, Path: "com.googlecode.go", Managed: true}},
	}
	if err := Remove(plan); err == nil {
		t.Fatal("Remove() error = nil, want combined failure")
	}
	if len(commands) != 2 || !strings.Contains(commands[0], "--forget org.golang.go") {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestRemoveContinuesAfterFilesystemFailure(t *testing.T) {
	old := runCommandPrivileged
	called := 0
	runCommandPrivileged = func(string, ...string) error {
		called++
		return nil
	}
	t.Cleanup(func() { runCommandPrivileged = old })

	missing := filepath.Join(t.TempDir(), "missing")
	err := Remove([]PlannedCommand{
		{Artifact: Artifact{Kind: KindFile, Path: missing, Managed: true}},
		{Artifact: Artifact{Kind: KindReceipt, Path: "org.golang.go", Managed: true}},
	})
	if err == nil || called != 1 {
		t.Fatalf("error=%v privileged calls=%d", err, called)
	}
}

func TestRemoveRejectsUnsafeManagedDirectory(t *testing.T) {
	if err := Remove([]PlannedCommand{{Artifact: Artifact{Kind: KindDir, Path: t.TempDir(), Managed: true}}}); err == nil {
		t.Fatal("Remove() error = nil for directory without Go markers")
	}
}
