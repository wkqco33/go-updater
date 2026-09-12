package cmd

import (
	"fmt"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/privatecache"
)

var (
	privateCleanAll      bool
	privateCleanStaleDay int
	privateCleanMaxSize  int64
)

var privateCleanCmd = &cli.Command{
	Use:   "clean",
	Short: "프라이빗 모듈 캐시를 정리합니다.",
	Long: `보존 기간이나 최대 용량을 기준으로 캐시를 정리하거나, --all로 전체를 삭제합니다.
캐시 전체 삭제는 확인 프롬프트를 거치며 --yes 또는 --dry-run으로 제어할 수 있습니다.

예시:
  gu private clean --stale-days 30
  gu private clean --max-size-mb 2048
  gu private clean --all --yes

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.NoArgs,
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

		if privateCleanAll && !globals.DryRun {
			approved, err := confirm(cmd, "프라이빗 모듈 캐시 전체를 삭제할까요?")
			if err != nil {
				return err
			}
			if !approved {
				fmt.Fprintln(cmd.ErrOrStderr(), "취소되었습니다.")
				return nil
			}
		}

		result, err := privatecache.CleanCache(cfg, metadataPath, privatecache.CleanOptions{
			All:       privateCleanAll,
			StaleDays: privateCleanStaleDay,
			MaxSizeMB: privateCleanMaxSize,
			DryRun:    globals.DryRun,
		})
		if err != nil {
			return fmt.Errorf("failed to clean cache: %w", err)
		}
		summary := "정리 완료"
		if globals.DryRun {
			summary = "dry-run: 정리 예정"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s: 삭제 파일 %d개, 확보 용량 %.2f MB, 최종 크기 %.2f MB\n",
			summary, result.RemovedFiles, float64(result.FreedBytes)/1024/1024, float64(result.FinalSize)/1024/1024)
		return nil
	},
}

func init() {
	privateCleanCmd.Flags().BoolVar(&privateCleanAll, "all", "", false, "캐시 전체 및 메타데이터를 삭제")
	privateCleanCmd.Flags().IntVar(&privateCleanStaleDay, "stale-days", "", 0, "N일 이전 파일 삭제")
	privateCleanCmd.Flags().Int64Var(&privateCleanMaxSize, "max-size-mb", "", 0, "캐시 최대 크기(MB) 초과 시 오래된 파일부터 삭제")
	privateCmd.AddCommand(privateCleanCmd)
}
