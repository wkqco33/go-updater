package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandRoutesVersionAndInheritsOutput(t *testing.T) {
	oldOut := rootCmd.OutOrStdout()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	t.Cleanup(func() { rootCmd.SetOut(oldOut) })

	if err := rootCmd.ExecuteArgs([]string{"version"}); err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if !strings.Contains(out.String(), "gu version 0.1.0") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRootCommandRejectsInvalidArgumentCount(t *testing.T) {
	if err := rootCmd.ExecuteArgs([]string{"use"}); err == nil {
		t.Fatal("ExecuteArgs() error = nil, want argument validation error")
	}
}

func TestRootCommandRoutesNestedPrivateCommand(t *testing.T) {
	var out bytes.Buffer
	oldOut := rootCmd.OutOrStdout()
	rootCmd.SetOut(&out)
	t.Cleanup(func() { rootCmd.SetOut(oldOut) })

	if err := rootCmd.ExecuteArgs([]string{"private", "env"}); err != nil {
		t.Fatalf("ExecuteArgs() error = %v", err)
	}
	if !strings.Contains(out.String(), "GOMODCACHE=") {
		t.Fatalf("output = %q", out.String())
	}
}
