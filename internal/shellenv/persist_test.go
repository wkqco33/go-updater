package shellenv

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testPaths builds one env file and one startup file under a temp home so no
// test ever touches a real shell configuration.
func testPaths(t *testing.T) (home, envFile, rc string) {
	t.Helper()
	home = t.TempDir()
	envFile = filepath.Join(home, ".config", "gu", "env.sh")
	rc = filepath.Join(home, ".zshrc")
	return home, envFile, rc
}

func testPersistOptions(envFile string, rcs ...string) PersistOptions {
	return PersistOptions{
		Setup:   Setup{Shell: ShellZsh, BinDir: "/home/u/.go/current/bin"},
		EnvFile: envFile,
		RCFiles: rcs,
	}
}

func TestPersistCreatesEnvFileAndRCBlock(t *testing.T) {
	_, envFile, rc := testPaths(t)

	result, err := Persist(testPersistOptions(envFile, rc))
	if err != nil {
		t.Fatal(err)
	}
	if !result.EnvFileChanged {
		t.Fatal("EnvFileChanged = false, want true")
	}
	if len(result.RCFiles) != 1 || !result.RCFiles[0].Changed {
		t.Fatalf("RCFiles = %+v, want one changed entry", result.RCFiles)
	}

	envContent, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envContent), "export PATH='/home/u/.go/current/bin':\"$PATH\"") {
		t.Fatalf("env file = %q", envContent)
	}

	rcContent, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !HasBlock(string(rcContent)) {
		t.Fatalf("rc file lacks gu block: %q", rcContent)
	}
	if !strings.Contains(string(rcContent), envFile) {
		t.Fatalf("rc file does not source %q: %q", envFile, rcContent)
	}
}

func TestPersistIsIdempotent(t *testing.T) {
	_, envFile, rc := testPaths(t)
	if _, err := Persist(testPersistOptions(envFile, rc)); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Persist(testPersistOptions(envFile, rc))
	if err != nil {
		t.Fatal(err)
	}
	if result.EnvFileChanged {
		t.Fatal("EnvFileChanged = true on second persist")
	}
	if result.RCFiles[0].Changed {
		t.Fatal("rc file changed on second persist")
	}
	if _, err := os.Stat(rc + backupSuffix); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("second persist created a backup for an unchanged file")
	}
	after, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("rc content changed on second persist:\nbefore=%q\nafter=%q", before, after)
	}
}

func TestPersistDryRunWritesNothing(t *testing.T) {
	_, envFile, rc := testPaths(t)
	opts := testPersistOptions(envFile, rc)
	opts.DryRun = true

	result, err := Persist(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !result.EnvFileChanged || !result.RCFiles[0].Changed {
		t.Fatalf("dry-run should report pending changes: %+v", result)
	}
	if _, err := os.Stat(envFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("dry-run created the env file")
	}
	if _, err := os.Stat(rc); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("dry-run created the rc file")
	}
}

func TestPersistBacksUpExistingRC(t *testing.T) {
	_, envFile, rc := testPaths(t)
	original := "# user config\nexport EDITOR=vim\n"
	if err := os.WriteFile(rc, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(rc)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Persist(testPersistOptions(envFile, rc))
	if err != nil {
		t.Fatal(err)
	}
	if result.RCFiles[0].Backup == "" {
		t.Fatal("Backup = empty, want a backup path")
	}
	backup, err := os.ReadFile(result.RCFiles[0].Backup)
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != original {
		t.Fatalf("backup = %q, want %q", backup, original)
	}
	// Compare against the mode the file had before instead of a hardcoded
	// 0600: Windows does not map POSIX permission bits, so only preservation
	// is portable across the CI matrix.
	after, err := os.Stat(rc)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm() != before.Mode().Perm() {
		t.Fatalf("rc mode = %v, want preserved %v", after.Mode().Perm(), before.Mode().Perm())
	}
	if !strings.HasPrefix(mustRead(t, rc), "# user config\n") {
		t.Fatalf("existing rc content was not preserved: %q", mustRead(t, rc))
	}
}

func TestPersistRejectsCorruptedRCWithoutWriting(t *testing.T) {
	_, envFile, rc := testPaths(t)
	corrupted := "# hi\n" + markerEnd + "\n"
	if err := os.WriteFile(rc, []byte(corrupted), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Persist(testPersistOptions(envFile, rc))
	if !errors.Is(err, ErrCorruptedBlock) {
		t.Fatalf("err = %v, want ErrCorruptedBlock", err)
	}
	if got := mustRead(t, rc); got != corrupted {
		t.Fatalf("rc file was modified despite the error: %q", got)
	}
}

func TestPersistCreatesMissingRCDirectory(t *testing.T) {
	home := t.TempDir()
	envFile := filepath.Join(home, ".config", "gu", "env.fish")
	rc := filepath.Join(home, ".config", "fish", "config.fish")

	opts := testPersistOptions(envFile, rc)
	opts.Setup.Shell = ShellFish
	if _, err := Persist(opts); err != nil {
		t.Fatal(err)
	}
	if !HasBlock(mustRead(t, rc)) {
		t.Fatalf("fish startup file lacks gu block: %q", mustRead(t, rc))
	}
	if !strings.Contains(mustRead(t, envFile), "fish_add_path") {
		t.Fatalf("env file = %q", mustRead(t, envFile))
	}
}

func TestPersistEditsSymlinkTarget(t *testing.T) {
	home, envFile, _ := testPaths(t)
	target := filepath.Join(home, "dotfiles", "zshrc")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("# dotfiles\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".zshrc")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if _, err := Persist(testPersistOptions(envFile, link)); err != nil {
		t.Fatal(err)
	}
	if !HasBlock(mustRead(t, target)) {
		t.Fatalf("symlink target lacks gu block: %q", mustRead(t, target))
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink was replaced by a regular file")
	}
}

func TestUnpersistRemovesBlockAndEnvFile(t *testing.T) {
	_, envFile, rc := testPaths(t)
	original := "# user config\nexport EDITOR=vim\n"
	if err := os.WriteFile(rc, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Persist(testPersistOptions(envFile, rc)); err != nil {
		t.Fatal(err)
	}

	result, err := Unpersist(testPersistOptions(envFile, rc))
	if err != nil {
		t.Fatal(err)
	}
	if !result.EnvFileChanged {
		t.Fatal("EnvFileChanged = false, want removed env file")
	}
	if !result.RCFiles[0].Changed {
		t.Fatal("rc file was not reported as changed")
	}
	if got := mustRead(t, rc); got != original {
		t.Fatalf("rc after unpersist = %q, want %q", got, original)
	}
	if _, err := os.Stat(envFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("env file still exists after unpersist")
	}
}

func TestUnpersistWithoutPersistIsNoop(t *testing.T) {
	_, envFile, rc := testPaths(t)
	if err := os.WriteFile(rc, []byte("# untouched\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Unpersist(testPersistOptions(envFile, rc))
	if err != nil {
		t.Fatal(err)
	}
	if result.EnvFileChanged || result.RCFiles[0].Changed {
		t.Fatalf("noop unpersist reported changes: %+v", result)
	}
	if got := mustRead(t, rc); got != "# untouched\n" {
		t.Fatalf("rc changed: %q", got)
	}
}

func TestUnpersistDryRunKeepsFiles(t *testing.T) {
	_, envFile, rc := testPaths(t)
	if _, err := Persist(testPersistOptions(envFile, rc)); err != nil {
		t.Fatal(err)
	}
	opts := testPersistOptions(envFile, rc)
	opts.DryRun = true

	result, err := Unpersist(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !result.EnvFileChanged || !result.RCFiles[0].Changed {
		t.Fatalf("dry-run unpersist did not report changes: %+v", result)
	}
	if !HasBlock(mustRead(t, rc)) {
		t.Fatal("dry-run unpersist removed the block")
	}
	if _, err := os.Stat(envFile); err != nil {
		t.Fatal("dry-run unpersist removed the env file")
	}
}

func TestPersistReportsUnchangedRCFiles(t *testing.T) {
	_, envFile, rc := testPaths(t)
	if _, err := Persist(testPersistOptions(envFile, rc)); err != nil {
		t.Fatal(err)
	}
	result, err := Persist(testPersistOptions(envFile, rc))
	if err != nil {
		t.Fatal(err)
	}
	if result.RCFiles[0].Exists != true {
		t.Fatalf("Exists = %v, want true", result.RCFiles[0].Exists)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPersistReportsWhetherEnvFileExisted(t *testing.T) {
	_, envFile, rc := testPaths(t)

	first, err := Persist(testPersistOptions(envFile, rc))
	if err != nil {
		t.Fatal(err)
	}
	if first.EnvFileExisted {
		t.Fatal("first persist reported EnvFileExisted = true")
	}

	second, err := Persist(testPersistOptions(envFile, rc))
	if err != nil {
		t.Fatal(err)
	}
	if !second.EnvFileExisted {
		t.Fatal("second persist reported EnvFileExisted = false")
	}
}
