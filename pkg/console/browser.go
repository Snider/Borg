// Package console provides an encrypted PWA demo server with browser integration.
package console

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the default browser to the specified URL.
// Supports macOS, Linux, and Windows.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
