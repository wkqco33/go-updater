package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/shellenv"
	"github.com/wkqco33/go-updater/internal/systemgo"
)

// withQuietSystemGo removes the system-Go detection boundary so env tests
// never inspect real system paths.
func withQuietSystemGo(t *testing.T) {
	t.Helper()
	withSystemSeams(t, func() ([]systemgo.Present, error) { return nil, nil }, nil)
}

func TestEnvInitPrintsSnippetForPositionalShell(t *testing.T) {
	root := t.TempDir()
	withGlobals(t, GlobalOptions{Home: root})
	withQuietSystemGo(t)

	out, _, err := runCopiedCommand(t, envInitCmd, "", []string{"zsh"})
	if err != nil {
		t.Fatal(err)
	}
	want := "export PATH='" + filepath.Join(root, "current", "bin") + "':\"$PATH\"\n"
	if out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

func TestEnvInitAutoDetectsFromShellEnv(t *testing.T) {
	root := t.TempDir()
	withGlobals(t, GlobalOptions{Home: root})
	t.Setenv("SHELL", "/usr/bin/fish")

	out, _, err := runCopiedCommand(t, envInitCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "fish_add_path") {
		t.Fatalf("output = %q, want fish snippet", out)
	}
}

func TestEnvInitUnsupportedShellIsUsageError(t *testing.T) {
	withGlobals(t, GlobalOptions{Home: t.TempDir()})

	_, _, err := runCopiedCommand(t, envInitCmd, "", []string{"tcsh"})
	if err == nil {
		t.Fatal("error = nil, want usage error")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
}

func TestEnvInitGorootFlagAddsGoroot(t *testing.T) {
	root := t.TempDir()
	withGlobals(t, GlobalOptions{Home: root})
	envGoroot = true
	t.Cleanup(func() { envGoroot = false })

	out, _, err := runCopiedCommand(t, envInitCmd, "", []string{"bash"})
	if err != nil {
		t.Fatal(err)
	}
	want := "export GOROOT='" + filepath.Join(root, "current") + "'\n"
	if !strings.Contains(out, want) {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

func TestEnvInitModuleFlagAddsPrivateEnv(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", root)
	writePrivateConfig(t, root, "/tmp/modcache")
	withGlobals(t, GlobalOptions{Home: root})
	envModule = true
	t.Cleanup(func() { envModule = false })

	out, _, err := runCopiedCommand(t, envInitCmd, "", []string{"bash"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "export GOMODCACHE='/tmp/modcache'") {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, "export GOPRIVATE='github.com/acme/*'") {
		t.Fatalf("output = %q", out)
	}
}

func TestEnvInitViaRootParsesFlags(t *testing.T) {
	root := t.TempDir()
	withQuietSystemGo(t)

	out, _, err := runRootCommand(t, "env", "init", "--shell", "bash", "--gopath", "/home/u/go", "--home", root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "export PATH='"+filepath.Join(root, "current", "bin")+"'") {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, "export GOPATH='/home/u/go'\n") {
		t.Fatalf("output = %q", out)
	}
}

func TestEnvShowReportsStatus(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PATH", filepath.Join(root, "current", "bin")+string(os.PathListSeparator)+"/usr/bin")
	withGlobals(t, GlobalOptions{Home: root})
	withQuietSystemGo(t)

	out, _, err := runCopiedCommand(t, envShowCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "PATH 포함:         예") {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, "gu 루트:           "+root) {
		t.Fatalf("output = %q", out)
	}
}

func TestEnvShowJSONIsMachineReadable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	withGlobals(t, GlobalOptions{Home: root})
	withQuietSystemGo(t)
	envJSON = true
	t.Cleanup(func() { envJSON = false })

	out, _, err := runCopiedCommand(t, envShowCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var parsed envShowOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("env show --json is not JSON (%v): %q", err, out)
	}
	if parsed.Root != root || parsed.BinDir != filepath.Join(root, "current", "bin") {
		t.Fatalf("parsed = %+v", parsed)
	}
	if parsed.EnvFile == "" {
		t.Fatalf("parsed.EnvFile empty: %+v", parsed)
	}
}

func TestEnvContainerRejectsPositionalArgs(t *testing.T) {
	_, _, err := runRootCommand(t, "env", "bogus")
	if err == nil {
		t.Fatal("error = nil, want usage error")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
}

// envTestHome isolates HOME/XDG so env set/unset never touch real dotfiles.
func envTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	return home
}

func TestEnvSetWritesEnvFileAndRC(t *testing.T) {
	root := t.TempDir()
	home := envTestHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("# user config\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withGlobals(t, GlobalOptions{Home: root})
	envShell = "zsh"
	envRC = rc
	t.Cleanup(func() { envShell, envRC = "", "" })

	out, _, err := runCopiedCommand(t, envSetCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "환경 파일:") || !strings.Contains(out, "생성됨") {
		t.Fatalf("first set should report creation: %q", out)
	}
	envFile := filepath.Join(home, ".config", "gu", "env.sh")
	if _, err := os.Stat(envFile); err != nil {
		t.Fatalf("env file missing: %v", err)
	}
	rcContent, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rcContent), envFile) {
		t.Fatalf("rc content = %q", rcContent)
	}
}

func TestEnvSetDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	home := envTestHome(t)
	rc := filepath.Join(home, ".zshrc")
	withGlobals(t, GlobalOptions{Home: root, DryRun: true})
	envShell = "zsh"
	envRC = rc
	t.Cleanup(func() { envShell, envRC = "", "" })

	out, _, err := runCopiedCommand(t, envSetCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dry-run:") {
		t.Fatalf("output = %q", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "gu", "env.sh")); !os.IsNotExist(err) {
		t.Fatal("dry-run created the env file")
	}
	if _, err := os.Stat(rc); !os.IsNotExist(err) {
		t.Fatal("dry-run created the rc file")
	}
}

func TestEnvUnsetRequiresConfirmation(t *testing.T) {
	root := t.TempDir()
	home := envTestHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("# user\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withGlobals(t, GlobalOptions{Home: root, NoInput: true})
	envShell = "zsh"
	envRC = rc
	t.Cleanup(func() { envShell, envRC = "", "" })

	// Seed a persisted setup without confirmation.
	if _, _, err := runCopiedCommand(t, envSetCmd, "", nil); err != nil {
		t.Fatal(err)
	}
	_, _, err := runCopiedCommand(t, envUnsetCmd, "", nil)
	if err == nil {
		t.Fatal("unset error = nil without confirmation")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
	if !strings.Contains(mustReadString(t, rc), envFileMarker) {
		t.Fatal("block was removed even though confirmation failed")
	}
}

func TestEnvUnsetRemovesWithYes(t *testing.T) {
	root := t.TempDir()
	home := envTestHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("# user\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withGlobals(t, GlobalOptions{Home: root, Yes: true})
	envShell = "zsh"
	envRC = rc
	t.Cleanup(func() { envShell, envRC = "", "" })

	if _, _, err := runCopiedCommand(t, envSetCmd, "", nil); err != nil {
		t.Fatal(err)
	}
	out, _, err := runCopiedCommand(t, envUnsetCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "삭제됨") {
		t.Fatalf("output = %q", out)
	}
	if strings.Contains(mustReadString(t, rc), envFileMarker) {
		t.Fatalf("rc still has gu block: %q", mustReadString(t, rc))
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "gu", "env.sh")); !os.IsNotExist(err) {
		t.Fatal("env file still exists")
	}
}

func TestEnvUnsetCancelledExitsZero(t *testing.T) {
	root := t.TempDir()
	home := envTestHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("# user\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withGlobals(t, GlobalOptions{Home: root})
	withTTY(t, true)
	envShell = "zsh"
	envRC = rc
	t.Cleanup(func() { envShell, envRC = "", "" })

	if _, _, err := runCopiedCommand(t, envSetCmd, "", nil); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runCopiedCommand(t, envUnsetCmd, "n\n", nil)
	if err != nil {
		t.Fatalf("cancelled unset error = %v", err)
	}
	if !strings.Contains(stderr, "취소") {
		t.Fatalf("stderr = %q", stderr)
	}
	if !strings.Contains(mustReadString(t, rc), envFileMarker) {
		t.Fatal("cancelled unset removed the block")
	}
}

func TestEnvSetRejectsCorruptedRC(t *testing.T) {
	root := t.TempDir()
	home := envTestHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("# >>> gu initialize >>>\nunterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withGlobals(t, GlobalOptions{Home: root})
	envShell = "zsh"
	envRC = rc
	t.Cleanup(func() { envShell, envRC = "", "" })

	_, _, err := runCopiedCommand(t, envSetCmd, "", nil)
	if err == nil {
		t.Fatal("error = nil, want corrupted block error")
	}
	if got := cli.ExitCode(err); got != 1 {
		t.Fatalf("ExitCode() = %d, want 1 (%v)", got, err)
	}
}

func TestEnvSetViaRootParsesRCFlag(t *testing.T) {
	root := t.TempDir()
	home := envTestHome(t)
	rc := filepath.Join(home, ".myrc")

	out, _, err := runRootCommand(t, "env", "set", "--shell", "bash", "--rc", rc, "--home", root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, rc) {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(mustReadString(t, rc), envFileMarker) {
		t.Fatalf("rc content = %q", mustReadString(t, rc))
	}
}

const envFileMarker = "gu initialize"

func mustReadString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// withEnvWindows swaps the Windows routing seams and restores them on cleanup.
func withEnvWindows(t *testing.T, isWindows bool, persist func(shellenv.WindowsOptions) (shellenv.WindowsResult, error), unpersist func(shellenv.WindowsOptions) (shellenv.WindowsResult, error)) {
	t.Helper()
	previousWin, previousPersist, previousUnpersist := envIsWindows, envPersistWindows, envUnpersistWindows
	envIsWindows = isWindows
	if persist != nil {
		envPersistWindows = persist
	}
	if unpersist != nil {
		envUnpersistWindows = unpersist
	}
	t.Cleanup(func() {
		envIsWindows, envPersistWindows, envUnpersistWindows = previousWin, previousPersist, previousUnpersist
	})
}

func TestUseWindowsUserPathDecision(t *testing.T) {
	tests := []struct {
		name      string
		isWindows bool
		shell     shellenv.Shell
		rc        string
		want      bool
	}{
		{"windows powershell", true, shellenv.ShellPowerShell, "", true},
		{"windows cmd", true, shellenv.ShellCmd, "", true},
		{"windows git bash with rc", true, shellenv.ShellBash, "/tmp/rc", false},
		{"linux powershell", false, shellenv.ShellPowerShell, "", false},
		{"windows with explicit rc", true, shellenv.ShellPowerShell, "/tmp/rc", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withEnvWindows(t, tt.isWindows, nil, nil)
			envRC = tt.rc
			t.Cleanup(func() { envRC = "" })
			if got := useWindowsUserPath(tt.shell); got != tt.want {
				t.Fatalf("useWindowsUserPath(%q) = %v, want %v", tt.shell, got, tt.want)
			}
		})
	}
}

func TestEnvSetWindowsUsesUserPath(t *testing.T) {
	root := t.TempDir()
	withGlobals(t, GlobalOptions{Home: root})
	var captured shellenv.WindowsOptions
	withEnvWindows(t, true, func(opts shellenv.WindowsOptions) (shellenv.WindowsResult, error) {
		captured = opts
		return shellenv.WindowsResult{BinDir: opts.BinDir, Changed: true}, nil
	}, nil)
	envShell = "powershell"
	t.Cleanup(func() { envShell = "" })

	out, _, err := runCopiedCommand(t, envSetCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "current", "bin")
	if captured.BinDir != want {
		t.Fatalf("BinDir = %q, want %q", captured.BinDir, want)
	}
	if !strings.Contains(out, want) {
		t.Fatalf("output = %q", out)
	}
}

func TestEnvSetWindowsReportsAlreadySet(t *testing.T) {
	root := t.TempDir()
	withGlobals(t, GlobalOptions{Home: root})
	withEnvWindows(t, true, func(opts shellenv.WindowsOptions) (shellenv.WindowsResult, error) {
		return shellenv.WindowsResult{BinDir: opts.BinDir, AlreadySet: true}, nil
	}, nil)
	envShell = "powershell"
	t.Cleanup(func() { envShell = "" })

	out, _, err := runCopiedCommand(t, envSetCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "이미 포함") {
		t.Fatalf("output = %q", out)
	}
}

func TestEnvUnsetWindowsRequiresConfirmation(t *testing.T) {
	root := t.TempDir()
	withGlobals(t, GlobalOptions{Home: root, NoInput: true})
	called := false
	withEnvWindows(t, true, nil, func(shellenv.WindowsOptions) (shellenv.WindowsResult, error) {
		called = true
		return shellenv.WindowsResult{}, nil
	})
	envShell = "powershell"
	t.Cleanup(func() { envShell = "" })

	_, _, err := runCopiedCommand(t, envUnsetCmd, "", nil)
	if err == nil {
		t.Fatal("error = nil, want usage error")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
	if called {
		t.Fatal("unpersist ran without confirmation")
	}
}

func TestEnvUnsetWindowsRemovesWithYes(t *testing.T) {
	root := t.TempDir()
	withGlobals(t, GlobalOptions{Home: root, Yes: true})
	withEnvWindows(t, true, nil, func(opts shellenv.WindowsOptions) (shellenv.WindowsResult, error) {
		return shellenv.WindowsResult{BinDir: opts.BinDir, Changed: true}, nil
	})
	envShell = "powershell"
	t.Cleanup(func() { envShell = "" })

	out, _, err := runCopiedCommand(t, envUnsetCmd, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "제거했습니다") {
		t.Fatalf("output = %q", out)
	}
}

func TestEnvInitModuleUsesConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", t.TempDir())
	writePrivateConfig(t, root, "/root-cache")
	withGlobals(t, GlobalOptions{Home: root})
	envModule = true
	t.Cleanup(func() { envModule = false })

	out, _, err := runCopiedCommand(t, envInitCmd, "", []string{"bash"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "export GOMODCACHE='/root-cache'") {
		t.Fatalf("output = %q", out)
	}
}
