package cmd

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/prompt"
)

func TestConfirmSkipsPromptWithYesFlag(t *testing.T) {
	withGlobals(t, GlobalOptions{Yes: true})

	cmd := *cleanCmd
	var errOut strings.Builder
	cmd.SetErr(&errOut)

	approved, err := confirm(&cmd, "계속?")
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Fatal("confirm() = false, want true")
	}
	if errOut.String() != "" {
		t.Fatalf("prompt should be skipped with --yes, got %q", errOut.String())
	}
}

func TestConfirmReportsMissingInputAsUsageError(t *testing.T) {
	withGlobals(t, GlobalOptions{NoInput: true})

	cmd := *cleanCmd
	cmd.SetErr(&strings.Builder{})

	_, err := confirm(&cmd, "계속?")
	if !errors.Is(err, prompt.ErrInputRequired) {
		t.Fatalf("confirm() error = %v, want ErrInputRequired", err)
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2", got)
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("error should mention the approve flag: %v", err)
	}
}

func TestConfirmReadsInteractiveAnswer(t *testing.T) {
	withGlobals(t, GlobalOptions{})
	withTTY(t, true)

	cmd := *cleanCmd
	var errOut strings.Builder
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetErr(&errOut)

	approved, err := confirm(&cmd, "계속?")
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Fatal("confirm() = false, want true")
	}
	if !strings.Contains(errOut.String(), "계속? [y/N]: ") {
		t.Fatalf("prompt = %q", errOut.String())
	}
}

func TestConfirmFailsOutsideTerminal(t *testing.T) {
	withGlobals(t, GlobalOptions{})
	withTTY(t, false)

	cmd := *cleanCmd
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetErr(io.Discard)

	_, err := confirm(&cmd, "계속?")
	if !errors.Is(err, prompt.ErrInputRequired) {
		t.Fatalf("confirm() error = %v, want ErrInputRequired", err)
	}
}
