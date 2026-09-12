package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/wkqco33/go-updater/internal/cli"
)

var versionJSON bool

type versionInfo struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

var versionCmd = &cli.Command{
	Use:   "version",
	Short: "gu의 버전 정보를 출력합니다.",
	Long: `gu 자체의 버전과 실행 중인 OS/아키텍처를 출력합니다.

예시:
  gu version
  gu version --json

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.NoArgs,
	RunE: func(cmd *cli.Command, args []string) error {
		slog.Debug("version command called")
		info := versionInfo{Version: buildVersion, OS: runtime.GOOS, Arch: runtime.GOARCH}
		if versionJSON {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(info)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "gu version %s\n", info.Version)
		fmt.Fprintf(cmd.OutOrStdout(), "OS/Arch: %s/%s\n", info.OS, info.Arch)
		return nil
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionJSON, "json", "", false, "결과를 JSON으로 출력합니다")
	rootCmd.AddCommand(versionCmd)
}
