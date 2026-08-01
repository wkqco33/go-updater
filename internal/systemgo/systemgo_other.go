//go:build !darwin

package systemgo

import (
	"os"
	"os/exec"
	"runtime"
)

// systemGoRoot returns the default installation path used by go.dev installers.
func systemGoRoot() string {
	if runtime.GOOS == "windows" {
		return `C:\Go`
	}
	return "/usr/local/go"
}

// Detect looks for a system Go installation at the go.dev default path. It
// returns an empty slice if nothing is found there.
func Detect() ([]Present, error) {
	var items []Present

	path := systemGoRoot()
	if _, err := os.Stat(path); err == nil {
		items = append(items, Present{
			Artifact: Artifact{Kind: KindDir, Path: path, Managed: true},
			Exists:   true,
		})
	}

	if runtime.GOOS == "windows" {
		winget, err := detectWingetGo()
		if err == nil && winget {
			items = append(items, Present{
				Artifact: Artifact{Kind: KindWinget, Path: "GoLang.Go", Managed: true, Detail: "winget으로 설치된 Go"},
				Exists:   true,
			})
		}
	}

	if len(items) == 0 {
		return nil, nil
	}
	return items, nil
}

// detectWingetGo checks if Go is installed via winget by running
// "winget list GoLang.Go".
func detectWingetGo() (bool, error) {
	wingetPath, err := exec.LookPath("winget")
	if err != nil {
		return false, nil
	}
	cmd := exec.Command(wingetPath, "list", "GoLang.Go")
	if err := cmd.Run(); err != nil {
		return false, nil
	}
	return true, nil
}
