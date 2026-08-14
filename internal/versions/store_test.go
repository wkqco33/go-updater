package versions

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func setupStore(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"go1.9.9", "go1.10.0", "go1.10.2", "not-a-version"} {
		if err := os.MkdirAll(filepath.Join(root, "versions", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return NewStore(root)
}

func TestListSortsVersionsAndMarksActive(t *testing.T) {
	store := setupStore(t)
	if err := store.Activate("go1.10.0"); err != nil {
		t.Fatal(err)
	}
	got, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	want := []Version{{Name: "go1.10.2"}, {Name: "go1.10.0", Active: true}, {Name: "go1.9.9"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestResolveUsesHighestMatchingPatch(t *testing.T) {
	store := setupStore(t)
	got, err := store.Resolve("1.10")
	if err != nil {
		t.Fatal(err)
	}
	if got != "go1.10.2" {
		t.Fatalf("Resolve() = %q, want go1.10.2", got)
	}
}

func TestRemoveActiveVersionIsRejected(t *testing.T) {
	store := setupStore(t)
	if err := store.Activate("go1.10.0"); err != nil {
		t.Fatal(err)
	}
	if err := store.Remove("go1.10.0"); err == nil {
		t.Fatal("Remove() error = nil for active version")
	}
}

func TestRemoveUnusedKeepsActiveVersion(t *testing.T) {
	store := setupStore(t)
	if err := store.Activate("go1.10.0"); err != nil {
		t.Fatal(err)
	}
	removed, err := store.RemoveUnused()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(removed, []string{"go1.10.2", "go1.9.9"}) {
		t.Fatalf("removed = %#v", removed)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "versions", "go1.10.0")); err != nil {
		t.Fatal("active version was removed")
	}
}
