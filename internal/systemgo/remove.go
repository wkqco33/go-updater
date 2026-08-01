package systemgo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// PlannedCommand is one privileged removal step, shown to the user before
// execution so they know exactly what will run under sudo.
type PlannedCommand struct {
	Artifact Artifact
	Display  string
}

// Fixed binary paths so the command shown to the user before confirmation
// matches exactly what actually runs under sudo.
const (
	binRm      = "/bin/rm"
	binPkgutil = "/usr/sbin/pkgutil"
)

// Plan returns the ordered removal steps for artifacts that are both managed
// (gu is allowed to delete them) and present on disk. Homebrew installs and
// missing artifacts never produce a step.
func Plan(items []Present) []PlannedCommand {
	var plan []PlannedCommand
	for _, item := range items {
		if !item.Artifact.Managed || !item.Exists {
			continue
		}
		switch item.Artifact.Kind {
		case KindDir:
			if isSymlink(item.Artifact.Path) {
				plan = append(plan, PlannedCommand{item.Artifact, "sudo " + binRm + " -f " + item.Artifact.Path})
			} else {
				plan = append(plan, PlannedCommand{item.Artifact, "sudo " + binRm + " -rf " + item.Artifact.Path})
			}
		case KindFile:
			plan = append(plan, PlannedCommand{item.Artifact, "sudo " + binRm + " -f " + item.Artifact.Path})
		case KindReceipt:
			plan = append(plan, PlannedCommand{item.Artifact, "sudo " + binPkgutil + " --forget " + item.Artifact.Path})
		}
	}
	return plan
}

// Remove executes every step in plan, escalating to sudo when a plain
// attempt fails with a permission error. It keeps going after a failure so
// one stuck step doesn't leave the rest of the cleanup undone, then returns
// a combined error describing anything that could not be removed.
func Remove(plan []PlannedCommand) error {
	var failures []error
	for _, step := range plan {
		if err := removeOne(step.Artifact); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", step.Artifact.Path, err))
		}
	}
	return errors.Join(failures...)
}

func removeOne(a Artifact) error {
	switch a.Kind {
	case KindDir:
		return removePathWithSudoFallback(a.Path, true)
	case KindFile:
		return removePathWithSudoFallback(a.Path, false)
	case KindReceipt:
		return runCommandPrivileged(binPkgutil, "--forget", a.Path)
	default:
		return fmt.Errorf("unsupported artifact kind for %s", a.Path)
	}
}

// removePathWithSudoFallback tries an unprivileged removal first (the path
// may already be user-owned), and only escalates to sudo on EACCES/EPERM.
func removePathWithSudoFallback(path string, recursive bool) error {
	if isSymlink(path) {
		if err := os.Remove(path); err == nil || !isPermissionErr(err) {
			return err
		}
		return runCommandPrivileged(binRm, "-f", path)
	}

	if recursive {
		if err := isSafeGoRoot(path); err != nil {
			return err
		}
		if err := os.RemoveAll(path); err == nil || !isPermissionErr(err) {
			return err
		}
		return runCommandPrivileged(binRm, "-rf", path)
	}

	if err := os.Remove(path); err == nil || !isPermissionErr(err) {
		return err
	}
	return runCommandPrivileged(binRm, "-f", path)
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func isPermissionErr(err error) bool {
	return errors.Is(err, os.ErrPermission)
}

// runCommandPrivileged is overridden in tests so no real sudo/pkgutil/rm
// invocation happens off of this process.
var runCommandPrivileged = func(name string, args ...string) error {
	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		return fmt.Errorf("sudo를 찾을 수 없습니다. 다음 명령을 직접 실행하세요: %s %v", name, args)
	}
	cmd := exec.Command(sudoPath, append([]string{name}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
