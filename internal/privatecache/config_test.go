package privatecache

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseCSVPatterns(t *testing.T) {
	got := ParseCSVPatterns(" github.com/acme/*,git.example.com/*,github.com/acme/* ")
	want := []string{"github.com/acme/*", "git.example.com/*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseCSVPatterns()=%v want=%v", got, want)
	}
}

func TestBuildEnv(t *testing.T) {
	cfg := Config{
		PrivatePatterns: []string{"github.com/acme/*"},
		NoSumDBPatterns: []string{"github.com/acme/*"},
		NoProxyPatterns: []string{"github.com/acme/*"},
		CacheDir:        filepath.Join("tmp", "cache"),
	}
	env := BuildEnv(cfg, true)
	if env["GOPROXY"] != "off" {
		t.Fatalf("expected GOPROXY=off got %q", env["GOPROXY"])
	}
	if env["GOMODCACHE"] == "" {
		t.Fatalf("expected GOMODCACHE to be set")
	}
}

func TestLoadConfigUsesDefaultsWhenFileIsMissing(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("expected default version 1, got %d", cfg.Version)
	}
	if cfg.CacheDir == "" {
		t.Fatal("expected default cache dir to be set")
	}
}

func TestSaveConfigPersistsNormalizedValues(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cfg := Config{
		PrivatePatterns: []string{" github.com/acme/* ", "", "github.com/acme/*"},
		CacheDir:        "  /tmp/go-cache  ",
	}

	if err := SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.CacheDir != "/tmp/go-cache" {
		t.Fatalf("expected trimmed cache dir, got %q", loaded.CacheDir)
	}
	if len(loaded.PrivatePatterns) != 1 || loaded.PrivatePatterns[0] != "github.com/acme/*" {
		t.Fatalf("expected normalized private patterns, got %v", loaded.PrivatePatterns)
	}
	if len(loaded.NoSumDBPatterns) != 1 || loaded.NoSumDBPatterns[0] != "github.com/acme/*" {
		t.Fatalf("expected default nosumdb patterns, got %v", loaded.NoSumDBPatterns)
	}
}

func TestSaveConfigRejectsEmptyCacheDir(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	err := SaveConfig(cfgPath, Config{CacheDir: "   "})
	if err == nil {
		t.Fatal("expected error for empty cache dir")
	}
	if _, statErr := os.Stat(cfgPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected config file to not be created, stat err = %v", statErr)
	}
}

func TestDefaultPathsHonorGuHomeAndXDGCacheHome(t *testing.T) {
	root := t.TempDir()
	xdg := t.TempDir()
	t.Setenv("GU_HOME", root)
	t.Setenv("XDG_CACHE_HOME", xdg)

	base, err := DefaultBaseDir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "private"); base != want {
		t.Fatalf("DefaultBaseDir() = %q, want %q", base, want)
	}

	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(xdg, "go-updater", "modcache"); cfg.CacheDir != want {
		t.Fatalf("DefaultConfig().CacheDir = %q, want %q", cfg.CacheDir, want)
	}

	configPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "private", "config.json"); configPath != want {
		t.Fatalf("DefaultConfigPath() = %q, want %q", configPath, want)
	}
}

func TestDefaultConfigKeepsCacheUnderRootWithoutXDG(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GU_HOME", root)
	t.Setenv("XDG_CACHE_HOME", "")

	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "private", "modcache"); cfg.CacheDir != want {
		t.Fatalf("DefaultConfig().CacheDir = %q, want %q", cfg.CacheDir, want)
	}
}
