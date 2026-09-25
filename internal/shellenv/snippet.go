package shellenv

import (
	"fmt"
	"sort"
	"strings"
)

// Setup is the resolved environment gu wants to configure for one shell.
// Goroot and Gopath are optional; Module carries module-cache variables such
// as GOMODCACHE and GOPRIVATE from the private cache configuration.
type Setup struct {
	Shell  Shell
	BinDir string
	Goroot string
	Gopath string
	Module map[string]string
}

// Snippet returns shell code that applies Setup to the current session. The
// output is safe to eval or source verbatim: every value is quoted for the
// target shell so a path containing $, backticks, or quotes cannot expand.
func Snippet(setup Setup) (string, error) {
	if strings.TrimSpace(setup.BinDir) == "" {
		return "", fmt.Errorf("Go bin 디렉터리를 결정하지 못했습니다")
	}
	var lines []string
	switch setup.Shell {
	case ShellZsh, ShellBash, ShellPosix:
		lines = append(lines, fmt.Sprintf("export PATH=%s:\"$PATH\"", SingleQuote(setup.BinDir)))
		if setup.Goroot != "" {
			lines = append(lines, fmt.Sprintf("export GOROOT=%s", SingleQuote(setup.Goroot)))
		}
		if setup.Gopath != "" {
			lines = append(lines, fmt.Sprintf("export GOPATH=%s", SingleQuote(setup.Gopath)))
		}
		for _, key := range sortedKeys(setup.Module) {
			lines = append(lines, fmt.Sprintf("export %s=%s", key, SingleQuote(setup.Module[key])))
		}
	case ShellFish:
		lines = append(lines, fmt.Sprintf("fish_add_path %s", fishQuote(setup.BinDir)))
		if setup.Goroot != "" {
			lines = append(lines, fmt.Sprintf("set -gx GOROOT %s", fishQuote(setup.Goroot)))
		}
		if setup.Gopath != "" {
			lines = append(lines, fmt.Sprintf("set -gx GOPATH %s", fishQuote(setup.Gopath)))
		}
		for _, key := range sortedKeys(setup.Module) {
			lines = append(lines, fmt.Sprintf("set -gx %s %s", key, fishQuote(setup.Module[key])))
		}
	case ShellPowerShell:
		lines = append(lines, fmt.Sprintf("$env:Path = %s + ';' + $env:Path", powerShellQuote(setup.BinDir)))
		if setup.Goroot != "" {
			lines = append(lines, fmt.Sprintf("$env:GOROOT = %s", powerShellQuote(setup.Goroot)))
		}
		if setup.Gopath != "" {
			lines = append(lines, fmt.Sprintf("$env:GOPATH = %s", powerShellQuote(setup.Gopath)))
		}
		for _, key := range sortedKeys(setup.Module) {
			lines = append(lines, fmt.Sprintf("$env:%s = %s", key, powerShellQuote(setup.Module[key])))
		}
	case ShellCmd:
		lines = append(lines, fmt.Sprintf("set \"PATH=%s;%%PATH%%\"", setup.BinDir))
		if setup.Goroot != "" {
			lines = append(lines, fmt.Sprintf("set \"GOROOT=%s\"", setup.Goroot))
		}
		if setup.Gopath != "" {
			lines = append(lines, fmt.Sprintf("set \"GOPATH=%s\"", setup.Gopath))
		}
		for _, key := range sortedKeys(setup.Module) {
			lines = append(lines, fmt.Sprintf("set \"%s=%s\"", key, setup.Module[key]))
		}
	default:
		return "", fmt.Errorf("지원하지 않는 셸입니다: %q", setup.Shell)
	}
	return strings.Join(lines, "\n") + "\n", nil
}

// EnvFileContent returns the body of the gu-owned env file: a managed-by
// header followed by the same code Snippet produces.
func EnvFileContent(setup Setup) (string, error) {
	snippet, err := Snippet(setup)
	if err != nil {
		return "", err
	}
	return managedHeader(setup.Shell) + "\n" + snippet, nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func managedHeader(shell Shell) string {
	if shell == ShellCmd {
		return "REM Managed by gu. Do not edit; run \"gu env unset\" to remove."
	}
	return "# Managed by gu. Do not edit; run \"gu env unset\" to remove."
}
