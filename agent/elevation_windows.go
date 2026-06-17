//go:build windows

package main

import "golang.org/x/sys/windows"

// isElevated returns true if the current process has administrator privileges.
func isElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
