package systemgo

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// PathsDGo is the file the go.dev macOS installer drops into /etc/paths.d
// to add its GOROOT/bin to every login shell's PATH.
const PathsDGo = "/etc/paths.d/go"

// protectedPaths must never be recursively removed, even if some detection
// path derives one of them as a "GOROOT" candidate.
var protectedPaths = []string{"/", "/usr", "/usr/local", "/etc", "/opt", "/bin", "/sbin"}

// isSafeGoRoot verifies path is safe to recursively remove as a GOROOT: it
// must not be a protected system directory or a symlink, and it must look
// like an actual Go installation.
func isSafeGoRoot(path string) error {
	clean := filepath.Clean(path)
	if clean == "" || clean == "/" || clean == "." {
		return fmt.Errorf("refusing to remove invalid path %q", path)
	}
	if slices.Contains(protectedPaths, clean) {
		return fmt.Errorf("refusing to remove protected system path %q", clean)
	}

	info, err := os.Lstat(clean)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to remove %q: it is a symlink, not a Go installation", clean)
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing to remove %q: not a directory", clean)
	}

	if _, err := os.Stat(filepath.Join(clean, "bin", "go")); err == nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(clean, "VERSION")); err == nil {
		return nil
	}
	return fmt.Errorf("refusing to remove %q: does not look like a Go installation", clean)
}
