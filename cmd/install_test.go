package cmd

import (
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/wkqco33/go-updater/internal/fetcher"
)

func TestInstallRunnerUsesProvidedDirectory(t *testing.T) {
	var gotTargetDir string
	runner := installRunner{
		fetchDownloadURL: func(version string) (string, *fetcher.GoFile, error) {
			return "https://example.com/go.tar.gz", &fetcher.GoFile{Version: "go1.21.5", Sha256: "abc"}, nil
		},
		installGo: func(url, sha, targetDir, version string) error {
			gotTargetDir = targetDir
			return nil
		},
		userHomeDir: func() (string, error) {
			return "/home/test", nil
		},
	}

	if err := runner.Run([]string{"1.21.5"}, "/custom/dir", io.Discard); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if gotTargetDir != "/custom/dir" {
		t.Fatalf("expected custom target dir, got %q", gotTargetDir)
	}
}

func TestInstallRunnerUsesHomeDirectoryWhenNotProvided(t *testing.T) {
	var gotTargetDir string
	runner := installRunner{
		fetchDownloadURL: func(version string) (string, *fetcher.GoFile, error) {
			return "https://example.com/go.tar.gz", &fetcher.GoFile{Version: "go1.21.5", Sha256: "abc"}, nil
		},
		installGo: func(url, sha, targetDir, version string) error {
			gotTargetDir = targetDir
			return nil
		},
		userHomeDir: func() (string, error) {
			return "/home/test", nil
		},
	}

	if err := runner.Run(nil, "", io.Discard); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	want := filepath.Join("/home/test", ".go")
	if gotTargetDir != want {
		t.Fatalf("expected %q, got %q", want, gotTargetDir)
	}
}

func TestInstallRunnerReturnsDownloadError(t *testing.T) {
	runner := installRunner{
		fetchDownloadURL: func(version string) (string, *fetcher.GoFile, error) {
			return "", nil, errors.New("boom")
		},
		installGo: func(url, sha, targetDir, version string) error {
			return nil
		},
		userHomeDir: func() (string, error) {
			return "/home/test", nil
		},
	}

	if err := runner.Run(nil, "", io.Discard); err == nil {
		t.Fatal("expected error but got nil")
	}
}
