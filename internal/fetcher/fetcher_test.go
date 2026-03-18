package fetcher

import (
	"testing"
)

func TestFindMatchingFile(t *testing.T) {
	release := &GoRelease{
		Version: "go1.21.0",
		Stable:  true,
		Files: []GoFile{
			{Filename: "go1.21.0.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive"},
			{Filename: "go1.21.0.darwin-arm64.tar.gz", OS: "darwin", Arch: "arm64", Kind: "archive"},
			{Filename: "go1.21.0.windows-amd64.zip", OS: "windows", Arch: "amd64", Kind: "archive"},
			{Filename: "go1.21.0.src.tar.gz", OS: "", Arch: "", Kind: "source"},
		},
	}

	tests := []struct {
		name     string
		os       string
		arch     string
		expected string
		wantErr  bool
	}{
		{"Linux AMD64", "linux", "amd64", "go1.21.0.linux-amd64.tar.gz", false},
		{"Darwin ARM64", "darwin", "arm64", "go1.21.0.darwin-arm64.tar.gz", false},
		{"Windows AMD64", "windows", "amd64", "go1.21.0.windows-amd64.zip", false},
		{"Unsupported OS", "freebsd", "amd64", "", true},
		{"Unsupported Arch", "linux", "386", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := FindMatchingFile(release, tt.os, tt.arch)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindMatchingFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && file.Filename != tt.expected {
				t.Errorf("FindMatchingFile() = %v, want %v", file.Filename, tt.expected)
			}
		})
	}
}
