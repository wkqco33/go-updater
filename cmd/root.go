package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	debug   bool
	rootCmd = &cobra.Command{
		Use:   "go_updater",
		Short: "Go 언어를 설치하거나 최신 버전으로 업데이트하는 도구입니다.",
		Long:  `go.dev에서 최신 Go 릴리스 정보를 가져와 사용자의 시스템 환경에 맞는 버전을 자동으로 다운로드하고 설치해주는 빠르고 유연한 CLI 도구입니다.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := slog.LevelInfo
			if debug {
				level = slog.LevelDebug
			}
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
			slog.SetDefault(logger)
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "디버그 로깅 활성화")
}
