package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/prompt"
)

// GlobalOptions holds the flags shared by every command.
type GlobalOptions struct {
	Debug   bool
	Yes     bool
	NoInput bool
	Quiet   bool
	NoColor bool
	DryRun  bool
	Home    string
}

// globals is the single home for command-shared flag values. Tests replace it
// through withGlobals, which restores the previous value in t.Cleanup.
var globals GlobalOptions

// promptIsTTY is the terminal seam used by confirm; tests override it to
// exercise the interactive path without a real terminal.
var promptIsTTY = prompt.IsTerminal

func registerGlobalFlags(root *cli.Command) {
	flags := root.PersistentFlags()
	flags.BoolVar(&globals.Debug, "debug", "", false, "디버그 로그를 stderr로 출력합니다")
	flags.BoolVar(&globals.Yes, "yes", "y", false, "확인 프롬프트를 자동 승인합니다 (비대화형/CI)")
	flags.BoolVar(&globals.NoInput, "no-input", "", false, "모든 대화형 프롬프트를 금지합니다")
	flags.BoolVar(&globals.Quiet, "quiet", "q", false, "진행/상태 메시지를 출력하지 않습니다")
	flags.BoolVar(&globals.NoColor, "no-color", "", false, "색상 출력을 끕니다 (NO_COLOR 환경변수와 동일)")
	flags.BoolVar(&globals.DryRun, "dry-run", "n", false, "삭제·설치를 수행하지 않고 계획만 출력합니다")
	flags.StringVar(&globals.Home, "home", "", "", "gu가 버전과 캐시를 관리할 루트 (기본값: $GU_HOME 또는 ~/.go)")
}

// statusf writes progress or status text to stderr so stdout stays reserved
// for results. --quiet suppresses it.
func statusf(cmd *cli.Command, format string, args ...any) {
	if globals.Quiet {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
}

// colorEnabled reports whether human-readable output may use ANSI color.
func colorEnabled(w io.Writer) bool {
	if globals.NoColor || os.Getenv("NO_COLOR") != "" {
		return false
	}
	return prompt.IsTerminalWriter(w)
}

// confirm asks the user to approve a destructive action. A missing input
// capability is reported as a usage error so scripts fail loudly instead of
// silently skipping the operation.
func confirm(cmd *cli.Command, question string) (bool, error) {
	approved, err := prompt.Confirm(prompt.Options{
		In:        cmd.InOrStdin(),
		Out:       cmd.ErrOrStderr(),
		AssumeYes: globals.Yes,
		NoInput:   globals.NoInput,
		IsTTY:     promptIsTTY,
	}, question, "--yes")
	if errors.Is(err, prompt.ErrInputRequired) {
		return false, cli.NewUsageError(err)
	}
	return approved, err
}
