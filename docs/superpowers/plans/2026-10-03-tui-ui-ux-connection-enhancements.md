# TUI UI/UX & Connection Informativeness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Overhaul `sshtui`'s UI/UX with a responsive split-pane layout, real-time connection inspector, native Go latency & socket telemetry, search/filter, log scrolling, clipboard copy, and a help modal.

**Architecture:** A native Go telemetry engine in `internal/tunnel` periodically probes SSH host latency and local port listener states non-blockingly, reporting atomic snapshots to Bubble Tea. The TUI in `internal/tui` renders a responsive split-pane layout with unified keyboard navigation, real-time log scrolling, and modal overlays.

**Tech Stack:** Go 1.22+, Bubble Tea (`github.com/charmbracelet/bubbletea`), Lipgloss (`github.com/charmbracelet/lipgloss`), Bubbles textinput (`github.com/charmbracelet/bubbles/textinput`), Standard library `net`, `time`, `os/exec`.

**Spec:** `docs/superpowers/specs/2026-10-03-tui-ui-ux-connection-enhancements-design.md`

## Global Constraints

- 100% native Go probes for latency and socket checks (`net.DialTimeout`), no external ping or shell subprocess dependencies.
- Zero breaking changes to existing `~/.config/sshtui/config.toml` structure.
- Responsive layout: split-pane for terminal width >= 90 columns, graceful stacked fallback for width < 90 columns.
- Unified navigation: cursor navigation stays on tunnel list (`↑↓/jk`), while log history scrolls via `PgUp/PgDn` or `J/K` without pane switching.

---

### Task 1: Connection Telemetry & Health Engine in `internal/tunnel`

**Files:**
- Create: `internal/tunnel/telemetry.go`
- Create: `internal/tunnel/telemetry_test.go`
- Modify: `internal/tunnel/tunnel.go`

**Interfaces:**
- Produces:
  ```go
  type Snapshot struct {
      Status      Status
      Uptime      time.Duration
      Logs        []string
      LastError   string
      RetryIn     time.Duration
      RetryCount  int
      LocalBound  bool
      LatencyRTT  time.Duration
      LatencyErr  error
  }
  func ProbeHost(c config.Config, host string) (time.Duration, error)
  func CheckLocalPort(bind string, port int) bool
  ```

- [ ] **Step 1: Write failing tests for telemetry probes**

Create `internal/tunnel/telemetry_test.go`:
```go
package tunnel

import (
	"net"
	"strconv"
	"testing"
	"time"

	"sshtui/internal/config"
)

func TestCheckLocalPort(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	_, portStr, _ := net.SplitHostPort(l.Addr().String())
	port, _ := strconv.Atoi(portStr)

	if !CheckLocalPort("127.0.0.1", port) {
		t.Errorf("expected port %d to be listening", port)
	}

	if CheckLocalPort("127.0.0.1", 59999) {
		t.Errorf("expected unused port 59999 to not be listening")
	}
}

func TestProbeHost(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	_, portStr, _ := net.SplitHostPort(l.Addr().String())
	port, _ := strconv.Atoi(portStr)

	cfg := config.Config{
		Hosts: map[string]config.HostCfg{
			"testhost": {Address: "127.0.0.1", Port: port},
		},
	}

	rtt, err := ProbeHost(cfg, "testhost")
	if err != nil {
		t.Fatalf("unexpected error probing host: %v", err)
	}
	if rtt <= 0 {
		t.Errorf("expected positive rtt, got %v", rtt)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tunnel`  
Expected: FAIL with undefined `CheckLocalPort` and `ProbeHost`.

- [ ] **Step 3: Implement telemetry functions in `internal/tunnel/telemetry.go`**

Create `internal/tunnel/telemetry.go`:
```go
package tunnel

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"sshtui/internal/config"
)

var (
	cacheMu     sync.RWMutex
	latencyMap  = make(map[string]latencyEntry)
	cacheExpiry = 5 * time.Second
)

type latencyEntry struct {
	rtt       time.Duration
	err       error
	checkedAt time.Time
}

// CheckLocalPort checks if a local address:port is actively listening.
func CheckLocalPort(bind string, port int) bool {
	if bind == "" {
		bind = "127.0.0.1"
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", bind, port), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ProbeHost measures round-trip connection time to the SSH host.
func ProbeHost(c config.Config, host string) (time.Duration, error) {
	cacheMu.RLock()
	entry, exists := latencyMap[host]
	cacheMu.RUnlock()

	if exists && time.Since(entry.checkedAt) < cacheExpiry {
		return entry.rtt, entry.err
	}

	// Resolve destination and port from config
	targetAddr := host
	port := 22
	if h, ok := c.Hosts[host]; ok {
		targetAddr = h.Address
		if h.Port != 0 {
			port = h.Port
		}
	} else if strings.Contains(host, "@") {
		parts := strings.SplitN(host, "@", 2)
		targetAddr = parts[1]
	}

	addr := net.JoinHostPort(targetAddr, fmt.Sprint(port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	rtt := time.Since(start)

	if err == nil {
		_ = conn.Close()
	}

	cacheMu.Lock()
	latencyMap[host] = latencyEntry{rtt: rtt, err: err, checkedAt: time.Now()}
	cacheMu.Unlock()

	return rtt, err
}
```

- [ ] **Step 4: Update `Runtime` and `Snapshot` in `internal/tunnel/tunnel.go`**

Update `Runtime` struct with:
```go
type Snapshot struct {
	Status     Status
	Uptime     time.Duration
	Logs       []string
	LastError  string
	RetryIn    time.Duration
	RetryCount int
	LocalBound bool
	LatencyRTT time.Duration
	LatencyErr error
}
```
Track `lastError`, `retryCount`, `nextRetryAt` in `Runtime.loop()` and populate `Snapshot()`.

- [ ] **Step 5: Run tests and verify they pass**

Run: `go test -v ./internal/tunnel`  
Expected: PASS.

---

### Task 2: UI Helpers (Clipboard Copy, Help Modal & Route Formatter) in `internal/tui`

**Files:**
- Create: `internal/tui/helpers.go`
- Create: `internal/tui/helpers_test.go`
- Create: `internal/tui/modal.go`

**Interfaces:**
- Produces:
  ```go
  func copyToClipboard(text string) error
  func routeDiagram(t config.TunnelCfg, hostDest string) string
  func viewHelpModal(w, h int) string
  ```

- [ ] **Step 1: Write tests for routeDiagram and helpers**

Create `internal/tui/helpers_test.go`:
```go
package tui

import (
	"strings"
	"testing"

	"sshtui/internal/config"
)

func TestRouteDiagram(t *testing.T) {
	tunnel := config.TunnelCfg{
		Name: "Test", Host: "homelab", Type: "local",
		LocalPort: 8891, RemoteHost: "localhost", RemotePort: 20128,
	}
	diag := routeDiagram(tunnel, "homelab.domain.com:22")
	if !strings.Contains(diag, "8891") || !strings.Contains(diag, "20128") {
		t.Errorf("route diagram missing ports: %s", diag)
	}
}
```

- [ ] **Step 2: Implement `internal/tui/helpers.go`**

Implement `copyToClipboard` (using `pbcopy` on darwin, `wl-copy`/`xclip` on linux) and `routeDiagram`:
```go
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
```

- [ ] **Step 3: Implement `internal/tui/modal.go` for Help Modal**

Create `internal/tui/modal.go`:
Render floating help dialog containing table of hotkeys grouped into Navigation, Tunnel Controls, Quick Actions, and Application.

- [ ] **Step 4: Run tests to verify**

Run: `go test -v ./internal/tui`  
Expected: PASS.

---

### Task 3: Search / Filter & Viewport Scrolling in `internal/tui`

**Files:**
- Modify: `internal/tui/tui.go`

**Interfaces:**
- Consumes: `textinput.Model`
- Produces:
  - Dynamic filtering logic `m.visibleTunnels()`
  - Search trigger on `/` key
  - Viewport scroll index calculation

- [ ] **Step 1: Add filter state & methods in `internal/tui/tui.go`**

Add fields to `model`:
```go
filterInput textinput.Model
filtering   bool
filterQuery string
showHelp    bool
logScroll   int
```

- [ ] **Step 2: Implement `visibleTunnels()` method**

Filter tunnels based on `filterQuery` checking `Name`, `Host`, `Type`, `LocalPort`, `RemotePort`.

- [ ] **Step 3: Handle KeyMsg for `/` and search input**

- Pressing `/` sets `m.filtering = true` and focuses `m.filterInput`.
- Typing updates `m.filterQuery` and recalculates visible list.
- Pressing `Esc` clears filter or leaves search input.
- Pressing `?` toggles `m.showHelp`.
- Pressing `c` copies `http://localhost:<LocalPort>` to clipboard and flashes notification.

- [ ] **Step 4: Verify build**

Run: `go build ./cmd/sshtui`  
Expected: PASS.

---

### Task 4: Responsive Split-Pane Layout & Live Inspector in `internal/tui`

**Files:**
- Modify: `internal/tui/tui.go`

**Interfaces:**
- Consumes: `Snapshot`, `ProbeHost`, `CheckLocalPort`, `routeDiagram`, `viewHelpModal`
- Produces:
  - `viewSplit(w, inner int) string`
  - `viewInspector(t config.TunnelCfg, snap tunnel.Snapshot, inner int) string`
  - `viewLogPane(t config.TunnelCfg, logs []string, inner, height int) string`
  - Responsive switch based on `m.width >= 90`

- [ ] **Step 1: Implement `viewInspector` card**

Renders:
- Route arrow diagram
- SSH Host Latency with color pills (`● 18ms (Sangat Cepat)`)
- Local listener status (`● 127.0.0.1:8891 listening`)
- Status, uptime duration, or reconnection countdown
- Credential details (user, address, ssh key path)

- [ ] **Step 2: Implement `viewLogPane` with scroll support**

Renders:
- Styled header with log line count and scroll hint (`PgUp/PgDn or J/K to scroll`)
- Slice logs by `m.logScroll` offset
- Key handlers for `pgup`/`pgdown` and `K`/`J` modifying `m.logScroll`.

- [ ] **Step 3: Implement `viewSplit` and Header Summary pill**

- Assemble left pane (Tunnel list + Filter) and right pane (Inspector + Log) side-by-side using `lipgloss.JoinHorizontal(lipgloss.Top, left, right)`.
- Header shows `● X Aktif  ·  ○ Y Berhenti  ·  ▲ Z Error`.
- If `m.width < 90`, fallback gracefully to stacked layout.

- [ ] **Step 4: Form animated spinner for `ctrl+d`**

Add spinner ticks or animated indicator `⠋ ⠙ ⠹ ⠸` in `viewForm()` when `f.scanning` is true.

- [ ] **Step 5: Verify build & tests**

Run: `go test ./...` and `make build`  
Expected: PASS.

---

### Task 5: Documentation, CLI Usage & End-to-End Verification

**Files:**
- Modify: `cmd/sshtui/main.go`
- Modify: `README.md`

- [ ] **Step 1: Update `README.md` and CLI usage**

Update shortcuts table in `README.md` to document:
- `/` Instant search/filter
- `c` Copy URL to clipboard
- `?` Help modal
- `PgUp`/`PgDn` or `J`/`K` Log scrolling
- Split-pane layout explanation

- [ ] **Step 2: Run all tests and build check**

Run:
```bash
make test
make build
```
Verify binary is produced at `./bin/sshtui` and all tests pass with zero warnings.

- [ ] **Step 3: Verification walkthrough**

Create `walkthrough.md` summarizing new capabilities, visual layout, and test results.
