//go:build !windows

package setup

import (
	"os/exec"
	"runtime"
)

func openBrowser(url string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}
