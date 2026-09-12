package cmd

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/guenv"
	"github.com/wkqco33/go-updater/internal/installer"
	"github.com/wkqco33/go-updater/internal/versions"
)

var useCmd = &cli.Command{
	Use:   "use <version>",
	Short: "설치된 특정 버전의 Go를 활성화합니다.",
	Long: `설치된 버전 중 하나를 선택해 current 링크를 전환합니다.
마이너 버전만 지정하면 해당 마이너의 가장 높은 설치 버전을 사용합니다.

예시:
  gu use 1.20.5
  gu use 1.20

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.ExactArgs(1),
	RunE: func(cmd *cli.Command, args []string) error {
		root, err := guenv.Resolve(globals.Home)
		if err != nil {
			return err
		}
		store := versions.NewStore(root)
		version, err := store.Resolve(args[0])
		if err != nil {
			return err
		}
		target := filepath.Join(root, "versions", version)
		if err := installer.UpdateCurrentSymlink(root, target); err != nil {
			return fmt.Errorf("버전 변경 실패: %w", err)
		}
		slog.Debug("active version switched", "version", version, "root", root)
		fmt.Fprintf(cmd.OutOrStdout(), "현재 Go 버전이 %s(으)로 변경되었습니다.\n", version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
