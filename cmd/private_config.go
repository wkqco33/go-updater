package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"go_updater/internal/privatecache"

	"github.com/spf13/cobra"
)

var (
	privatePatterns string
	noSumDBPatterns string
	noProxyPatterns string
	cacheDirFlag    string
)

var privateConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "프라이빗 모듈 캐시 설정을 관리합니다.",
}

var privateConfigInitCmd = &cobra.Command{
	Use:   "init",
	Short: "기본 프라이빗 캐시 설정 파일을 생성합니다.",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := privatecache.DefaultConfigPath()
		if err != nil {
			slog.Error("failed to resolve config path", "error", err)
			os.Exit(1)
		}
		cfg, err := privatecache.DefaultConfig()
		if err != nil {
			slog.Error("failed to get default config", "error", err)
			os.Exit(1)
		}
		if err := privatecache.SaveConfig(configPath, cfg); err != nil {
			slog.Error("failed to save config", "error", err)
			os.Exit(1)
		}
		fmt.Printf("설정 파일이 생성되었습니다: %s\n", configPath)
	},
}

var privateConfigSetCmd = &cobra.Command{
	Use:   "set",
	Short: "프라이빗 모듈 캐시 설정 값을 갱신합니다.",
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

		if cmd.Flags().Changed("private") {
			cfg.PrivatePatterns = privatecache.ParseCSVPatterns(privatePatterns)
		}
		if cmd.Flags().Changed("nosumdb") {
			cfg.NoSumDBPatterns = privatecache.ParseCSVPatterns(noSumDBPatterns)
		}
		if cmd.Flags().Changed("noproxy") {
			cfg.NoProxyPatterns = privatecache.ParseCSVPatterns(noProxyPatterns)
		}
		if cmd.Flags().Changed("cache-dir") {
			cfg.CacheDir = cacheDirFlag
		}

		if err := privatecache.SaveConfig(configPath, cfg); err != nil {
			slog.Error("failed to save config", "error", err)
			os.Exit(1)
		}
		fmt.Printf("설정이 저장되었습니다: %s\n", configPath)
	},
}

var privateConfigShowCmd = &cobra.Command{
	Use:   "show",
	Short: "현재 프라이빗 모듈 캐시 설정을 출력합니다.",
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
		b, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			slog.Error("failed to render config", "error", err)
			os.Exit(1)
		}
		fmt.Println(string(b))
	},
}

func init() {
	privateConfigSetCmd.Flags().StringVar(&privatePatterns, "private", "", "GOPRIVATE 패턴 (쉼표 구분)")
	privateConfigSetCmd.Flags().StringVar(&noSumDBPatterns, "nosumdb", "", "GONOSUMDB 패턴 (쉼표 구분)")
	privateConfigSetCmd.Flags().StringVar(&noProxyPatterns, "noproxy", "", "GONOPROXY 패턴 (쉼표 구분)")
	privateConfigSetCmd.Flags().StringVar(&cacheDirFlag, "cache-dir", "", "모듈 캐시 디렉토리")

	privateConfigCmd.AddCommand(privateConfigInitCmd)
	privateConfigCmd.AddCommand(privateConfigSetCmd)
	privateConfigCmd.AddCommand(privateConfigShowCmd)
	privateCmd.AddCommand(privateConfigCmd)
}
