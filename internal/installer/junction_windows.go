//go:build windows

package installer

import (
	"fmt"
	"os"
	"os/exec"
)

func createWindowsJunction(link, target string) error {
	cmdPath, err := exec.LookPath("cmd")
	if err != nil {
		return err
	}
	// StartProcess receives each argument separately; link and target are not
	// interpolated into a shell command string.
	argv := []string{"cmd", "/d", "/c", "mklink", "/J", link, target}
	proc, err := os.StartProcess(cmdPath, argv, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		return err
	}
	status, err := proc.Wait()
	if err != nil {
		return err
	}
	if !status.Success() {
		return fmt.Errorf("mklink exited with status %s", status.String())
	}
	return nil
}
