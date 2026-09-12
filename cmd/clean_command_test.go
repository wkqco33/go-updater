package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wkqco33/go-updater/internal/cli"
)

func setupVersions(t *testing.T, versions ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, version := range versions {
		if err := os.MkdirAll(filepath.Join(root, "versions", version), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func activateVersion(t *testing.T, root, version string) {
	t.Helper()
	if err := os.Symlink(filepath.Join(root, "versions", version), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
}

func TestCleanCommandRemovesUnusedVersionsAfterConfirmation(t *testing.T) {
	root := setupVersions(t, "go1.22.0", "go1.23.0")
	activateVersion(t, root, "go1.23.0")
	withGlobals(t, GlobalOptions{Home: root, Yes: true})
	cleanUnused = true
	t.Cleanup(func() { cleanUnused = false })

	out, _, err := runCopiedCommand(t, cleanCmd, "", nil)
	if err != nil {
		t.Fatalf("clean RunE() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "go1.22.0")); !os.IsNotExist(err) {
		t.Fatal("unused version still exists")
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "go1.23.0")); err != nil {
		t.Fatal("active version was removed")
	}
	if !strings.Contains(out, "총 1개의") {
		t.Fatalf("output = %q", out)
	}
}

func TestCleanCommandAllRemovesCurrentAndVersionsAfterConfirmation(t *testing.T) {
	root := setupVersions(t, "go1.23.0")
	activateVersion(t, root, "go1.23.0")
	withGlobals(t, GlobalOptions{Home: root, Yes: true})
	cleanAll = true
	t.Cleanup(func() { cleanAll = false })

	if _, _, err := runCopiedCommand(t, cleanCmd, "", nil); err != nil {
		t.Fatalf("clean RunE() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "versions")); !os.IsNotExist(err) {
		t.Fatal("versions directory still exists")
	}
	if _, err := os.Lstat(filepath.Join(root, "current")); !os.IsNotExist(err) {
		t.Fatal("current link still exists")
	}
}

func TestCleanCommandAllRequiresConfirmationWithoutTTY(t *testing.T) {
	root := setupVersions(t, "go1.23.0")
	withGlobals(t, GlobalOptions{Home: root, NoInput: true})
	cleanAll = true
	t.Cleanup(func() { cleanAll = false })

	_, _, err := runCopiedCommand(t, cleanCmd, "", nil)
	if err == nil {
		t.Fatal("clean --all error = nil without confirmation")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "versions", "go1.23.0")); statErr != nil {
		t.Fatal("versions were deleted even though confirmation failed")
	}
}

func TestCleanCommandAllCancelsOnDeclinedPrompt(t *testing.T) {
	root := setupVersions(t, "go1.23.0")
	withGlobals(t, GlobalOptions{Home: root})
	withTTY(t, true)
	cleanAll = true
	t.Cleanup(func() { cleanAll = false })

	out, errOut, err := runCopiedCommand(t, cleanCmd, "n\n", nil)
	if err != nil {
		t.Fatalf("clean RunE() error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "versions", "go1.23.0")); statErr != nil {
		t.Fatal("versions were deleted even though the prompt was declined")
	}
	if !strings.Contains(errOut, "취소되었습니다") {
		t.Fatalf("stderr = %q", errOut)
	}
	if strings.Contains(out, "삭제되었습니다") {
		t.Fatalf("stdout = %q", out)
	}
}

func TestCleanCommandAllDryRunKeepsVersions(t *testing.T) {
	root := setupVersions(t, "go1.23.0")
	withGlobals(t, GlobalOptions{Home: root, DryRun: true})
	cleanAll = true
	t.Cleanup(func() { cleanAll = false })

	out, _, err := runCopiedCommand(t, cleanCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "versions", "go1.23.0")); statErr != nil {
		t.Fatal("dry-run deleted a version")
	}
	if !strings.Contains(out, "dry-run") {
		t.Fatalf("output = %q", out)
	}
}

func TestCleanCommandRemovesRequestedVersion(t *testing.T) {
	root := setupVersions(t, "go1.22.0", "go1.23.0")
	activateVersion(t, root, "go1.23.0")
	withGlobals(t, GlobalOptions{Home: root})

	out, _, err := runCopiedCommand(t, cleanCmd, "", []string{"1.22"})
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "versions", "go1.22.0")); !os.IsNotExist(statErr) {
		t.Fatal("requested version still exists")
	}
	if !strings.Contains(out, "go1.22.0") {
		t.Fatalf("output = %q", out)
	}
}

func TestCleanCommandWarnsForActiveVersionOnStderr(t *testing.T) {
	root := setupVersions(t, "go1.23.0")
	activateVersion(t, root, "go1.23.0")
	withGlobals(t, GlobalOptions{Home: root})

	out, errOut, err := runCopiedCommand(t, cleanCmd, "", []string{"1.23.0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "versions", "go1.23.0")); statErr != nil {
		t.Fatal("active version was removed")
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
	if !strings.Contains(errOut, "활성화") {
		t.Fatalf("stderr = %q", errOut)
	}
}

func TestCleanCommandWarnsForMissingVersionOnStderr(t *testing.T) {
	root := setupVersions(t, "go1.23.0")
	withGlobals(t, GlobalOptions{Home: root})

	out, errOut, err := runCopiedCommand(t, cleanCmd, "", []string{"9.9.9"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
	if !strings.Contains(errOut, "is not installed") {
		t.Fatalf("stderr = %q", errOut)
	}
}

func TestCleanCommandQuietHidesStatusButKeepsResult(t *testing.T) {
	root := setupVersions(t, "go1.22.0", "go1.23.0")
	activateVersion(t, root, "go1.23.0")
	withGlobals(t, GlobalOptions{Home: root, Yes: true, Quiet: true})
	cleanUnused = true
	t.Cleanup(func() { cleanUnused = false })

	out, errOut, err := runCopiedCommand(t, cleanCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty with --quiet", errOut)
	}
	if !strings.Contains(out, "총 1개의") {
		t.Fatalf("stdout = %q", out)
	}
}

func TestCleanCommandHelpWithoutFlags(t *testing.T) {
	withGlobals(t, GlobalOptions{Home: t.TempDir()})

	out, _, err := runCopiedCommand(t, cleanCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "clean") {
		t.Fatalf("help output = %q", out)
	}
}
