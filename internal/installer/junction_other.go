//go:build !windows

package installer

import "fmt"

func createWindowsJunction(_, _ string) error {
	return fmt.Errorf("Windows junctions are only supported on Windows")
}
