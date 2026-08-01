package cmd

import (
	"fmt"
	"log/slog"
	"os"

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

		result, err := privatecache.CleanCache(cfg, metadataPath, privatecache.CleanOptions{
			All:       privateCleanAll,
			StaleDays: privateCleanStaleDay,
			MaxSizeMB: privateCleanMaxSize,
		})
		if err != nil {
			slog.Error("failed to clean cache", "error", err)
			os.Exit(1)
		}

		fmt.Printf("정리 완료: 삭제 파일 %d개, 확보 용량 %.2f MB, 최종 크기 %.2f MB\n",
			result.RemovedFiles,
			float64(result.FreedBytes)/1024/1024,
			float64(result.FinalSize)/1024/1024,
		)
	},
}

func init() {
	privateCleanCmd.Flags().BoolVar(&privateCleanAll, "all", false, "캐시 전체 및 메타데이터를 삭제")
	privateCleanCmd.Flags().IntVar(&privateCleanStaleDay, "stale-days", 0, "N일 이전 파일 삭제")
	privateCleanCmd.Flags().Int64Var(&privateCleanMaxSize, "max-size-mb", 0, "캐시 최대 크기(MB) 초과 시 오래된 파일부터 삭제")
	privateCmd.AddCommand(privateCleanCmd)
}
