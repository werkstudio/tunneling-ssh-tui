package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"sshtui/internal/config"
)

var clipboardRunner = func(cmd *exec.Cmd) error {
	return cmd.Run()
}

func clipboardCmd(goos string) (string, []string) {
	switch goos {
	case "darwin":
		return "pbcopy", nil
	default:
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return "wl-copy", nil
		}
		return "xclip", []string{"-selection", "clipboard"}
	}
}

func copyToClipboard(text string) error {
	prog, args := clipboardCmd(runtime.GOOS)
	cmd := exec.Command(prog, args...)
	cmd.Stdin = strings.NewReader(text)
	return clipboardRunner(cmd)
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
