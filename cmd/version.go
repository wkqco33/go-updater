package cmd

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "gu의 버전 정보를 출력합니다.",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Debug("version command called")
		fmt.Printf("gu version 0.1.0\n")
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
