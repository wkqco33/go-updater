package prompt

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func alwaysTTY(io.Reader) bool { return true }
func neverTTY(io.Reader) bool  { return false }

func TestConfirmApprovesWithYesFlagWithoutPrompting(t *testing.T) {
	var out strings.Builder
	got, err := Confirm(Options{In: strings.NewReader(""), Out: &out, AssumeYes: true, IsTTY: neverTTY}, "삭제할까요?", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("Confirm() = false, want true")
	}
	if out.String() != "" {
		t.Fatalf("prompt should be skipped, got %q", out.String())
	}
}

func TestConfirmFailsWithoutTTY(t *testing.T) {
	for _, opts := range []Options{
		{In: strings.NewReader("y\n"), Out: io.Discard, IsTTY: neverTTY},
		{In: strings.NewReader("y\n"), Out: io.Discard, NoInput: true, IsTTY: alwaysTTY},
	} {
		_, err := Confirm(opts, "삭제할까요?", "--yes")
		if !errors.Is(err, ErrInputRequired) {
			t.Fatalf("Confirm() error = %v, want ErrInputRequired", err)
		}
		if !strings.Contains(err.Error(), "--yes") {
			t.Fatalf("error should point at the approve flag: %v", err)
		}
	}
}

func TestConfirmFailsWhenInputCannotBeRead(t *testing.T) {
	var out strings.Builder
	_, err := Confirm(Options{In: strings.NewReader(""), Out: &out, IsTTY: alwaysTTY}, "삭제할까요?", "--yes")
	if !errors.Is(err, ErrInputRequired) {
		t.Fatalf("Confirm() error = %v, want ErrInputRequired", err)
	}
}

func TestConfirmReadsAnswers(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"lowercase yes", "y\n", true},
		{"uppercase yes", "Y\n", true},
		{"no", "n\n", false},
		{"empty", "\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			got, err := Confirm(Options{In: strings.NewReader(tt.in), Out: &out, IsTTY: alwaysTTY}, "삭제할까요?", "--yes")
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("Confirm() = %v, want %v", got, tt.want)
			}
			if !strings.Contains(out.String(), "삭제할까요? [y/N]: ") {
				t.Fatalf("prompt = %q", out.String())
			}
		})
	}
}

func TestIsTerminalRejectsNonFiles(t *testing.T) {
	if IsTerminal(strings.NewReader("y")) {
		t.Fatal("IsTerminal() = true for a string reader")
	}
}

func TestIsTerminalRejectsRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(path, []byte("y\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if IsTerminal(file) {
		t.Fatal("IsTerminal() = true for a regular file")
	}
}
