//go:build windows

package installer

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallGoCompletesWindowsZipInstallation(t *testing.T) {
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	file, err := zw.Create("go/bin/go.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("go executable")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	data := archive.Bytes()
	sum := sha256.Sum256(data)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(data)
	}))
	defer server.Close()

	target := t.TempDir()
	if err := InstallGo(server.URL+"/go.zip", hex.EncodeToString(sum[:]), target, "go1.23.0", Options{Out: io.Discard, Err: io.Discard}); err != nil {
		t.Fatalf("InstallGo() error = %v", err)
	}
	installed := filepath.Join(target, "versions", "go1.23.0", "bin", "go.exe")
	if got, err := os.ReadFile(installed); err != nil || string(got) != "go executable" {
		t.Fatalf("installed executable = %q, error=%v", got, err)
	}
}
