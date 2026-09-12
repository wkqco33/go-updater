package cmd

import (
	"fmt"
	"log/slog"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/guenv"
	"github.com/wkqco33/go-updater/internal/systemgo"
	"github.com/wkqco33/go-updater/internal/versions"
)

var (
	cleanAll    bool
	cleanUnused bool
	cleanSystem bool
)

// detectSystemGo and removeSystemGo are seams so the system cleanup flow can be
// tested without touching real system paths or invoking sudo.
var (
	detectSystemGo = systemgo.Detect
	removeSystemGo = systemgo.Remove
)

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
func runCleanSystem(cmd *cli.Command) error {
	out := cmd.OutOrStdout()
	items, err := detectSystemGo()
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

	if globals.DryRun {
		fmt.Fprintln(out, "dry-run: 위 명령을 실행하지 않았습니다.")
		return nil
	}

	approved, err := confirm(cmd, "\n계속하시겠습니까?")
	if err != nil {
		return err
	}
	if !approved {
		fmt.Fprintln(cmd.ErrOrStderr(), "취소되었습니다.")
		return nil
	}

	if err := removeSystemGo(plan); err != nil {
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

var cleanCmd = &cli.Command{
	Use:   "clean [version]",
	Short: "설치된 Go 버전들을 삭제하여 용량을 확보합니다.",
	Long: `지정한 버전, 사용하지 않는 버전(--unused), 또는 모든 버전(--all)을 삭제합니다.
여러 버전을 한 번에 지우는 작업은 확인 프롬프트를 거치며, 비대화형 환경에서는
--yes로 승인하거나 --dry-run으로 계획만 확인할 수 있습니다.

예시:
  gu clean 1.20.5        특정 버전 삭제
  gu clean --unused      사용하지 않는 버전 삭제
  gu clean --all --yes   모든 버전 삭제 (비대화형)
  gu clean --system      go.dev 시스템 설치 정리
  gu clean --all --dry-run  삭제 대상만 확인

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.MaximumNArgs(1),
	RunE: func(cmd *cli.Command, args []string) error {
		slog.Debug("clean command started")

		root, err := guenv.Resolve(globals.Home)
		if err != nil {
			return err
		}
		store := versions.NewStore(root)
		currentVersion, err := store.Active()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()

		// 1. Handle --all flag
		if cleanAll {
			statusf(cmd, "모든 설치된 Go 버전을 삭제합니다...\n")
			if globals.DryRun {
				fmt.Fprintf(out, "dry-run: %s 의 모든 버전과 current 링크를 삭제합니다\n", root)
				return nil
			}
			approved, err := confirm(cmd, "모든 Go 버전을 삭제할까요?")
			if err != nil {
				return err
			}
			if !approved {
				fmt.Fprintln(cmd.ErrOrStderr(), "취소되었습니다.")
				return nil
			}
			if err := store.RemoveAll(); err != nil {
				return err
			}
			fmt.Fprintln(out, "모든 버전이 삭제되었습니다.")
			return nil
		}

		// 2. Handle specific version deletion
		if len(args) > 0 {
			resolved, err := store.Resolve(args[0])
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "경고: %v\n", err)
				return nil
			}
			if resolved == currentVersion {
				fmt.Fprintf(cmd.ErrOrStderr(), "경고: %s는 현재 활성화되어 사용 중이므로 삭제할 수 없습니다. 'use' 명령어로 다른 버전으로 전환 후 삭제하세요.\n", resolved)
				return nil
			}
			if globals.DryRun {
				fmt.Fprintf(out, "dry-run: %s 를 삭제합니다\n", resolved)
				return nil
			}
			statusf(cmd, "버전 %s를 삭제합니다...\n", resolved)
			if err := store.Remove(resolved); err != nil {
				return err
			}
			fmt.Fprintf(out, "버전 %s가 성공적으로 삭제되었습니다.\n", resolved)
			return nil
		}

		// 3. Handle --unused flag
		if cleanUnused {
			if currentVersion == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "현재 활성화된 버전 정보가 없습니다. 모든 버전을 삭제하시려면 --all을 사용하세요.")
				return nil
			}

			statusf(cmd, "현재 사용 중인 버전(%s)을 제외한 모든 버전을 삭제합니다...\n", currentVersion)
			if globals.DryRun {
				list, err := store.List()
				if err != nil {
					return err
				}
				for _, version := range list {
					if version.Name != currentVersion {
						fmt.Fprintf(out, "dry-run: %s 를 삭제합니다\n", version.Name)
					}
				}
				return nil
			}
			approved, err := confirm(cmd, "현재 사용 중인 버전을 제외한 모든 버전을 삭제할까요?")
			if err != nil {
				return err
			}
			if !approved {
				fmt.Fprintln(cmd.ErrOrStderr(), "취소되었습니다.")
				return nil
			}

			removed, err := store.RemoveUnused()
			if err != nil {
				return err
			}
			for _, version := range removed {
				fmt.Fprintf(out, "  삭제됨: %s\n", version)
			}
			fmt.Fprintf(out, "총 %d개의 사용하지 않는 버전이 삭제되었습니다.\n", len(removed))
			return nil
		}

		// 4. Handle --system flag: remove go.dev system installation
		if cleanSystem {
			return runCleanSystem(cmd)
		}

		// 5. Default: No args and no flags
		cmd.Help()
		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", "", false, "모든 설치된 Go 버전을 삭제합니다.")
	cleanCmd.Flags().BoolVar(&cleanUnused, "unused", "", false, "현재 사용 중인 버전을 제외한 모든 설치된 버전을 삭제합니다.")
	cleanCmd.Flags().BoolVar(&cleanSystem, "system", "", false, "go.dev에서 설치된 시스템 Go를 삭제합니다 (macOS: /usr/local/go, /etc/paths.d/go, pkgutil 리시트 포함 / Windows: C:\\Go).")
	rootCmd.AddCommand(cleanCmd)
}
