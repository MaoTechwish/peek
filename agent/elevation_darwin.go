//go:build darwin

package main

import (
	"fmt"
	"os"
)

// isElevated returns true if running as root (uid 0).
// On macOS, screen recording permission is handled by the OS on first screenshot attempt;
// root is required to ensure the process can always obtain that permission.
func isElevated() bool {
	return os.Getuid() == 0
}

// reportElevationRequired prints to stderr; on macOS the agent is launched from a
// terminal with sudo, so stderr is visible.
func reportElevationRequired() {
	fmt.Fprintln(os.Stderr, "error: peek requires administrator privileges (run with sudo)")
}
