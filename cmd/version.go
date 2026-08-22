package cmd

import (
	"fmt"
	"log/slog"
	"runtime"

	"go_updater/internal/cli"
)

var versionCmd = &cli.Command{
	Use:   "version",
	Short: "gu의 버전 정보를 출력합니다.",
	RunE: func(cmd *cli.Command, args []string) error {
		slog.Debug("version command called")
		fmt.Fprintf(cmd.OutOrStdout(), "gu version 0.1.0\n")
		fmt.Fprintf(cmd.OutOrStdout(), "OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
