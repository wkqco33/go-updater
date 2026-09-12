package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/privatecache"
)

var privateOffline bool

var privateEnvCmd = &cli.Command{
	Use:   "env",
	Short: "다른 프로젝트에서 사용할 프라이빗 모듈 환경변수를 출력합니다.",
	Long: `현재 설정으로 만든 환경변수를 export 문으로 출력합니다.
출력은 셸에서 그대로 평가할 수 있도록 단일 인용부호로 감싸며, 값에 포함된 $나
백틱 같은 문자가 셸에 의해 확장되지 않습니다.

예시:
  eval $(gu private env)
  eval $(gu private env --offline)

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.NoArgs,
	RunE: func(cmd *cli.Command, args []string) error {
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
			fmt.Fprintf(cmd.OutOrStdout(), "export %s=%s\n", k, shellQuote(envMap[k]))
		}
		return nil
	},
}

// shellQuote wraps value in single quotes for POSIX shells, escaping embedded
// single quotes, so an evaluated export statement cannot expand $, backticks,
// or other metacharacters that appear in a user-supplied cache path.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func init() {
	privateEnvCmd.Flags().BoolVar(&privateOffline, "offline", "", false, "오프라인 모드용 GOPROXY=off를 함께 출력")
	privateCmd.AddCommand(privateEnvCmd)
}
