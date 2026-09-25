// Package shellenv turns the gu-managed Go toolchain location into
// shell-specific setup: an eval-able snippet for the current session and a
// small, revertible block in the user's shell startup files. It never writes
// to a process stream on its own and exposes the platform differences (POSIX
// shells, fish, PowerShell, cmd) behind one Shell value.
package shellenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Shell identifies a supported shell family.
type Shell string

const (
	ShellZsh        Shell = "zsh"
	ShellBash       Shell = "bash"
	ShellPosix      Shell = "sh"
	ShellFish       Shell = "fish"
	ShellPowerShell Shell = "powershell"
	ShellCmd        Shell = "cmd"
)

// Parse normalizes a user-supplied shell name or path.
func Parse(name string) (Shell, error) {
	switch strings.ToLower(filepath.Base(strings.TrimSpace(name))) {
	case "zsh":
		return ShellZsh, nil
	case "bash":
		return ShellBash, nil
	case "sh", "dash", "ash", "ksh", "posix":
		return ShellPosix, nil
	case "fish":
		return ShellFish, nil
	case "powershell", "pwsh", "ps1":
		return ShellPowerShell, nil
	case "cmd", "cmd.exe", "batch":
		return ShellCmd, nil
	}
	return "", fmt.Errorf("지원하지 않는 셸입니다: %q", name)
}

// Detect maps the login shell reported by the environment to a supported
// shell. On Windows there is no login-shell variable, so PowerShell (which gu
// also uses to update the user PATH) is the default unless shellEnv names one.
func Detect(goos, shellEnv string) (Shell, error) {
	if goos == "windows" {
		if trimmed := strings.TrimSpace(shellEnv); trimmed != "" {
			if shell, err := Parse(trimmed); err == nil {
				return shell, nil
			}
		}
		return ShellPowerShell, nil
	}
	trimmed := strings.TrimSpace(shellEnv)
	if trimmed == "" {
		return "", fmt.Errorf("셸을 감지할 수 없습니다: SHELL 환경변수가 비어 있습니다 (--shell로 지정하세요)")
	}
	shell, err := Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w (--shell로 지정하세요)", err)
	}
	return shell, nil
}

// SingleQuote wraps value so a POSIX shell evaluates it literally, escaping
// embedded single quotes. The canonical implementation is shared with
// `gu private env` so both commands quote identically.
func SingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// fishQuote wraps value in fish single quotes, which escape backslash and
// single quote with a backslash instead of POSIX concatenation.
func fishQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "'", `\'`)
	return "'" + value + "'"
}

// powerShellQuote wraps value in a PowerShell single-quoted string, where an
// embedded single quote is doubled.
func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// EnvFile returns the gu-owned file that holds the environment for shell.
// configHome is $XDG_CONFIG_HOME; when empty it falls back to home/.config.
// cmd has no startup file gu can own, so it is rejected.
func EnvFile(shell Shell, home, configHome string) (string, error) {
	base := strings.TrimSpace(configHome)
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "gu")
	switch shell {
	case ShellZsh, ShellBash, ShellPosix:
		return filepath.Join(dir, "env.sh"), nil
	case ShellFish:
		return filepath.Join(dir, "env.fish"), nil
	case ShellPowerShell:
		return filepath.Join(dir, "env.ps1"), nil
	}
	return "", fmt.Errorf("셸 %q은(는) 전용 환경 파일을 사용하지 않습니다", shell)
}

// RCFiles returns the shell startup files gu may insert its source line into.
// Only files that already exist are returned so gu never creates a spurious
// dotfile; when none exist the primary file is returned for creation.
func RCFiles(shell Shell, home string) ([]string, error) {
	switch shell {
	case ShellZsh:
		return []string{filepath.Join(home, ".zshrc")}, nil
	case ShellFish:
		return []string{filepath.Join(home, ".config", "fish", "config.fish")}, nil
	case ShellPosix:
		return []string{filepath.Join(home, ".profile")}, nil
	case ShellBash:
		candidates := []string{
			filepath.Join(home, ".bashrc"),
			filepath.Join(home, ".bash_profile"),
			filepath.Join(home, ".profile"),
		}
		existing := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				existing = append(existing, candidate)
			}
		}
		if len(existing) > 0 {
			return existing, nil
		}
		return candidates[:1], nil
	}
	return nil, fmt.Errorf("셸 %q은(는) 시작 파일에 설정을 저장하지 않습니다", shell)
}

// SourceLine returns the one-line startup-file entry that loads envFile.
func SourceLine(shell Shell, envFile string) (string, error) {
	switch shell {
	case ShellZsh, ShellBash, ShellPosix:
		return fmt.Sprintf("[ -f %s ] && . %s", SingleQuote(envFile), SingleQuote(envFile)), nil
	case ShellFish:
		return fmt.Sprintf("test -f %s; and source %s", fishQuote(envFile), fishQuote(envFile)), nil
	case ShellPowerShell:
		return fmt.Sprintf(". %s", powerShellQuote(envFile)), nil
	}
	return "", fmt.Errorf("셸 %q에는 시작 파일 한 줄을 만들 수 없습니다", shell)
}
