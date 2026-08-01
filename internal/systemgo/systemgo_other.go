//go:build !darwin

package systemgo

import (
	"os"
	"runtime"
)

// systemGoRoot returns the default installation path used by go.dev installers.
func systemGoRoot() string {
	if runtime.GOOS == "windows" {
		return `C:\Go`
	}
	return "/usr/local/go"
}

// Detect looks for a system Go installation at the go.dev default path. It
// returns an empty slice if nothing is found there.
func Detect() ([]Present, error) {
	path := systemGoRoot()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	return []Present{
		{Artifact: Artifact{Kind: KindDir, Path: path, Managed: true}, Exists: true},
	}, nil
}
