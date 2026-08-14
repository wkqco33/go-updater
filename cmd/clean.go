package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"go_updater/internal/systemgo"
	"go_updater/internal/versions"

	"github.com/spf13/cobra"
)

var (
	cleanAll    bool
	cleanUnused bool
	cleanSystem bool
)

// confirmAction asks the user to confirm by typing 'y' or 'Y'.
func confirmAction(prompt string, in io.Reader, out io.Writer) bool {
	fmt.Fprintf(out, "%s [y/N]: ", prompt)
	scanner := bufio.NewScanner(in)
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
	case systemgo.KindWinget:
		return "winget Go (" + a.Path + ")"
	default:
		return a.Path
	}
}

// runCleanSystem detects go.dev pkg / Homebrew Go installations and, after
// showing exactly which commands will run, removes what gu is allowed to
// manage. Root-owned paths are removed via sudo, prompting for a password.
func runCleanSystem(in io.Reader, out io.Writer) error {
	items, err := systemgo.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect system Go installation: %w", err)
	}
	if len(items) == 0 {
		fmt.Fprintln(out, "감지된 시스템 Go 설치가 없습니다.")
		return nil
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
		fmt.Fprintln(out, "감지된 go.dev pkg 설치:")
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
			fmt.Fprintln(out, line)
		}
	}
	for _, item := range homebrewItems {
		fmt.Fprintf(out, "Homebrew로 설치된 Go가 감지되었습니다: %s\n", item.Artifact.Path)
		fmt.Fprintln(out, "gu는 Homebrew 설치를 삭제하지 않습니다. 삭제하려면 'brew uninstall go'를 사용하세요.")
	}

	plan := systemgo.Plan(items)
	if len(plan) == 0 {
		return nil
	}

	fmt.Fprintln(out, "\n다음 명령이 실행됩니다:")
	for _, step := range plan {
		fmt.Fprintln(out, "  "+step.Display)
	}

	if !confirmAction("\n계속하시겠습니까?", in, out) {
		fmt.Fprintln(out, "취소되었습니다.")
		return nil
	}

	if err := systemgo.Remove(plan); err != nil {
		return fmt.Errorf("failed to remove system Go installation: %w", err)
	}

	fmt.Fprintln(out, "시스템 Go 설치가 성공적으로 삭제되었습니다.")
	for _, step := range plan {
		if step.Artifact.Kind == systemgo.KindFile && step.Artifact.Path == systemgo.PathsDGo {
			fmt.Fprintln(out, "참고: 현재 열려 있는 셸의 PATH에는 여전히 이전 경로가 남아있을 수 있습니다. 새 터미널 세션을 여세요.")
		}
	}
	return nil
}

var cleanCmd = &cobra.Command{
	Use:   "clean [version]",
	Short: "설치된 Go 버전들을 삭제하여 용량을 확보합니다.",
	Long:  `지정한 특정 버전, 사용하지 않는 모든 버전(--unused), 또는 모든 버전(--all)을 삭제합니다.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Debug("clean command started")

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		targetDir := filepath.Join(homeDir, ".go")
		store := versions.NewStore(targetDir)
		currentVersion, err := store.Active()
		if err != nil {
			return err
		}

		// 1. Handle --all flag
		if cleanAll {
			fmt.Println("모든 설치된 Go 버전을 삭제합니다...")
			if err := store.RemoveAll(); err != nil {
				return err
			}
			fmt.Println("모든 버전이 삭제되었습니다.")
			return nil
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
				return nil
			}

			if _, err := store.Resolve(targetVersion); err != nil {
				fmt.Printf("버전 %s가 설치되어 있지 않습니다.\n", targetVersion)
				return nil
			}

			fmt.Printf("버전 %s를 삭제합니다...\n", targetVersion)
			if err := store.Remove(targetVersion); err != nil {
				return err
			}
			fmt.Printf("버전 %s가 성공적으로 삭제되었습니다.\n", targetVersion)
			return nil
		}

		// 3. Handle --unused flag
		if cleanUnused {
			if currentVersion == "" {
				fmt.Println("현재 활성화된 버전 정보가 없습니다. 모든 버전을 삭제하시려면 --all을 사용하세요.")
				return nil
			}

			fmt.Printf("현재 사용 중인 버전(%s)을 제외한 모든 버전을 삭제합니다...\n", currentVersion)
			removed, err := store.RemoveUnused()
			if err != nil {
				return err
			}
			for _, version := range removed {
				fmt.Printf("  삭제됨: %s\n", version)
			}
			fmt.Printf("총 %d개의 사용하지 않는 버전이 삭제되었습니다.\n", len(removed))
			return nil
		}

		// 4. Handle --system flag: remove go.dev system installation
		if cleanSystem {
			return runCleanSystem(cmd.InOrStdin(), cmd.OutOrStdout())
		}

		// 5. Default: No args and no flags
		return cmd.Help()
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "모든 설치된 Go 버전을 삭제합니다.")
	cleanCmd.Flags().BoolVar(&cleanUnused, "unused", false, "현재 사용 중인 버전을 제외한 모든 설치된 버전을 삭제합니다.")
	cleanCmd.Flags().BoolVar(&cleanSystem, "system", false, "go.dev에서 설치된 시스템 Go를 삭제합니다 (macOS: /usr/local/go, /etc/paths.d/go, pkgutil 리시트 포함 / Windows: C:\\Go).")
	rootCmd.AddCommand(cleanCmd)
}
