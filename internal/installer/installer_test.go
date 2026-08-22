package installer

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

func TestInstallGoCompletesSuccessfulInstallation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar.gz is not the Windows installer format")
	}
	archive := makeTarGz(t, map[string]string{
		"go/VERSION": "go1.23.0",
		"go/bin/go":  "go executable",
	})
	sum := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	target := t.TempDir()
	if err := InstallGo(server.URL+"/go.tar.gz", hex.EncodeToString(sum[:]), target, "go1.23.0"); err != nil {
		t.Fatalf("InstallGo() error = %v", err)
	}
	installed := filepath.Join(target, "versions", "go1.23.0")
	if got, err := os.ReadFile(filepath.Join(installed, "bin", "go")); err != nil || string(got) != "go executable" {
		t.Fatalf("installed executable = %q, error=%v", got, err)
	}
	link, err := os.Readlink(filepath.Join(target, "current"))
	if err != nil {
		t.Fatal(err)
	}
	if link != installed {
		t.Fatalf("current link = %q, want %q", link, installed)
	}
}

func TestInstallGoKeepsExistingVersionWhenArchiveIsInvalid(t *testing.T) {
	archive := makeTarGz(t, map[string]string{"readme.txt": "not a Go distribution"})
	sum := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	target := t.TempDir()
	final := filepath.Join(target, "versions", "go1.23.0")
	if err := os.MkdirAll(final, 0o755); err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(final, "old.txt")
	if err := os.WriteFile(oldFile, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := InstallGo(server.URL+"/go.tar.gz", hex.EncodeToString(sum[:]), target, "go1.23.0")
	if err == nil {
		t.Fatal("InstallGo() error = nil for invalid distribution")
	}
	got, readErr := os.ReadFile(oldFile)
	if readErr != nil {
		t.Fatalf("existing installation was not preserved: %v", readErr)
	}
	if string(got) != "keep me" {
		t.Fatalf("existing file = %q", got)
	}
}

func makeTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
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

	otherDir := filepath.Join(dir, "versions", "go1.24.0")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := UpdateCurrentSymlink(dir, otherDir); err != nil {
		t.Fatalf("second update error = %v", err)
	}
	got, err = os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != otherDir {
		t.Fatalf("updated symlink target = %q, want %q", got, otherDir)
	}
}
