package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <version>",
	Short: "설치된 특정 버전의 Go를 활성화합니다.",
	Long:  `설치된 버전 목록 중 하나를 선택하여 활성화합니다. (예: 1.20, 1.20.5)`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := args[0]
		targetVersion := version
		if !strings.HasPrefix(targetVersion, "go") {
			targetVersion = "go" + targetVersion
		}
		slog.Debug("use command started", "version", targetVersion)

		homeDir, err := os.UserHomeDir()
		if err != nil {
			slog.Error("failed to get home directory", "error", err)
			os.Exit(1)
		}
		targetDir := filepath.Join(homeDir, ".go")
		versionsDir := filepath.Join(targetDir, "versions")
		targetGoDir := filepath.Join(versionsDir, targetVersion)

		// 1. Check if the exact version exists
		if _, err := os.Stat(targetGoDir); err != nil {
			if os.IsNotExist(err) {
				// 2. If exact version doesn't exist, try prefix match (e.g. "go1.20" matches "go1.20.5")
				entries, readErr := os.ReadDir(versionsDir)
				if readErr == nil {
					var matchedVersion string
					for _, entry := range entries {
						if entry.IsDir() && strings.HasPrefix(entry.Name(), targetVersion) {
							// Find the highest version if multiple matches exist, but for simplicity here we take the first match or exact.
							// Assuming dir reading order might not be semantic version order, simple prefix match for now.
							if matchedVersion == "" || entry.Name() > matchedVersion {
								matchedVersion = entry.Name()
							}
						}
					}
					if matchedVersion != "" {
						targetGoDir = filepath.Join(versionsDir, matchedVersion)
						targetVersion = matchedVersion
						slog.Debug("found prefix matched installed version", "matched", targetVersion)
					} else {
						fmt.Printf("버전 '%s'가 설치되어 있지 않습니다. 'go_updater list'로 설치된 목록을 확인하거나 'go_updater install %s'로 설치하세요.\n", version, version)
						os.Exit(1)
					}
				} else {
					fmt.Printf("버전 '%s'가 설치되어 있지 않습니다.\n", version)
					os.Exit(1)
				}
			} else {
				slog.Error("error checking version directory", "error", err)
				os.Exit(1)
			}
		}

		currentLink := filepath.Join(targetDir, "current")
		slog.Debug("updating symlink", "link", currentLink, "target", targetGoDir)

		os.Remove(currentLink) // Remove existing symlink
		
		if err := os.Symlink(targetGoDir, currentLink); err != nil {
			slog.Error("failed to create symlink", "error", err)
			fmt.Printf("버전 변경 실패: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("현재 Go 버전이 %s(으)로 변경되었습니다.\n", targetVersion)
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
