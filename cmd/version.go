package cmd

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "go_updater의 버전 정보를 출력합니다.",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("version command called")
		fmt.Printf("go_updater version 0.1.0\n")
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
