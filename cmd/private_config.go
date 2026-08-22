package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/wkqco33/go-updater/internal/privatecache"

	"github.com/wkqco33/go-updater/internal/cli"
)

var (
	privatePatterns string
	noSumDBPatterns string
	noProxyPatterns string
	cacheDirFlag    string
)

var privateConfigCmd = &cli.Command{Use: "config", Short: "프라이빗 모듈 캐시 설정을 관리합니다."}

var privateConfigInitCmd = &cli.Command{
	Use: "init", Short: "기본 프라이빗 캐시 설정 파일을 생성합니다.",
	RunE: func(cmd *cli.Command, args []string) error {
		configPath, err := privatecache.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}
		cfg, err := privatecache.DefaultConfig()
		if err != nil {
			return fmt.Errorf("failed to get default config: %w", err)
		}
		if err := privatecache.SaveConfig(configPath, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "설정 파일이 생성되었습니다: %s\n", configPath)
		return nil
	},
}

var privateConfigSetCmd = &cli.Command{
	Use: "set", Short: "프라이빗 모듈 캐시 설정 값을 갱신합니다.",
	RunE: func(cmd *cli.Command, args []string) error {
		configPath, err := privatecache.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}
		cfg, err := privatecache.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
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
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "설정이 저장되었습니다: %s\n", configPath)
		return nil
	},
}

var privateConfigShowCmd = &cli.Command{
	Use: "show", Short: "현재 프라이빗 모듈 캐시 설정을 출력합니다.",
	RunE: func(cmd *cli.Command, args []string) error {
		configPath, err := privatecache.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}
		cfg, err := privatecache.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		b, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to render config: %w", err)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return err
	},
}

func init() {
	privateConfigSetCmd.Flags().StringVar(&privatePatterns, "private", "", "", "GOPRIVATE 패턴 (쉼표 구분)")
	privateConfigSetCmd.Flags().StringVar(&noSumDBPatterns, "nosumdb", "", "", "GONOSUMDB 패턴 (쉼표 구분)")
	privateConfigSetCmd.Flags().StringVar(&noProxyPatterns, "noproxy", "", "", "GONOPROXY 패턴 (쉼표 구분)")
	privateConfigSetCmd.Flags().StringVar(&cacheDirFlag, "cache-dir", "", "", "모듈 캐시 디렉토리")
	privateConfigCmd.AddCommand(privateConfigInitCmd, privateConfigSetCmd, privateConfigShowCmd)
	privateCmd.AddCommand(privateConfigCmd)
}
