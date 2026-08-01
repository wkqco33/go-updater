package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"sort"

	"go_updater/internal/privatecache"

	"github.com/spf13/cobra"
)

var privateOffline bool

var privateEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "다른 프로젝트에서 사용할 프라이빗 모듈 환경변수를 출력합니다.",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := privatecache.DefaultConfigPath()
		if err != nil {
			slog.Error("failed to resolve config path", "error", err)
			os.Exit(1)
		}
		cfg, err := privatecache.LoadConfig(configPath)
		if err != nil {
			slog.Error("failed to load config", "error", err)
			os.Exit(1)
		}

		envMap := privatecache.BuildEnv(cfg, privateOffline)
		keys := make([]string, 0, len(envMap))
		for k := range envMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("export %s=%q\n", k, envMap[k])
		}
	},
}

func init() {
	privateEnvCmd.Flags().BoolVar(&privateOffline, "offline", false, "오프라인 모드용 GOPROXY=off를 함께 출력")
	privateCmd.AddCommand(privateEnvCmd)
}
