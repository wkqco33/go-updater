package privatecache

import "testing"

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
