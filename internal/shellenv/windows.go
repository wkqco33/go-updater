package shellenv

import (
	"fmt"
	"os/exec"
	"strings"
)

// powershellBinary is the program gu uses to read and update the Windows user
// environment. setx is deliberately never used: it truncates PATH at 1024
// characters and expands embedded %VAR% references.
const powershellBinary = "powershell.exe"

// Runner executes an external command and returns its stdout. It is injected
// so tests never spawn PowerShell or change the real user environment.
type Runner func(name string, args ...string) ([]byte, error)

// DefaultRunner executes the command through os/exec.
func DefaultRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// WindowsOptions configures adding BinDir to the Windows user PATH through the
// environment API instead of a shell startup file.
type WindowsOptions struct {
	BinDir string
	DryRun bool
	Runner Runner
}

func (o WindowsOptions) runner() Runner {
	if o.Runner != nil {
		return o.Runner
	}
	return DefaultRunner
}

// WindowsResult reports the user PATH read during the operation.
type WindowsResult struct {
	BinDir string
	// UserPath is the user PATH before the change.
	UserPath string
	// Changed is true when the PATH was updated (or would be under DryRun).
	Changed bool
	// AlreadySet is true when BinDir was already present.
	AlreadySet bool
}

// PersistWindows prepends BinDir to the user PATH and reports the result.
func PersistWindows(opts WindowsOptions) (WindowsResult, error) {
	result := WindowsResult{BinDir: opts.BinDir}
	if strings.TrimSpace(opts.BinDir) == "" {
		return result, fmt.Errorf("Go bin 디렉터리를 결정하지 못했습니다")
	}

	current, err := readWindowsUserPath(opts)
	if err != nil {
		return result, err
	}
	result.UserPath = current

	if windowsPathContains(current, opts.BinDir) {
		result.AlreadySet = true
		return result, nil
	}
	result.Changed = true
	if opts.DryRun {
		return result, nil
	}

	next := opts.BinDir
	if strings.TrimSpace(current) != "" {
		next = opts.BinDir + ";" + current
	}
	if err := setWindowsUserPath(opts, next); err != nil {
		return result, err
	}
	return result, nil
}

// UnpersistWindows removes BinDir from the user PATH, leaving every other
// entry in place.
func UnpersistWindows(opts WindowsOptions) (WindowsResult, error) {
	result := WindowsResult{BinDir: opts.BinDir}

	current, err := readWindowsUserPath(opts)
	if err != nil {
		return result, err
	}
	result.UserPath = current

	next, removed := removeWindowsPathEntry(current, opts.BinDir)
	if !removed {
		return result, nil
	}
	result.Changed = true
	if opts.DryRun {
		return result, nil
	}
	if err := setWindowsUserPath(opts, next); err != nil {
		return result, err
	}
	return result, nil
}

func readWindowsUserPath(opts WindowsOptions) (string, error) {
	out, err := opts.runner()(powershellBinary, "-NoProfile", "-NonInteractive", "-Command",
		`[Environment]::GetEnvironmentVariable('Path','User')`)
	if err != nil {
		return "", fmt.Errorf("failed to read Windows user PATH: %w", err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func setWindowsUserPath(opts WindowsOptions, value string) error {
	literal := "$null"
	if strings.TrimSpace(value) != "" {
		literal = powerShellQuote(value)
	}
	script := fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', %s, 'User')", literal)
	if _, err := opts.runner()(powershellBinary, "-NoProfile", "-NonInteractive", "-Command", script); err != nil {
		return fmt.Errorf("failed to update Windows user PATH: %w", err)
	}
	return nil
}

// windowsPathContains reports whether dir is one of the user PATH entries,
// ignoring case, surrounding quotes, and blank entries.
func windowsPathContains(pathEnv, dir string) bool {
	for _, entry := range splitWindowsPath(pathEnv) {
		if strings.EqualFold(entry, dir) {
			return true
		}
	}
	return false
}

func removeWindowsPathEntry(pathEnv, dir string) (string, bool) {
	entries := splitWindowsPath(pathEnv)
	kept := make([]string, 0, len(entries))
	removed := false
	for _, entry := range entries {
		if strings.EqualFold(entry, dir) {
			removed = true
			continue
		}
		kept = append(kept, entry)
	}
	if !removed {
		return pathEnv, false
	}
	return strings.Join(kept, ";"), true
}

func splitWindowsPath(pathEnv string) []string {
	parts := strings.Split(pathEnv, ";")
	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		trimmed = strings.Trim(trimmed, `"`)
		if trimmed == "" {
			continue
		}
		entries = append(entries, trimmed)
	}
	return entries
}
