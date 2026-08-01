package cmd

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"go_updater/internal/systemgo"

	"github.com/spf13/cobra"
)

var (
	cleanAll    bool
	cleanUnused bool
	cleanSystem bool
)

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

// artifactLabel returns the human-readable name shown in the detection
// checklist for one artifact.
func artifactLabel(a systemgo.Artifact) string {
	switch a.Kind {
	case systemgo.KindReceipt:
		return "pkgutil 리시트 " + a.Path
	case systemgo.KindHomebrew:
		return "Homebrew Go (" + a.Path + ")"
	default:
		return a.Path
	}
}

// runCleanSystem detects go.dev pkg / Homebrew Go installations and, after
// showing exactly which commands will run, removes what gu is allowed to
// manage. Root-owned paths are removed via sudo, prompting for a password.
func runCleanSystem() {
	items, err := systemgo.Detect()
	if err != nil {
		slog.Error("failed to detect system Go installation", "error", err)
		os.Exit(1)
	}
	if len(items) == 0 {
		fmt.Println("감지된 시스템 Go 설치가 없습니다.")
		return
	}

	var pkgItems, homebrewItems []systemgo.Present
	for _, item := range items {
		if item.Artifact.Kind == systemgo.KindHomebrew {
			homebrewItems = append(homebrewItems, item)
		} else {
			pkgItems = append(pkgItems, item)
		}
	}

	if len(pkgItems) > 0 {
		fmt.Println("감지된 go.dev pkg 설치:")
		for _, item := range pkgItems {
			line := "  [x] " + artifactLabel(item.Artifact)
			if !item.Exists {
				line += " (없음)"
			} else {
				line = "  [v] " + artifactLabel(item.Artifact)
				if item.Artifact.Detail != "" {
					line += " " + item.Artifact.Detail
				}
			}
			fmt.Println(line)
		}
	}
	for _, item := range homebrewItems {
		fmt.Printf("Homebrew로 설치된 Go가 감지되었습니다: %s\n", item.Artifact.Path)
		fmt.Println("gu는 Homebrew 설치를 삭제하지 않습니다. 삭제하려면 'brew uninstall go'를 사용하세요.")
	}

	plan := systemgo.Plan(items)
	if len(plan) == 0 {
		return
	}

	fmt.Println("\n다음 명령이 실행됩니다:")
	for _, step := range plan {
		fmt.Println("  " + step.Display)
	}

	if !confirmAction("\n계속하시겠습니까?") {
		fmt.Println("취소되었습니다.")
		return
	}

	if err := systemgo.Remove(plan); err != nil {
		slog.Error("failed to remove system Go installation", "error", err)
		fmt.Printf("일부 항목을 삭제하지 못했습니다: %v\n권한이 필요한 경우 sudo 비밀번호를 다시 확인하세요.\n", err)
		os.Exit(1)
	}

	fmt.Println("시스템 Go 설치가 성공적으로 삭제되었습니다.")
	for _, step := range plan {
		if step.Artifact.Kind == systemgo.KindFile && step.Artifact.Path == systemgo.PathsDGo {
			fmt.Println("참고: 현재 열려 있는 셸의 PATH에는 여전히 이전 경로가 남아있을 수 있습니다. 새 터미널 세션을 여세요.")
		}
	}
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
			runCleanSystem()
			return
		}

		// 5. Default: No args and no flags
		cmd.Help()
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "모든 설치된 Go 버전을 삭제합니다.")
	cleanCmd.Flags().BoolVar(&cleanUnused, "unused", false, "현재 사용 중인 버전을 제외한 모든 설치된 버전을 삭제합니다.")
	cleanCmd.Flags().BoolVar(&cleanSystem, "system", false, "go.dev에서 설치된 시스템 Go를 삭제합니다 (macOS: /usr/local/go, /etc/paths.d/go, pkgutil 리시트 포함 / Windows: C:\\Go).")
	rootCmd.AddCommand(cleanCmd)
}
