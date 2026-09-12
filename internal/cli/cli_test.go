package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestCommandRoutesNestedCommandAndInheritsOutput(t *testing.T) {
	var out bytes.Buffer
	root := &Command{Use: "app"}
	child := &Command{Use: "child", RunE: func(cmd *Command, args []string) error {
		if len(args) != 1 || args[0] != "value" {
			t.Fatalf("args = %#v", args)
		}
		_, err := cmd.OutOrStdout().Write([]byte("ok"))
		return err
	}}
	root.AddCommand(child)
	root.SetOut(&out)

	if err := root.ExecuteArgs([]string{"child", "value"}); err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if out.String() != "ok" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestArgumentValidators(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*Command, []string) error
		args []string
		want bool
	}{
		{"exact success", ExactArgs(1), []string{"x"}, false},
		{"exact failure", ExactArgs(1), nil, true},
		{"minimum failure", MinimumNArgs(2), []string{"x"}, true},
		{"maximum failure", MaximumNArgs(1), []string{"x", "y"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(&Command{}, tt.args) != nil; got != tt.want {
				t.Fatalf("error = %v, want error=%v", got, tt.want)
			}
		})
	}
}

func TestCommandPassesExecutionContext(t *testing.T) {
	want := context.WithValue(context.Background(), "key", "value")
	root := &Command{Use: "app", RunE: func(cmd *Command, args []string) error {
		if cmd.Context().Value("key") != "value" {
			t.Fatalf("context value missing")
		}
		return nil
	}}
	if err := root.ExecuteContext(want, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCommandResetsFlagsBetweenExecutions(t *testing.T) {
	var value bool
	root := &Command{Use: "app", RunE: func(cmd *Command, args []string) error { return nil }}
	root.Flags().BoolVar(&value, "flag", "", false, "test")
	if err := root.ExecuteArgs([]string{"--flag"}); err != nil {
		t.Fatal(err)
	}
	if !value {
		t.Fatal("flag was not parsed")
	}
	if err := root.ExecuteArgs(nil); err != nil {
		t.Fatal(err)
	}
	if value {
		t.Fatal("flag value was not reset")
	}
}

func TestCommandVersionFlagPrintsVersion(t *testing.T) {
	var out bytes.Buffer
	root := &Command{Use: "app", Version: "1.2.3"}
	root.SetOut(&out)

	if err := root.ExecuteArgs([]string{"--version"}); err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "1.2.3" {
		t.Fatalf("version output = %q, want 1.2.3", got)
	}
}

func newFlaggedRoot(run func(*Command, []string) error) *Command {
	root := &Command{Use: "app", RunE: run}
	root.Flags().BoolVar(new(bool), "flag", "", false, "test")
	return root
}

func TestExitCodeSeparatesUsageAndRuntimeErrors(t *testing.T) {
	runtimeErr := errors.New("boom")
	tests := []struct {
		name string
		root *Command
		args []string
		want int
	}{
		{
			name: "success",
			root: &Command{Use: "app", RunE: func(*Command, []string) error { return nil }},
			want: 0,
		},
		{
			name: "argument count",
			root: &Command{Use: "app", Args: ExactArgs(1), RunE: func(*Command, []string) error { return nil }},
			want: 2,
		},
		{
			name: "unknown flag",
			root: newFlaggedRoot(func(*Command, []string) error { return nil }),
			args: []string{"--nope"},
			want: 2,
		},
		{
			name: "unknown command",
			root: &Command{Use: "app", Args: NoArgs, RunE: func(*Command, []string) error { return nil }},
			args: []string{"missing"},
			want: 2,
		},
		{
			name: "runtime failure",
			root: &Command{Use: "app", RunE: func(*Command, []string) error { return runtimeErr }},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.root.SetOut(io.Discard)
			tt.root.SetErr(io.Discard)
			err := tt.root.ExecuteArgs(tt.args)
			if got := ExitCode(err); got != tt.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", err, got, tt.want)
			}
		})
	}
}

func TestHelpShowsFullCommandPath(t *testing.T) {
	var out bytes.Buffer
	root := &Command{Use: "gu"}
	child := &Command{Use: "install [version]", Short: "install Go"}
	root.AddCommand(child)
	root.SetOut(&out)

	child.Help()
	if !strings.Contains(out.String(), "gu install [version]") {
		t.Fatalf("help output = %q, want full command path", out.String())
	}
}
