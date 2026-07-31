package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go_updater/internal/privatecache"

	"github.com/spf13/cobra"
)

var (
	privateSyncRetries        int
	privateSyncLatestIfMissed bool
	privateSyncSource         string
)

var privateSyncCmd = &cobra.Command{
	Use:   "sync <module[@version]> [module[@version] ...]",
	Short: "프라이빗 Go 모듈을 미리 다운로드해 공용 캐시에 적재합니다.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := privatecache.DefaultConfigPath()
		if err != nil {
			slog.Error("failed to resolve config path", "error", err)
			os.Exit(1)
		}
		metadataPath, err := privatecache.DefaultMetadataPath()
		if err != nil {
			slog.Error("failed to resolve metadata path", "error", err)
			os.Exit(1)
		}
		cfg, err := privatecache.LoadConfig(configPath)
		if err != nil {
			slog.Error("failed to load config", "error", err)
			os.Exit(1)
		}

		result, err := privatecache.SyncModules(context.Background(), cfg, metadataPath, args, privateSyncRetries, privateSyncLatestIfMissed, privateSyncSource)
		if err != nil {
			slog.Error("sync failed", "error", err)
			os.Exit(1)
		}

		for _, rec := range result.Downloaded {
			fmt.Printf("downloaded: %s@%s (resolved=%s)\n", rec.Module, rec.RequestedVersion, rec.ResolvedVersion)
		}
		for _, rec := range result.Skipped {
			fmt.Printf("skipped: %s@%s\n", rec.Module, rec.RequestedVersion)
		}
		for _, reason := range result.Failed {
			fmt.Printf("failed: %s\n", reason)
		}

		if len(result.Failed) > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	privateSyncCmd.Flags().IntVar(&privateSyncRetries, "retries", 3, "모듈 다운로드 재시도 횟수")
	privateSyncCmd.Flags().BoolVar(&privateSyncLatestIfMissed, "latest-if-missing", true, "버전 미지정 시 latest 사용")
	privateSyncCmd.Flags().StringVar(&privateSyncSource, "source", "manual", "동기화 소스 식별자(예: github-enterprise)")
	privateCmd.AddCommand(privateSyncCmd)
}
