//go:build windows

package main

import "golang.org/x/sys/windows"

// isElevated returns true if the current process has administrator privileges.
func isElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// reportElevationRequired shows a one-time message box. The agent is built as a
// GUI-subsystem app (no console), so stderr would be invisible — a message box
// is the only way to tell the user to relaunch as administrator.
func reportElevationRequired() {
	const (
		mbOK        = 0x00000000
		mbIconError = 0x00000010
	)
	text, _ := windows.UTF16PtrFromString(
		"Peek requires administrator privileges.\n\n" +
			"Right-click the app and choose \"Run as administrator\".")
	title, _ := windows.UTF16PtrFromString("Peek")
	windows.MessageBox(0, text, title, mbOK|mbIconError)
}
