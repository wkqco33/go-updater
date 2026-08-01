package privatecache

import (
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
