package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "설치된 Go 버전 목록을 출력합니다.",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("list command started")

		homeDir, err := os.UserHomeDir()
		if err != nil {
			slog.Error("failed to get home directory", "error", err)
			os.Exit(1)
		}
		targetDir := filepath.Join(homeDir, ".go")
		versionsDir := filepath.Join(targetDir, "versions")
		currentLink := filepath.Join(targetDir, "current")

		entries, err := os.ReadDir(versionsDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("설치된 Go 버전이 없습니다. 'go_updater install' 명령어를 사용하여 설치하세요.")
				return
			}
			slog.Error("failed to read versions directory", "error", err)
			os.Exit(1)
		}

		currentVersion := ""
		resolvedLink, err := os.Readlink(currentLink)
		if err == nil {
			currentVersion = filepath.Base(resolvedLink)
		}

		fmt.Println("설치된 Go 버전 목록:")
		found := false
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "go") {
				found = true
				if entry.Name() == currentVersion {
					fmt.Printf("  * %s (활성화됨)\n", entry.Name())
				} else {
					fmt.Printf("    %s\n", entry.Name())
				}
			}
		}

		if !found {
			fmt.Println("  설치된 버전이 없습니다.")
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
