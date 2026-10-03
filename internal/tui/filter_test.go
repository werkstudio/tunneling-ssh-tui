package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"sshtui/internal/config"
)

func sampleTunnels() []config.TunnelCfg {
	return []config.TunnelCfg{
		{
			Name:       "api-prod",
			Host:       "bastion-prod.example.com",
			Type:       "local",
			LocalPort:  8080,
			RemoteHost: "localhost",
			RemotePort: 80,
		},
		{
			Name:       "db-staging",
			Host:       "staging-box",
			Type:       "local",
			LocalPort:  5432,
			RemoteHost: "localhost",
			RemotePort: 5432,
		},
		{
			Name:       "webhook-rev",
			Host:       "vps.remote.net",
			Type:       "reverse",
			LocalPort:  3000,
			RemoteHost: "localhost",
			RemotePort: 9000,
		},
	}
}

func TestMatchTunnel(t *testing.T) {
	tunnels := sampleTunnels()

	tests := []struct {
		name    string
		tunnel  config.TunnelCfg
		query   string
		matched bool
	}{
		// Empty / whitespace
		{"empty query matches all", tunnels[0], "", true},
		{"whitespace query matches all", tunnels[0], "   ", true},

		// By Name
		{"match name exact", tunnels[0], "api-prod", true},
		{"match name substring", tunnels[0], "api", true},
		{"match name case insensitive", tunnels[0], "API-PROD", true},
		{"match name staging", tunnels[1], "staging", true},

		// By Host
		{"match host substring", tunnels[0], "bastion", true},
		{"match host case insensitive", tunnels[0], "BASTION-PROD", true},
		{"match host box", tunnels[1], "staging-box", true},
		{"match host vps", tunnels[2], "remote.net", true},

		// By Type
		{"match type local", tunnels[0], "local", true},
		{"match type reverse", tunnels[2], "reverse", true},
		{"match type uppercase REVERSE", tunnels[2], "REVERSE", true},

		// By LocalPort
		{"match local port 8080", tunnels[0], "8080", true},
		{"match local port partial 54", tunnels[1], "54", true},
		{"match local port 3000", tunnels[2], "3000", true},

		// By RemotePort
		{"match remote port 80", tunnels[0], "80", true},
		{"match remote port 5432", tunnels[1], "5432", true},
		{"match remote port 9000", tunnels[2], "9000", true},

		// Non-matching
		{"no match unknown string", tunnels[0], "nonexistent", false},
		{"no match wrong port", tunnels[0], "9999", false},
		{"no match wrong host", tunnels[1], "production", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchTunnel(tt.tunnel, tt.query)
			if got != tt.matched {
				t.Errorf("matchTunnel(%q, %q) = %v, want %v", tt.tunnel.Name, tt.query, got, tt.matched)
			}
		})
	}
}

func TestVisibleTunnelsAndIndices(t *testing.T) {
	tunnels := sampleTunnels()
	m := model{
		cfg: config.Config{Tunnels: tunnels},
	}

	// 1. Empty filter
	m.filterQuery = ""
	vis := m.visibleTunnels()
	indices := m.visibleIndices()
	if len(vis) != 3 {
		t.Fatalf("expected 3 visible tunnels for empty query, got %d", len(vis))
	}
	if len(indices) != 3 || indices[0] != 0 || indices[1] != 1 || indices[2] != 2 {
		t.Fatalf("expected indices [0, 1, 2], got %v", indices)
	}

	// 2. Filter by name "api"
	m.filterQuery = "api"
	vis = m.visibleTunnels()
	indices = m.visibleIndices()
	if len(vis) != 1 || vis[0].Name != "api-prod" {
		t.Fatalf("expected ['api-prod'], got %+v", vis)
	}
	if len(indices) != 1 || indices[0] != 0 {
		t.Fatalf("expected indices [0], got %v", indices)
	}

	// 3. Filter by type "local" -> matches 2 tunnels
	m.filterQuery = "local"
	vis = m.visibleTunnels()
	indices = m.visibleIndices()
	if len(vis) != 2 {
		t.Fatalf("expected 2 visible tunnels for 'local', got %d", len(vis))
	}
	if indices[0] != 0 || indices[1] != 1 {
		t.Fatalf("expected indices [0, 1], got %v", indices)
	}

	// 4. Filter by port "9000" -> matches webhook-rev at index 2
	m.filterQuery = "9000"
	vis = m.visibleTunnels()
	indices = m.visibleIndices()
	if len(vis) != 1 || vis[0].Name != "webhook-rev" {
		t.Fatalf("expected ['webhook-rev'], got %+v", vis)
	}
	if len(indices) != 1 || indices[0] != 2 {
		t.Fatalf("expected indices [2], got %v", indices)
	}

	// 5. Filter matching nothing
	m.filterQuery = "nomatch"
	vis = m.visibleTunnels()
	indices = m.visibleIndices()
	if len(vis) != 0 || len(indices) != 0 {
		t.Fatalf("expected 0 matches, got %d tunnels, %d indices", len(vis), len(indices))
	}
}

func TestSelectedWithFilter(t *testing.T) {
	tunnels := sampleTunnels()
	m := model{
		cfg: config.Config{Tunnels: tunnels},
	}

	// Filter matches tunnel index 1 ("db-staging")
	m.filterQuery = "5432"
	m.cursor = 0

	sel, ok := m.selected()
	if !ok || sel.Name != "db-staging" {
		t.Fatalf("expected selected tunnel 'db-staging', got ok=%v, sel=%+v", ok, sel)
	}
	if idx := m.selectedIndex(); idx != 1 {
		t.Fatalf("expected selectedIndex 1, got %d", idx)
	}

	// Cursor out of bounds when no matches
	m.filterQuery = "nomatch"
	_, ok = m.selected()
	if ok {
		t.Fatal("expected ok=false when no tunnels match filter")
	}
	if idx := m.selectedIndex(); idx != -1 {
		t.Fatalf("expected selectedIndex -1, got %d", idx)
	}
}

func TestFilterKeyboardLifecycle(t *testing.T) {
	cfg := config.Config{Tunnels: sampleTunnels()}
	mModel := New(cfg, "", nil)
	m := mModel.(model)

	// 1. Initial state
	if m.filtering || m.filterQuery != "" {
		t.Fatal("expected filtering=false and filterQuery='' initially")
	}

	// 2. Press '/' to enter filter mode
	mRes, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mRes.(model)
	if !m.filtering {
		t.Fatal("expected m.filtering=true after pressing '/'")
	}

	// 3. Type "api"
	for _, r := range "api" {
		mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mRes.(model)
	}
	if m.filterQuery != "api" {
		t.Fatalf("expected filterQuery='api', got %q", m.filterQuery)
	}
	if len(m.visibleTunnels()) != 1 {
		t.Fatalf("expected 1 visible tunnel, got %d", len(m.visibleTunnels()))
	}

	// 4. Press Esc to leave search input while retaining filterQuery
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mRes.(model)
	if m.filtering {
		t.Fatal("expected filtering=false after Esc in input mode")
	}
	if m.filterQuery != "api" {
		t.Fatalf("expected filterQuery to remain 'api', got %q", m.filterQuery)
	}

	// 5. Press Esc again in list mode to clear filter
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mRes.(model)
	if m.filterQuery != "" {
		t.Fatalf("expected filterQuery='' after second Esc, got %q", m.filterQuery)
	}
	if len(m.visibleTunnels()) != 3 {
		t.Fatalf("expected all 3 tunnels visible after clear, got %d", len(m.visibleTunnels()))
	}

	// 6. Enter filter mode and press Enter to exit input mode
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = mRes.(model)
	for _, r := range "staging" {
		mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mRes.(model)
	}
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mRes.(model)
	if m.filtering {
		t.Fatal("expected filtering=false after Enter")
	}
	if m.filterQuery != "staging" {
		t.Fatalf("expected filterQuery='staging', got %q", m.filterQuery)
	}
}

func TestHelpModalToggleAndDismiss(t *testing.T) {
	cfg := config.Config{Tunnels: sampleTunnels()}
	mModel := New(cfg, "", nil)
	m := mModel.(model)

	// 1. Initial state
	if m.showHelp {
		t.Fatal("expected showHelp=false initially")
	}

	// 2. Press '?' to toggle help modal
	mRes, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = mRes.(model)
	if !m.showHelp {
		t.Fatal("expected showHelp=true after pressing '?'")
	}

	// 3. View() should contain help modal content
	rendered := m.View()
	if !strings.Contains(rendered, "Bantuan & Shortcut") {
		t.Fatal("expected View() to render help modal when showHelp=true")
	}

	// 4. Press '?' again to dismiss
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = mRes.(model)
	if m.showHelp {
		t.Fatal("expected showHelp=false after toggling '?'")
	}

	// 5. Open and dismiss with Esc
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = mRes.(model)
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mRes.(model)
	if m.showHelp {
		t.Fatal("expected showHelp=false after Esc")
	}

	// 6. Open and dismiss with Enter
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = mRes.(model)
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mRes.(model)
	if m.showHelp {
		t.Fatal("expected showHelp=false after Enter")
	}
}

func TestLogScrollKeys(t *testing.T) {
	cfg := config.Config{Tunnels: sampleTunnels()}
	mModel := New(cfg, "", nil)
	m := mModel.(model)

	if m.logScroll != 0 {
		t.Fatalf("expected initial logScroll=0, got %d", m.logScroll)
	}

	// Test K and pgup
	mRes, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	m = mRes.(model)
	if m.logScroll != 1 {
		t.Fatalf("expected logScroll=1 after 'K', got %d", m.logScroll)
	}

	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = mRes.(model)
	if m.logScroll != 6 {
		t.Fatalf("expected logScroll=6 after 'pgup', got %d", m.logScroll)
	}

	// Test J and pgdown
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	m = mRes.(model)
	if m.logScroll != 5 {
		t.Fatalf("expected logScroll=5 after 'J', got %d", m.logScroll)
	}

	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = mRes.(model)
	if m.logScroll != 0 {
		t.Fatalf("expected logScroll=0 after 'pgdown', got %d", m.logScroll)
	}

	// Doesn't go negative
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = mRes.(model)
	if m.logScroll < 0 {
		t.Fatalf("expected logScroll >= 0, got %d", m.logScroll)
	}

	// Cursor movement resets logScroll
	m.logScroll = 10
	mRes, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = mRes.(model)
	if m.logScroll != 0 {
		t.Fatalf("expected cursor move to reset logScroll to 0, got %d", m.logScroll)
	}
}

func TestClipboardKeyHandler(t *testing.T) {
	cfg := config.Config{Tunnels: sampleTunnels()}
	mModel := New(cfg, "", nil)
	m := mModel.(model)

	// Press 'c' to copy selected tunnel URL
	mRes, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = mRes.(model)

	if !strings.Contains(m.msg, "http://localhost:8080") {
		t.Fatalf("expected flash message containing copied URL, got %q", m.msg)
	}
}
