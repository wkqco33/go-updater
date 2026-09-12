package cmd

import (
	"strings"
	"testing"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/systemgo"
)

func presentDir(path string) []systemgo.Present {
	return []systemgo.Present{{
		Artifact: systemgo.Artifact{Kind: systemgo.KindDir, Path: path, Managed: true},
		Exists:   true,
	}}
}

func TestRunCleanSystemReportsWhenNothingIsDetected(t *testing.T) {
	withSystemSeams(t, func() ([]systemgo.Present, error) { return nil, nil }, func([]systemgo.PlannedCommand) error {
		t.Fatal("removeSystemGo should not run when nothing is detected")
		return nil
	})

	out, _, err := runRootCommand(t, "clean", "--system")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "감지된 시스템 Go 설치가 없습니다") {
		t.Fatalf("output = %q", out)
	}
}

func TestRunCleanSystemShowsPlanAndRemovesWithYes(t *testing.T) {
	var planned []systemgo.PlannedCommand
	withSystemSeams(t,
		func() ([]systemgo.Present, error) {
			return append(presentDir("/usr/local/go"), systemgo.Present{
				Artifact: systemgo.Artifact{Kind: systemgo.KindFile, Path: systemgo.PathsDGo, Managed: true},
				Exists:   true,
			}), nil
		},
		func(steps []systemgo.PlannedCommand) error {
			planned = steps
			return nil
		})

	out, _, err := runRootCommand(t, "clean", "--system", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if len(planned) != 2 {
		t.Fatalf("planned steps = %d, want 2", len(planned))
	}
	if !strings.Contains(out, "다음 명령이 실행됩니다") {
		t.Fatalf("output does not show the plan: %q", out)
	}
	if !strings.Contains(out, "시스템 Go 설치가 성공적으로 삭제되었습니다") {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, "새 터미널 세션") {
		t.Fatalf("paths.d note missing: %q", out)
	}
}

func TestRunCleanSystemDryRunSkipsRemoval(t *testing.T) {
	withSystemSeams(t,
		func() ([]systemgo.Present, error) { return presentDir("/usr/local/go"), nil },
		func([]systemgo.PlannedCommand) error {
			t.Fatal("removeSystemGo should not run with --dry-run")
			return nil
		})

	out, _, err := runRootCommand(t, "clean", "--system", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dry-run") {
		t.Fatalf("output = %q", out)
	}
}

func TestRunCleanSystemRequiresConfirmationWithoutTTY(t *testing.T) {
	withTTY(t, false)
	withSystemSeams(t,
		func() ([]systemgo.Present, error) { return presentDir("/usr/local/go"), nil },
		func([]systemgo.PlannedCommand) error {
			t.Fatal("removeSystemGo should not run without confirmation")
			return nil
		})

	_, _, err := runRootCommand(t, "clean", "--system")
	if err == nil {
		t.Fatal("clean --system error = nil without confirmation")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Fatalf("ExitCode() = %d, want 2 (%v)", got, err)
	}
}

func TestRunCleanSystemReportsHomebrewOnly(t *testing.T) {
	withSystemSeams(t,
		func() ([]systemgo.Present, error) {
			return []systemgo.Present{{
				Artifact: systemgo.Artifact{Kind: systemgo.KindHomebrew, Path: "/opt/homebrew/opt/go"},
				Exists:   true,
			}}, nil
		},
		func([]systemgo.PlannedCommand) error {
			t.Fatal("Homebrew installs must never be removed")
			return nil
		})

	out, _, err := runRootCommand(t, "clean", "--system")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "brew uninstall go") {
		t.Fatalf("output = %q", out)
	}
}

func TestArtifactLabelDescribesEachKind(t *testing.T) {
	tests := []struct {
		artifact systemgo.Artifact
		want     string
	}{
		{systemgo.Artifact{Kind: systemgo.KindReceipt, Path: "org.golang.go"}, "pkgutil 리시트 org.golang.go"},
		{systemgo.Artifact{Kind: systemgo.KindHomebrew, Path: "/opt/homebrew/opt/go"}, "Homebrew Go (/opt/homebrew/opt/go)"},
		{systemgo.Artifact{Kind: systemgo.KindWinget, Path: "GoLang.Go"}, "winget Go (GoLang.Go)"},
		{systemgo.Artifact{Kind: systemgo.KindDir, Path: "/usr/local/go"}, "/usr/local/go"},
	}
	for _, tt := range tests {
		if got := artifactLabel(tt.artifact); got != tt.want {
			t.Fatalf("artifactLabel(%+v) = %q, want %q", tt.artifact, got, tt.want)
		}
	}
}
