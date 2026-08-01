package privatecache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildMetadataIndex(t *testing.T) {
	records := []ModuleRecord{
		{Module: "github.com/acme/lib", ResolvedVersion: "v1.2.3"},
		{Module: "github.com/acme/lib2", ResolvedVersion: "v0.1.0"},
	}
	index := BuildMetadataIndex(records)
	if _, ok := index["github.com/acme/lib@v1.2.3"]; !ok {
		t.Fatalf("expected index to contain github.com/acme/lib@v1.2.3")
	}
	if _, ok := index["github.com/acme/lib2@v0.1.0"]; !ok {
		t.Fatalf("expected index to contain github.com/acme/lib2@v0.1.0")
	}
}

func TestMetadataRoundTrip(t *testing.T) {
	metadataPath := filepath.Join(t.TempDir(), "metadata.json")
	metadata := Metadata{Modules: []ModuleRecord{{Module: "github.com/acme/lib", ResolvedVersion: "v1.2.3"}}}

	if err := SaveMetadata(metadataPath, metadata); err != nil {
		t.Fatalf("SaveMetadata() error = %v", err)
	}

	loaded, err := LoadMetadata(metadataPath)
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v", err)
	}
	if len(loaded.Modules) != 1 {
		t.Fatalf("expected one module, got %d", len(loaded.Modules))
	}
	if loaded.Modules[0].Module != "github.com/acme/lib" {
		t.Fatalf("expected module github.com/acme/lib, got %q", loaded.Modules[0].Module)
	}
	if loaded.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be populated")
	}
	if _, err := os.Stat(metadataPath); err != nil {
		t.Fatalf("expected metadata file to exist: %v", err)
	}
}
