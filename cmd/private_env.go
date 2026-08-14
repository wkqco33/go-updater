package cmd

import (
	"fmt"
	"sort"

	"go_updater/internal/privatecache"

	"github.com/spf13/cobra"
)

var privateOffline bool

var privateEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "다른 프로젝트에서 사용할 프라이빗 모듈 환경변수를 출력합니다.",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := privatecache.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}
		cfg, err := privatecache.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		envMap := privatecache.BuildEnv(cfg, privateOffline)
		keys := make([]string, 0, len(envMap))
		for k := range envMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(cmd.OutOrStdout(), "export %s=%q\n", k, envMap[k])
		}
		return nil
	},
}

func init() {
	privateEnvCmd.Flags().BoolVar(&privateOffline, "offline", false, "오프라인 모드용 GOPROXY=off를 함께 출력")
	privateCmd.AddCommand(privateEnvCmd)
}
