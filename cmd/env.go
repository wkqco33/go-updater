package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/guenv"
	"github.com/wkqco33/go-updater/internal/privatecache"
	"github.com/wkqco33/go-updater/internal/shellenv"
)

var (
	envShell  string
	envJSON   bool
	envGoroot bool
	envGopath string
	envModule bool
	envRC     string
)

var envCmd = &cli.Command{
	Use:   "env",
	Short: "Go 셸 환경변수(PATH)를 출력하거나 설정합니다.",
	Long: `current 링크의 bin 디렉터리를 PATH에 추가하는 셸 설정을 다룹니다.

예시:
  eval "$(gu env init)"        현재 셸에 즉시 적용
  gu env init zsh             zsh용 코드만 출력
  gu env show                 현재 설정 상태 확인
  gu env set                  ~/.config/gu/env.sh 생성 후 시작 파일에 연결
  gu env unset                시작 파일의 gu 블록 제거

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.NoArgs,
	RunE: func(cmd *cli.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

var envInitCmd = &cli.Command{
	Use:   "init [shell]",
	Short: "현재 셸에 적용할 환경변수 코드를 출력합니다.",
	Long: `설치된 Go의 bin 디렉터리를 PATH 앞에 추가하는 셸 코드를 stdout으로 출력합니다.
파일을 수정하지 않으므로 eval로 즉시 적용하거나 시작 파일에 직접 넣을 수 있습니다.

예시:
  eval "$(gu env init)"
  gu env init fish > ~/.config/fish/conf.d/gu.fish

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.MaximumNArgs(1),
	RunE: runEnvInit,
}

var envShowCmd = &cli.Command{
	Use:   "show",
	Short: "현재 셸 환경 설정 상태를 출력합니다.",
	Long: `감지된 셸, Go bin 디렉터리, PATH 포함 여부, gu가 관리하는 환경 파일과 시작
파일의 설정 여부를 출력합니다. --json으로 기계 판독 출력을 사용할 수 있습니다.

예시:
  gu env show
  gu env show --json

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.NoArgs,
	RunE: runEnvShow,
}

var envSetCmd = &cli.Command{
	Use:   "set",
	Short: "환경 파일을 만들고 시작 파일에 연결합니다.",
	Long: `gu가 소유한 환경 파일(~/.config/gu/env.sh)을 만들고, 시작 파일에 source 한 줄을
추가합니다. 기존 사용자 설정은 그대로 두고 gu 블록만 삽입하며, 수정 전에 백업을
남깁니다. 반복 실행해도 블록이 중복되지 않습니다.

예시:
  gu env set
  gu env set --shell fish
  gu env set --dry-run
  gu env set --goroot --module

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.NoArgs,
	RunE: runEnvSet,
}

func runEnvInit(cmd *cli.Command, args []string) error {
	shell, err := resolveEnvShell(args)
	if err != nil {
		return cli.NewUsageError(err)
	}
	setup, err := resolveEnvSetup(shell)
	if err != nil {
		return err
	}
	snippet, err := shellenv.Snippet(setup)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), snippet)
	return err
}

// resolveEnvShell picks the target shell from the positional argument, then
// the --shell flag, then the login shell. Unknown shells are reported to the
// caller so it can classify them as usage errors.
func resolveEnvShell(args []string) (shellenv.Shell, error) {
	if len(args) > 0 {
		return shellenv.Parse(args[0])
	}
	if strings.TrimSpace(envShell) != "" {
		return shellenv.Parse(envShell)
	}
	return shellenv.Detect(runtime.GOOS, os.Getenv("SHELL"))
}

// resolveEnvSetup builds the environment gu wants to configure. Only PATH is
// always set; GOROOT, GOPATH, and the private module cache variables are
// opt-in so gu never overrides a user's existing Go configuration by default.
func resolveEnvSetup(shell shellenv.Shell) (shellenv.Setup, error) {
	root, err := guenv.Resolve(globals.Home)
	if err != nil {
		return shellenv.Setup{}, err
	}
	setup := shellenv.Setup{
		Shell:  shell,
		BinDir: filepath.Join(root, "current", "bin"),
	}
	if envGoroot {
		setup.Goroot = filepath.Join(root, "current")
	}
	if trimmed := strings.TrimSpace(envGopath); trimmed != "" {
		setup.Gopath = trimmed
	}
	if envModule {
		cfg, err := privatecache.LoadConfigForRoot(root, privatecache.ConfigPath(root))
		if err != nil {
			return shellenv.Setup{}, fmt.Errorf("failed to load config: %w", err)
		}
		setup.Module = privatecache.BuildEnv(cfg, false)
	}
	return setup, nil
}

type envRCStatus struct {
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	Configured bool   `json:"configured"`
}

type envShowOutput struct {
	Shell         string        `json:"shell"`
	Root          string        `json:"root"`
	BinDir        string        `json:"bin_dir"`
	PathHasBin    bool          `json:"path_has_bin"`
	EnvFile       string        `json:"env_file"`
	EnvFileExists bool          `json:"env_file_exists"`
	RCFiles       []envRCStatus `json:"rc_files"`
	SystemGo      []string      `json:"system_go,omitempty"`
}

func runEnvShow(cmd *cli.Command, args []string) error {
	shell, err := resolveEnvShell(nil)
	if err != nil {
		return cli.NewUsageError(err)
	}
	root, err := guenv.Resolve(globals.Home)
	if err != nil {
		return err
	}
	home, err := guenv.HomeDir()
	if err != nil {
		return err
	}
	configHome, err := guenv.ConfigDir()
	if err != nil {
		return err
	}

	output := envShowOutput{
		Shell:      string(shell),
		Root:       root,
		BinDir:     filepath.Join(root, "current", "bin"),
		PathHasBin: pathContains(os.Getenv("PATH"), filepath.Join(root, "current", "bin")),
	}
	if envFile, err := shellenv.EnvFile(shell, home, configHome); err == nil {
		output.EnvFile = envFile
		_, statErr := os.Stat(envFile)
		output.EnvFileExists = statErr == nil
	}

	if rcFiles, err := shellenv.RCFiles(shell, home); err == nil {
		output.RCFiles = make([]envRCStatus, 0, len(rcFiles))
		for _, rc := range rcFiles {
			status := envRCStatus{Path: rc}
			data, readErr := os.ReadFile(rc)
			if readErr == nil {
				status.Exists = true
				status.Configured = shellenv.HasBlock(string(data))
			}
			output.RCFiles = append(output.RCFiles, status)
		}
	}

	// A leftover go.dev/system install can shadow gu's PATH entry, so surface
	// it as a warning instead of silently preferring whichever comes first.
	if items, err := detectSystemGo(); err == nil {
		for _, item := range items {
			if item.Exists {
				output.SystemGo = append(output.SystemGo, item.Artifact.Path)
			}
		}
	}

	if envJSON {
		encoded, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to render env status: %w", err)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "셸:               %s\n", output.Shell)
	fmt.Fprintf(out, "gu 루트:           %s\n", output.Root)
	fmt.Fprintf(out, "Go bin 디렉터리:   %s\n", output.BinDir)
	fmt.Fprintf(out, "PATH 포함:         %s\n", yesNo(output.PathHasBin))
	if output.EnvFile != "" {
		fmt.Fprintf(out, "환경 파일:         %s (%s)\n", output.EnvFile, existsLabel(output.EnvFileExists))
	} else {
		fmt.Fprintln(out, "환경 파일:         이 셸은 전용 파일을 사용하지 않습니다")
	}
	for _, rc := range output.RCFiles {
		fmt.Fprintf(out, "시작 파일:         %s (%s)\n", rc.Path, configuredLabel(rc.Exists, rc.Configured))
	}
	if !output.PathHasBin {
		fmt.Fprintln(out, "\n적용 방법: eval \"$(gu env init)\" 또는 gu env set")
	}
	if len(output.SystemGo) > 0 {
		fmt.Fprintf(out, "\n주의: 시스템 Go 설치가 감지되었습니다: %s\n", strings.Join(output.SystemGo, ", "))
		fmt.Fprintln(out, "      PATH 순서에 따라 gu가 관리하는 Go 대신 사용될 수 있습니다.")
	}
	return nil
}

func pathContains(pathEnv, dir string) bool {
	if dir == "" {
		return false
	}
	return slices.Contains(filepath.SplitList(pathEnv), dir)
}

func yesNo(value bool) string {
	if value {
		return "예"
	}
	return "아니오"
}

func existsLabel(exists bool) string {
	if exists {
		return "있음"
	}
	return "없음"
}

func configuredLabel(exists, configured bool) string {
	switch {
	case configured:
		return "설정됨"
	case exists:
		return "미설정"
	default:
		return "파일 없음"
	}
}

var envUnsetCmd = &cli.Command{
	Use:   "unset",
	Short: "시작 파일에서 gu 환경 설정 블록을 제거합니다.",
	Long: `gu가 추가한 시작 파일 블록과 환경 파일만 제거하며, 사용자가 직접 작성한
설정은 건드리지 않습니다. 실행 전에 확인 프롬프트를 거칩니다.

예시:
  gu env unset
  gu env unset --dry-run
  gu env unset --yes

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.NoArgs,
	RunE: runEnvUnset,
}

// envIsWindows and the Windows persistence seams let tests exercise the
// platform branch without PowerShell or the real user environment.
var (
	envIsWindows        = runtime.GOOS == "windows"
	envPersistWindows   = shellenv.PersistWindows
	envUnpersistWindows = shellenv.UnpersistWindows
)

// useWindowsUserPath reports whether the Windows user PATH is the right target:
// only on Windows, only for shells without a startup file gu can own, and never
// when the user pinned an explicit --rc (e.g. Git Bash on Windows).
func useWindowsUserPath(shell shellenv.Shell) bool {
	if !envIsWindows || strings.TrimSpace(envRC) != "" {
		return false
	}
	return shell == shellenv.ShellPowerShell || shell == shellenv.ShellCmd
}

func runEnvSetWindows(cmd *cli.Command, setup shellenv.Setup) error {
	result, err := envPersistWindows(shellenv.WindowsOptions{BinDir: setup.BinDir, DryRun: globals.DryRun})
	if err != nil {
		return err
	}
	prefix := ""
	if globals.DryRun {
		prefix = "dry-run: "
	}
	out := cmd.OutOrStdout()
	switch {
	case result.AlreadySet:
		fmt.Fprintf(out, "%s사용자 PATH에 이미 포함되어 있습니다: %s\n", prefix, result.BinDir)
	case result.Changed:
		fmt.Fprintf(out, "%s사용자 PATH에 추가합니다: %s\n", prefix, result.BinDir)
	default:
		fmt.Fprintf(out, "사용자 PATH 변경 없음: %s\n", result.BinDir)
	}
	if !globals.DryRun && result.Changed {
		statusf(cmd, "새 터미널(또는 새 프로세스)에서 적용됩니다.\n")
	}
	return nil
}

func runEnvUnsetWindows(cmd *cli.Command) error {
	root, err := guenv.Resolve(globals.Home)
	if err != nil {
		return err
	}
	binDir := filepath.Join(root, "current", "bin")

	if !globals.DryRun {
		approved, err := confirm(cmd, "Windows 사용자 PATH에서 gu가 추가한 항목을 제거할까요?")
		if err != nil {
			return err
		}
		if !approved {
			fmt.Fprintln(cmd.ErrOrStderr(), "취소되었습니다.")
			return nil
		}
	}

	result, err := envUnpersistWindows(shellenv.WindowsOptions{BinDir: binDir, DryRun: globals.DryRun})
	if err != nil {
		return err
	}
	prefix := ""
	if globals.DryRun {
		prefix = "dry-run: "
	}
	if result.Changed {
		fmt.Fprintf(cmd.OutOrStdout(), "%s사용자 PATH에서 제거했습니다: %s\n", prefix, binDir)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "사용자 PATH 변경 없음: %s\n", binDir)
	}
	return nil
}

func runEnvSet(cmd *cli.Command, args []string) error {
	shell, err := resolveEnvShell(nil)
	if err != nil {
		return cli.NewUsageError(err)
	}
	setup, err := resolveEnvSetup(shell)
	if err != nil {
		return err
	}
	if useWindowsUserPath(shell) {
		return runEnvSetWindows(cmd, setup)
	}
	envFile, rcFiles, err := resolveEnvPaths(shell)
	if err != nil {
		return cli.NewUsageError(err)
	}

	result, err := shellenv.Persist(shellenv.PersistOptions{
		Setup:   setup,
		EnvFile: envFile,
		RCFiles: rcFiles,
		DryRun:  globals.DryRun,
	})
	if err != nil {
		return err
	}

	prefix := ""
	if globals.DryRun {
		prefix = "dry-run: "
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s환경 파일: %s (%s)\n", prefix, result.EnvFile, envFileLabel(result.EnvFileExisted, result.EnvFileChanged))
	for _, rc := range result.RCFiles {
		fmt.Fprintf(out, "%s시작 파일: %s (%s)\n", prefix, rc.Path, rcChangeLabel(rc))
		if rc.Backup != "" {
			fmt.Fprintf(out, "  백업: %s\n", rc.Backup)
		}
	}
	if !globals.DryRun {
		statusf(cmd, "새 터미널을 열거나 'source %s'로 적용하세요.\n", result.EnvFile)
	}
	return nil
}

func runEnvUnset(cmd *cli.Command, args []string) error {
	shell, err := resolveEnvShell(nil)
	if err != nil {
		return cli.NewUsageError(err)
	}
	if useWindowsUserPath(shell) {
		return runEnvUnsetWindows(cmd)
	}
	envFile, rcFiles, err := resolveEnvPaths(shell)
	if err != nil {
		return cli.NewUsageError(err)
	}

	if !globals.DryRun {
		approved, err := confirm(cmd, "시작 파일에서 gu 설정 블록을 제거하고 환경 파일을 삭제할까요?")
		if err != nil {
			return err
		}
		if !approved {
			fmt.Fprintln(cmd.ErrOrStderr(), "취소되었습니다.")
			return nil
		}
	}

	result, err := shellenv.Unpersist(shellenv.PersistOptions{
		EnvFile: envFile,
		RCFiles: rcFiles,
		DryRun:  globals.DryRun,
	})
	if err != nil {
		return err
	}

	prefix := ""
	if globals.DryRun {
		prefix = "dry-run: "
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s환경 파일: %s (%s)\n", prefix, result.EnvFile, removalLabel(result.EnvFileChanged))
	for _, rc := range result.RCFiles {
		fmt.Fprintf(out, "%s시작 파일: %s (%s)\n", prefix, rc.Path, rcRemovalLabel(rc))
		if rc.Backup != "" {
			fmt.Fprintf(out, "  백업: %s\n", rc.Backup)
		}
	}
	return nil
}

// resolveEnvPaths resolves the managed env file and the startup files to edit.
// --rc overrides shell detection so a user with a non-standard layout stays in
// control, and an unsupported shell is reported as a usage error.
func resolveEnvPaths(shell shellenv.Shell) (string, []string, error) {
	home, err := guenv.HomeDir()
	if err != nil {
		return "", nil, err
	}
	configHome, err := guenv.ConfigDir()
	if err != nil {
		return "", nil, err
	}
	envFile, err := shellenv.EnvFile(shell, home, configHome)
	if err != nil {
		return "", nil, err
	}
	if trimmed := strings.TrimSpace(envRC); trimmed != "" {
		return envFile, []string{trimmed}, nil
	}
	rcFiles, err := shellenv.RCFiles(shell, home)
	if err != nil {
		return "", nil, fmt.Errorf("%w (--rc로 대상 파일을 지정하세요)", err)
	}
	return envFile, rcFiles, nil
}

func envFileLabel(existed, changed bool) string {
	switch {
	case changed && existed:
		return "갱신됨"
	case changed:
		return "생성됨"
	default:
		return "변경 없음"
	}
}

func removalLabel(removed bool) string {
	if removed {
		return "삭제됨"
	}
	return "변경 없음"
}

func rcChangeLabel(rc shellenv.RCResult) string {
	switch {
	case rc.Changed && rc.Exists:
		return "갱신됨"
	case rc.Changed:
		return "생성됨"
	default:
		return "변경 없음"
	}
}

func rcRemovalLabel(rc shellenv.RCResult) string {
	if rc.Changed {
		return "제거됨"
	}
	return "변경 없음"
}

func init() {
	envInitCmd.Flags().StringVar(&envShell, "shell", "", "", "대상 셸 (zsh, bash, sh, fish, powershell, cmd)")
	envInitCmd.Flags().BoolVar(&envGoroot, "goroot", "", false, "GOROOT도 current로 설정합니다")
	envInitCmd.Flags().StringVar(&envGopath, "gopath", "", "", "GOPATH도 함께 설정합니다")
	envInitCmd.Flags().BoolVar(&envModule, "module", "", false, "프라이빗 모듈 캐시 환경변수도 함께 출력합니다")
	envShowCmd.Flags().StringVar(&envShell, "shell", "", "", "대상 셸 (zsh, bash, sh, fish, powershell, cmd)")
	envShowCmd.Flags().BoolVar(&envJSON, "json", "", false, "JSON으로 출력합니다")
	for _, command := range []*cli.Command{envSetCmd, envUnsetCmd} {
		command.Flags().StringVar(&envShell, "shell", "", "", "대상 셸 (zsh, bash, sh, fish, powershell, cmd)")
		command.Flags().StringVar(&envRC, "rc", "", "", "시작 파일 경로를 직접 지정합니다")
	}
	envSetCmd.Flags().BoolVar(&envGoroot, "goroot", "", false, "GOROOT도 current로 설정합니다")
	envSetCmd.Flags().StringVar(&envGopath, "gopath", "", "", "GOPATH도 함께 설정합니다")
	envSetCmd.Flags().BoolVar(&envModule, "module", "", false, "프라이빗 모듈 캐시 환경변수도 함께 저장합니다")
	envCmd.AddCommand(envInitCmd, envShowCmd, envSetCmd, envUnsetCmd)
	rootCmd.AddCommand(envCmd)
}
