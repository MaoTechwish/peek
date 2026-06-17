//go:build darwin

package main

import "os"

// isElevated returns true if running as root (uid 0).
// On macOS, screen recording permission is handled by the OS on first screenshot attempt;
// root is required to ensure the process can always obtain that permission.
func isElevated() bool {
	return os.Getuid() == 0
}
