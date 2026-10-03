# Design Specification: sshtui UI/UX & Connection Informativeness Overhaul

**Date:** 2026-10-03  
**Status:** Approved  
**Author:** Pair Programming Agent & User  

---

## 1. Overview & Problem Statement

`sshtui` is a modern TUI written in Go (Bubble Tea + Lipgloss) for managing SSH tunnels.
While the foundational functionality (tunnel management, auto-reconnect, and remote port detection) is solid, the existing UI/UX has key limitations:
1. **Connection Opacity**: Tunnel status is limited to a simple process check (`Running`, `Stopped`, `Reconnecting`, `Failed`). Users cannot see actual SSH host latency/reachability, cannot verify if the local port is actively bound and listening for traffic, and failure reasons are obscured within raw logs.
2. **Rigid Layout**: Tunnel table and logs are stacked vertically. As tunnel count grows, log view shrinks to an unscrollable 4-line window, leaving wide terminal space unutilized.
3. **Ergonomic Gaps**: Lacks quick filter/search (`/`), clipboard copy (`c`), log scrolling, and an interactive keyboard help modal (`?`).

This design specifies a **Responsive Split-Pane Layout with an Integrated Connection Inspector and Native Go Telemetry Engine**.

---

## 2. Visual Architecture & Layout

### 2.1 Split-Pane Layout (Terminal Width >= 90 columns)

```text
┌─ sshtui ─ SSH Tunnel Manager ──────────── [ ● 2 Aktif  ·  ○ 1 Berhenti  ·  ▲ 0 Error ] ─ ~/.config/sshtui/config.toml ┐
│                                                                                                                       │
│  [ DAFTAR TUNNEL ] (/ cari)                         [ INSPEKTOR KONEKSI: DMS Mazda ]                                  │
│ ┌──────────────────────────────────────────────┐   ┌────────────────────────────────────────────────────────────────┐ │
│ │  STATUS  TIPE   NAMA TUNNEL     PORT LOKAL   │   │  Rute Koneksi:                                                 │ │
│ │ ──────────────────────────────────────────── │   │  127.0.0.1:8891  ──────►  [homelab.arry.my.id:22]              │ │
│ │► ● ON   [LOCAL] DMS Mazda       :8891   ↻    │   │                           └──► 127.0.0.1:20128 (Remote Target) │ │
│ │  ● ON   [LOCAL] Postgres Dev    :5433        │   │                                                                │ │
│ │  ○ OFF  [REV]   Webhook Test    :3000        │   │  Telemetri & Kesehatan:                                        │ │
│ │                                              │   │  ● SSH Host Latency  : 18ms (Sangat Baik / Terhubung)          │ │
│ │                                              │   │  ● Local Listener    : 127.0.0.1:8891 aktif (Siap menerima)    │ │
│ │                                              │   │  ● Status Siklus     : Berjalan normal (Uptime: 24m 10s)       │ │
│ │                                              │   │  ● Kredensial        : user: arry  |  key: ~/.ssh/id_ed25519   │ │
│ │                                              │   └────────────────────────────────────────────────────────────────┘ │
│ │                                              │   [ LOG REALTIME — DMS Mazda ] (PgUp/PgDn / J/K scroll)              │ │
│ │                                              │   ┌────────────────────────────────────────────────────────────────┐ │
│ │                                              │   │ 13:50:12  ssh -N -o BatchMode=yes ... arry@homelab.arry.my.id  │ │
│ │                                              │   │ 13:50:14  tunnel aktif dan siap melayani traffic               │ │
│ │                                              │   │ 13:52:00  ping RTT: 18ms (koneksi stabil)                      │ │
│ └──────────────────────────────────────────────┘   └────────────────────────────────────────────────────────────────┘ │
│                                                                                                                       │
│ [Pesan Flash / Status Notifikasi] Tersimpan: DMS Mazda                                                                │
│ [Space] On/Off  [r] Restart  [o] Browser  [c] Salin URL  [/] Cari  [n/e/d] Baru/Edit/Hapus  [?] Bantuan  [q] Keluar    │
└───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Layout Components
1. **Header Bar**:
   - Left: Styled app title badge `sshtui` and subtitle.
   - Center: Global status summary pill: `● X Aktif  ·  ○ Y Berhenti  ·  ▲ Z Error`.
   - Right: Current config path in dim text.
2. **Left Column (~45% width)**:
   - Filter input box when `/` is pressed.
   - Scrollable viewport for tunnel list if tunnels exceed vertical height.
   - Column layout: Selection marker (`►`), Status dot (`●`/`○`), Type badge (`[LOCAL]`/`[REV]`), Tunnel Name (truncated if needed), Local Port (`:8891`), and Autostart indicator (`↻`).
3. **Right Column (~55% width)**:
   - **Connection Inspector Card (Top)**:
     - Visual route arrow diagram: `Local Bind -> SSH Server -> Remote Target`.
     - SSH Host Latency indicator with color-coded classification.
     - Local Listener verification status.
     - Lifecycle details (Uptime duration or Reconnection countdown `retry in Xs (attempt #N)`).
     - SSH connection parameters (effective user, address, port, and key path).
   - **Live Log Stream (Bottom)**:
     - Log lines formatted with timestamps.
     - Scrollable via `PgUp`/`PgDn` or `J`/`K` keys.
4. **Responsive Fallback (Terminal Width < 90 columns)**:
   - Collapses gracefully into a stacked view (Tunnel list on top, Inspector & Log on bottom) to prevent text clipping.

---

## 3. Connection Telemetry & Health Engine

### 3.1 Host Latency RTT Engine
- **Mechanism**:
  - Uses standard library `net.DialTimeout("tcp", hostAddr, 2*time.Second)`.
  - Non-blocking goroutine triggered by periodic tick (every 5 seconds) for the active/selected tunnel host.
  - Cached by host address to avoid redundant probes.
- **Visual Thresholds**:
  - `< 50ms`: Green `● <RTT>ms (Sangat Cepat)`
  - `50ms - 150ms`: Cyan/Green `● <RTT>ms (Stabil)`
  - `150ms - 300ms`: Yellow `▲ <RTT>ms (Sedang)`
  - `> 300ms`: Orange `▲ <RTT>ms (Lambat)`
  - Timeout / Offline: Red `✗ Tidak Terjangkau`

### 3.2 Local Listener Verification
- **Mechanism**:
  - For `local` tunnels, once SSH process is launched and running, verify that `127.0.0.1:<LocalPort>` is listening via `net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", t.LocalPort), 200*time.Millisecond)`.
- **Status Reporting**:
  - If bound & accepting: `● 127.0.0.1:<Port> aktif (Siap menerima koneksi)`
  - If starting/waiting: `◌ Menunggu bind port...`
  - If failed/conflict: `✗ Port bentrok / belum terbuka`

### 3.3 Reconnection & Error Telemetry
- `tunnel.Runtime` enhanced with:
  - `lastError string`: Captured from SSH stderr line (e.g. `Address already in use`, `Permission denied (publickey)`).
  - `retryCount int`: Cumulative reconnect attempts.
  - `nextRetryIn time.Duration`: Countdown remaining until next backoff attempt.
- `tunnel.Snapshot` struct updated:
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
  ```

---

## 4. Navigation, Controls & Modals

### 4.1 Keybindings
- `↑` / `↓` or `k` / `j`: Move tunnel cursor
- `Space` or `Enter`: Start / Stop selected tunnel
- `r`: Restart selected tunnel
- `a` / `x`: Start all / Stop all
- `o`: Open `http://localhost:<LocalPort>` in default web browser
- `c`: Copy `http://localhost:<LocalPort>` to clipboard using system tool (`pbcopy` on macOS, `wl-copy`/`xclip` on Linux) with flash message feedback
- `PgUp` / `PgDn` or `J` / `K`: Scroll log lines without changing cursor selection
- `/`: Enter instant search/filter mode
- `n` / `e` / `d`: New tunnel / Edit tunnel / Delete tunnel
- `?`: Toggle Help Modal overlay
- `q` or `Ctrl+C`: Gracefully exit and stop running tunnels

### 4.2 Search / Filter Mode (`/`)
- Pressing `/` focuses an inline filter text input above the tunnel list.
- Real-time filtering matching against:
  - Tunnel name
  - Host alias or address
  - Local port or remote port
- Pressing `Esc` or `Enter` leaves search input while maintaining active filter, pressing `Esc` again clears filter.

### 4.3 Help Modal Overlay (`?`)
- Centered popup rendered with `lipgloss.RoundedBorder()`.
- Categorized tables:
  - **Navigasi**: `↑↓ / jk`, `PgUp/PgDn / JK`, `/`
  - **Kontrol Tunnel**: `Space / Enter`, `r`, `a`, `x`
  - **Aksi Cepat**: `o`, `c`, `n`, `e`, `d`
  - **Aplikasi**: `?`, `q`
- Closes on `Esc`, `?`, or `Enter`.

### 4.4 Form Improvements
- **Animated Spinner**: During port scan (`ctrl+d`), display animated spinner (`⠋ ⠙ ⠹ ⠸`) rather than static text.
- **Host Hints**: Displays known hosts from `[host.*]` in config as prompt hints.
- **Inline Validation**: Highlights invalid fields directly with clear guidance.

---

## 5. File Modifications & Structure

1. `internal/tunnel/tunnel.go`:
   - Add `lastError`, `retryCount`, `nextRetryAt` to `Runtime`.
   - Update `Snapshot()` to return the rich telemetry struct.
   - Add non-blocking latency prober function `ProbeHost(c config.Config, host string) (time.Duration, error)`.
   - Add local port listener prober `CheckLocalPort(port int) bool`.
2. `internal/tui/tui.go`:
   - Implement responsive split-pane rendering (left: tunnel list & filter; right: inspector card & live scrollable logs).
   - Add clipboard copy helper (`copyToClipboard`).
   - Implement filter mode (`/`) and text filter logic.
   - Implement help modal (`?`) view.
   - Add animated spinner for `ctrl+d` port scanning.
3. `cmd/sshtui/main.go` & `README.md`:
   - Document new keybindings (`c`, `/`, `?`, `PgUp/PgDn`).

---

## 6. Verification Plan

1. **Unit & Integration Tests**:
   - Add tests for `ProbeHost` and `CheckLocalPort` in `internal/tunnel/`.
   - Run `go test ./...` to ensure clean builds and test passes.
2. **Build Verification**:
   - `make build` generates clean `./bin/sshtui` binary without errors or lint issues.
3. **Manual Interactive Verification**:
   - Start `./bin/sshtui` in terminal with sample config.
   - Verify split layout, inspector data (latency, local port status, route arrows), log scrolling with `PgUp`/`PgDn`, search filter with `/`, copy URL with `c`, and help modal with `?`.
   - Test terminal resizing down to < 90 cols to verify responsive fallback.
