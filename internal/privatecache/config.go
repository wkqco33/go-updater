package privatecache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wkqco33/go-updater/internal/guenv"
)

const (
	configFileName   = "config.json"
	metadataFileName = "metadata.json"
)

type Config struct {
	Version         int      `json:"version"`
	PrivatePatterns []string `json:"private_patterns"`
	NoSumDBPatterns []string `json:"nosumdb_patterns"`
	NoProxyPatterns []string `json:"noproxy_patterns"`
	CacheDir        string   `json:"cache_dir"`
}

// DefaultBaseDir returns $GU_HOME/private or ~/.go/private.
func DefaultBaseDir() (string, error) {
	root, err := guenv.Root()
	if err != nil {
		return "", err
	}
	return guenv.PrivateDir(root), nil
}

// ConfigPath returns the private cache config path under an explicit gu root,
// so callers that resolved a root (e.g. --home) do not silently read the
// GU_HOME environment override instead.
func ConfigPath(root string) string {
	return filepath.Join(guenv.PrivateDir(root), configFileName)
}

func DefaultConfigPath() (string, error) {
	root, err := guenv.Root()
	if err != nil {
		return "", err
	}
	return ConfigPath(root), nil
}

func DefaultMetadataPath() (string, error) {
	baseDir, err := DefaultBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, metadataFileName), nil
}

// DefaultConfig returns the configuration used when no config file exists yet.
func DefaultConfig() (Config, error) {
	root, err := guenv.Root()
	if err != nil {
		return Config{}, err
	}
	return DefaultConfigForRoot(root), nil
}

// DefaultConfigForRoot is DefaultConfig for an explicit gu root.
func DefaultConfigForRoot(root string) Config {
	return Config{
		Version:  1,
		CacheDir: guenv.CacheDir(root),
	}
}

func LoadConfig(path string) (Config, error) {
	root, err := guenv.Root()
	if err != nil {
		return Config{}, err
	}
	return LoadConfigForRoot(root, path)
}

// LoadConfigForRoot loads the config at path, falling back to defaults rooted
// at root rather than the GU_HOME environment override.
func LoadConfigForRoot(root, path string) (Config, error) {
	defaultCfg := DefaultConfigForRoot(root)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultCfg, nil
		}
		return Config{}, fmt.Errorf("failed to read private cache config: %w", err)
	}
	cfg := defaultCfg
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse private cache config: %w", err)
	}
	normalizeConfig(&cfg)
	return cfg, nil
}

func SaveConfig(path string, cfg Config) error {
	normalizeConfig(&cfg)
	if cfg.CacheDir == "" {
		return fmt.Errorf("cache_dir must not be empty")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal private cache config: %w", err)
	}

	if err := os.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("failed to write private cache config: %w", err)
	}
	return nil
}

func normalizeConfig(cfg *Config) {
	cfg.PrivatePatterns = normalizePatterns(cfg.PrivatePatterns)
	cfg.NoSumDBPatterns = normalizePatterns(cfg.NoSumDBPatterns)
	cfg.NoProxyPatterns = normalizePatterns(cfg.NoProxyPatterns)
	cfg.CacheDir = strings.TrimSpace(cfg.CacheDir)
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if len(cfg.NoSumDBPatterns) == 0 {
		cfg.NoSumDBPatterns = slices.Clone(cfg.PrivatePatterns)
	}
	if len(cfg.NoProxyPatterns) == 0 {
		cfg.NoProxyPatterns = slices.Clone(cfg.PrivatePatterns)
	}
}

func normalizePatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(patterns))
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		pp := strings.TrimSpace(p)
		if pp == "" {
			continue
		}
		if _, exists := seen[pp]; exists {
			continue
		}
		seen[pp] = struct{}{}
		out = append(out, pp)
	}
	return out
}

func ParseCSVPatterns(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return normalizePatterns(strings.Split(v, ","))
}
