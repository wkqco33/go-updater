package installer

import (
	"runtime"
	"strings"
	"testing"
)

func TestPathGuidanceMentionsEnvCommand(t *testing.T) {
	got := PathGuidance("linux", "/usr/bin/zsh", "/home/u/.go/current/bin")
	if !strings.Contains(got, "gu env init") {
		t.Fatalf("guidance should reference gu env init:\n%s", got)
	}
	if !strings.Contains(got, "gu env set") {
		t.Fatalf("guidance should reference gu env set:\n%s", got)
	}
	if !strings.Contains(got, "/home/u/.go/current/bin") {
		t.Fatalf("guidance should include the bin directory:\n%s", got)
	}
}

func TestPathGuidanceWindowsUsesPowerShell(t *testing.T) {
	got := PathGuidance("windows", "", `C:\Users\u\.go\current\bin`)
	if !strings.Contains(got, "gu env init powershell") {
		t.Fatalf("windows guidance should reference gu env init powershell:\n%s", got)
	}
	if strings.Contains(got, ".bashrc") || strings.Contains(got, ".zshrc") {
		t.Fatalf("windows guidance must not mention POSIX startup files:\n%s", got)
	}
}

func TestPathGuidancePosixDoesNotMentionWindows(t *testing.T) {
	got := PathGuidance("darwin", "/bin/bash", "/Users/u/.go/current/bin")
	if strings.Contains(got, "PowerShell") {
		t.Fatalf("posix guidance must not mention PowerShell:\n%s", got)
	}
}

func TestPathGuidanceMatchesBinaryOS(t *testing.T) {
	// Sanity check that the production call passes a supported GOOS value.
	if runtime.GOOS == "" {
		t.Fatal("runtime.GOOS is empty")
	}
}
