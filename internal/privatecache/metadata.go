package privatecache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ModuleRecord struct {
	Module           string    `json:"module"`
	RequestedVersion string    `json:"requested_version"`
	ResolvedVersion  string    `json:"resolved_version"`
	Source           string    `json:"source"`
	SyncedAt         time.Time `json:"synced_at"`
}

type Metadata struct {
	UpdatedAt time.Time      `json:"updated_at"`
	Modules   []ModuleRecord `json:"modules"`
}

func LoadMetadata(path string) (Metadata, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Metadata{}, nil
		}
		return Metadata{}, fmt.Errorf("failed to read metadata: %w", err)
	}
	var m Metadata
	if err := json.Unmarshal(b, &m); err != nil {
		return Metadata{}, fmt.Errorf("failed to parse metadata: %w", err)
	}
	return m, nil
}

func SaveMetadata(path string, metadata Metadata) error {
	metadata.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	return nil
}

func BuildMetadataIndex(records []ModuleRecord) map[string]ModuleRecord {
	index := make(map[string]ModuleRecord, len(records))
	for _, r := range records {
		index[r.Module+"@"+r.ResolvedVersion] = r
	}
	return index
}
