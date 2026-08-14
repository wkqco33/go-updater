// Package versions owns the filesystem state of gu-managed Go versions.
package versions

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Version struct {
	Name   string
	Active bool
}

type Store struct {
	Root string
}

func NewStore(root string) Store { return Store{Root: root} }

func (s Store) versionsDir() string { return filepath.Join(s.Root, "versions") }
func (s Store) currentPath() string { return filepath.Join(s.Root, "current") }

func (s Store) List() ([]Version, error) {
	entries, err := os.ReadDir(s.versionsDir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read versions directory: %w", err)
	}
	active, _ := s.Active()
	result := make([]Version, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && validVersionName(entry.Name()) {
			result = append(result, Version{Name: entry.Name(), Active: entry.Name() == active})
		}
	}
	sort.Slice(result, func(i, j int) bool { return compare(result[i].Name, result[j].Name) > 0 })
	return result, nil
}

func (s Store) Active() (string, error) {
	target, err := os.Readlink(s.currentPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read current version: %w", err)
	}
	return filepath.Base(target), nil
}

func (s Store) Resolve(requested string) (string, error) {
	if !strings.HasPrefix(requested, "go") {
		requested = "go" + requested
	}
	if !validVersionName(requested) {
		return "", fmt.Errorf("invalid Go version: %s", requested)
	}
	if _, err := os.Stat(filepath.Join(s.versionsDir(), requested)); err == nil {
		return requested, nil
	}
	list, err := s.List()
	if err != nil {
		return "", err
	}
	for _, version := range list {
		if strings.HasPrefix(version.Name, requested) {
			return version.Name, nil
		}
	}
	return "", fmt.Errorf("version %q is not installed", requested)
}

func (s Store) Activate(name string) error {
	if !validVersionName(name) {
		return fmt.Errorf("invalid Go version: %s", name)
	}
	target := filepath.Join(s.versionsDir(), name)
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("version %q is not installed: %w", name, err)
	}
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return err
	}
	tmp := s.currentPath() + ".tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return fmt.Errorf("create current symlink: %w", err)
	}
	if err := os.Rename(tmp, s.currentPath()); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace current symlink: %w", err)
	}
	return nil
}

func (s Store) RemoveAll() error {
	if err := os.Remove(s.currentPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove current version link: %w", err)
	}
	if err := os.RemoveAll(s.versionsDir()); err != nil {
		return fmt.Errorf("remove versions directory: %w", err)
	}
	return nil
}

func (s Store) Remove(name string) error {
	if !validVersionName(name) {
		return fmt.Errorf("invalid Go version: %s", name)
	}
	active, err := s.Active()
	if err != nil {
		return err
	}
	if active == name {
		return fmt.Errorf("version %q is currently active", name)
	}
	return os.RemoveAll(filepath.Join(s.versionsDir(), name))
}

func (s Store) RemoveUnused() ([]string, error) {
	active, err := s.Active()
	if err != nil {
		return nil, err
	}
	list, err := s.List()
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, version := range list {
		if version.Name == active {
			continue
		}
		if err := os.RemoveAll(filepath.Join(s.versionsDir(), version.Name)); err != nil {
			return removed, err
		}
		removed = append(removed, version.Name)
	}
	return removed, nil
}

func validVersionName(name string) bool {
	return strings.HasPrefix(name, "go") && !strings.ContainsAny(name, `/\\`) && len(name) > 2
}

func compare(a, b string) int {
	ap := versionParts(a)
	bp := versionParts(b)
	for i := 0; i < len(ap) && i < len(bp); i++ {
		if ap[i] != bp[i] {
			if ap[i] > bp[i] {
				return 1
			}
			return -1
		}
	}
	if len(ap) > len(bp) {
		return 1
	}
	if len(ap) < len(bp) {
		return -1
	}
	return strings.Compare(a, b)
}

func versionParts(name string) []int {
	parts := strings.Split(strings.TrimPrefix(name, "go"), ".")
	result := make([]int, len(parts))
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return []int{-1}
		}
		result[i] = n
	}
	return result
}
