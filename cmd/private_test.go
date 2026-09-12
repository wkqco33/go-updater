package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/privatecache"
)

// writePrivateConfig persists a real config file so tests parse what the
// command writes instead of hand-built JSON, which breaks on Windows paths.
func writePrivateConfig(t *testing.T, root, cacheDir string) {
	t.Helper()
	if err := privatecache.SaveConfig(filepath.Join(root, "private", "config.json"), privatecache.Config{
		Version:         1,
		PrivatePatterns: []string{"github.com/acme/*"},
		CacheDir:        cacheDir,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestShellQuoteEscapesMetacharacters(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", "'plain'"},
		{"/tmp/$HOME/modcache", "'/tmp/$HOME/modcache'"},
		{"it's", `'it'\''s'`},
		{"`whoami`", "'`whoami`'"},
	}
	for _, tt := range tests {
		if got := shellQuote(tt.in); got != tt.want {
			t.Fatalf("shellQuote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPrivateEnvQuotesValues(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", root)
	writePrivateConfig(t, root, "/tmp/a$HOME'b")
	withGlobals(t, GlobalOptions{})

	out, _, err := runCopiedCommand(t, privateEnvCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `export GOMODCACHE='/tmp/a$HOME'\''b'`) {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, `export GOPRIVATE='github.com/acme/*'`) {
		t.Fatalf("output = %q", out)
	}
}

func TestPrivateEnvOfflineAddsGoproxy(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", root)
	withGlobals(t, GlobalOptions{})
	privateOffline = true
	t.Cleanup(func() { privateOffline = false })

	out, _, err := runCopiedCommand(t, privateEnvCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "export GOPROXY='off'") {
		t.Fatalf("output = %q", out)
	}
}

func TestPrivateCleanAllRequiresConfirmation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", root)
	cacheDir := filepath.Join(root, "modcache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "module.zip"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	writePrivateConfig(t, root, cacheDir)
	withGlobals(t, GlobalOptions{NoInput: true})
	privateCleanAll = true
	t.Cleanup(func() { privateCleanAll = false })

	_, _, err := runCopiedCommand(t, privateCleanCmd, "", nil)
	if err == nil {
		t.Fatal("private clean --all error = nil without confirmation")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
	if _, statErr := os.Stat(filepath.Join(cacheDir, "module.zip")); statErr != nil {
		t.Fatal("cache was deleted even though confirmation failed")
	}
}

func TestPrivateCleanAllDryRunKeepsCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", root)
	cacheDir := filepath.Join(root, "modcache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writePrivateConfig(t, root, cacheDir)
	withGlobals(t, GlobalOptions{DryRun: true})
	privateCleanAll = true
	t.Cleanup(func() { privateCleanAll = false })

	out, _, err := runCopiedCommand(t, privateCleanCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dry-run") {
		t.Fatalf("output = %q", out)
	}
	if _, statErr := os.Stat(cacheDir); statErr != nil {
		t.Fatal("dry-run removed the cache directory")
	}
}

func TestPrivateCleanStaleDaysDryRunKeepsFiles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", root)
	cacheDir := filepath.Join(root, "modcache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(cacheDir, "module.zip")
	if err := os.WriteFile(stale, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	writePrivateConfig(t, root, cacheDir)
	withGlobals(t, GlobalOptions{DryRun: true})
	privateCleanStaleDay = 1
	t.Cleanup(func() { privateCleanStaleDay = 0 })

	out, _, err := runCopiedCommand(t, privateCleanCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dry-run") || !strings.Contains(out, "삭제 파일 1개") {
		t.Fatalf("output = %q", out)
	}
	if _, statErr := os.Stat(stale); statErr != nil {
		t.Fatal("dry-run removed a stale cache file")
	}
}

func TestPrivateCleanAllRemovesCacheWithYes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", root)
	cacheDir := filepath.Join(root, "modcache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "module.zip"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	writePrivateConfig(t, root, cacheDir)
	withGlobals(t, GlobalOptions{Yes: true})
	privateCleanAll = true
	t.Cleanup(func() { privateCleanAll = false })

	out, _, err := runCopiedCommand(t, privateCleanCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "정리 완료") {
		t.Fatalf("output = %q", out)
	}
	if _, statErr := os.Stat(cacheDir); !os.IsNotExist(statErr) {
		t.Fatal("cache directory still exists")
	}
}

func TestPrivateSyncCommandRequiresModuleArgument(t *testing.T) {
	t.Setenv("GU_HOME", t.TempDir())

	_, _, err := runRootCommand(t, "private", "sync")
	if err == nil {
		t.Fatal("private sync error = nil without a module argument")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
}

func TestPrivateConfigSetPersistsPatterns(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", root)

	if _, _, err := runRootCommand(t, "private", "config", "set", "--private", "github.com/acme/*,git.example.com/*"); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(filepath.Join(root, "private", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"github.com/acme/*", "git.example.com/*", "nosumdb_patterns", "noproxy_patterns"} {
		if !strings.Contains(text, want) {
			t.Fatalf("config = %s, want %q", text, want)
		}
	}
}
