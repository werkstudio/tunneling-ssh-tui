package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"sshtui/internal/config"
)

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func routeDiagram(t config.TunnelCfg, hostDest string) string {
	bind := t.BindAddr()
	if t.Type == "reverse" {
		return fmt.Sprintf("%s:%d  ◄──────  [%s]  ◄──────  %s:%d (Local Target)",
			bind, t.RemotePort, hostDest, bind, t.LocalPort)
	}
	remote := t.RemoteAddr()
	return fmt.Sprintf("%s:%d  ──────►  [%s]  ──────►  %s:%d (%s Target)",
		bind, t.LocalPort, hostDest, remote, t.RemotePort, remote)
}
