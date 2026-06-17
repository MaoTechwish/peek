//go:build windows

package setup

import (
	"os/exec"
	"syscall"
)

// createNoWindow prevents the helper cmd.exe from flashing a console window.
const createNoWindow = 0x08000000

func openBrowser(url string) error {
	// `start "" <url>` launches the default browser; the empty first arg is the
	// window title that `start` would otherwise consume.
	cmd := exec.Command("cmd", "/c", "start", "", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd.Start()
}
