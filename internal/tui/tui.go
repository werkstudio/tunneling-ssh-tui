package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sshtui/internal/config"
	"sshtui/internal/detect"
	"sshtui/internal/tunnel"
)

var (
	cAccent = lipgloss.Color("#7D56F4")
	cGreen  = lipgloss.Color("#04B575")
	cRed    = lipgloss.Color("#FF5F87")
	cYellow = lipgloss.Color("#FFCC66")
	cDim    = lipgloss.Color("#767676")
	cText   = lipgloss.Color("#E4E4E4")

	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(cAccent).Padding(0, 1)
	dimStyle    = lipgloss.NewStyle().Foreground(cDim)
	selStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#3B2D7A"))
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cDim).Padding(0, 1)
	errStyle    = lipgloss.NewStyle().Foreground(cRed)
	okStyle     = lipgloss.NewStyle().Foreground(cGreen)
	labelStyle  = lipgloss.NewStyle().Foreground(cDim).Width(14)
	focusLabel  = lipgloss.NewStyle().Foreground(cAccent).Bold(true).Width(14)
	keyStyle    = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	statusStyle = map[tunnel.Status]lipgloss.Style{
		tunnel.Stopped:      lipgloss.NewStyle().Foreground(cDim),
		tunnel.Starting:     lipgloss.NewStyle().Foreground(cYellow),
		tunnel.Running:      lipgloss.NewStyle().Foreground(cGreen),
		tunnel.Reconnecting: lipgloss.NewStyle().Foreground(cYellow),
		tunnel.Failed:       lipgloss.NewStyle().Foreground(cRed),
	}
)

type mode int

const (
	modeList mode = iota
	modeForm
	modeConfirm
	modePicker
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸"}

type tickMsg time.Time
type spinMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func spinTick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg { return spinMsg(t) })
}

// ---------- form ----------

type field struct {
	label   string
	input   textinput.Model
	options []string // jika diisi: field toggle
	hint    string
}

type form struct {
	fields    []field
	focus     int
	editing   int // index tunnel yang diedit, -1 = baru
	err       string
	scanning  bool
	spinFrame int
}

func newField(label, value, hint string) field {
	ti := textinput.New()
	ti.Prompt = ""
	ti.SetValue(value)
	ti.CharLimit = 80
	ti.Width = 34
	return field{label: label, input: ti, hint: hint}
}

func newForm(t config.TunnelCfg, editing int) *form {
	typ := t.Type
	if typ == "" {
		typ = "local"
	}
	auto := "no"
	if t.Autostart {
		auto = "yes"
	}
	port := func(p int) string {
		if p == 0 {
			return ""
		}
		return strconv.Itoa(p)
	}
	toggle := func(label, val string, opts ...string) field {
		f := newField(label, val, "")
		f.options = opts
		return f
	}
	f := &form{editing: editing, fields: []field{
		newField("Nama", t.Name, ""),
		newField("Host", t.Host, "alias di [host.*], alias ~/.ssh/config, atau user@host"),
		toggle("Tipe", typ, "local", "reverse"),
		newField("Bind", t.Bind, "kosong = 127.0.0.1"),
		newField("Port lokal", port(t.LocalPort), "port di Mac"),
		newField("Remote host", t.RemoteHost, "kosong = localhost (sisi server)"),
		newField("Port remote", port(t.RemotePort), "port di server — tekan ctrl+d untuk mendeteksi"),
		toggle("Autostart", auto, "no", "yes"),
	}}
	f.setFocus(0)
	return f
}

func (f *form) setFocus(i int) {
	n := len(f.fields)
	f.focus = (i + n) % n
	for j := range f.fields {
		if j == f.focus && f.fields[j].options == nil {
			f.fields[j].input.Focus()
		} else {
			f.fields[j].input.Blur()
		}
	}
}

func (f *form) val(i int) string { return strings.TrimSpace(f.fields[i].input.Value()) }

func (f *form) toTunnel() (config.TunnelCfg, error) {
	atoi := func(s string) int { n, _ := strconv.Atoi(s); return n }
	t := config.TunnelCfg{
		Name: f.val(0), Host: f.val(1), Type: f.val(2), Bind: f.val(3),
		LocalPort: atoi(f.val(4)), RemoteHost: f.val(5), RemotePort: atoi(f.val(6)),
		Autostart: f.val(7) == "yes",
	}
	return t, t.Validate()
}

func (f *form) toggle(dir int) {
	fl := &f.fields[f.focus]
	cur := 0
	for i, o := range fl.options {
		if o == fl.input.Value() {
			cur = i
		}
	}
	fl.input.SetValue(fl.options[(cur+dir+len(fl.options))%len(fl.options)])
}

// ---------- model ----------

type model struct {
	cfg     config.Config
	path    string
	mgr     *tunnel.Manager
	cursor  int
	mode    mode
	form    *form
	width   int
	height  int
	msg     string
	msgErr  bool
	confirm string
	picker  []detect.Port
	pickCur int

	filterInput textinput.Model
	filtering   bool
	filterQuery string
	showHelp    bool
	logScroll   int
}

// New membuat model TUI.
func New(cfg config.Config, path string, mgr *tunnel.Manager) tea.Model {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "cari nama, host, tipe, port..."
	ti.CharLimit = 50
	return model{
		cfg:         cfg,
		path:        path,
		mgr:         mgr,
		width:       100,
		height:      30,
		filterInput: ti,
	}
}

func (m model) Init() tea.Cmd { return tick() }

func (m *model) flash(s string, isErr bool) { m.msg, m.msgErr = s, isErr }

func (m *model) initFilterInput() {
	if m.filterInput.Prompt == "" {
		ti := textinput.New()
		ti.Prompt = "/ "
		ti.Placeholder = "cari nama, host, tipe, port..."
		ti.CharLimit = 50
		m.filterInput = ti
	}
}

func matchTunnel(t config.TunnelCfg, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(t.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(t.Host), q) {
		return true
	}
	if strings.Contains(strings.ToLower(t.Type), q) {
		return true
	}
	if t.LocalPort > 0 && strings.Contains(strconv.Itoa(t.LocalPort), q) {
		return true
	}
	if t.RemotePort > 0 && strings.Contains(strconv.Itoa(t.RemotePort), q) {
		return true
	}
	return false
}

// visibleTunnels mengembalikan daftar tunnel yang cocok dengan filterQuery.
func (m model) visibleTunnels() []config.TunnelCfg {
	var res []config.TunnelCfg
	for _, idx := range m.visibleIndices() {
		res = append(res, m.cfg.Tunnels[idx])
	}
	return res
}

// visibleIndices mengembalikan indeks tunnel di m.cfg.Tunnels yang cocok dengan filterQuery.
func (m model) visibleIndices() []int {
	var indices []int
	for i, t := range m.cfg.Tunnels {
		if matchTunnel(t, m.filterQuery) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m *model) selectedIndex() int {
	vis := m.visibleIndices()
	if len(vis) == 0 || m.cursor < 0 || m.cursor >= len(vis) {
		return -1
	}
	return vis[m.cursor]
}

func (m *model) selected() (config.TunnelCfg, bool) {
	idx := m.selectedIndex()
	if idx < 0 || idx >= len(m.cfg.Tunnels) {
		return config.TunnelCfg{}, false
	}
	return m.cfg.Tunnels[idx], true
}

func (m model) logHeight() int {
	if m.width >= 90 {
		h := m.height - 18
		if h < 4 {
			h = 4
		}
		return h
	}
	visCount := len(m.visibleIndices())
	h := m.height - visCount - 16
	if m.filtering || m.filterQuery != "" {
		h -= 2
	}
	if h < 4 {
		h = 4
	}
	return h
}

func (m *model) clampLogScroll() {
	if m.mgr != nil {
		if t, ok := m.selected(); ok {
			rt := m.mgr.Get(t.Name)
			if rt != nil {
				snap := rt.Snapshot()
				n := m.logHeight()
				maxScroll := len(snap.Logs) - n
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.logScroll > maxScroll {
					m.logScroll = maxScroll
				}
			}
		}
	}
	if m.logScroll < 0 {
		m.logScroll = 0
	}
}

func (m *model) save() {
	if err := config.Save(m.path, m.cfg); err != nil {
		m.flash("gagal menyimpan config: "+err.Error(), true)
	}
}

func (m *model) toggleTunnel(t config.TunnelCfg) {
	if m.mgr == nil {
		return
	}
	rt := m.mgr.Get(t.Name)
	if rt.Active() {
		go rt.Stop()
		m.flash("menghentikan "+t.Name, false)
		return
	}
	if err := m.mgr.StartTunnel(m.cfg, t); err != nil {
		m.flash(err.Error(), true)
		return
	}
	m.flash("memulai "+t.Name, false)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		if m.mode == modeForm && m.form != nil && m.form.scanning {
			m.form.spinFrame++
		}
		return m, tick()
	case spinMsg:
		if m.mode == modeForm && m.form != nil && m.form.scanning {
			m.form.spinFrame++
			return m, spinTick()
		}
		return m, nil
	case scanMsg:
		f := m.form
		if m.mode != modeForm || f == nil || !f.scanning {
			return m, nil
		}
		f.scanning = false
		if msg.err != nil {
			f.err = "deteksi gagal: " + msg.err.Error()
			return m, nil
		}
		m.picker, m.pickCur = msg.ports, 0
		for i, p := range msg.ports {
			if strconv.Itoa(p.Port) == f.val(6) {
				m.pickCur = i
			}
		}
		m.mode = modePicker
		return m, nil
	case tea.KeyMsg:
		if m.showHelp {
			switch msg.String() {
			case "ctrl+c":
				if m.mgr != nil {
					m.mgr.StopAll()
				}
				return m, tea.Quit
			case "esc", "?", "enter", "q", " ":
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}
		switch m.mode {
		case modeForm:
			return m.updateForm(msg)
		case modePicker:
			return m.updatePicker(msg)
		case modeConfirm:
			if msg.String() == "y" {
				if t, ok := m.selected(); ok {
					idx := m.selectedIndex()
					if m.mgr != nil {
						go m.mgr.StopTunnel(t.Name)
						m.mgr.Forget(t.Name)
					}
					if idx >= 0 && idx < len(m.cfg.Tunnels) {
						m.cfg.Tunnels = append(m.cfg.Tunnels[:idx], m.cfg.Tunnels[idx+1:]...)
					}
					vis := m.visibleIndices()
					if m.cursor >= len(vis) && m.cursor > 0 {
						m.cursor--
					}
					m.save()
					m.flash("tunnel dihapus", false)
				}
			}
			m.mode = modeList
		default:
			if m.filtering {
				return m.updateFilter(msg)
			}
			return m.updateList(msg)
		}
	}
	return m, nil
}

func (m model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		if m.mgr != nil {
			m.mgr.StopAll()
		}
		return m, tea.Quit
	case "esc":
		m.filtering = false
		m.filterInput.Blur()
		return m, nil
	case "enter":
		m.filtering = false
		m.filterInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterQuery = m.filterInput.Value()
	visLen := len(m.visibleIndices())
	if visLen == 0 {
		m.cursor = 0
	} else if m.cursor >= visLen {
		m.cursor = visLen - 1
	}
	m.logScroll = 0
	return m, cmd
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	t, has := m.selected()
	switch msg.String() {
	case "ctrl+c", "q":
		if m.mgr != nil {
			m.mgr.StopAll()
		}
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.logScroll = 0
		}
	case "down", "j":
		visLen := len(m.visibleIndices())
		if visLen > 0 && m.cursor < visLen-1 {
			m.cursor++
			m.logScroll = 0
		}
	case "pgup":
		m.logScroll += 5
		m.clampLogScroll()
	case "pgdown":
		m.logScroll -= 5
		if m.logScroll < 0 {
			m.logScroll = 0
		}
	case "K":
		m.logScroll++
		m.clampLogScroll()
	case "J":
		if m.logScroll > 0 {
			m.logScroll--
		}
	case "/":
		m.filtering = true
		m.initFilterInput()
		m.filterInput.SetValue(m.filterQuery)
		cmd := m.filterInput.Focus()
		return m, cmd
	case "esc":
		if m.filterQuery != "" {
			m.filterQuery = ""
			m.filterInput.SetValue("")
			m.cursor = 0
			m.logScroll = 0
			return m, nil
		}
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "c":
		if has {
			url := fmt.Sprintf("http://localhost:%d", t.LocalPort)
			if err := copyToClipboard(url); err != nil {
				m.flash("gagal menyalin ke clipboard: "+err.Error(), true)
			} else {
				m.flash(fmt.Sprintf("disalin ke clipboard: %s", url), false)
			}
		}
	case "enter", " ":
		if has {
			m.toggleTunnel(t)
		}
	case "r":
		if has && m.mgr != nil {
			rt := m.mgr.Get(t.Name)
			cfg := m.cfg
			go func() {
				rt.Stop()
				m.mgr.StartTunnel(cfg, t)
			}()
			m.flash("restart "+t.Name, false)
		}
	case "a":
		if m.mgr != nil {
			for _, t := range m.cfg.Tunnels {
				if !m.mgr.Get(t.Name).Active() {
					m.mgr.StartTunnel(m.cfg, t)
				}
			}
		}
		m.flash("semua tunnel dimulai", false)
	case "x":
		if m.mgr != nil {
			for _, t := range m.cfg.Tunnels {
				go m.mgr.StopTunnel(t.Name)
			}
		}
		m.flash("semua tunnel dihentikan", false)
	case "o":
		if has && t.Type == "local" {
			openBrowser(fmt.Sprintf("http://localhost:%d", t.LocalPort))
			m.flash(fmt.Sprintf("membuka http://localhost:%d", t.LocalPort), false)
		}
	case "n":
		m.form, m.mode = newForm(config.TunnelCfg{RemoteHost: "localhost"}, -1), modeForm
	case "e":
		if has {
			m.form, m.mode = newForm(t, m.selectedIndex()), modeForm
		}
	case "d":
		if has {
			m.mode = modeConfirm
		}
	}
	return m, nil
}

type scanMsg struct {
	ports []detect.Port
	err   error
}

func scanCmd(cfg config.Config, host string) tea.Cmd {
	return func() tea.Msg {
		ports, err := detect.Scan(context.Background(), cfg, host)
		return scanMsg{ports, err}
	}
}

func (m model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "ctrl+c":
		m.mode = modeForm
	case "up", "k":
		if m.pickCur > 0 {
			m.pickCur--
		}
	case "down", "j":
		if m.pickCur < len(m.picker)-1 {
			m.pickCur++
		}
	case "enter":
		p, f := m.picker[m.pickCur], m.form
		f.fields[6].input.SetValue(strconv.Itoa(p.Port))
		f.fields[5].input.SetValue(p.Target())
		if f.val(4) == "" {
			f.fields[4].input.SetValue(strconv.Itoa(p.Port))
		}
		if f.val(0) == "" && p.Process != "" {
			f.fields[0].input.SetValue(p.Process)
		}
		f.err = ""
		f.setFocus(4)
		m.mode = modeForm
	}
	return m, nil
}

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := m.form
	switch msg.String() {
	case "esc":
		m.mode = modeList
		return m, nil
	case "tab", "down":
		f.setFocus(f.focus + 1)
		return m, nil
	case "shift+tab", "up":
		f.setFocus(f.focus - 1)
		return m, nil
	case "ctrl+d":
		if f.val(1) == "" {
			f.err = "isi Host dulu sebelum mendeteksi port"
			return m, nil
		}
		f.err, f.scanning = "", true
		f.spinFrame = 0
		return m, tea.Batch(scanCmd(m.cfg, f.val(1)), spinTick())
	case "ctrl+s", "enter":
		if msg.String() == "enter" && f.focus != len(f.fields)-1 {
			f.setFocus(f.focus + 1)
			return m, nil
		}
		t, err := f.toTunnel()
		if err == nil {
			for i, o := range m.cfg.Tunnels {
				if o.Name == t.Name && i != f.editing {
					err = fmt.Errorf("nama sudah dipakai")
				}
			}
		}
		if err != nil {
			f.err = err.Error()
			return m, nil
		}
		if f.editing >= 0 {
			old := m.cfg.Tunnels[f.editing]
			go m.mgr.StopTunnel(old.Name)
			m.mgr.Forget(old.Name)
			m.cfg.Tunnels[f.editing] = t
		} else {
			m.cfg.Tunnels = append(m.cfg.Tunnels, t)
			m.filterQuery = ""
			m.filterInput.SetValue("")
			m.cursor = len(m.cfg.Tunnels) - 1
			m.logScroll = 0
		}
		m.save()
		m.flash("tersimpan: "+t.Name, false)
		m.mode = modeList
		return m, nil
	}
	if fl := &f.fields[f.focus]; fl.options != nil {
		switch msg.String() {
		case " ", "right", "l":
			f.toggle(1)
		case "left", "h":
			f.toggle(-1)
		}
		return m, nil
	}
	var cmd tea.Cmd
	f.fields[f.focus].input, cmd = f.fields[f.focus].input.Update(msg)
	return m, cmd
}

// ---------- view ----------

func openBrowser(url string) {
	cmd := "xdg-open"
	if runtime.GOOS == "darwin" {
		cmd = "open"
	}
	_ = exec.Command(cmd, url).Start()
}

func fmtDur(d time.Duration) string {
	d = d.Round(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

func help(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, keyStyle.Render(pairs[i])+" "+dimStyle.Render(pairs[i+1]))
	}
	return strings.Join(parts, dimStyle.Render("  ·  "))
}

func (m model) summaryCounts() (aktif, berhenti, errCount int) {
	for _, t := range m.cfg.Tunnels {
		var st tunnel.Status = tunnel.Stopped
		if m.mgr != nil {
			if rt := m.mgr.Get(t.Name); rt != nil {
				st = rt.Snapshot().Status
			}
		}
		switch st {
		case tunnel.Running:
			aktif++
		case tunnel.Failed, tunnel.Reconnecting:
			errCount++
		case tunnel.Starting:
			aktif++
		default:
			berhenti++
		}
	}
	return
}

func (m model) viewHeader(w int) string {
	aktif, berhenti, errCount := m.summaryCounts()
	title := titleStyle.Render("sshtui")
	subtitle := dimStyle.Render("SSH tunnel manager")
	left := title + " " + subtitle

	pillAktif := okStyle.Render(fmt.Sprintf("● %d Aktif", aktif))
	pillBerhenti := dimStyle.Render(fmt.Sprintf("○ %d Berhenti", berhenti))
	pillErrStyle := dimStyle
	if errCount > 0 {
		pillErrStyle = errStyle
	}
	pillErr := pillErrStyle.Render(fmt.Sprintf("▲ %d Error", errCount))
	pill := fmt.Sprintf("[ %s  ·  %s  ·  %s ]", pillAktif, pillBerhenti, pillErr)

	displayPath := m.path
	if home, err := os.UserHomeDir(); err == nil && home != "" && (displayPath == home || strings.HasPrefix(displayPath, home+string(os.PathSeparator))) {
		displayPath = "~" + strings.TrimPrefix(displayPath, home)
	}

	if w >= 90 {
		maxPath := w - lipgloss.Width(left) - lipgloss.Width(pill) - 4
		if maxPath >= 10 {
			if len([]rune(displayPath)) > maxPath {
				displayPath = truncate(displayPath, maxPath)
			}
			cfgPath := dimStyle.Render(displayPath)
			gap := w - lipgloss.Width(left) - lipgloss.Width(pill) - lipgloss.Width(cfgPath)
			if gap >= 4 {
				gap1 := gap / 2
				gap2 := gap - gap1
				return left + strings.Repeat(" ", gap1) + pill + strings.Repeat(" ", gap2) + cfgPath + "\n\n"
			}
			return left + "  " + pill + "  " + cfgPath + "\n\n"
		}
	}

	if lipgloss.Width(left)+lipgloss.Width(pill)+2 <= w {
		return left + "  " + pill + "\n\n"
	}
	return left + "\n" + pill + "\n\n"
}

func (m model) viewInspector(t config.TunnelCfg, snap tunnel.Snapshot, inner int) string {
	header := lipgloss.NewStyle().Foreground(cAccent).Bold(true).Render(fmt.Sprintf("[ Inspektor Koneksi: %s ]", t.Name))

	hostDest := t.Host
	if h, ok := m.cfg.Hosts[t.Host]; ok {
		port := 22
		if h.Port != 0 {
			port = h.Port
		}
		addr := h.Address
		if addr == "" {
			addr = t.Host
		}
		hostDest = fmt.Sprintf("%s:%d", addr, port)
	} else if !strings.Contains(hostDest, ":") {
		hostDest = fmt.Sprintf("%s:22", hostDest)
	}
	diag := routeDiagram(t, hostDest)
	diag = truncate(diag, inner)

	var latencyBadge string
	if rtt, err := tunnel.ProbeHost(m.cfg, t.Host); err != nil {
		latencyBadge = errStyle.Render("✗ Tidak Terjangkau")
	} else {
		ms := rtt.Milliseconds()
		switch {
		case ms < 50:
			latencyBadge = lipgloss.NewStyle().Foreground(cGreen).Render(fmt.Sprintf("● %dms (Sangat Cepat)", ms))
		case ms <= 150:
			latencyBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D7D7")).Render(fmt.Sprintf("● %dms (Stabil)", ms))
		case ms <= 300:
			latencyBadge = lipgloss.NewStyle().Foreground(cYellow).Render(fmt.Sprintf("▲ %dms (Sedang)", ms))
		default:
			latencyBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8700")).Render(fmt.Sprintf("▲ %dms (Lambat)", ms))
		}
	}

	isListening := tunnel.CheckLocalPort(t.BindAddr(), t.LocalPort)
	var listenerBadge string
	if isListening {
		listenerBadge = okStyle.Render(fmt.Sprintf("● %s:%d listening", t.BindAddr(), t.LocalPort))
	} else {
		switch snap.Status {
		case tunnel.Running:
			listenerBadge = errStyle.Render(fmt.Sprintf("✗ %s:%d belum terbuka", t.BindAddr(), t.LocalPort))
		case tunnel.Starting:
			listenerBadge = lipgloss.NewStyle().Foreground(cYellow).Render(fmt.Sprintf("◌ %s:%d menunggu bind...", t.BindAddr(), t.LocalPort))
		default:
			listenerBadge = dimStyle.Render(fmt.Sprintf("○ %s:%d not listening", t.BindAddr(), t.LocalPort))
		}
	}

	var statusText string
	switch snap.Status {
	case tunnel.Running:
		statusText = okStyle.Render(fmt.Sprintf("● Berjalan (Uptime: %s)", fmtDur(snap.Uptime)))
	case tunnel.Reconnecting:
		retrySec := int(snap.RetryIn.Seconds())
		if retrySec <= 0 {
			retrySec = 1
		}
		statusText = lipgloss.NewStyle().Foreground(cYellow).Render(fmt.Sprintf("▲ Reconnecting (#%d, coba lagi dlm %ds)", snap.RetryCount, retrySec))
		if snap.LastError != "" {
			statusText += " " + errStyle.Render(fmt.Sprintf("[%s]", truncate(snap.LastError, 25)))
		}
	case tunnel.Failed:
		statusText = errStyle.Render("✗ Gagal")
		if snap.LastError != "" {
			statusText += " " + errStyle.Render(fmt.Sprintf("[%s]", truncate(snap.LastError, 30)))
		}
	case tunnel.Starting:
		statusText = lipgloss.NewStyle().Foreground(cYellow).Render("◌ Memulai...")
	default:
		statusText = dimStyle.Render("○ Berhenti")
	}

	user := "-"
	key := "-"
	hostAddr := t.Host
	if h, ok := m.cfg.Hosts[t.Host]; ok {
		if h.User != "" {
			user = h.User
		}
		if h.Key != "" {
			key = h.Key
		}
		if h.Address != "" {
			hostAddr = h.Address
		}
	} else if strings.Contains(t.Host, "@") {
		parts := strings.SplitN(t.Host, "@", 2)
		user = parts[0]
		hostAddr = parts[1]
	}
	credText := fmt.Sprintf("user: %s  |  host: %s  |  key: %s", user, hostAddr, key)

	inspLabel := lipgloss.NewStyle().Foreground(cDim).Width(18)
	var rows []string
	rows = append(rows, header, "")
	rows = append(rows, dimStyle.Render("Rute Koneksi:"))
	rows = append(rows, "  "+diag, "")
	rows = append(rows, dimStyle.Render("Telemetri & Kesehatan:"))
	rows = append(rows, inspLabel.Render("  SSH Latency")+": "+latencyBadge)
	rows = append(rows, inspLabel.Render("  Local Listener")+": "+listenerBadge)
	rows = append(rows, inspLabel.Render("  Siklus Hidup")+": "+statusText)
	rows = append(rows, inspLabel.Render("  Kredensial")+": "+dimStyle.Render(credText))

	return boxStyle.Width(inner).Render(strings.Join(rows, "\n"))
}

func viewInspector(t config.TunnelCfg, snap tunnel.Snapshot, inner int) string {
	return model{}.viewInspector(t, snap, inner)
}

func (m model) viewLogPane(t config.TunnelCfg, logs []string, inner, height int) string {
	title := fmt.Sprintf("Log — %s", t.Name)
	if m.logScroll > 0 {
		title += fmt.Sprintf(" [▲ %d]", m.logScroll)
	}
	header := lipgloss.NewStyle().Foreground(cAccent).Bold(true).Render(title) +
		" " + dimStyle.Render(fmt.Sprintf("(%d baris) · PgUp/PgDn / J/K scroll", len(logs)))

	if height <= 0 {
		height = 6
	}

	var logLines []string
	if len(logs) == 0 {
		logLines = append(logLines, dimStyle.Render("(belum ada log)"))
	} else {
		totalLogs := len(logs)
		if totalLogs > height {
			maxScroll := totalLogs - height
			scroll := m.logScroll
			if scroll > maxScroll {
				scroll = maxScroll
			}
			if scroll < 0 {
				scroll = 0
			}
			end := totalLogs - scroll
			start := end - height
			if start < 0 {
				start = 0
			}
			logs = logs[start:end]
		}
		for _, l := range logs {
			logLines = append(logLines, dimStyle.Render(truncate(l, inner)))
		}
	}
	var rows []string
	rows = append(rows, header)
	rows = append(rows, logLines...)
	return boxStyle.Width(inner).Render(strings.Join(rows, "\n"))
}

func viewLogPane(t config.TunnelCfg, logs []string, inner, height int) string {
	return model{}.viewLogPane(t, logs, inner, height)
}

func (m model) viewTunnelTable(inner int) string {
	contentWidth := inner - 2
	if contentWidth < 30 {
		contentWidth = 30
	}

	var rows []string
	title := lipgloss.NewStyle().Foreground(cAccent).Bold(true).Render("[ DAFTAR TUNNEL ]")
	if m.filtering {
		rows = append(rows, title+" "+m.filterInput.View())
	} else if m.filterQuery != "" {
		filterHdr := dimStyle.Render("Filter: ") + keyStyle.Render(m.filterQuery) + dimStyle.Render(" (esc reset)")
		rows = append(rows, title+" "+filterHdr)
	} else {
		rows = append(rows, title+" "+dimStyle.Render("(/ cari)"))
	}

	nameWidth := contentWidth - 21
	if nameWidth < 10 {
		nameWidth = 10
	}
	hdr := dimStyle.Render(fmt.Sprintf("  ST  %-7s %-*s %-6s", "TIPE", nameWidth, "NAMA", "PORT"))
	rows = append(rows, hdr)
	rows = append(rows, dimStyle.Render(strings.Repeat("─", contentWidth)))

	vis := m.visibleIndices()
	if len(m.cfg.Tunnels) == 0 {
		rows = append(rows, dimStyle.Render("Belum ada tunnel. Tekan n untuk menambah."))
	} else if len(vis) == 0 {
		rows = append(rows, dimStyle.Render(fmt.Sprintf("Tidak ada tunnel cocok %q. Tekan esc untuk reset.", m.filterQuery)))
	} else {
		for displayIdx, origIdx := range vis {
			t := m.cfg.Tunnels[origIdx]
			var snap tunnel.Snapshot
			if m.mgr != nil {
				if rt := m.mgr.Get(t.Name); rt != nil {
					snap = rt.Snapshot()
				}
			}
			st := snap.Status
			dot := statusStyle[st].Render("●")
			if st == tunnel.Stopped {
				dot = dimStyle.Render("○")
			}
			marker := " "
			if displayIdx == m.cursor {
				marker = "►"
			}
			typBadge := "[LOCAL]"
			if strings.ToLower(t.Type) == "reverse" {
				typBadge = "[REV]  "
			}
			portStr := fmt.Sprintf(":%d", t.LocalPort)
			auto := " "
			if t.Autostart {
				auto = "↻"
			}
			nameStr := truncate(t.Name, nameWidth)
			line := fmt.Sprintf("%s %s %s %-*s %-6s %s", marker, dot, typBadge, nameWidth, nameStr, portStr, dimStyle.Render(auto))
			if displayIdx == m.cursor {
				line = selStyle.Width(contentWidth).Render(fmt.Sprintf("%s %s %s %-*s %-6s %s", marker, dot, typBadge, nameWidth, nameStr, portStr, auto))
			}
			rows = append(rows, line)
		}
	}
	return boxStyle.Width(inner).Render(strings.Join(rows, "\n"))
}

func (m model) viewSplit(w, inner int) string {
	wLeft := (w * 45) / 100
	innerLeft := wLeft - 4
	if innerLeft < 34 {
		innerLeft = 34
		wLeft = innerLeft + 4
	}
	wRight := w - wLeft - 1
	innerRight := wRight - 4
	if innerRight < 40 {
		innerRight = 40
	}

	leftBox := m.viewTunnelTable(innerLeft)

	var rightCol string
	if t, ok := m.selected(); ok {
		var snap tunnel.Snapshot
		var logs []string
		if m.mgr != nil {
			if rt := m.mgr.Get(t.Name); rt != nil {
				snap = rt.Snapshot()
				logs = snap.Logs
			}
		}
		insp := m.viewInspector(t, snap, innerRight)
		logPane := m.viewLogPane(t, logs, innerRight, m.logHeight())
		rightCol = lipgloss.JoinVertical(lipgloss.Left, insp, logPane)
	} else {
		placeholder := boxStyle.Width(innerRight).Render(dimStyle.Render("Belum ada tunnel yang dipilih."))
		emptyLog := boxStyle.Width(innerRight).Render(dimStyle.Render("(belum ada log)"))
		rightCol = lipgloss.JoinVertical(lipgloss.Left, placeholder, emptyLog)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightCol)
}

func viewSplit(w, inner int) string {
	return model{width: w}.viewSplit(w, inner)
}

func (m model) viewStacked(w, inner int) string {
	var parts []string
	tableBox := m.viewTunnelTable(inner)
	parts = append(parts, tableBox)

	if t, ok := m.selected(); ok {
		var snap tunnel.Snapshot
		var logs []string
		if m.mgr != nil {
			if rt := m.mgr.Get(t.Name); rt != nil {
				snap = rt.Snapshot()
				logs = snap.Logs
			}
		}
		insp := m.viewInspector(t, snap, inner)
		parts = append(parts, insp)
		logPane := m.viewLogPane(t, logs, inner, m.logHeight())
		parts = append(parts, logPane)
	}
	return strings.Join(parts, "\n")
}

func (m model) View() string {
	if m.showHelp {
		return viewHelpModal(m.width, m.height)
	}

	w := m.width
	if w < 60 {
		w = 60
	}
	inner := w - 4

	var b strings.Builder
	b.WriteString(m.viewHeader(w))

	if m.mode == modePicker {
		b.WriteString(m.viewPicker(inner))
		return b.String()
	}
	if m.mode == modeForm {
		b.WriteString(m.viewForm(inner))
		return b.String()
	}

	if m.width >= 90 {
		b.WriteString(m.viewSplit(w, inner))
	} else {
		b.WriteString(m.viewStacked(w, inner))
	}

	// footer
	switch {
	case m.mode == modeConfirm:
		t, _ := m.selected()
		b.WriteString("\n" + errStyle.Render(fmt.Sprintf("Hapus tunnel %q? ", t.Name)) + help("y", "ya", "lain", "batal"))
	case m.filtering:
		b.WriteString("\n" + help("enter", "selesai filter", "esc", "batal filter", "ctrl+c", "keluar"))
	default:
		if m.msg != "" {
			st := okStyle
			if m.msgErr {
				st = errStyle
			}
			b.WriteString("\n" + st.Render(m.msg))
		}
		b.WriteString("\n" + help("↑↓", "pilih", "/", "filter", "enter", "start/stop", "c", "salin", "?", "bantuan", "n/e/d", "kelola", "q", "keluar"))
	}
	return b.String()
}

func (m model) viewForm(inner int) string {
	f := m.form
	title := "Tunnel baru"
	if f.editing >= 0 {
		title = "Edit tunnel"
	}
	var rows []string
	rows = append(rows, lipgloss.NewStyle().Foreground(cAccent).Bold(true).Render(title), "")
	for i, fl := range f.fields {
		ls := labelStyle
		if i == f.focus {
			ls = focusLabel
		}
		var v string
		if fl.options != nil {
			var opts []string
			for _, o := range fl.options {
				if o == fl.input.Value() {
					opts = append(opts, selStyle.Padding(0, 1).Render(o))
				} else {
					opts = append(opts, dimStyle.Padding(0, 1).Render(o))
				}
			}
			v = strings.Join(opts, " ")
		} else {
			v = fl.input.View()
			if fl.input.Value() == "" && i != f.focus {
				v = dimStyle.Render(fl.hint)
			}
		}
		rows = append(rows, ls.Render(fl.label)+v)
	}
	rows = append(rows, "")
	if f.fields[f.focus].hint != "" {
		rows = append(rows, dimStyle.Render("ℹ "+f.fields[f.focus].hint))
	}
	if f.scanning {
		spin := spinnerFrames[f.spinFrame%len(spinnerFrames)]
		rows = append(rows, okStyle.Render(fmt.Sprintf("%s mendeteksi port di %s …", spin, f.val(1))))
	}
	if f.err != "" {
		rows = append(rows, errStyle.Render("✗ "+f.err))
	}
	box := boxStyle.Width(inner).Render(strings.Join(rows, "\n"))
	return box + "\n" + help("tab", "next", "space/←→", "pilihan", "ctrl+d", "deteksi port", "ctrl+s", "simpan", "esc", "batal")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if n < 1 || len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func (m model) viewPicker(inner int) string {
	host := m.form.val(1)
	rows := []string{
		lipgloss.NewStyle().Foreground(cAccent).Bold(true).Render("Port terdeteksi di " + host),
		dimStyle.Render(fmt.Sprintf("%-7s %-16s %-22s %s", "PORT", "BIND", "PROSES", "")),
	}
	for i, p := range m.picker {
		note := ""
		if p.Loopback() {
			note = "loopback"
		}
		for _, t := range m.cfg.Tunnels {
			if t.Host == host && t.Type == "local" && t.RemotePort == p.Port {
				note = "✓ sudah ada: " + t.Name
			}
		}
		proc := p.Process
		if proc == "" {
			proc = "-"
		}
		line := fmt.Sprintf("%-7d %-16s %-22s %s", p.Port, truncate(p.Addr, 15), truncate(proc, 21), note)
		if i == m.pickCur {
			line = selStyle.Render(line)
		}
		rows = append(rows, line)
	}
	// batasi tinggi: tampilkan jendela di sekitar kursor
	max := m.height - 10
	if max < 6 {
		max = 6
	}
	if body := rows[2:]; len(body) > max {
		start := m.pickCur - max/2
		if start < 0 {
			start = 0
		}
		if start+max > len(body) {
			start = len(body) - max
		}
		rows = append(rows[:2], body[start:start+max]...)
	}
	box := boxStyle.Width(inner).Render(strings.Join(rows, "\n"))
	return box + "\n" + help("↑↓", "pilih", "enter", "pakai", "esc", "kembali")
}

// Run menjalankan TUI sampai pengguna keluar.
func Run(cfg config.Config, path string, mgr *tunnel.Manager) error {
	_, err := tea.NewProgram(New(cfg, path, mgr), tea.WithAltScreen()).Run()
	return err
}
