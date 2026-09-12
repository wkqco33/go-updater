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
	// DryRun reports what would be removed without deleting anything.
	DryRun bool
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
		freed, count, err := scanAndMaybeRemove(cfg.CacheDir, !opts.DryRun)
		if err != nil {
			return CleanResult{}, fmt.Errorf("failed to remove cache directory: %w", err)
		}
		if !opts.DryRun {
			_ = os.Remove(metadataPath)
		}
		return CleanResult{
			RemovedFiles: count,
			FreedBytes:   freed,
			FinalSize:    0,
		}, nil
	}

	files, totalSize, err := listCacheFiles(cfg.CacheDir)
	if err != nil {
		return CleanResult{}, err
	}
	result := CleanResult{FinalSize: totalSize}

	remove := func(f cacheFile) {
		if !opts.DryRun {
			if err := os.Remove(f.path); err != nil {
				return
			}
		}
		result.RemovedFiles++
		result.FreedBytes += f.size
		result.FinalSize -= f.size
	}

	if opts.StaleDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -opts.StaleDays)
		for _, f := range files {
			if f.modTime.After(cutoff) {
				continue
			}
			remove(f)
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
				remove(f)
			}
		}
	}
	return result, nil
}

// scanAndMaybeRemove walks path once, summing the size of every file. When
// remove is true it also deletes the files and then the now-empty directories,
// so cleaning does not need a separate size scan.
func scanAndMaybeRemove(path string, remove bool) (freed int64, count int, err error) {
	var dirs []string
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			if remove {
				dirs = append(dirs, p)
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if remove {
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		freed += info.Size()
		count++
		return nil
	})
	if err != nil {
		return freed, count, err
	}
	if remove {
		// Deepest directories first so each parent is empty when it is removed.
		for i := len(dirs) - 1; i >= 0; i-- {
			if err := os.Remove(dirs[i]); err != nil && !os.IsNotExist(err) {
				return freed, count, err
			}
		}
	}
	return freed, count, nil
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
