package systemgo

import (
	"runtime"
	"strings"
	"testing"
)

func TestPlanSkipsUnmanagedAndMissingArtifacts(t *testing.T) {
	items := []Present{
		{Artifact: Artifact{Kind: KindHomebrew, Path: "/brew/opt/go", Managed: false}, Exists: true},
		{Artifact: Artifact{Kind: KindDir, Path: "/usr/local/go", Managed: true}, Exists: false},
	}
	if got := Plan(items); len(got) != 0 {
		t.Fatalf("Plan() = %+v, want no commands", got)
	}
}

func TestPlanIncludesManagedArtifacts(t *testing.T) {
	items := []Present{
		{Artifact: Artifact{Kind: KindDir, Path: "/tmp/go", Managed: true}, Exists: true},
		{Artifact: Artifact{Kind: KindFile, Path: "/tmp/paths-go", Managed: true}, Exists: true},
		{Artifact: Artifact{Kind: KindReceipt, Path: "org.golang.go", Managed: true}, Exists: true},
		{Artifact: Artifact{Kind: KindWinget, Path: "GoLang.Go", Managed: true}, Exists: true},
	}
	got := Plan(items)
	if len(got) != len(items) {
		t.Fatalf("Plan() returned %d commands, want %d", len(got), len(items))
	}
	if !strings.Contains(got[2].Display, "org.golang.go") {
		t.Fatalf("receipt command = %q", got[2].Display)
	}
	if runtime.GOOS != "windows" && !strings.Contains(got[0].Display, "rm") {
		t.Fatalf("directory command = %q", got[0].Display)
	}
}
