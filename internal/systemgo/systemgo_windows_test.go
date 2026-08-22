//go:build windows

package systemgo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectWingetGoFindsSuccessfulWingetCommand(t *testing.T) {
	dir := t.TempDir()
	winget := filepath.Join(dir, "winget.cmd")
	if err := os.WriteFile(winget, []byte("@echo off\r\nexit /b 0\r\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	found, err := detectWingetGo()
	if err != nil {
		t.Fatalf("detectWingetGo() error = %v", err)
	}
	if !found {
		t.Fatal("detectWingetGo() = false, want true")
	}
}

func TestRunWingetUninstallUsesWingetOnPath(t *testing.T) {
	dir := t.TempDir()
	winget := filepath.Join(dir, "winget.cmd")
	if err := os.WriteFile(winget, []byte("@echo off\r\nexit /b 0\r\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := runWingetUninstall(); err != nil {
		t.Fatalf("runWingetUninstall() error = %v", err)
	}
}

func TestRunWingetUninstallReportsMissingCommand(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := runWingetUninstall(); err == nil {
		t.Fatal("runWingetUninstall() error = nil, want missing command error")
	}
}

func TestDetectWingetGoReturnsFalseWhenCommandFails(t *testing.T) {
	dir := t.TempDir()
	winget := filepath.Join(dir, "winget.cmd")
	if err := os.WriteFile(winget, []byte("@echo off\r\nexit /b 1\r\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	found, err := detectWingetGo()
	if err != nil {
		t.Fatalf("detectWingetGo() error = %v", err)
	}
	if found {
		t.Fatal("detectWingetGo() = true, want false")
	}
}
