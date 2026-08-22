package privatecache

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseModuleSpec(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		wantModule  string
		wantVersion string
		wantErr     bool
	}{
		{"with version", "github.com/acme/lib@v1.0.0", "github.com/acme/lib", "v1.0.0", false},
		{"without version", "github.com/acme/lib", "github.com/acme/lib", "", false},
		{"invalid", "@v1.0.0", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, version, err := ParseModuleSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseModuleSpec() err=%v wantErr=%v", err, tt.wantErr)
			}
			if module != tt.wantModule || version != tt.wantVersion {
				t.Fatalf("ParseModuleSpec() module=%q version=%q want module=%q version=%q", module, version, tt.wantModule, tt.wantVersion)
			}
		})
	}
}

func TestSyncModulesDownloadsAndPersistsMetadata(t *testing.T) {
	old := downloadModuleFn
	var calls []string
	downloadModuleFn = func(_ context.Context, _ Config, target string) (string, error) {
		calls = append(calls, target)
		return "v1.2.3", nil
	}
	t.Cleanup(func() { downloadModuleFn = old })

	dir := t.TempDir()
	cfg := Config{CacheDir: filepath.Join(dir, "cache")}
	metadataPath := filepath.Join(dir, "metadata.json")
	result, err := SyncModules(context.Background(), cfg, metadataPath,
		[]string{"github.com/acme/lib@v1.0.0", "github.com/acme/other"},
		2, true, "test")
	if err != nil {
		t.Fatalf("SyncModules() error = %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"github.com/acme/lib@v1.0.0", "github.com/acme/other@latest"}) {
		t.Fatalf("download calls = %#v", calls)
	}
	if len(result.Downloaded) != 2 || len(result.Failed) != 0 {
		t.Fatalf("result = %#v", result)
	}
	metadata, err := LoadMetadata(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Modules) != 2 || metadata.Modules[0].Source != "test" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestSyncModulesReportsInvalidAndMissingSpecsWithoutDownloading(t *testing.T) {
	old := downloadModuleFn
	called := false
	downloadModuleFn = func(context.Context, Config, string) (string, error) {
		called = true
		return "", nil
	}
	t.Cleanup(func() { downloadModuleFn = old })

	dir := t.TempDir()
	result, err := SyncModules(context.Background(), Config{CacheDir: filepath.Join(dir, "cache")},
		filepath.Join(dir, "metadata.json"), []string{"@bad", "github.com/acme/lib"}, 1, false, "")
	if err != nil {
		t.Fatalf("SyncModules() error = %v", err)
	}
	if called || len(result.Failed) != 2 {
		t.Fatalf("result = %#v, called=%v", result, called)
	}
}

func TestDownloadModuleWithRetryRetriesAndReturnsLastError(t *testing.T) {
	old := downloadModuleFn
	calls := 0
	wantErr := errors.New("temporary")
	downloadModuleFn = func(context.Context, Config, string) (string, error) {
		calls++
		return "", wantErr
	}
	t.Cleanup(func() { downloadModuleFn = old })

	_, err := downloadModuleWithRetry(context.Background(), Config{}, "github.com/acme/lib", "latest", 3)
	if !errors.Is(err, wantErr) || calls != 3 {
		t.Fatalf("error=%v calls=%d", err, calls)
	}
}

func TestDownloadModuleUsesInjectedCommandRunner(t *testing.T) {
	old := runCommandFn
	var calls []string
	runCommandFn = func(_ context.Context, dir string, env []string, name string, args ...string) ([]byte, []byte, error) {
		calls = append(calls, name+" "+filepath.Base(dir)+" "+strings.Join(args, " "))
		if args[1] == "download" && !strings.Contains(strings.Join(env, "\n"), "GO111MODULE=on") {
			return nil, nil, errors.New("GO111MODULE was not configured")
		}
		if args[1] == "init" {
			return nil, nil, nil
		}
		return []byte(`{"Path":"github.com/acme/lib","Version":"v1.2.3"}`), nil, nil
	}
	t.Cleanup(func() { runCommandFn = old })

	resolved, err := downloadModule(context.Background(), Config{CacheDir: filepath.Join(t.TempDir(), "cache")}, "github.com/acme/lib@latest")
	if err != nil || resolved != "v1.2.3" {
		t.Fatalf("resolved=%q err=%v", resolved, err)
	}
	if len(calls) != 2 || !strings.Contains(calls[1], "download -json github.com/acme/lib@latest") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestSyncModulesSkipsExistingResolvedVersion(t *testing.T) {
	old := downloadModuleFn
	called := false
	downloadModuleFn = func(context.Context, Config, string) (string, error) {
		called = true
		return "v1.0.0", nil
	}
	t.Cleanup(func() { downloadModuleFn = old })

	dir := t.TempDir()
	metadataPath := filepath.Join(dir, "metadata.json")
	if err := SaveMetadata(metadataPath, Metadata{Modules: []ModuleRecord{{
		Module: "github.com/acme/lib", RequestedVersion: "v1.0.0", ResolvedVersion: "v1.0.0",
	}}}); err != nil {
		t.Fatal(err)
	}
	result, err := SyncModules(context.Background(), Config{CacheDir: filepath.Join(dir, "cache")},
		metadataPath, []string{"github.com/acme/lib@v1.0.0"}, 1, false, "test")
	if err != nil || called || len(result.Skipped) != 1 {
		t.Fatalf("result=%#v err=%v called=%v", result, err, called)
	}
}
