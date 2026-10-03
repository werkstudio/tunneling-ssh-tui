package tunnel

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"sshtui/internal/config"
)

type Status int

const (
	Stopped Status = iota
	Starting
	Running
	Reconnecting
	Failed
)

func (s Status) String() string {
	return [...]string{"stopped", "starting", "running", "reconnecting", "error"}[s]
}

const maxLogLines = 200

// Runtime mengelola satu proses ssh -N beserta auto-reconnect.
type Runtime struct {
	mu      sync.Mutex
	status  Status
	since   time.Time // kapan masuk status Running
	logs    []string
	cancel  context.CancelFunc
	done    chan struct{}
	Echo    func(string) // opsional, dipakai mode CLI
	partial string
}

func (r *Runtime) logf(format string, a ...any) {
	line := fmt.Sprintf("%s  %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
	r.mu.Lock()
	r.logs = append(r.logs, line)
	if len(r.logs) > maxLogLines {
		r.logs = r.logs[len(r.logs)-maxLogLines:]
	}
	echo := r.Echo
	r.mu.Unlock()
	if echo != nil {
		echo(line)
	}
}

// Write memecah output ssh per baris (io.Writer untuk cmd.Stderr).
func (r *Runtime) Write(p []byte) (int, error) {
	r.mu.Lock()
	text := r.partial + string(p)
	lines := strings.Split(text, "\n")
	r.partial = lines[len(lines)-1]
	r.mu.Unlock()
	for _, l := range lines[:len(lines)-1] {
		if l = strings.TrimSpace(l); l != "" {
			r.logf("ssh: %s", l)
		}
	}
	return len(p), nil
}

func (r *Runtime) setStatus(s Status) {
	r.mu.Lock()
	r.status = s
	if s == Running {
		r.since = time.Now()
	}
	r.mu.Unlock()
}

func (r *Runtime) Snapshot() (Status, time.Duration, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var up time.Duration
	if r.status == Running {
		up = time.Since(r.since)
	}
	return r.status, up, append([]string(nil), r.logs...)
}

func (r *Runtime) Active() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cancel != nil
}

// Start menjalankan tunnel. checkAddr (opsional) dicek bebas sebelum start.
func (r *Runtime) Start(args []string, checkAddr string) {
	r.mu.Lock()
	if r.cancel != nil {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.done = make(chan struct{})
	r.status = Starting
	r.mu.Unlock()
	go r.loop(ctx, args, checkAddr)
}

// Stop menghentikan tunnel dan menunggu proses selesai.
func (r *Runtime) Stop() {
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}

func (r *Runtime) finish(s Status) {
	r.mu.Lock()
	r.status = s
	r.cancel = nil
	done := r.done
	r.mu.Unlock()
	close(done)
}

func (r *Runtime) loop(ctx context.Context, args []string, checkAddr string) {
	if checkAddr != "" {
		l, err := net.Listen("tcp", checkAddr)
		if err != nil {
			r.logf("port %s sudah dipakai proses lain", checkAddr)
			r.finish(Failed)
			return
		}
		l.Close()
	}
	backoff := 2 * time.Second
	first := true
	for ctx.Err() == nil {
		if first {
			r.setStatus(Starting)
		} else {
			r.setStatus(Reconnecting)
		}
		first = false
		r.logf("ssh %s", strings.Join(args, " "))
		cmd := exec.CommandContext(ctx, "ssh", args...)
		cmd.Stderr = r
		if err := cmd.Start(); err != nil {
			r.logf("gagal menjalankan ssh: %v", err)
			r.finish(Failed)
			return
		}
		started := time.Now()
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()

		var err error
		select {
		case err = <-exited:
		case <-time.After(2 * time.Second):
			r.setStatus(Running)
			r.logf("tunnel aktif")
			err = <-exited
		}
		if ctx.Err() != nil {
			break
		}
		r.logf("ssh berhenti: %v", err)
		if time.Since(started) > 30*time.Second {
			backoff = 2 * time.Second
		}
		r.setStatus(Reconnecting)
		r.logf("mencoba lagi dalam %s", backoff)
		select {
		case <-ctx.Done():
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
	r.logf("tunnel dihentikan")
	r.finish(Stopped)
}

// Manager menyimpan Runtime per nama tunnel.
type Manager struct {
	mu sync.Mutex
	rt map[string]*Runtime
}

func NewManager() *Manager { return &Manager{rt: map[string]*Runtime{}} }

func (m *Manager) Get(name string) *Runtime {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rt[name]
	if !ok {
		r = &Runtime{}
		m.rt[name] = r
	}
	return r
}

func (m *Manager) StartTunnel(c config.Config, t config.TunnelCfg) error {
	args, err := config.SSHArgs(c, t)
	if err != nil {
		return err
	}
	check := ""
	if t.Type == "local" {
		check = fmt.Sprintf("%s:%d", t.BindAddr(), t.LocalPort)
	}
	m.Get(t.Name).Start(args, check)
	return nil
}

func (m *Manager) StopTunnel(name string) { m.Get(name).Stop() }

func (m *Manager) Forget(name string) {
	m.mu.Lock()
	delete(m.rt, name)
	m.mu.Unlock()
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	all := make([]*Runtime, 0, len(m.rt))
	for _, r := range m.rt {
		all = append(all, r)
	}
	m.mu.Unlock()
	var wg sync.WaitGroup
	for _, r := range all {
		wg.Add(1)
		go func(r *Runtime) { defer wg.Done(); r.Stop() }(r)
	}
	wg.Wait()
}
