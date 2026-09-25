// Package guenv resolves the on-disk locations gu manages so commands and
// tests share one definition instead of repeating ~/.go.
package guenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// EnvHome overrides the gu root (versions, current link, private state).
	EnvHome = "GU_HOME"
	// EnvCacheHome is the XDG base directory for rebuildable cache data.
	EnvCacheHome = "XDG_CACHE_HOME"
	// EnvConfigHome is the XDG base directory for user configuration.
	EnvConfigHome = "XDG_CONFIG_HOME"
)

// HomeDir returns the current user's home directory. Commands must call this
// instead of os.UserHomeDir so shell setup and tests share one definition.
func HomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return homeDir, nil
}

// ConfigDir returns $XDG_CONFIG_HOME when set, otherwise ~/.config.
func ConfigDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv(EnvConfigHome)); override != "" {
		return override, nil
	}
	homeDir, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config"), nil
}

// Root returns the gu root directory: $GU_HOME when set, otherwise ~/.go.
func Root() (string, error) {
	if override := strings.TrimSpace(os.Getenv(EnvHome)); override != "" {
		return override, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".go"), nil
}

// Resolve returns override when it is non-empty, otherwise Root.
func Resolve(override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return override, nil
	}
	return Root()
}

// PrivateDir returns the directory holding private cache config and metadata.
func PrivateDir(root string) string { return filepath.Join(root, "private") }

// CacheDir returns the default private module cache directory. XDG_CACHE_HOME
// wins when set because the module cache is rebuildable; otherwise the cache
// stays next to the private state for backward compatibility.
func CacheDir(root string) string {
	if xdg := strings.TrimSpace(os.Getenv(EnvCacheHome)); xdg != "" {
		return filepath.Join(xdg, "go-updater", "modcache")
	}
	return filepath.Join(PrivateDir(root), "modcache")
}
