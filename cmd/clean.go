package cmd

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cleanAll    bool
	cleanUnused bool
	cleanSystem bool
)

// systemGoPath returns the default installation path used by go.dev installers.
func systemGoPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Go`
	}
	return "/usr/local/go"
}

// confirmAction asks the user to confirm by typing 'y' or 'Y'.
func confirmAction(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(scanner.Text())
		return strings.EqualFold(answer, "y")
	}
	return false
}

var cleanCmd = &cobra.Command{
	Use:   "clean [version]",
	Short: "설치된 Go 버전들을 삭제하여 용량을 확보합니다.",
	Long:  `지정한 특정 버전, 사용하지 않는 모든 버전(--unused), 또는 모든 버전(--all)을 삭제합니다.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("clean command started")

		homeDir, err := os.UserHomeDir()
		if err != nil {
			slog.Error("failed to get home directory", "error", err)
			os.Exit(1)
		}
		targetDir := filepath.Join(homeDir, ".go")
		versionsDir := filepath.Join(targetDir, "versions")
		currentLink := filepath.Join(targetDir, "current")

		resolvedLink, err := os.Readlink(currentLink)
		currentVersion := ""
		if err == nil {
			currentVersion = filepath.Base(resolvedLink)
		}

		// 1. Handle --all flag
		if cleanAll {
			fmt.Println("모든 설치된 Go 버전을 삭제합니다...")
			os.Remove(currentLink)
			if err := os.RemoveAll(versionsDir); err != nil {
				slog.Error("failed to remove versions directory", "error", err)
				os.Exit(1)
			}
			fmt.Println("모든 버전이 삭제되었습니다.")
			return
		}

		// 2. Handle specific version deletion
		if len(args) > 0 {
			version := args[0]
			targetVersion := version
			if !strings.HasPrefix(targetVersion, "go") {
				targetVersion = "go" + targetVersion
			}
			
			if targetVersion == currentVersion {
				fmt.Printf("버전 %s는 현재 활성화되어 사용 중이므로 삭제할 수 없습니다. 'use' 명령어로 다른 버전으로 전환 후 삭제하세요.\n", targetVersion)
				return
			}

			targetPath := filepath.Join(versionsDir, targetVersion)
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				fmt.Printf("버전 %s가 설치되어 있지 않습니다.\n", targetVersion)
				return
			}

			fmt.Printf("버전 %s를 삭제합니다...\n", targetVersion)
			if err := os.RemoveAll(targetPath); err != nil {
				slog.Error("failed to remove version folder", "version", targetVersion, "error", err)
				os.Exit(1)
			}
			fmt.Printf("버전 %s가 성공적으로 삭제되었습니다.\n", targetVersion)
			return
		}

		// 3. Handle --unused flag
		if cleanUnused {
			if currentVersion == "" {
				fmt.Println("현재 활성화된 버전 정보가 없습니다. 모든 버전을 삭제하시려면 --all을 사용하세요.")
				return
			}

			entries, err := os.ReadDir(versionsDir)
			if err != nil {
				slog.Error("failed to read versions directory", "error", err)
				os.Exit(1)
			}

			fmt.Printf("현재 사용 중인 버전(%s)을 제외한 모든 버전을 삭제합니다...\n", currentVersion)
			count := 0
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "go") {
					if entry.Name() != currentVersion {
						slog.Debug("deleting unused version", "version", entry.Name())
						if err := os.RemoveAll(filepath.Join(versionsDir, entry.Name())); err != nil {
							slog.Warn("failed to delete version", "version", entry.Name(), "error", err)
						} else {
							fmt.Printf("  삭제됨: %s\n", entry.Name())
							count++
						}
					}
				}
			}
			fmt.Printf("총 %d개의 사용하지 않는 버전이 삭제되었습니다.\n", count)
			return
		}

		// 4. Handle --system flag: remove go.dev system installation
		if cleanSystem {
			sysPath := systemGoPath()
			if _, err := os.Stat(sysPath); os.IsNotExist(err) {
				fmt.Printf("go.dev 시스템 설치 경로(%s)에서 Go를 찾을 수 없습니다.\n", sysPath)
				return
			}

			fmt.Printf("go.dev에서 설치된 Go가 다음 경로에서 감지되었습니다: %s\n", sysPath)
			if !confirmAction("해당 경로의 Go를 삭제하시겠습니까?") {
				fmt.Println("취소되었습니다.")
				return
			}

			fmt.Printf("시스템 Go 설치를 삭제합니다: %s\n", sysPath)
			if err := os.RemoveAll(sysPath); err != nil {
				slog.Error("failed to remove system Go installation", "path", sysPath, "error", err)
				fmt.Printf("삭제 실패: %v\n권한이 필요한 경우 sudo를 사용하세요.\n", err)
				os.Exit(1)
			}
			fmt.Printf("시스템 Go 설치(%s)가 성공적으로 삭제되었습니다.\n", sysPath)
			return
		}

		// 5. Default: No args and no flags
		cmd.Help()
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "모든 설치된 Go 버전을 삭제합니다.")
	cleanCmd.Flags().BoolVar(&cleanUnused, "unused", false, "현재 사용 중인 버전을 제외한 모든 설치된 버전을 삭제합니다.")
	cleanCmd.Flags().BoolVar(&cleanSystem, "system", false, "go.dev에서 설치된 시스템 Go(/usr/local/go 또는 C:\\Go)를 삭제합니다.")
	rootCmd.AddCommand(cleanCmd)
}
