package cmd

import "github.com/spf13/cobra"

var privateCmd = &cobra.Command{
	Use:   "private",
	Short: "프라이빗 Go 모듈 캐시를 관리합니다.",
}

func init() {
	rootCmd.AddCommand(privateCmd)
}
