package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"go_updater/internal/fetcher"
	"go_updater/internal/installer"

	"github.com/spf13/cobra"
)

var installDir string

type installRunner struct {
	fetchDownloadURL func(version string) (string, *fetcher.GoFile, error)
	installGo        func(url, sha256, targetDir, version string) error
	userHomeDir      func() (string, error)
	stdout           io.Writer
}

func (r installRunner) Run(args []string, explicitDir string, stdout io.Writer) error {
	slog.Debug("install command started")
	version := ""
	if len(args) > 0 {
		version = args[0]
		fmt.Fprintf(stdout, "Fetching Go release information for version %s...\n", version)
	} else {
		fmt.Fprintln(stdout, "Fetching latest Go release information...")
	}

	url, fileInfo, err := r.fetchDownloadURL(version)
	if err != nil {
		slog.Error("failed to get download URL", "error", err)
		return err
	}

	fmt.Fprintf(stdout, "Found version: %s\n", fileInfo.Version)
	slog.Debug("release info", "version", fileInfo.Version, "url", url, "sha256", fileInfo.Sha256)

	targetDir := explicitDir
	if targetDir == "" {
		homeDir, err := r.userHomeDir()
		if err != nil {
			slog.Error("failed to get home directory", "error", err)
			return err
		}
		targetDir = filepath.Join(homeDir, ".go")
	}

	slog.Debug("target configuration", "base_dir", targetDir)

	if err := r.installGo(url, fileInfo.Sha256, targetDir, fileInfo.Version); err != nil {
		slog.Error("installation failed", "error", err)
		return err
	}
	slog.Debug("install command finished successfully")
	return nil
}

func newInstallRunner() installRunner {
	return installRunner{
		fetchDownloadURL: fetcher.GetDownloadURL,
		installGo:        installer.InstallGo,
		userHomeDir:      os.UserHomeDir,
		stdout:           os.Stdout,
	}
}

var installCmd = &cobra.Command{
	Use:   "install [version]",
	Short: "Go를 설치하거나 특정 버전으로 업데이트합니다.",
	Long:  `go.dev에서 릴리스 정보를 확인하고, 적절한 압축 파일을 다운로드하여 설치합니다. 버전을 명시하지 않으면 최신 버전을 설치합니다 (예: 1.20, 1.20.5).`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runner := newInstallRunner()
		runner.stdout = cmd.OutOrStdout()
		return runner.Run(args, installDir, runner.stdout)
	},
}

func init() {
	installCmd.Flags().StringVarP(&installDir, "dir", "d", "", "Go가 설치될 최상위 디렉토리 (기본값: ~/.go)")
	rootCmd.AddCommand(installCmd)
}
