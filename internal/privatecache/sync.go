package privatecache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type SyncResult struct {
	Downloaded []ModuleRecord
	Skipped    []ModuleRecord
	Failed     []string
}

type moduleSpec struct {
	module  string
	version string
}

type downloadJSON struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
}

func ParseModuleSpec(spec string) (string, string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", "", fmt.Errorf("module spec is empty")
	}
	parts := strings.Split(spec, "@")
	if len(parts) > 2 {
		return "", "", fmt.Errorf("invalid module spec: %s", spec)
	}
	module := strings.TrimSpace(parts[0])
	if module == "" {
		return "", "", fmt.Errorf("invalid module spec: %s", spec)
	}
	version := ""
	if len(parts) == 2 {
		version = strings.TrimSpace(parts[1])
	}
	return module, version, nil
}

func SyncModules(ctx context.Context, cfg Config, metadataPath string, specs []string, retries int, latestIfMissing bool, source string) (SyncResult, error) {
	if retries < 1 {
		retries = 1
	}
	if err := os.MkdirAll(cfg.CacheDir, 0755); err != nil {
		return SyncResult{}, fmt.Errorf("failed to create cache directory: %w", err)
	}

	metadata, err := LoadMetadata(metadataPath)
	if err != nil {
		return SyncResult{}, err
	}
	existing := BuildMetadataIndex(metadata.Modules)
	result := SyncResult{}

	for _, raw := range specs {
		module, version, err := ParseModuleSpec(raw)
		if err != nil {
			result.Failed = append(result.Failed, err.Error())
			continue
		}
		if version == "" && latestIfMissing {
			version = "latest"
		}
		if version == "" {
			result.Failed = append(result.Failed, fmt.Sprintf("missing version for module: %s", module))
			continue
		}

		cacheKey := module + "@" + version
		if _, ok := existing[cacheKey]; ok {
			result.Skipped = append(result.Skipped, ModuleRecord{
				Module:           module,
				RequestedVersion: version,
				ResolvedVersion:  version,
				Source:           source,
			})
			continue
		}

		resolvedVersion, err := downloadModuleWithRetry(ctx, cfg, module, version, retries)
		if err != nil {
			result.Failed = append(result.Failed, err.Error())
			continue
		}

		record := ModuleRecord{
			Module:           module,
			RequestedVersion: version,
			ResolvedVersion:  resolvedVersion,
			Source:           source,
			SyncedAt:         time.Now().UTC(),
		}
		metadata.Modules = append(metadata.Modules, record)
		existing[module+"@"+resolvedVersion] = record
		result.Downloaded = append(result.Downloaded, record)
	}

	if err := SaveMetadata(metadataPath, metadata); err != nil {
		return result, err
	}
	return result, nil
}

func downloadModuleWithRetry(ctx context.Context, cfg Config, module, version string, retries int) (string, error) {
	target := module + "@" + version
	var lastErr error
	for i := 1; i <= retries; i++ {
		resolved, err := downloadModule(ctx, cfg, target)
		if err == nil {
			return resolved, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("failed to download %s after %d attempts: %w", target, retries, lastErr)
}

func downloadModule(ctx context.Context, cfg Config, target string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "go-updater-private-sync-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	initCmd := exec.CommandContext(ctx, "go", "mod", "init", "go_updater_private_sync_tmp")
	initCmd.Dir = tmpDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go mod init failed: %s: %w", sanitizeOutput(string(out)), err)
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "download", "-json", target)
	cmd.Dir = tmpDir
	cmd.Env = buildCommandEnv(cfg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go mod download failed for %s: %s: %w", target, sanitizeOutput(stderr.String()), err)
	}

	var decoded downloadJSON
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		return "", fmt.Errorf("failed to parse go mod download result for %s: %w", target, err)
	}
	if decoded.Version == "" {
		return "", fmt.Errorf("resolved version is empty for %s", target)
	}
	return decoded.Version, nil
}

func buildCommandEnv(cfg Config) []string {
	cmdEnv := os.Environ()
	for k, v := range BuildEnv(cfg, false) {
		if v == "" {
			continue
		}
		cmdEnv = append(cmdEnv, k+"="+v)
	}
	cmdEnv = append(cmdEnv, "GO111MODULE=on")
	return cmdEnv
}

func sanitizeOutput(s string) string {
	out := strings.TrimSpace(s)
	if out == "" {
		return out
	}
	out = strings.ReplaceAll(out, "\n", " ")
	return out
}
