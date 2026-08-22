package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wkqco33/go-updater/internal/installer"
	"github.com/wkqco33/go-updater/internal/versions"

	"github.com/wkqco33/go-updater/internal/cli"
)

var useHomeDir = os.UserHomeDir

var useCmd = &cli.Command{
	Use:   "use <version>",
	Short: "설치된 특정 버전의 Go를 활성화합니다.",
	Long:  `설치된 버전 목록 중 하나를 선택하여 활성화합니다. (예: 1.20, 1.20.5)`,
	Args:  cli.ExactArgs(1),
	RunE: func(cmd *cli.Command, args []string) error {
		homeDir, err := useHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		root := filepath.Join(homeDir, ".go")
		store := versions.NewStore(root)
		version, err := store.Resolve(args[0])
		if err != nil {
			return err
		}
		target := filepath.Join(root, "versions", version)
		if err := installer.UpdateCurrentSymlink(root, target); err != nil {
			return fmt.Errorf("버전 변경 실패: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "현재 Go 버전이 %s(으)로 변경되었습니다.\n", version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
