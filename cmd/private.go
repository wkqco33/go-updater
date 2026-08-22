package cmd

import "go_updater/internal/cli"

var privateCmd = &cli.Command{
	Use:   "private",
	Short: "프라이빗 Go 모듈 캐시를 관리합니다.",
}

func init() {
	rootCmd.AddCommand(privateCmd)
}
