package cmd

import (
	"fmt"

	"go_updater/internal/privatecache"

	"github.com/spf13/cobra"
)

var (
	privateCleanAll      bool
	privateCleanStaleDay int
	privateCleanMaxSize  int64
)

var privateCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "프라이빗 모듈 캐시를 정리합니다.",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		result, err := privatecache.CleanCache(cfg, metadataPath, privatecache.CleanOptions{
			All: privateCleanAll, StaleDays: privateCleanStaleDay, MaxSizeMB: privateCleanMaxSize,
		})
		if err != nil {
			return fmt.Errorf("failed to clean cache: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "정리 완료: 삭제 파일 %d개, 확보 용량 %.2f MB, 최종 크기 %.2f MB\n",
			result.RemovedFiles, float64(result.FreedBytes)/1024/1024, float64(result.FinalSize)/1024/1024)
		return nil
	},
}

func init() {
	privateCleanCmd.Flags().BoolVar(&privateCleanAll, "all", false, "캐시 전체 및 메타데이터를 삭제")
	privateCleanCmd.Flags().IntVar(&privateCleanStaleDay, "stale-days", 0, "N일 이전 파일 삭제")
	privateCleanCmd.Flags().Int64Var(&privateCleanMaxSize, "max-size-mb", 0, "캐시 최대 크기(MB) 초과 시 오래된 파일부터 삭제")
	privateCmd.AddCommand(privateCleanCmd)
}
