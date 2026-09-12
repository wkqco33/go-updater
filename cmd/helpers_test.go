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

// flagSnapshot captures every package-level flag value that executing a
// command can mutate. CI runs the suite with -shuffle=on, so flag state left
// behind by one test must never decide another test's outcome.
type flagSnapshot struct {
	globals                   GlobalOptions
	cleanAll                  bool
	cleanUnused               bool
	cleanSystem               bool
	installDir                string
	listJSON                  bool
	versionJSON               bool
	privateCleanAll           bool
	privateCleanStaleDay      int
	privateCleanMaxSize       int64
	privateOffline            bool
	privateSyncRetries        int
	privateSyncLatestIfMissed bool
	privateSyncSource         string
	privatePatterns           string
	noSumDBPatterns           string
	noProxyPatterns           string
	cacheDirFlag              string
}

func snapshotFlags() flagSnapshot {
	return flagSnapshot{
		globals:                   globals,
		cleanAll:                  cleanAll,
		cleanUnused:               cleanUnused,
		cleanSystem:               cleanSystem,
		installDir:                installDir,
		listJSON:                  listJSON,
		versionJSON:               versionJSON,
		privateCleanAll:           privateCleanAll,
		privateCleanStaleDay:      privateCleanStaleDay,
		privateCleanMaxSize:       privateCleanMaxSize,
		privateOffline:            privateOffline,
		privateSyncRetries:        privateSyncRetries,
		privateSyncLatestIfMissed: privateSyncLatestIfMissed,
		privateSyncSource:         privateSyncSource,
		privatePatterns:           privatePatterns,
		noSumDBPatterns:           noSumDBPatterns,
		noProxyPatterns:           noProxyPatterns,
		cacheDirFlag:              cacheDirFlag,
	}
}

func (s flagSnapshot) restore() {
	globals = s.globals
	cleanAll = s.cleanAll
	cleanUnused = s.cleanUnused
	cleanSystem = s.cleanSystem
	installDir = s.installDir
	listJSON = s.listJSON
	versionJSON = s.versionJSON
	privateCleanAll = s.privateCleanAll
	privateCleanStaleDay = s.privateCleanStaleDay
	privateCleanMaxSize = s.privateCleanMaxSize
	privateOffline = s.privateOffline
	privateSyncRetries = s.privateSyncRetries
	privateSyncLatestIfMissed = s.privateSyncLatestIfMissed
	privateSyncSource = s.privateSyncSource
	privatePatterns = s.privatePatterns
	noSumDBPatterns = s.noSumDBPatterns
	noProxyPatterns = s.noProxyPatterns
	cacheDirFlag = s.cacheDirFlag
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
	snapshot := snapshotFlags()
	defer func() {
		rootCmd.SetOut(previousOut)
		rootCmd.SetErr(previousErr)
		snapshot.restore()
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
