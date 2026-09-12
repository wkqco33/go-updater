package cmd

import (
	"strings"
	"testing"
)

func TestRootCommandRoutesVersionCommand(t *testing.T) {
	out, _, err := runRootCommand(t, "version")
	if err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if !strings.Contains(out, "gu version dev") {
		t.Fatalf("output = %q", out)
	}
}

func TestRootCommandSupportsVersionFlag(t *testing.T) {
	out, _, err := runRootCommand(t, "--version")
	if err != nil {
		t.Fatalf("--version error = %v", err)
	}
	if !strings.Contains(out, "dev") {
		t.Fatalf("output = %q", out)
	}
}

func TestRootCommandRejectsInvalidArgumentCount(t *testing.T) {
	_, _, err := runRootCommand(t, "use")
	if err == nil {
		t.Fatal("ExecuteArgs() error = nil, want argument validation error")
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
}

func TestRootCommandRejectsUnknownCommand(t *testing.T) {
	_, _, err := runRootCommand(t, "definitely-not-a-command")
	if err == nil {
		t.Fatal("ExecuteArgs() error = nil, want unknown command error")
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
}

func TestRootCommandRoutesNestedPrivateCommand(t *testing.T) {
	t.Setenv("GU_HOME", t.TempDir())

	out, _, err := runRootCommand(t, "private", "env")
	if err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if !strings.Contains(out, "export GOMODCACHE=") {
		t.Fatalf("output = %q", out)
	}
}

func TestRootCommandPrintsHelpWithoutArguments(t *testing.T) {
	out, _, err := runRootCommand(t)
	if err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "install") {
		t.Fatalf("output = %q", out)
	}
}

func TestSubcommandHelpShowsFullPathAndLinks(t *testing.T) {
	out, _, err := runRootCommand(t, "install", "--help")
	if err != nil {
		t.Fatalf("install --help error = %v", err)
	}
	for _, want := range []string{"gu install [version]", "예시:", docsURL, issuesURL} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q:\n%s", want, out)
		}
	}
}

func TestRootHelpListsGlobalFlags(t *testing.T) {
	out, _, err := runRootCommand(t, "--help")
	if err != nil {
		t.Fatalf("--help error = %v", err)
	}
	for _, want := range []string{"--yes", "--no-input", "--quiet", "--no-color", "--dry-run", "--home", "--debug", "--version"} {
		if !strings.Contains(out, want) {
			t.Fatalf("root help missing %q:\n%s", want, out)
		}
	}
}
