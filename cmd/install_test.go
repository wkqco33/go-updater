package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wkqco33/go-updater/internal/fetcher"
	"github.com/wkqco33/go-updater/internal/installer"
)

func stubRelease(string) (string, *fetcher.GoFile, error) {
	return "https://example.com/go.tar.gz", &fetcher.GoFile{Version: "go1.21.5", Sha256: "abc"}, nil
}

func TestInstallRunnerUsesProvidedDirectory(t *testing.T) {
	var gotTargetDir string
	var gotOptions installer.Options
	runner := installRunner{
		fetchDownloadURL: stubRelease,
		installGo: func(url, sha, targetDir, version string, opts installer.Options) error {
			gotTargetDir = targetDir
			gotOptions = opts
			return nil
		},
		resolveRoot: func() (string, error) { return "/from/root", nil },
		out:         &bytes.Buffer{},
		err:         &bytes.Buffer{},
	}

	if err := runner.Run([]string{"1.21.5"}, "/custom/dir"); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if gotTargetDir != "/custom/dir" {
		t.Fatalf("expected custom target dir, got %q", gotTargetDir)
	}
	if gotOptions.Quiet {
		t.Fatal("quiet should default to false")
	}
}

func TestInstallRunnerUsesResolvedRootWhenDirectoryNotProvided(t *testing.T) {
	var gotTargetDir string
	runner := installRunner{
		fetchDownloadURL: stubRelease,
		installGo: func(url, sha, targetDir, version string, opts installer.Options) error {
			gotTargetDir = targetDir
			return nil
		},
		resolveRoot: func() (string, error) { return "/from/root", nil },
		out:         &bytes.Buffer{},
		err:         &bytes.Buffer{},
	}

	if err := runner.Run(nil, ""); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if gotTargetDir != "/from/root" {
		t.Fatalf("expected resolved root, got %q", gotTargetDir)
	}
}

func TestInstallRunnerReturnsDownloadError(t *testing.T) {
	installCalled := false
	runner := installRunner{
		fetchDownloadURL: func(version string) (string, *fetcher.GoFile, error) {
			return "", nil, errors.New("boom")
		},
		installGo: func(url, sha, targetDir, version string, opts installer.Options) error {
			installCalled = true
			return nil
		},
		resolveRoot: func() (string, error) { return "/from/root", nil },
		out:         &bytes.Buffer{},
		err:         &bytes.Buffer{},
	}

	if err := runner.Run(nil, ""); err == nil {
		t.Fatal("expected error but got nil")
	}
	if installCalled {
		t.Fatal("installGo was called after a download failure")
	}
}

func TestInstallRunnerReturnsRootResolutionError(t *testing.T) {
	runner := installRunner{
		fetchDownloadURL: stubRelease,
		installGo: func(url, sha, targetDir, version string, opts installer.Options) error {
			t.Fatal("installGo should not run when the root cannot be resolved")
			return nil
		},
		resolveRoot: func() (string, error) { return "", errors.New("no home") },
		out:         &bytes.Buffer{},
		err:         &bytes.Buffer{},
	}

	if err := runner.Run(nil, ""); err == nil {
		t.Fatal("expected root resolution error")
	}
}

func TestInstallRunnerDryRunSkipsInstall(t *testing.T) {
	var out bytes.Buffer
	runner := installRunner{
		fetchDownloadURL: stubRelease,
		installGo: func(url, sha, targetDir, version string, opts installer.Options) error {
			t.Fatal("installGo should not run with --dry-run")
			return nil
		},
		resolveRoot: func() (string, error) { return "/from/root", nil },
		out:         &out,
		err:         &bytes.Buffer{},
		dryRun:      true,
	}

	if err := runner.Run([]string{"1.21.5"}, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "dry-run") || !strings.Contains(out.String(), "go1.21.5") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestInstallRunnerSeparatesStatusFromResults(t *testing.T) {
	var out, errOut bytes.Buffer
	runner := installRunner{
		fetchDownloadURL: stubRelease,
		installGo: func(url, sha, targetDir, version string, opts installer.Options) error {
			_, _ = opts.Out.Write([]byte("result\n"))
			_, _ = opts.Err.Write([]byte("progress\n"))
			return nil
		},
		resolveRoot: func() (string, error) { return "/from/root", nil },
		out:         &out,
		err:         &errOut,
	}

	if err := runner.Run(nil, ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Fetching") || strings.Contains(out.String(), "Found version") {
		t.Fatalf("status leaked to stdout: %q", out.String())
	}
	if !strings.Contains(errOut.String(), "Fetching latest Go release information") {
		t.Fatalf("stderr = %q", errOut.String())
	}
	if !strings.Contains(errOut.String(), "Found version: go1.21.5") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestNewInstallRunnerWiresProductionSeams(t *testing.T) {
	runner := newInstallRunner()
	if runner.fetchDownloadURL == nil || runner.installGo == nil || runner.resolveRoot == nil {
		t.Fatalf("runner seams = %+v", runner)
	}
	if runner.out == nil || runner.err == nil {
		t.Fatal("runner writers must default to the process streams")
	}
	if runner.dryRun || runner.quiet {
		t.Fatalf("runner should not start in dry-run/quiet mode: %+v", runner)
	}
}

func TestInstallRunnerQuietSuppressesStatus(t *testing.T) {
	var out, errOut bytes.Buffer
	runner := installRunner{
		fetchDownloadURL: stubRelease,
		installGo: func(url, sha, targetDir, version string, opts installer.Options) error {
			if !opts.Quiet {
				t.Fatal("installer options should carry the quiet flag")
			}
			return nil
		},
		resolveRoot: func() (string, error) { return "/from/root", nil },
		out:         &out,
		err:         &errOut,
		quiet:       true,
	}

	if err := runner.Run(nil, ""); err != nil {
		t.Fatal(err)
	}
	if errOut.String() != "" {
		t.Fatalf("stderr = %q, want empty with quiet", errOut.String())
	}
}
