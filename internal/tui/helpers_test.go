package tui

import (
	"io"
	"os/exec"
	"strings"
	"testing"

	"sshtui/internal/config"
)

func TestRouteDiagram(t *testing.T) {
	// Local tunnel test
	localTunnel := config.TunnelCfg{
		Name:       "TestLocal",
		Host:       "homelab",
		Type:       "local",
		LocalPort:  8891,
		RemoteHost: "localhost",
		RemotePort: 20128,
	}
	diag := routeDiagram(localTunnel, "homelab.domain.com:22")
	if !strings.Contains(diag, "8891") || !strings.Contains(diag, "20128") {
		t.Errorf("local route diagram missing ports: %s", diag)
	}
	if !strings.Contains(diag, "──────►") {
		t.Errorf("local route diagram missing forward arrow: %s", diag)
	}
	if !strings.Contains(diag, "[homelab.domain.com:22]") {
		t.Errorf("local route diagram missing hostDest: %s", diag)
	}
	if !strings.Contains(diag, "localhost Target") {
		t.Errorf("local route diagram missing remote target label: %s", diag)
	}

	// Reverse tunnel test
	revTunnel := config.TunnelCfg{
		Name:       "TestReverse",
		Host:       "homelab",
		Type:       "reverse",
		LocalPort:  8891,
		RemoteHost: "localhost",
		RemotePort: 20128,
	}
	diagRev := routeDiagram(revTunnel, "homelab.domain.com:22")
	if !strings.Contains(diagRev, "8891") || !strings.Contains(diagRev, "20128") {
		t.Errorf("reverse route diagram missing ports: %s", diagRev)
	}
	if !strings.Contains(diagRev, "◄──────") {
		t.Errorf("reverse route diagram missing reverse arrow: %s", diagRev)
	}
	if !strings.Contains(diagRev, "[homelab.domain.com:22]") {
		t.Errorf("reverse route diagram missing hostDest: %s", diagRev)
	}
	if !strings.Contains(diagRev, "Local Target") {
		t.Errorf("reverse route diagram missing Local Target label: %s", diagRev)
	}

	// Custom bind and custom remote host
	customTunnel := config.TunnelCfg{
		Name:       "Custom",
		Host:       "srv",
		Type:       "local",
		Bind:       "0.0.0.0",
		LocalPort:  9000,
		RemoteHost: "192.168.1.50",
		RemotePort: 80,
	}
	diagCustom := routeDiagram(customTunnel, "srv:22")
	if !strings.Contains(diagCustom, "0.0.0.0:9000") {
		t.Errorf("custom route diagram missing bind addr: %s", diagCustom)
	}
	if !strings.Contains(diagCustom, "192.168.1.50:80") {
		t.Errorf("custom route diagram missing remote addr: %s", diagCustom)
	}
}

func TestCopyToClipboard(t *testing.T) {
	var capturedCmd *exec.Cmd
	var capturedInput string

	origRunner := clipboardRunner
	clipboardRunner = func(cmd *exec.Cmd) error {
		capturedCmd = cmd
		if cmd.Stdin != nil {
			b, _ := io.ReadAll(cmd.Stdin)
			capturedInput = string(b)
		}
		return nil
	}
	t.Cleanup(func() { clipboardRunner = origRunner })

	testStrings := []string{
		"sshtui-clipboard-test-token",
		"",
		"http://localhost:8891\nwith-newline",
	}

	for _, str := range testStrings {
		capturedInput = ""
		capturedCmd = nil
		err := copyToClipboard(str)
		if err != nil {
			t.Fatalf("unexpected copyToClipboard error for input %q: %v", str, err)
		}
		if capturedInput != str {
			t.Errorf("expected captured stdin %q, got %q", str, capturedInput)
		}
		if capturedCmd == nil {
			t.Errorf("expected clipboard command to be created")
		}
	}
}

func TestClipboardCmd(t *testing.T) {
	prog, args := clipboardCmd("darwin")
	if prog != "pbcopy" || len(args) != 0 {
		t.Errorf("expected darwin to use pbcopy, got %s %v", prog, args)
	}

	prog, args = clipboardCmd("linux")
	if prog != "wl-copy" && prog != "xclip" {
		t.Errorf("expected linux to use wl-copy or xclip, got %s", prog)
	}
	if prog == "xclip" {
		if len(args) != 2 || args[0] != "-selection" || args[1] != "clipboard" {
			t.Errorf("expected xclip args [-selection clipboard], got %v", args)
		}
	}
}

func TestViewHelpModal(t *testing.T) {
	expectedCategories := []string{
		"Navigasi",
		"Kontrol Tunnel",
		"Aksi Cepat",
		"Aplikasi",
	}
	expectedKeys := []string{
		"↑↓",
		"jk",
		"Space",
		"Enter",
		"r",
		"a",
		"x",
		"o",
		"c",
		"n",
		"e",
		"d",
		"?",
		"q",
	}

	// Wide layout (2 columns)
	modalWide := viewHelpModal(100, 30)
	if modalWide == "" {
		t.Fatal("expected non-empty help modal for wide layout")
	}
	for _, cat := range expectedCategories {
		if !strings.Contains(modalWide, cat) {
			t.Errorf("wide help modal missing category %q", cat)
		}
	}
	for _, key := range expectedKeys {
		if !strings.Contains(modalWide, key) {
			t.Errorf("wide help modal missing key shortcut %q", key)
		}
	}

	// Narrow layout (1 column stacked)
	modalNarrow := viewHelpModal(60, 25)
	if modalNarrow == "" {
		t.Fatal("expected non-empty help modal for narrow layout")
	}
	for _, cat := range expectedCategories {
		if !strings.Contains(modalNarrow, cat) {
			t.Errorf("narrow help modal missing category %q", cat)
		}
	}
	for _, key := range expectedKeys {
		if !strings.Contains(modalNarrow, key) {
			t.Errorf("narrow help modal missing key shortcut %q", key)
		}
	}

	// Boundary dimension tests
	if zeroModal := viewHelpModal(0, 0); zeroModal == "" {
		t.Error("expected non-empty modal even for zero dimensions")
	}
	if smallModal := viewHelpModal(40, 15); smallModal == "" {
		t.Error("expected non-empty modal for small dimensions")
	}
}
