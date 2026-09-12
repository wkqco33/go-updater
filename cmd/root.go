package cmd

import (
	"log/slog"

	"github.com/wkqco33/go-updater/internal/cli"
)

// buildVersion is overridden by release builds with -ldflags. Local builds
// report dev. It is declared before rootCmd so --version is populated at startup.
var buildVersion = "dev"

const (
	docsURL   = "https://github.com/wkqco33/go-updater#readme"
	issuesURL = "https://github.com/wkqco33/go-updater/issues"
)

// NewRoot builds the command tree with the shared global flags attached.
func NewRoot() *cli.Command {
	root := &cli.Command{
		Use:     "gu",
		Short:   "Go 언어를 설치하거나 최신 버전으로 업데이트하는 도구입니다.",
		Version: buildVersion,
		Long: `go.dev 릴리스 정보를 바탕으로 Go를 설치하고, 여러 버전을 전환·정리하는 CLI입니다.

예시:
  gu install          최신 안정 버전 설치
  gu install 1.20     1.20.x 최신 패치 설치
  gu use 1.20         설치된 1.20.x로 전환
  gu list             설치된 버전 확인
  gu clean --unused   사용하지 않는 버전 정리

문서: ` + docsURL + `
이슈: ` + issuesURL,
		Args: cli.NoArgs,
		PersistentPreRun: func(cmd *cli.Command, args []string) {
			level := slog.LevelInfo
			if globals.Debug {
				level = slog.LevelDebug
			}
			slog.SetDefault(slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: level})))
		},
		RunE: func(cmd *cli.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}
	registerGlobalFlags(root)
	return root
}

var rootCmd = NewRoot()

// Execute returns command errors to the process boundary instead of exiting
// from inside the CLI package.
func Execute() error { return rootCmd.Execute() }

// ExitCode maps an Execute error to the documented process exit code.
func ExitCode(err error) int { return cli.ExitCode(err) }
