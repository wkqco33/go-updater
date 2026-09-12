package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/fetcher"
	"github.com/wkqco33/go-updater/internal/guenv"
	"github.com/wkqco33/go-updater/internal/installer"
	"github.com/wkqco33/go-updater/internal/prompt"
)

var installDir string

type installRunner struct {
	fetchDownloadURL func(version string) (string, *fetcher.GoFile, error)
	installGo        func(url, sha256, targetDir, version string, opts installer.Options) error
	resolveRoot      func() (string, error)
	out              io.Writer
	err              io.Writer
	quiet            bool
	dryRun           bool
	progress         bool
}

func (r installRunner) outWriter() io.Writer {
	if r.out == nil {
		return io.Discard
	}
	return r.out
}

func (r installRunner) errWriter() io.Writer {
	if r.err == nil {
		return io.Discard
	}
	return r.err
}

func (r installRunner) status(format string, args ...any) {
	if r.quiet {
		return
	}
	fmt.Fprintf(r.errWriter(), format, args...)
}

func (r installRunner) Run(args []string, explicitDir string) error {
	var version string
	if len(args) > 0 {
		version = args[0]
		r.status("Fetching Go release information for version %s...\n", version)
	} else {
		r.status("Fetching latest Go release information...\n")
	}

	url, fileInfo, err := r.fetchDownloadURL(version)
	if err != nil {
		return err
	}
	slog.Debug("release info", "version", fileInfo.Version, "url", url, "sha256", fileInfo.Sha256)
	r.status("Found version: %s\n", fileInfo.Version)

	targetDir := strings.TrimSpace(explicitDir)
	if targetDir == "" {
		targetDir, err = r.resolveRoot()
		if err != nil {
			return err
		}
	}
	slog.Debug("target configuration", "base_dir", targetDir)

	if r.dryRun {
		fmt.Fprintf(r.outWriter(), "dry-run: Go %s를 %s 에서 %s 에 설치합니다\n", fileInfo.Version, url, targetDir)
		return nil
	}

	return r.installGo(url, fileInfo.Sha256, targetDir, fileInfo.Version, installer.Options{
		Out:      r.outWriter(),
		Err:      r.errWriter(),
		Quiet:    r.quiet,
		Progress: r.progress,
	})
}

func newInstallRunner() installRunner {
	return installRunner{
		fetchDownloadURL: fetcher.GetDownloadURL,
		installGo:        installer.InstallGo,
		resolveRoot:      func() (string, error) { return guenv.Resolve(globals.Home) },
		out:              os.Stdout,
		err:              os.Stderr,
	}
}

var installCmd = &cli.Command{
	Use:   "install [version]",
	Short: "Go를 설치하거나 특정 버전으로 업데이트합니다.",
	Long: `go.dev에서 릴리스 정보를 확인하고 알맞은 압축 파일을 설치합니다.
버전을 명시하지 않으면 최신 안정 버전을 설치하며, 마이너 버전만 지정하면 해당
마이너의 최신 안정 패치를 설치합니다.

예시:
  gu install             최신 안정 버전 설치
  gu install 1.20        1.20.x 최신 패치 설치
  gu install 1.20.5      정확한 버전 설치
  gu install -d /opt/go  설치 루트 지정 (gu --home과 동일)
  gu install --dry-run   설치하지 않고 버전/URL만 확인

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.MaximumNArgs(1),
	RunE: func(cmd *cli.Command, args []string) error {
		runner := newInstallRunner()
		runner.out = cmd.OutOrStdout()
		runner.err = cmd.ErrOrStderr()
		runner.quiet = globals.Quiet
		runner.dryRun = globals.DryRun
		runner.progress = prompt.IsTerminalWriter(cmd.ErrOrStderr())
		return runner.Run(args, installDir)
	},
}

func init() {
	installCmd.Flags().StringVarP(&installDir, "dir", "d", "", "Go가 설치될 최상위 디렉토리 (기본값: $GU_HOME 또는 ~/.go)")
	rootCmd.AddCommand(installCmd)
}
