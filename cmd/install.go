package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go_updater/internal/fetcher"
	"go_updater/internal/installer"
)

var installDir string

var installCmd = &cobra.Command{
	Use:   "install [version]",
	Short: "Go를 설치하거나 특정 버전으로 업데이트합니다.",
	Long:  `go.dev에서 릴리스 정보를 확인하고, 적절한 압축 파일을 다운로드하여 설치합니다. 버전을 명시하지 않으면 최신 버전을 설치합니다 (예: 1.20, 1.20.5).`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("install command started")
		version := ""
		if len(args) > 0 {
			version = args[0]
			fmt.Printf("Fetching Go release information for version %s...\n", version)
		} else {
			fmt.Println("Fetching latest Go release information...")
		}
		
		url, fileInfo, err := fetcher.GetDownloadURL(version)
		if err != nil {
			slog.Error("failed to get download URL", "error", err)
			os.Exit(1)
		}

		fmt.Printf("Found version: %s\n", fileInfo.Version)
		slog.Debug("release info", "version", fileInfo.Version, "url", url, "sha256", fileInfo.Sha256)
		
		targetDir := installDir
		if targetDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				slog.Error("failed to get home directory", "error", err)
				os.Exit(1)
			}
			targetDir = filepath.Join(homeDir, ".go")
		}

		slog.Debug("target configuration", "base_dir", targetDir)
		
		if err := installer.InstallGo(url, fileInfo.Sha256, targetDir, fileInfo.Version); err != nil {
			slog.Error("installation failed", "error", err)
			os.Exit(1)
		}
		slog.Debug("install command finished successfully")
	},
}

func init() {
	installCmd.Flags().StringVarP(&installDir, "dir", "d", "", "Go가 설치될 최상위 디렉토리 (기본값: ~/.go)")
	rootCmd.AddCommand(installCmd)
}
