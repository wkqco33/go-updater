package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"go_updater/internal/versions"

	"go_updater/internal/cli"
)

var listHomeDir = os.UserHomeDir

var listCmd = &cli.Command{
	Use:   "list",
	Short: "설치된 Go 버전 목록을 출력합니다.",
	RunE: func(cmd *cli.Command, args []string) error {
		homeDir, err := listHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		items, err := versions.NewStore(filepath.Join(homeDir, ".go")).List()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if len(items) == 0 {
			fmt.Fprintln(out, "설치된 Go 버전이 없습니다. 'gu install' 명령어를 사용하여 설치하세요.")
			return nil
		}
		fmt.Fprintln(out, "설치된 Go 버전 목록:")
		for _, item := range items {
			if item.Active {
				fmt.Fprintf(out, "  * %s (활성화됨)\n", item.Name)
			} else {
				fmt.Fprintf(out, "    %s\n", item.Name)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
