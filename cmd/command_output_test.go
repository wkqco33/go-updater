package cmd

import (
	"bytes"
	"encoding/json"
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
	if !strings.Contains(out.String(), "gu version dev") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestVersionCommandEmitsJSON(t *testing.T) {
	withGlobals(t, GlobalOptions{})
	versionJSON = true
	t.Cleanup(func() { versionJSON = false })

	var out bytes.Buffer
	cmd := *versionCmd
	cmd.SetOut(&out)
	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatal(err)
	}
	var info versionInfo
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		t.Fatalf("version --json output is not JSON (%v): %q", err, out.String())
	}
	if info.Version != "dev" || info.OS == "" || info.Arch == "" {
		t.Fatalf("info = %+v", info)
	}
}

func TestUseCommandActivatesRequestedVersion(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "versions", "go1.23.0")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	withGlobals(t, GlobalOptions{Home: root})

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

func TestListCommandUsesConfiguredRootAndOutput(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "versions", "go1.23.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	withGlobals(t, GlobalOptions{Home: root})

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

func TestListCommandEmitsJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "versions", "go1.23.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "versions", "go1.23.0"), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	withGlobals(t, GlobalOptions{Home: root})
	listJSON = true
	t.Cleanup(func() { listJSON = false })

	var out bytes.Buffer
	cmd := *listCmd
	cmd.SetOut(&out)
	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatal(err)
	}
	var parsed listOutput
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("list --json output is not JSON (%v): %q", err, out.String())
	}
	if parsed.Root != root || parsed.Current != "go1.23.0" || len(parsed.Versions) != 1 || !parsed.Versions[0].Active {
		t.Fatalf("parsed = %+v", parsed)
	}
}

func TestListCommandReportsEmptyState(t *testing.T) {
	withGlobals(t, GlobalOptions{Home: t.TempDir()})

	out, _, err := runCopiedCommand(t, listCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "설치된 Go 버전이 없습니다") {
		t.Fatalf("output = %q", out)
	}
}

func TestListCommandOmitsColorWhenDisabled(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "versions", "go1.23.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "versions", "go1.23.0"), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	withGlobals(t, GlobalOptions{Home: root, NoColor: true})
	t.Setenv("NO_COLOR", "")

	out, _, err := runCopiedCommand(t, listCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("color escape found in output: %q", out)
	}
}
