package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/systemgo"
)

// withGlobals swaps the shared flag values for one test and restores them on
// cleanup so tests never depend on execution order.
func withGlobals(t *testing.T, opts GlobalOptions) {
	t.Helper()
	previous := globals
	globals = opts
	t.Cleanup(func() { globals = previous })
}

// withSystemSeams replaces the system detection/removal boundary so tests
// never inspect or delete real system paths.
func withSystemSeams(t *testing.T, detect func() ([]systemgo.Present, error), remove func([]systemgo.PlannedCommand) error) {
	t.Helper()
	previousDetect, previousRemove := detectSystemGo, removeSystemGo
	detectSystemGo, removeSystemGo = detect, remove
	t.Cleanup(func() {
		detectSystemGo, removeSystemGo = previousDetect, previousRemove
	})
}

// withTTY replaces the terminal seam so confirmation prompts can be exercised
// without a real terminal.
func withTTY(t *testing.T, isTTY bool) {
	t.Helper()
	previous := promptIsTTY
	promptIsTTY = func(io.Reader) bool { return isTTY }
	t.Cleanup(func() { promptIsTTY = previous })
}

// runRootCommand executes the real root command tree so flag parsing, argument
// validation, and exit-code classification are exercised end to end. Writer
// injection targets the shared root command because child commands resolve
// their writers through the parent chain.
func runRootCommand(t *testing.T, args ...string) (stdout string, stderr string, err error) {
	t.Helper()
	previousOut, previousErr := rootCmd.OutOrStdout(), rootCmd.ErrOrStderr()
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	defer func() {
		rootCmd.SetOut(previousOut)
		rootCmd.SetErr(previousErr)
	}()

	err = rootCmd.ExecuteArgs(args)
	return out.String(), errOut.String(), err
}

// runCopiedCommand runs a command copy with injected streams.
func runCopiedCommand(t *testing.T, command *cli.Command, in string, args []string) (stdout string, stderr string, err error) {
	t.Helper()
	cmd := *command
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if in != "" {
		cmd.SetIn(strings.NewReader(in))
	}
	err = cmd.RunE(&cmd, args)
	return out.String(), errOut.String(), err
}
