package cmd

import (
	"fmt"

	"github.com/wkqco33/go-updater/internal/privatecache"

	"github.com/wkqco33/go-updater/internal/cli"
)

var (
	privateSyncRetries        int
	privateSyncLatestIfMissed bool
	privateSyncSource         string
)

var privateSyncCmd = &cli.Command{
	Use:   "sync <module[@version]> [module[@version] ...]",
	Short: "프라이빗 Go 모듈을 미리 다운로드해 공용 캐시에 적재합니다.",
	Args:  cli.MinimumNArgs(1),
	RunE: func(cmd *cli.Command, args []string) error {
		configPath, err := privatecache.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}
		metadataPath, err := privatecache.DefaultMetadataPath()
		if err != nil {
			return fmt.Errorf("failed to resolve metadata path: %w", err)
		}
		cfg, err := privatecache.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		result, err := privatecache.SyncModules(cmd.Context(), cfg, metadataPath, args, privateSyncRetries, privateSyncLatestIfMissed, privateSyncSource)
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		out := cmd.OutOrStdout()
		for _, rec := range result.Downloaded {
			fmt.Fprintf(out, "downloaded: %s@%s (resolved=%s)\n", rec.Module, rec.RequestedVersion, rec.ResolvedVersion)
		}
		for _, rec := range result.Skipped {
			fmt.Fprintf(out, "skipped: %s@%s\n", rec.Module, rec.RequestedVersion)
		}
		for _, reason := range result.Failed {
			fmt.Fprintf(out, "failed: %s\n", reason)
		}
		if len(result.Failed) > 0 {
			return fmt.Errorf("%d module(s) failed to sync", len(result.Failed))
		}
		return nil
	},
}

func init() {
	privateSyncCmd.Flags().IntVar(&privateSyncRetries, "retries", "", 3, "모듈 다운로드 재시도 횟수")
	privateSyncCmd.Flags().BoolVar(&privateSyncLatestIfMissed, "latest-if-missing", "", true, "버전 미지정 시 latest 사용")
	privateSyncCmd.Flags().StringVar(&privateSyncSource, "source", "", "manual", "동기화 소스 식별자(예: github-enterprise)")
	privateCmd.AddCommand(privateSyncCmd)
}
