package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"sshtui/internal/config"
	"sshtui/internal/tunnel"
)

func TestHeaderBar(t *testing.T) {
	cfg := config.Config{
		Hosts: map[string]config.HostCfg{
			"vps": {Address: "127.0.0.1", User: "root", Port: 22},
		},
		Tunnels: []config.TunnelCfg{
			{Name: "T1", Host: "vps", Type: "local", LocalPort: 8081, RemotePort: 80},
			{Name: "T2", Host: "vps", Type: "local", LocalPort: 8082, RemotePort: 81},
		},
	}
	mgr := tunnel.NewManager()
	m := New(cfg, "/path/to/config.toml", mgr).(model)
	m.width = 100
	m.height = 30

	view := m.View()

	// 1. Title badge & subtitle
	if !strings.Contains(view, "sshtui") {
		t.Errorf("expected view to contain app title 'sshtui'")
	}
	if !strings.Contains(view, "SSH tunnel manager") {
		t.Errorf("expected view to contain subtitle 'SSH tunnel manager'")
	}

	// 2. Config path
	if !strings.Contains(view, "/path/to/config.toml") {
		t.Errorf("expected view to contain config path")
	}

	// 3. Summary pill
	// Both tunnels stopped -> 0 Aktif · 2 Berhenti · 0 Error
	if !strings.Contains(view, "0 Aktif") {
		t.Errorf("expected summary pill to contain '0 Aktif', got:\n%s", view)
	}
	if !strings.Contains(view, "2 Berhenti") {
		t.Errorf("expected summary pill to contain '2 Berhenti', got:\n%s", view)
	}
	if !strings.Contains(view, "0 Error") {
		t.Errorf("expected summary pill to contain '0 Error', got:\n%s", view)
	}
}

func TestHeaderBar_LongConfigPathTruncation(t *testing.T) {
	cfg := config.Config{
		Hosts: map[string]config.HostCfg{
			"vps": {Address: "127.0.0.1", User: "root", Port: 22},
		},
		Tunnels: []config.TunnelCfg{
			{Name: "T1", Host: "vps", Type: "local", LocalPort: 8081, RemotePort: 80},
		},
	}
	mgr := tunnel.NewManager()
	longPath := "/very/long/nested/path/to/directory/with/lots/of/levels/and/sublevels/config.toml"
	m := New(cfg, longPath, mgr).(model)
	m.width = 95
	m.height = 30

	view := m.View()

	// Should contain truncated path with ellipsis "…"
	if !strings.Contains(view, "…") {
		t.Errorf("expected view to contain truncated path with ellipsis '…', got:\n%s", view)
	}
	// Shouldn't contain full path
	if strings.Contains(view, longPath) {
		t.Errorf("expected long path to be truncated, but full path was found")
	}
}

func TestViewInspector(t *testing.T) {
	cfg := config.Config{
		Hosts: map[string]config.HostCfg{
			"homelab": {Address: "homelab.my.id", User: "arry", Port: 2222, Key: "~/.ssh/id_ed25519"},
		},
		Tunnels: []config.TunnelCfg{
			{
				Name:       "DevAPI",
				Host:       "homelab",
				Type:       "local",
				LocalPort:  8891,
				RemoteHost: "127.0.0.1",
				RemotePort: 20128,
				Autostart:  true,
			},
		},
	}
	tun := cfg.Tunnels[0]
	snap := tunnel.Snapshot{
		Status:     tunnel.Running,
		Uptime:     15 * time.Minute,
		Logs:       []string{"12:00:00 tunnel ready"},
		LocalBound: true,
	}

	m := New(cfg, "/cfg", nil).(model)
	m.width = 120

	insp := m.viewInspector(tun, snap, 100)

	// 1. Route diagram
	if !strings.Contains(insp, "8891") || !strings.Contains(insp, "20128") {
		t.Errorf("expected inspector to contain route ports, got:\n%s", insp)
	}

	// 2. Latency check line
	if !strings.Contains(insp, "Latency") {
		t.Errorf("expected inspector to contain Latency line, got:\n%s", insp)
	}

	// 3. Local listener status
	if !strings.Contains(insp, "Local Listener") || !strings.Contains(insp, "8891") {
		t.Errorf("expected inspector to contain local listener info, got:\n%s", insp)
	}

	// 4. Lifecycle status & uptime
	if !strings.Contains(insp, "15m00s") || !strings.Contains(insp, "Berjalan") {
		t.Errorf("expected inspector to contain uptime '15m00s' and 'Berjalan', got:\n%s", insp)
	}

	// 5. Credential info
	if !strings.Contains(insp, "arry") || !strings.Contains(insp, "homelab.my.id") || !strings.Contains(insp, "~/.ssh/id_ed25519") {
		t.Errorf("expected inspector to contain credentials (user, host, key), got:\n%s", insp)
	}

	// Also verify package function viewInspector
	pkgInsp := viewInspector(tun, snap, 100)
	if !strings.Contains(pkgInsp, "8891") {
		t.Errorf("package viewInspector should also work")
	}

	// Truncation on narrow width
	narrowInsp := m.viewInspector(tun, snap, 35)
	if !strings.Contains(narrowInsp, "…") {
		t.Errorf("expected route diagram to be truncated in narrow inspector, got:\n%s", narrowInsp)
	}
}

func TestViewInspectorReconnectingAndError(t *testing.T) {
	cfg := config.Config{
		Tunnels: []config.TunnelCfg{
			{Name: "Webhook", Host: "testuser@remote.com", Type: "reverse", LocalPort: 3000, RemotePort: 9000},
		},
	}
	tun := cfg.Tunnels[0]
	snap := tunnel.Snapshot{
		Status:     tunnel.Reconnecting,
		RetryCount: 3,
		RetryIn:    5 * time.Second,
		LastError:  "connection refused",
	}

	m := New(cfg, "/cfg", nil).(model)
	insp := m.viewInspector(tun, snap, 60)

	if !strings.Contains(insp, "Reconnecting") || !strings.Contains(insp, "#3") {
		t.Errorf("expected reconnecting attempt #3 in inspector, got:\n%s", insp)
	}
	if !strings.Contains(insp, "connection refused") {
		t.Errorf("expected last error 'connection refused' in inspector, got:\n%s", insp)
	}
	if !strings.Contains(insp, "testuser") || !strings.Contains(insp, "remote.com") {
		t.Errorf("expected user and remote host in credentials, got:\n%s", insp)
	}
}

func TestViewLogPane(t *testing.T) {
	tun := config.TunnelCfg{Name: "App"}
	logs := []string{
		"line 1: connecting",
		"line 2: authenticating",
		"line 3: forwarding established",
		"line 4: ping ok",
		"line 5: traffic 12kb",
		"line 6: keepalive sent",
	}

	m := New(config.Config{}, "/cfg", nil).(model)
	m.logScroll = 2

	// Height 3 with scroll 2 on 6 lines:
	// total 6, maxScroll = 3. scroll = 2. end = 6 - 2 = 4. start = 4 - 3 = 1.
	// should show lines[1:4] => line 2, line 3, line 4
	pane := m.viewLogPane(tun, logs, 50, 3)

	if !strings.Contains(pane, "Log") || !strings.Contains(pane, "App") {
		t.Errorf("expected pane header with tunnel name, got:\n%s", pane)
	}
	if !strings.Contains(pane, "PgUp/PgDn") || !strings.Contains(pane, "J/K") {
		t.Errorf("expected scroll hint in header, got:\n%s", pane)
	}
	if !strings.Contains(pane, "line 2") || !strings.Contains(pane, "line 3") || !strings.Contains(pane, "line 4") {
		t.Errorf("expected sliced lines [line 2, line 3, line 4], got:\n%s", pane)
	}
	if strings.Contains(pane, "line 6") {
		t.Errorf("expected scrolled view to not show newest line 6, got:\n%s", pane)
	}

	// Empty logs check
	emptyPane := m.viewLogPane(tun, nil, 50, 3)
	if !strings.Contains(emptyPane, "belum ada log") {
		t.Errorf("expected '(belum ada log)' in empty log pane, got:\n%s", emptyPane)
	}

	// Package function check
	pkgPane := viewLogPane(tun, logs, 50, 3)
	if !strings.Contains(pkgPane, "Log") {
		t.Errorf("package viewLogPane should work")
	}
}

func TestSplitPaneLayout(t *testing.T) {
	cfg := config.Config{
		Tunnels: []config.TunnelCfg{
			{Name: "Postgres", Host: "db.internal", Type: "local", LocalPort: 5433, RemotePort: 5432, Autostart: true},
			{Name: "WebHook", Host: "api.ext", Type: "reverse", LocalPort: 3000, RemotePort: 8080},
		},
	}
	m := New(cfg, "/cfg", nil).(model)
	m.width = 110
	m.height = 32

	view := m.View()

	// Wide mode (width = 110 >= 90):
	// Left column components
	if !strings.Contains(view, "►") {
		t.Errorf("expected selection marker '►' in split table, got:\n%s", view)
	}
	if !strings.Contains(view, "[LOCAL]") {
		t.Errorf("expected '[LOCAL]' type badge in table, got:\n%s", view)
	}
	if !strings.Contains(view, "[REV]") {
		t.Errorf("expected '[REV]' type badge in table, got:\n%s", view)
	}
	if !strings.Contains(view, ":5433") {
		t.Errorf("expected local port ':5433' in table, got:\n%s", view)
	}
	if !strings.Contains(view, "↻") {
		t.Errorf("expected autostart indicator '↻' in table, got:\n%s", view)
	}

	// Right column components
	if !strings.Contains(view, "Inspektor") {
		t.Errorf("expected inspector in right column, got:\n%s", view)
	}
	if !strings.Contains(view, "Log") {
		t.Errorf("expected log pane in right column, got:\n%s", view)
	}

	// Verify viewSplit package function
	splitOut := viewSplit(110, 106)
	if splitOut == "" {
		t.Errorf("expected viewSplit to return rendered string")
	}
}

func TestResponsiveFallback(t *testing.T) {
	cfg := config.Config{
		Tunnels: []config.TunnelCfg{
			{Name: "Postgres", Host: "db.internal", Type: "local", LocalPort: 5433, RemotePort: 5432},
		},
	}
	m := New(cfg, "/cfg", nil).(model)
	m.width = 75 // Narrow (< 90)
	m.height = 25

	view := m.View()

	// Should not panic, should render stacked view
	if !strings.Contains(view, "Postgres") {
		t.Errorf("expected tunnel name in fallback view")
	}
	if !strings.Contains(view, "Log") {
		t.Errorf("expected log section in fallback view")
	}
}

func TestFormAnimatedSpinner(t *testing.T) {
	cfg := config.Config{
		Hosts: map[string]config.HostCfg{"srv": {Address: "1.2.3.4"}},
	}
	m := New(cfg, "/cfg", nil).(model)
	m.mode = modeForm
	m.form = newForm(config.TunnelCfg{Host: "srv"}, -1)
	m.form.scanning = true
	m.form.spinFrame = 0

	view0 := m.viewForm(80)
	if !strings.Contains(view0, "⠋") && !strings.Contains(view0, "mendeteksi port") {
		t.Errorf("expected spinner frame ⠋ when scanning, got:\n%s", view0)
	}

	// Advance frame via tickMsg
	mUpdated, _ := m.Update(tickMsg(time.Now()))
	m2 := mUpdated.(model)
	if m2.form.spinFrame != 1 {
		t.Errorf("expected spinFrame to advance to 1 on tickMsg, got %d", m2.form.spinFrame)
	}

	view1 := m2.viewForm(80)
	if !strings.Contains(view1, "⠙") {
		t.Errorf("expected spinner frame ⠙ at frame 1, got:\n%s", view1)
	}
}

func TestFormNewTunnelClearsFilter(t *testing.T) {
	cfg := config.Config{
		Tunnels: []config.TunnelCfg{
			{Name: "OldTunnel", Host: "box", Type: "local", LocalPort: 8080, RemotePort: 80},
		},
	}
	m := New(cfg, "/cfg", nil).(model)
	m.filterQuery = "old"
	m.filterInput.SetValue("old")

	// Open new tunnel form
	m.form, m.mode = newForm(config.TunnelCfg{RemoteHost: "localhost"}, -1), modeForm
	m.form.fields[0].input.SetValue("NewTunnel")
	m.form.fields[1].input.SetValue("box")
	m.form.fields[2].input.SetValue("local")
	m.form.fields[4].input.SetValue("9090")
	m.form.fields[6].input.SetValue("90")

	// Submit with enter or ctrl+s
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	res := updated.(model)

	if res.filterQuery != "" {
		t.Errorf("expected filterQuery to be cleared on new tunnel, got %q", res.filterQuery)
	}
	if res.filterInput.Value() != "" {
		t.Errorf("expected filterInput to be empty, got %q", res.filterInput.Value())
	}
	if len(res.cfg.Tunnels) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(res.cfg.Tunnels))
	}
	if res.cursor != 1 {
		t.Errorf("expected newly added tunnel to be selected (cursor 1), got %d", res.cursor)
	}
	sel, ok := res.selected()
	if !ok || sel.Name != "NewTunnel" {
		t.Errorf("expected selected tunnel to be NewTunnel, got %+v", sel)
	}
}
