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
	for _, name := range []string{"go1.9.9", "go1.10.0", "go1.10.2", "go1.20.5", "not-a-version"} {
		if err := os.MkdirAll(filepath.Join(root, "versions", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return NewStore(root)
}

// activate points the current link at name the same way the use command does,
// without depending on installer code.
func activate(t *testing.T, store Store, name string) {
	t.Helper()
	if err := os.Symlink(filepath.Join(store.Root, "versions", name), filepath.Join(store.Root, "current")); err != nil {
		t.Fatal(err)
	}
}

func TestListSortsVersionsAndMarksActive(t *testing.T) {
	store := setupStore(t)
	activate(t, store, "go1.10.0")
	got, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	want := []Version{
		{Name: "go1.20.5"}, {Name: "go1.10.2"}, {Name: "go1.10.0", Active: true}, {Name: "go1.9.9"},
	}
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

func TestResolveDoesNotMatchDifferentMinorVersion(t *testing.T) {
	store := setupStore(t)
	if _, err := store.Resolve("1.2"); err == nil {
		t.Fatal("Resolve(\"1.2\") matched a 1.20 release; want not-installed error")
	}
}

func TestResolveRejectsUninstalledExactVersion(t *testing.T) {
	store := setupStore(t)
	if _, err := store.Resolve("1.10.20"); err == nil {
		t.Fatal("Resolve(\"1.10.20\") matched an unrelated prefix; want not-installed error")
	}
}

func TestRemoveActiveVersionIsRejected(t *testing.T) {
	store := setupStore(t)
	activate(t, store, "go1.10.0")
	if err := store.Remove("go1.10.0"); err == nil {
		t.Fatal("Remove() error = nil for active version")
	}
}

func TestRemoveAllRemovesVersionsAndCurrentLink(t *testing.T) {
	store := setupStore(t)
	activate(t, store, "go1.10.0")
	if err := store.RemoveAll(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "versions")); !os.IsNotExist(err) {
		t.Fatal("versions directory still exists")
	}
	if _, err := os.Lstat(filepath.Join(store.Root, "current")); !os.IsNotExist(err) {
		t.Fatal("current link still exists")
	}
}

func TestRemoveUnusedKeepsActiveVersion(t *testing.T) {
	store := setupStore(t)
	activate(t, store, "go1.10.0")
	removed, err := store.RemoveUnused()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(removed, []string{"go1.20.5", "go1.10.2", "go1.9.9"}) {
		t.Fatalf("removed = %#v", removed)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "versions", "go1.10.0")); err != nil {
		t.Fatal("active version was removed")
	}
}
