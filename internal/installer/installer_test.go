package installer

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.bin")
	data := []byte("installer test data")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if err := VerifyChecksum(path, hex.EncodeToString(sum[:])); err != nil {
		t.Fatalf("VerifyChecksum() error = %v", err)
	}
	if err := VerifyChecksum(path, "bad"); err == nil {
		t.Fatal("VerifyChecksum() error = nil for mismatched checksum")
	}
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.zip")
	archive, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(archive)
	file, err := zw.Create("go/bin/go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("go executable")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(dir, "dest")
	if err := extractZip(archivePath, dest); err != nil {
		t.Fatalf("extractZip() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "go/bin/go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "go executable" {
		t.Fatalf("extracted content = %q", got)
	}
}

func TestExtractZipRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "unsafe.zip")
	archive, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(archive)
	file, err := zw.Create("../outside")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte("must not escape"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(dir, "dest")
	if err := extractZip(archivePath, dest); err == nil {
		t.Fatal("extractZip() error = nil for path traversal archive")
	}
	if _, err := os.Stat(filepath.Join(dir, "outside")); !os.IsNotExist(err) {
		t.Fatal("path traversal created a file outside destination")
	}
}

func TestUpdateCurrentSymlink(t *testing.T) {
	dir := t.TempDir()
	goDir := filepath.Join(dir, "versions", "go1.23.0")
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := UpdateCurrentSymlink(dir, goDir); err != nil {
		t.Fatalf("UpdateCurrentSymlink() error = %v", err)
	}
	link := filepath.Join(dir, "current")
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != goDir {
		t.Fatalf("symlink target = %q, want %q", got, goDir)
	}
}
