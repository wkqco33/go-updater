package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go_updater/internal/installer"

	"github.com/spf13/cobra"
)

// parseGoVersion은 "go1.21.5" 형식을 [major, minor, patch]로 파싱한다.
// 파싱 실패 시 [-1, -1, -1]을 반환한다.
func parseGoVersion(v string) [3]int {
	s := strings.TrimPrefix(v, "go")
	parts := strings.SplitN(s, ".", 3)
	var nums [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{-1, -1, -1}
		}
		nums[i] = n
	}
	return nums
}

func goVersionGreater(a, b string) bool {
	va, vb := parseGoVersion(a), parseGoVersion(b)
	for i := range va {
		if va[i] != vb[i] {
			return va[i] > vb[i]
		}
	}
	return false
}

var useCmd = &cobra.Command{
	Use:   "use <version>",
	Short: "설치된 특정 버전의 Go를 활성화합니다.",
	Long:  `설치된 버전 목록 중 하나를 선택하여 활성화합니다. (예: 1.20, 1.20.5)`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		targetVersion := version
		if !strings.HasPrefix(targetVersion, "go") {
			targetVersion = "go" + targetVersion
		}
		slog.Debug("use command started", "version", targetVersion)

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
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
							if matchedVersion == "" || goVersionGreater(entry.Name(), matchedVersion) {
								matchedVersion = entry.Name()
							}
						}
					}
					if matchedVersion != "" {
						targetGoDir = filepath.Join(versionsDir, matchedVersion)
						targetVersion = matchedVersion
						slog.Debug("found prefix matched installed version", "matched", targetVersion)
					} else {
						return fmt.Errorf("버전 '%s'가 설치되어 있지 않습니다", version)
					}
				} else {
					return fmt.Errorf("버전 '%s'가 설치되어 있지 않습니다", version)
				}
			} else {
				return fmt.Errorf("error checking version directory: %w", err)
			}
		}

		if err := installer.UpdateCurrentSymlink(targetDir, targetGoDir); err != nil {
			return fmt.Errorf("버전 변경 실패: %w", err)
		}

		fmt.Printf("현재 Go 버전이 %s(으)로 변경되었습니다.\n", targetVersion)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
