package shellenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAcceptsShellAliases(t *testing.T) {
	tests := map[string]Shell{
		"zsh":        ShellZsh,
		"BASH":       ShellBash,
		" sh ":       ShellPosix,
		"dash":       ShellPosix,
		"fish":       ShellFish,
		"pwsh":       ShellPowerShell,
		"powershell": ShellPowerShell,
		"cmd.exe":    ShellCmd,
	}
	for input, want := range tests {
		got, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("Parse(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseRejectsUnknownShell(t *testing.T) {
	if _, err := Parse("tcsh"); err == nil {
		t.Fatal("Parse(tcsh) error = nil, want unsupported shell error")
	}
}

func TestDetectReadsLoginShell(t *testing.T) {
	got, err := Detect("linux", "/usr/bin/zsh")
	if err != nil {
		t.Fatal(err)
	}
	if got != ShellZsh {
		t.Fatalf("Detect() = %q, want zsh", got)
	}

	got, err = Detect("darwin", "/bin/bash")
	if err != nil {
		t.Fatal(err)
	}
	if got != ShellBash {
		t.Fatalf("Detect() = %q, want bash", got)
	}
}

func TestDetectWithoutShellEnvFails(t *testing.T) {
	if _, err := Detect("linux", ""); err == nil {
		t.Fatal("Detect() error = nil, want missing SHELL error")
	}
}

func TestDetectRejectsUnsupportedLoginShell(t *testing.T) {
	_, err := Detect("linux", "/usr/bin/tcsh")
	if err == nil {
		t.Fatal("Detect() error = nil, want unsupported shell error")
	}
	if !strings.Contains(err.Error(), "--shell") {
		t.Fatalf("error should point at --shell: %v", err)
	}
}

func TestDetectWindowsDefaultsToPowerShell(t *testing.T) {
	got, err := Detect("windows", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != ShellPowerShell {
		t.Fatalf("Detect(windows) = %q, want powershell", got)
	}
}

func TestDetectWindowsHonorsExplicitShell(t *testing.T) {
	got, err := Detect("windows", "cmd")
	if err != nil {
		t.Fatal(err)
	}
	if got != ShellCmd {
		t.Fatalf("Detect(windows, cmd) = %q, want cmd", got)
	}
}

func TestSingleQuoteEscapesMetacharacters(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", "'plain'"},
		{"/tmp/$HOME/bin", "'/tmp/$HOME/bin'"},
		{"it's", `'it'\''s'`},
		{"`whoami`", "'`whoami`'"},
	}
	for _, tt := range tests {
		if got := SingleQuote(tt.in); got != tt.want {
			t.Fatalf("SingleQuote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSnippetPosixPrependsPath(t *testing.T) {
	got, err := Snippet(Setup{Shell: ShellBash, BinDir: "/h/.go/current/bin"})
	if err != nil {
		t.Fatal(err)
	}
	want := "export PATH='/h/.go/current/bin':\"$PATH\"\n"
	if got != want {
		t.Fatalf("Snippet() = %q, want %q", got, want)
	}
}

func TestSnippetFishUsesFishAddPath(t *testing.T) {
	got, err := Snippet(Setup{Shell: ShellFish, BinDir: "/h/.go/current/bin"})
	if err != nil {
		t.Fatal(err)
	}
	want := "fish_add_path '/h/.go/current/bin'\n"
	if got != want {
		t.Fatalf("Snippet() = %q, want %q", got, want)
	}
}

func TestSnippetPowerShellPrependsPath(t *testing.T) {
	got, err := Snippet(Setup{Shell: ShellPowerShell, BinDir: `C:\Users\a\.go\current\bin`})
	if err != nil {
		t.Fatal(err)
	}
	want := "$env:Path = 'C:\\Users\\a\\.go\\current\\bin' + ';' + $env:Path\n"
	if got != want {
		t.Fatalf("Snippet() = %q, want %q", got, want)
	}
}

func TestSnippetCommandPrependsPath(t *testing.T) {
	got, err := Snippet(Setup{Shell: ShellCmd, BinDir: `C:\go\current\bin`})
	if err != nil {
		t.Fatal(err)
	}
	want := "set \"PATH=C:\\go\\current\\bin;%PATH%\"\n"
	if got != want {
		t.Fatalf("Snippet() = %q, want %q", got, want)
	}
}

func TestSnippetIncludesOptionalVariables(t *testing.T) {
	got, err := Snippet(Setup{
		Shell:  ShellZsh,
		BinDir: "/h/bin",
		Goroot: "/h/.go/current",
		Gopath: "/home/u/go",
		Module: map[string]string{"GOPRIVATE": "github.com/acme/*", "GOMODCACHE": "/cache"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`export PATH='/h/bin':"$PATH"`,
		`export GOROOT='/h/.go/current'`,
		`export GOPATH='/home/u/go'`,
		`export GOMODCACHE='/cache'`,
		`export GOPRIVATE='github.com/acme/*'`,
		"",
	}, "\n")
	if got != want {
		t.Fatalf("Snippet() = %q, want %q", got, want)
	}
}

func TestSnippetFishQuotesSingleQuote(t *testing.T) {
	got, err := Snippet(Setup{Shell: ShellFish, BinDir: "/h/it's/bin"})
	if err != nil {
		t.Fatal(err)
	}
	want := "fish_add_path '/h/it\\'s/bin'\n"
	if got != want {
		t.Fatalf("Snippet() = %q, want %q", got, want)
	}
}

func TestSnippetPowerShellDoublesSingleQuote(t *testing.T) {
	got, err := Snippet(Setup{Shell: ShellPowerShell, BinDir: `C:\it's\bin`})
	if err != nil {
		t.Fatal(err)
	}
	want := "$env:Path = 'C:\\it''s\\bin' + ';' + $env:Path\n"
	if got != want {
		t.Fatalf("Snippet() = %q, want %q", got, want)
	}
}

func TestSnippetRequiresBinDir(t *testing.T) {
	if _, err := Snippet(Setup{Shell: ShellZsh}); err == nil {
		t.Fatal("Snippet() error = nil, want missing BinDir error")
	}
}

func TestEnvFileHonorsXDGConfigHome(t *testing.T) {
	got, err := EnvFile(ShellZsh, "/home/u", "/xdg")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/xdg", "gu", "env.sh"); got != want {
		t.Fatalf("EnvFile() = %q, want %q", got, want)
	}

	got, err = EnvFile(ShellFish, "/home/u", "/xdg")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/xdg", "gu", "env.fish"); got != want {
		t.Fatalf("EnvFile(fish) = %q, want %q", got, want)
	}
}

func TestEnvFileFallsBackToHomeDotConfig(t *testing.T) {
	got, err := EnvFile(ShellBash, "/home/u", "")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/home/u", ".config", "gu", "env.sh"); got != want {
		t.Fatalf("EnvFile() = %q, want %q", got, want)
	}
}

func TestEnvFileRejectsCmd(t *testing.T) {
	if _, err := EnvFile(ShellCmd, "/home/u", ""); err == nil {
		t.Fatal("EnvFile(cmd) error = nil, want unsupported error")
	}
}

func TestRCFilesReturnsPrimaryWhenNoneExist(t *testing.T) {
	home := t.TempDir()
	got, err := RCFiles(ShellBash, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(home, ".bashrc") {
		t.Fatalf("RCFiles() = %v, want only .bashrc", got)
	}
}

func TestRCFilesPrefersExistingStartupFile(t *testing.T) {
	home := t.TempDir()
	profile := filepath.Join(home, ".profile")
	if err := os.WriteFile(profile, []byte("# profile"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := RCFiles(ShellBash, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != profile {
		t.Fatalf("RCFiles() = %v, want only %q", got, profile)
	}
}

func TestRCFilesForZshAndFish(t *testing.T) {
	home := t.TempDir()
	got, err := RCFiles(ShellZsh, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(home, ".zshrc") {
		t.Fatalf("RCFiles(zsh) = %v", got)
	}

	got, err = RCFiles(ShellFish, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(home, ".config", "fish", "config.fish") {
		t.Fatalf("RCFiles(fish) = %v", got)
	}
}

func TestRCFilesRejectsPowerShell(t *testing.T) {
	if _, err := RCFiles(ShellPowerShell, t.TempDir()); err == nil {
		t.Fatal("RCFiles(powershell) error = nil, want unsupported error")
	}
}

func TestSourceLinePerShell(t *testing.T) {
	got, err := SourceLine(ShellBash, "/h/.config/gu/env.sh")
	if err != nil {
		t.Fatal(err)
	}
	if want := `[ -f '/h/.config/gu/env.sh' ] && . '/h/.config/gu/env.sh'`; got != want {
		t.Fatalf("SourceLine() = %q, want %q", got, want)
	}

	got, err = SourceLine(ShellFish, "/h/.config/gu/env.fish")
	if err != nil {
		t.Fatal(err)
	}
	if want := `test -f '/h/.config/gu/env.fish'; and source '/h/.config/gu/env.fish'`; got != want {
		t.Fatalf("SourceLine(fish) = %q, want %q", got, want)
	}
}

func TestSourceLinePowerShell(t *testing.T) {
	got, err := SourceLine(ShellPowerShell, `C:\gu\env.ps1`)
	if err != nil {
		t.Fatal(err)
	}
	if want := `. 'C:\gu\env.ps1'`; got != want {
		t.Fatalf("SourceLine(powershell) = %q, want %q", got, want)
	}
}

func TestSourceLineRejectsCmd(t *testing.T) {
	if _, err := SourceLine(ShellCmd, "anything"); err == nil {
		t.Fatal("SourceLine(cmd) error = nil, want unsupported error")
	}
}

func TestSnippetRejectsUnknownShell(t *testing.T) {
	if _, err := Snippet(Setup{Shell: "tcsh", BinDir: "/x"}); err == nil {
		t.Fatal("Snippet() error = nil, want unsupported shell error")
	}
}

func TestRCFilesForPosixShUsesProfile(t *testing.T) {
	home := t.TempDir()
	got, err := RCFiles(ShellPosix, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(home, ".profile") {
		t.Fatalf("RCFiles(sh) = %v", got)
	}
}

func TestEnvFileContentUsesShellComment(t *testing.T) {
	posix, err := EnvFileContent(Setup{Shell: ShellBash, BinDir: "/bin"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(posix, "# Managed by gu.") {
		t.Fatalf("posix header = %q", posix)
	}

	cmd, err := EnvFileContent(Setup{Shell: ShellCmd, BinDir: `C:\bin`})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cmd, "REM Managed by gu.") {
		t.Fatalf("cmd header = %q", cmd)
	}
}
