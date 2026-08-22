package cmd

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/wkqco33/go-updater/internal/cli"
)

// version is overridden by release builds with -ldflags. Local builds report dev.
var version = "dev"

var versionCmd = &cli.Command{
	Use:   "version",
	Short: "gu의 버전 정보를 출력합니다.",
	RunE: func(cmd *cli.Command, args []string) error {
		slog.Debug("version command called")
		fmt.Fprintf(cmd.OutOrStdout(), "gu version %s\n", version)
		fmt.Fprintf(cmd.OutOrStdout(), "OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
