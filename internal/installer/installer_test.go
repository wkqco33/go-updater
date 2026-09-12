package installer

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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

func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "unsafe.tar.gz")
	if err := os.WriteFile(archivePath, makeTarGz(t, map[string]string{"../outside": "must not escape"}), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "dest")
	if err := extractTarGz(archivePath, dest); err == nil {
		t.Fatal("extractTarGz() error = nil for path traversal archive")
	}
	if _, err := os.Stat(filepath.Join(dir, "outside")); !os.IsNotExist(err) {
		t.Fatal("path traversal created a file outside destination")
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
	if err := InstallGo(server.URL+"/go.tar.gz", hex.EncodeToString(sum[:]), target, "go1.23.0", Options{Out: io.Discard, Err: io.Discard}); err != nil {
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

func TestInstallGoRollsBackWhenCurrentLinkReplacementFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory replacement semantics differ on Windows")
	}
	archive := makeTarGz(t, map[string]string{"go/bin/go": "new"})
	sum := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	target := t.TempDir()
	oldVersion := filepath.Join(target, "versions", "go1.23.0")
	if err := os.MkdirAll(oldVersion, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldVersion, "old.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(target, "current"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := InstallGo(server.URL+"/go.tar.gz", hex.EncodeToString(sum[:]), target, "go1.23.0", Options{Out: io.Discard, Err: io.Discard}); err == nil {
		t.Fatal("InstallGo() error = nil when current replacement fails")
	}
	if got, err := os.ReadFile(filepath.Join(oldVersion, "old.txt")); err != nil || string(got) != "keep" {
		t.Fatalf("old installation was not restored: %q, error=%v", got, err)
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

	err := InstallGo(server.URL+"/go.tar.gz", hex.EncodeToString(sum[:]), target, "go1.23.0", Options{Out: io.Discard, Err: io.Discard})
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

func TestInstallGoRoutesOutputToInjectedWriters(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar.gz is not the Windows installer format")
	}
	archive := makeTarGz(t, map[string]string{"go/bin/go": "go executable"})
	sum := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	target := t.TempDir()
	err := InstallGo(server.URL+"/go.tar.gz", hex.EncodeToString(sum[:]), target, "go1.23.0", Options{
		Out: &out, Err: &errOut, Progress: true,
	})
	if err != nil {
		t.Fatalf("InstallGo() error = %v", err)
	}
	if !strings.Contains(out.String(), "설치가 완료되었습니다") {
		t.Fatalf("stdout = %q, want the completion summary", out.String())
	}
	for _, want := range []string{"Downloading", "Checksum OK.", "Extracting"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr = %q, want %q", errOut.String(), want)
		}
	}
	if strings.Contains(out.String(), "Downloading") {
		t.Fatalf("progress leaked to stdout: %q", out.String())
	}
}

func TestInstallGoQuietSuppressesProgress(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar.gz is not the Windows installer format")
	}
	archive := makeTarGz(t, map[string]string{"go/bin/go": "go executable"})
	sum := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	target := t.TempDir()
	err := InstallGo(server.URL+"/go.tar.gz", hex.EncodeToString(sum[:]), target, "go1.23.0", Options{
		Out: &out, Err: &errOut, Quiet: true, Progress: true,
	})
	if err != nil {
		t.Fatalf("InstallGo() error = %v", err)
	}
	if errOut.String() != "" {
		t.Fatalf("stderr = %q, want empty with quiet", errOut.String())
	}
	if !strings.Contains(out.String(), "설치가 완료되었습니다") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestInstallGoChecksumMismatchFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar.gz is not the Windows installer format")
	}
	archive := makeTarGz(t, map[string]string{"go/bin/go": "go executable"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	target := t.TempDir()
	err := InstallGo(server.URL+"/go.tar.gz", "deadbeef", target, "go1.23.0", Options{})
	if err == nil {
		t.Fatal("InstallGo() error = nil for a checksum mismatch")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(target, "versions")); !os.IsNotExist(statErr) {
		t.Fatal("a failed checksum must not leave a version directory behind")
	}
}

func TestDownloadFileReportsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "archive.tar.gz")
	if _, err := DownloadFile(server.URL+"/go.tar.gz", dest, nil); err == nil {
		t.Fatal("DownloadFile() error = nil for a 500 response")
	}
}

func TestExtractArchiveReportsProgressToWriter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar.gz is not the Windows installer format")
	}
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "go.tar.gz")
	if err := os.WriteFile(archivePath, makeTarGz(t, map[string]string{"go/VERSION": "go1.23.0"}), 0o644); err != nil {
		t.Fatal(err)
	}

	var status bytes.Buffer
	if err := ExtractArchive(archivePath, filepath.Join(dir, "dest"), &status); err != nil {
		t.Fatalf("ExtractArchive() error = %v", err)
	}
	if !strings.Contains(status.String(), "Extracting") {
		t.Fatalf("status = %q", status.String())
	}

	if err := ExtractArchive(archivePath, filepath.Join(dir, "dest2"), nil); err != nil {
		t.Fatalf("ExtractArchive() with nil status error = %v", err)
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
