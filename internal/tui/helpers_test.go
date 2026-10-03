package tui

import (
	"runtime"
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
	testStrings := []string{
		"sshtui-clipboard-test-token",
		"",
		"http://localhost:8891\nwith-newline",
	}

	for _, str := range testStrings {
		err := copyToClipboard(str)
		if runtime.GOOS == "darwin" && err != nil {
			t.Fatalf("unexpected copyToClipboard error on darwin for input %q: %v", str, err)
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
