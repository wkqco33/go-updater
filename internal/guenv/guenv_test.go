package guenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootPrefersGuHomeOverride(t *testing.T) {
	override := t.TempDir()
	t.Setenv(EnvHome, override)

	got, err := Root()
	if err != nil {
		t.Fatal(err)
	}
	if got != override {
		t.Fatalf("Root() = %q, want %q", got, override)
	}
}

func TestRootFallsBackToHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv(EnvHome, "")
	t.Setenv("HOME", home)
	if os.Getenv("USERPROFILE") != "" {
		t.Setenv("USERPROFILE", home)
	}

	got, err := Root()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".go"); got != want {
		t.Fatalf("Root() = %q, want %q", got, want)
	}
}

func TestResolvePrefersExplicitOverride(t *testing.T) {
	t.Setenv(EnvHome, "/from/env")

	got, err := Resolve("/from/flag")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/from/flag" {
		t.Fatalf("Resolve() = %q, want /from/flag", got)
	}

	got, err = Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/from/env" {
		t.Fatalf("Resolve() = %q, want /from/env", got)
	}
}

func TestCacheDirHonorsXDGCacheHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv(EnvCacheHome, xdg)

	if got, want := CacheDir("/root"), filepath.Join(xdg, "go-updater", "modcache"); got != want {
		t.Fatalf("CacheDir() = %q, want %q", got, want)
	}
}

func TestCacheDirDefaultsUnderRootPrivate(t *testing.T) {
	t.Setenv(EnvCacheHome, "")

	if got, want := CacheDir("/root"), filepath.Join("/root", "private", "modcache"); got != want {
		t.Fatalf("CacheDir() = %q, want %q", got, want)
	}
}
