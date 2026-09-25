package shellenv

import (
	"errors"
	"strings"
	"testing"
)

type runnerStep struct {
	out string
	err error
}

// recordingRunner captures every command invocation and replays canned stdout
// so Windows PATH handling can be tested without spawning PowerShell.
type recordingRunner struct {
	calls  [][]string
	script []runnerStep
}

func (r *recordingRunner) run(name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if len(r.calls) > len(r.script) {
		return nil, errors.New("unexpected command invocation")
	}
	step := r.script[len(r.calls)-1]
	return []byte(step.out), step.err
}

func (r *recordingRunner) lastCommand() []string {
	if len(r.calls) == 0 {
		return nil
	}
	return r.calls[len(r.calls)-1]
}

func TestPersistWindowsPrependsBinDir(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{
		{out: `C:\Windows\system32;C:\Go\bin`},
		{},
	}}
	result, err := PersistWindows(WindowsOptions{
		BinDir: `C:\Users\u\.go\current\bin`,
		Runner: runner.run,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.AlreadySet {
		t.Fatalf("result = %+v", result)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("calls = %d, want 2 (read + set)", len(runner.calls))
	}
	if got := runner.calls[0][0]; got != powershellBinary {
		t.Fatalf("read command = %q, want %q", got, powershellBinary)
	}
	setArgs := strings.Join(runner.lastCommand(), " ")
	if !strings.Contains(setArgs, "SetEnvironmentVariable") {
		t.Fatalf("set command = %q", setArgs)
	}
	if !strings.Contains(setArgs, `C:\Users\u\.go\current\bin;C:\Windows\system32;C:\Go\bin`) {
		t.Fatalf("set command does not prepend the bin dir: %q", setArgs)
	}
	if strings.Contains(setArgs, "setx") {
		t.Fatalf("setx must never be used: %q", setArgs)
	}
}

func TestPersistWindowsIsIdempotent(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{
		{out: `C:\Users\u\.go\current\bin;C:\Windows`},
	}}
	result, err := PersistWindows(WindowsOptions{
		BinDir: `C:\Users\u\.go\current\bin`,
		Runner: runner.run,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadySet || result.Changed {
		t.Fatalf("result = %+v", result)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %d, want only the read", len(runner.calls))
	}
}

func TestPersistWindowsMatchesCaseInsensitively(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{
		{out: `C:\USERS\U\.GO\CURRENT\BIN`},
	}}
	result, err := PersistWindows(WindowsOptions{
		BinDir: `c:\users\u\.go\current\bin`,
		Runner: runner.run,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadySet {
		t.Fatalf("result = %+v, want AlreadySet", result)
	}
}

func TestPersistWindowsDryRunDoesNotSet(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{{out: `C:\Windows`}}}
	result, err := PersistWindows(WindowsOptions{
		BinDir: `C:\go\bin`,
		DryRun: true,
		Runner: runner.run,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatalf("result = %+v, want Changed", result)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("dry-run ran %d commands, want only the read", len(runner.calls))
	}
}

func TestPersistWindowsReadFailureIsReported(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{{err: errors.New("powershell missing")}}}
	_, err := PersistWindows(WindowsOptions{BinDir: `C:\go\bin`, Runner: runner.run})
	if err == nil {
		t.Fatal("error = nil, want read failure")
	}
	if !strings.Contains(err.Error(), "user PATH") {
		t.Fatalf("error = %v", err)
	}
}

func TestPersistWindowsSetFailureIsReported(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{{out: `C:\Windows`}, {err: errors.New("access denied")}}}
	_, err := PersistWindows(WindowsOptions{BinDir: `C:\go\bin`, Runner: runner.run})
	if err == nil {
		t.Fatal("error = nil, want set failure")
	}
}

func TestPersistWindowsEmptyUserPath(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{{out: "\r\n"}, {}}}
	result, err := PersistWindows(WindowsOptions{BinDir: `C:\go\bin`, Runner: runner.run})
	if err != nil {
		t.Fatal(err)
	}
	if result.UserPath != "" {
		t.Fatalf("UserPath = %q, want empty", result.UserPath)
	}
	if !strings.Contains(strings.Join(runner.lastCommand(), " "), `'C:\go\bin'`) {
		t.Fatalf("set command = %q", runner.lastCommand())
	}
}

func TestPersistWindowsEscapesSingleQuote(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{{out: ""}, {}}}
	if _, err := PersistWindows(WindowsOptions{BinDir: `C:\it's\bin`, Runner: runner.run}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(runner.lastCommand(), " "), `C:\it''s\bin`) {
		t.Fatalf("set command did not escape the quote: %q", runner.lastCommand())
	}
}

func TestPersistWindowsRejectsEmptyBinDir(t *testing.T) {
	runner := &recordingRunner{}
	if _, err := PersistWindows(WindowsOptions{Runner: runner.run}); err == nil {
		t.Fatal("error = nil, want missing bin dir error")
	}
	if len(runner.calls) != 0 {
		t.Fatal("validation failure still ran a command")
	}
}

func TestUnpersistWindowsRemovesEntry(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{
		{out: `C:\go\bin;C:\Windows;C:\Go\bin`},
		{},
	}}
	result, err := UnpersistWindows(WindowsOptions{BinDir: `C:\go\bin`, Runner: runner.run})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatalf("result = %+v, want Changed", result)
	}
	// Windows paths are case-insensitive, so every case variant of the same
	// directory is removed and the unrelated entries survive.
	setArgs := strings.Join(runner.lastCommand(), " ")
	if !strings.Contains(setArgs, `'C:\Windows'`) {
		t.Fatalf("set command = %q", setArgs)
	}
	if strings.Contains(setArgs, `C:\Go\bin`) {
		t.Fatalf("case variant survived: %q", setArgs)
	}
}

func TestUnpersistWindowsKeepsPathWhenEntryAbsent(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{{out: `C:\Windows`}}}
	result, err := UnpersistWindows(WindowsOptions{BinDir: `C:\go\bin`, Runner: runner.run})
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed {
		t.Fatalf("result = %+v, want unchanged", result)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %d, want only the read", len(runner.calls))
	}
}

func TestUnpersistWindowsClearsPathWhenLastEntry(t *testing.T) {
	runner := &recordingRunner{script: []runnerStep{{out: `C:\go\bin`}, {}}}
	if _, err := UnpersistWindows(WindowsOptions{BinDir: `C:\go\bin`, Runner: runner.run}); err != nil {
		t.Fatal(err)
	}
	setArgs := strings.Join(runner.lastCommand(), " ")
	if !strings.Contains(setArgs, "$null") {
		t.Fatalf("set command = %q, want $null for an empty PATH", setArgs)
	}
}

func TestWindowsPathContainsIgnoresBlankEntriesAndQuotes(t *testing.T) {
	path := `"C:\go\bin"; ;C:\Windows`
	if !windowsPathContains(path, `C:\go\bin`) {
		t.Fatal("quoted entry was not matched")
	}
	if windowsPathContains(path, `C:\Go`) {
		t.Fatal("partial entry matched")
	}
}
