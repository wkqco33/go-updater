package privatecache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeCacheFile(t *testing.T, dir, name string, size int, age time.Duration) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCleanCacheAllRemovesCacheAndMetadata(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	metadata := filepath.Join(dir, "metadata.json")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCacheFile(t, cacheDir, "module.zip", 10, 0)
	if err := os.WriteFile(metadata, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := CleanCache(Config{CacheDir: cacheDir}, metadata, CleanOptions{All: true})
	if err != nil {
		t.Fatalf("CleanCache() error = %v", err)
	}
	if result.FreedBytes != 10 || result.FinalSize != 0 {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatal("cache directory still exists")
	}
	if _, err := os.Stat(metadata); !os.IsNotExist(err) {
		t.Fatal("metadata file still exists")
	}
}

func TestCleanCacheRemovesStaleFiles(t *testing.T) {
	dir := t.TempDir()
	old := writeCacheFile(t, dir, "old", 7, 48*time.Hour)
	fresh := writeCacheFile(t, dir, "fresh", 11, time.Hour)

	result, err := CleanCache(Config{CacheDir: dir}, filepath.Join(dir, "metadata.json"), CleanOptions{StaleDays: 1})
	if err != nil {
		t.Fatalf("CleanCache() error = %v", err)
	}
	if result.RemovedFiles != 1 || result.FreedBytes != 7 || result.FinalSize != 11 {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("stale file still exists")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("fresh file was removed")
	}
}

func TestCleanCacheEnforcesMaxSizeByOldestFirst(t *testing.T) {
	dir := t.TempDir()
	old := writeCacheFile(t, dir, "old", 600*1024, 3*time.Hour)
	newer := writeCacheFile(t, dir, "newer", 600*1024, 2*time.Hour)
	newest := writeCacheFile(t, dir, "newest", 600*1024, time.Hour)

	result, err := CleanCache(Config{CacheDir: dir}, filepath.Join(dir, "metadata.json"), CleanOptions{MaxSizeMB: 1})
	if err != nil {
		t.Fatalf("CleanCache() error = %v", err)
	}
	if result.RemovedFiles != 2 || result.FinalSize != 600*1024 {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("oldest file still exists")
	}
	if _, err := os.Stat(newer); !os.IsNotExist(err) {
		t.Fatal("second-oldest file still exists")
	}
	if _, err := os.Stat(newest); err != nil {
		t.Fatal("newest file was removed")
	}
}
