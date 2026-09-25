package shellenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// backupSuffix is appended to a startup file before gu modifies it so the
// previous content can be restored by hand if anything goes wrong.
const backupSuffix = ".gu-backup"

// PersistOptions configures writing the gu-managed env file and connecting it
// to one or more shell startup files.
type PersistOptions struct {
	Setup   Setup
	EnvFile string
	RCFiles []string
	// DryRun computes the result without writing anything.
	DryRun bool
}

// RCResult reports what happened to one startup file.
type RCResult struct {
	Path string
	// Exists is true when the file existed before the operation.
	Exists bool
	// Changed is true when the gu block was added, updated, or removed.
	Changed bool
	// Backup is the path written before a modification; empty when unchanged.
	Backup string
}

// Result reports the files Persist wrote or Unpersist removed.
type Result struct {
	EnvFile string
	// EnvFileChanged is true when the env file was written (Persist) or
	// removed (Unpersist), or would be under DryRun.
	EnvFileChanged bool
	// EnvFileExisted is true when the env file was present before the
	// operation, so callers can distinguish creation from an update.
	EnvFileExisted bool
	RCFiles        []RCResult
}

// Persist writes the env file and ensures each startup file sources it. On the
// first startup-file error nothing else is touched, so a corrupted file never
// leaves a half-applied setup.
func Persist(opts PersistOptions) (Result, error) {
	result := Result{EnvFile: opts.EnvFile}

	content, err := EnvFileContent(opts.Setup)
	if err != nil {
		return result, err
	}
	envChanged, envExisted, err := writeIfChanged(opts.EnvFile, []byte(content), opts.DryRun)
	if err != nil {
		return result, fmt.Errorf("failed to write env file: %w", err)
	}
	result.EnvFileChanged = envChanged
	result.EnvFileExisted = envExisted

	sourceLine, err := SourceLine(opts.Setup.Shell, opts.EnvFile)
	if err != nil {
		return result, err
	}
	for _, rc := range opts.RCFiles {
		entry, err := upsertRC(rc, sourceLine, opts.DryRun)
		if err != nil {
			return result, err
		}
		result.RCFiles = append(result.RCFiles, entry)
	}
	return result, nil
}

// Unpersist removes the gu block from every startup file and deletes the
// managed env file, leaving the rest of the user's configuration intact.
func Unpersist(opts PersistOptions) (Result, error) {
	result := Result{EnvFile: opts.EnvFile}

	for _, rc := range opts.RCFiles {
		entry, err := removeRC(rc, opts.DryRun)
		if err != nil {
			return result, err
		}
		result.RCFiles = append(result.RCFiles, entry)
	}

	removed, existed, err := removeIfExists(opts.EnvFile, opts.DryRun)
	if err != nil {
		return result, fmt.Errorf("failed to remove env file: %w", err)
	}
	result.EnvFileChanged = removed
	result.EnvFileExisted = existed
	return result, nil
}

func upsertRC(path, body string, dryRun bool) (RCResult, error) {
	entry := RCResult{Path: path}
	data, exists, err := readIfExists(path)
	if err != nil {
		return entry, fmt.Errorf("failed to read %s: %w", path, err)
	}
	entry.Exists = exists

	updated, err := UpsertBlock(string(data), body)
	if err != nil {
		return entry, fmt.Errorf("%s: %w", path, err)
	}
	if updated == string(data) {
		return entry, nil
	}
	entry.Changed = true
	if dryRun {
		return entry, nil
	}

	target := resolveWritePath(path)
	if exists {
		backup, err := writeBackup(target, data)
		if err != nil {
			return entry, err
		}
		entry.Backup = backup
	}
	if err := writeFileAtomic(target, []byte(updated), fileMode(target)); err != nil {
		return entry, fmt.Errorf("failed to write %s: %w", target, err)
	}
	return entry, nil
}

func removeRC(path string, dryRun bool) (RCResult, error) {
	entry := RCResult{Path: path}
	data, exists, err := readIfExists(path)
	if err != nil {
		return entry, fmt.Errorf("failed to read %s: %w", path, err)
	}
	entry.Exists = exists
	if !exists {
		return entry, nil
	}

	updated, err := RemoveBlock(string(data))
	if err != nil {
		return entry, fmt.Errorf("%s: %w", path, err)
	}
	if updated == string(data) {
		return entry, nil
	}
	entry.Changed = true
	if dryRun {
		return entry, nil
	}

	target := resolveWritePath(path)
	backup, err := writeBackup(target, data)
	if err != nil {
		return entry, err
	}
	entry.Backup = backup
	if err := writeFileAtomic(target, []byte(updated), fileMode(target)); err != nil {
		return entry, fmt.Errorf("failed to write %s: %w", target, err)
	}
	return entry, nil
}

func readIfExists(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

// writeIfChanged reports whether the caller would change the file and whether
// the file already existed. Under DryRun it only compares content.
func writeIfChanged(path string, data []byte, dryRun bool) (bool, bool, error) {
	current, exists, err := readIfExists(path)
	if err != nil {
		return false, false, err
	}
	if exists && string(current) == string(data) {
		return false, true, nil
	}
	if dryRun {
		return true, exists, nil
	}
	if err := writeFileAtomic(path, data, fileMode(path)); err != nil {
		return false, exists, err
	}
	return true, exists, nil
}

func removeIfExists(path string, dryRun bool) (bool, bool, error) {
	_, exists, err := readIfExists(path)
	if err != nil {
		return false, false, err
	}
	if !exists {
		return false, false, nil
	}
	if dryRun {
		return true, true, nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, true, err
	}
	return true, true, nil
}

// resolveWritePath follows a symlinked startup file so gu edits the real file
// instead of replacing the user's symlink with a regular file.
func resolveWritePath(path string) string {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return path
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func fileMode(path string) os.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return 0o644
}

// writeFileAtomic writes data through a temporary file in the same directory
// and renames it into place, so an interrupted write cannot truncate the
// user's shell configuration.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".gu-env-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	return os.Rename(tmpName, path)
}

func writeBackup(path string, data []byte) (string, error) {
	backup := path + backupSuffix
	if err := os.WriteFile(backup, data, 0o600); err != nil {
		return "", fmt.Errorf("failed to write backup %s: %w", backup, err)
	}
	return backup, nil
}
