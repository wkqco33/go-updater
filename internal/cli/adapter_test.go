package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestUsageErrorWrapsAndUnwraps(t *testing.T) {
	cause := errors.New("accepts 1 arg(s)")
	usageErr := NewUsageError(cause)

	if !errors.Is(usageErr, cause) {
		t.Fatalf("errors.Is() = false, want the wrapped cause")
	}
	if usageErr.Error() != cause.Error() {
		t.Fatalf("Error() = %q, want %q", usageErr.Error(), cause.Error())
	}
}

func TestExitCodeTreatsNilAsSuccess(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Fatalf("ExitCode(nil) = %d, want 0", got)
	}
}

func TestCommandWritersAndInputInheritFromParent(t *testing.T) {
	root := &Command{Use: "gu"}
	child := &Command{Use: "list"}
	root.AddCommand(child)

	var out, errOut bytes.Buffer
	in := strings.NewReader("answer\n")
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(in)

	if child.OutOrStdout() != &out {
		t.Fatal("child did not inherit stdout")
	}
	if child.ErrOrStderr() != &errOut {
		t.Fatal("child did not inherit stderr")
	}
	if child.InOrStdin() != in {
		t.Fatal("child did not inherit stdin")
	}
	if want := "gu list"; child.fullUse() != want {
		t.Fatalf("fullUse() = %q, want %q", child.fullUse(), want)
	}
}

func TestCommandDefaultsToProcessStreams(t *testing.T) {
	command := &Command{Use: "gu"}
	if command.OutOrStdout() != os.Stdout {
		t.Fatal("OutOrStdout() should default to the process stdout")
	}
	if command.ErrOrStderr() != os.Stderr {
		t.Fatal("ErrOrStderr() should default to the process stderr")
	}
	if command.InOrStdin() != os.Stdin {
		t.Fatal("InOrStdin() should default to the process stdin")
	}
}

func TestFlagSetBindingsAndVisibility(t *testing.T) {
	command := &Command{Use: "gu", RunE: func(cmd *Command, args []string) error {
		if len(args) != 1 || args[0] != "arg" {
			t.Fatalf("args = %#v", args)
		}
		if cmd.Context() == nil {
			t.Fatal("Context() = nil")
		}
		return nil
	}}

	var str string
	var number int
	var big int64
	var persistent bool
	command.Flags().StringVar(&str, "str", "", "value", "test string")
	command.Flags().IntVar(&number, "number", "", 0, "test int")
	command.Flags().Int64Var(&big, "big", "", 0, "test int64")
	command.Flags().StringVarP(&str, "alias", "a", "", "test alias")
	command.PersistentFlags().BoolVar(&persistent, "persist", "", false, "test persistent")

	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	err := command.ExecuteArgs([]string{"--str=hello", "--number", "3", "--big", "9", "-a", "aliased", "--persist", "arg"})
	if err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if str != "aliased" || number != 3 || big != 9 || !persistent {
		t.Fatalf("flags = str:%q number:%d big:%d persist:%v", str, number, big, persistent)
	}
}

func TestChangedTracksFlagUse(t *testing.T) {
	command := &Command{Use: "gu", RunE: func(*Command, []string) error { return nil }}
	var value string
	command.Flags().StringVar(&value, "value", "", "", "test")

	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if command.Flags().Changed("value") {
		t.Fatal("Changed() = true before parsing")
	}
	if err := command.ExecuteArgs([]string{"--value", "set"}); err != nil {
		t.Fatal(err)
	}
	if !command.Flags().Changed("value") {
		t.Fatal("Changed() = false after the flag was set")
	}
	if err := command.ExecuteArgs(nil); err != nil {
		t.Fatal(err)
	}
	if command.Flags().Changed("value") {
		t.Fatal("Changed() = true after flags were reset")
	}
}

func TestNoArgsRejectsPositionalArguments(t *testing.T) {
	if err := NoArgs(&Command{}, nil); err != nil {
		t.Fatalf("NoArgs() with no arguments = %v", err)
	}
	err := NoArgs(&Command{}, []string{"bogus"})
	if err == nil {
		t.Fatal("NoArgs() = nil for a positional argument")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("error = %v, want the offending argument", err)
	}
}

func TestCommandExecuteContextUsesInjectedContext(t *testing.T) {
	type key struct{}
	command := &Command{Use: "gu", RunE: func(cmd *Command, args []string) error {
		if cmd.Context().Value(key{}) != "value" {
			t.Fatal("injected context value missing")
		}
		return nil
	}}
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.ExecuteContext(context.WithValue(context.Background(), key{}, "value"), nil); err != nil {
		t.Fatal(err)
	}
}
