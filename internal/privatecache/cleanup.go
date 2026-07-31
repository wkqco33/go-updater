package privatecache

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type CleanOptions struct {
	All       bool
	StaleDays int
	MaxSizeMB int64
}

type CleanResult struct {
	RemovedFiles int
	FreedBytes   int64
	FinalSize    int64
}

type cacheFile struct {
	path    string
	size    int64
	modTime time.Time
}

func CleanCache(cfg Config, metadataPath string, opts CleanOptions) (CleanResult, error) {
	if opts.All {
		sizeBefore, _ := DirSize(cfg.CacheDir)
		if err := os.RemoveAll(cfg.CacheDir); err != nil {
			return CleanResult{}, fmt.Errorf("failed to remove cache directory: %w", err)
		}
		_ = os.Remove(metadataPath)
		return CleanResult{
			FreedBytes: sizeBefore,
			FinalSize:  0,
		}, nil
	}

	files, totalSize, err := listCacheFiles(cfg.CacheDir)
	if err != nil {
		return CleanResult{}, err
	}
	result := CleanResult{FinalSize: totalSize}

	if opts.StaleDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -opts.StaleDays)
		for _, f := range files {
			if f.modTime.After(cutoff) {
				continue
			}
			if err := os.Remove(f.path); err == nil {
				result.RemovedFiles++
				result.FreedBytes += f.size
				result.FinalSize -= f.size
			}
		}
	}

	if opts.MaxSizeMB > 0 {
		limit := opts.MaxSizeMB * 1024 * 1024
		if result.FinalSize > limit {
			files, _, err = listCacheFiles(cfg.CacheDir)
			if err != nil {
				return result, err
			}
			sort.Slice(files, func(i, j int) bool {
				return files[i].modTime.Before(files[j].modTime)
			})
			for _, f := range files {
				if result.FinalSize <= limit {
					break
				}
				if err := os.Remove(f.path); err == nil {
					result.RemovedFiles++
					result.FreedBytes += f.size
					result.FinalSize -= f.size
				}
			}
		}
	}
	return result, nil
}

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return size, nil
}

func listCacheFiles(path string) ([]cacheFile, int64, error) {
	var files []cacheFile
	var total int64
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		size := info.Size()
		total += size
		files = append(files, cacheFile{
			path:    p,
			size:    size,
			modTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("failed to scan cache files: %w", err)
	}
	return files, total, nil
}
