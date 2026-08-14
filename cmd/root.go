package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var debug bool

// NewRoot creates the base Cobra command. Command implementations register
// themselves during package initialization for now; keeping construction in a
// function gives tests and the eventual dependency-injected command tree a
// single entry point.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "gu",
		Short: "Go 언어를 설치하거나 최신 버전으로 업데이트하는 도구입니다.",
		Long:  `go.dev에서 최신 Go 릴리스 정보를 가져와 사용자의 시스템 환경에 맞는 버전을 자동으로 다운로드하고 설치해주는 빠르고 유연한 CLI 도구입니다.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := slog.LevelInfo
			if debug {
				level = slog.LevelDebug
			}
			logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: level}))
			slog.SetDefault(logger)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.PersistentFlags().BoolVar(&debug, "debug", false, "디버그 로깅 활성화")
	return root
}

var rootCmd = NewRoot()

// Execute returns command errors to the process boundary instead of exiting
// from inside the CLI package. This makes command failures testable.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Command registration remains in the individual command files. The root
	// command itself is created by NewRoot so this can be replaced by a fresh
	// dependency-injected tree in a later refactor.
}
