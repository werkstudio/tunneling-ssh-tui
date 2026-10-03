// Package detect mendeteksi port yang sedang listen di server remote lewat ssh.
package detect

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"sshtui/internal/config"
)

// Port adalah satu port yang sedang listen di server.
type Port struct {
	Port    int
	Addr    string // alamat bind di server, mis. 0.0.0.0, 127.0.0.1, ::
	Process string // kosong jika tidak terlihat (butuh root untuk proses milik user lain)
}

// Loopback true jika port hanya bisa diakses dari server itu sendiri.
func (p Port) Loopback() bool {
	return p.Addr == "127.0.0.1" || p.Addr == "::1" || strings.HasPrefix(p.Addr, "127.")
}

// Target mengembalikan host tujuan forward: "localhost" untuk port yang bind
// ke semua interface/loopback, atau alamat spesifiknya.
func (p Port) Target() string {
	switch p.Addr {
	case "", "*", "0.0.0.0", "::", "[::]":
		return "localhost"
	}
	if p.Loopback() {
		return "localhost"
	}
	return p.Addr
}

// Perintah di server: coba ss, lalu netstat, lalu lsof.
const remoteCmd = `ss -H -ltnp 2>/dev/null || netstat -ltnp 2>/dev/null || netstat -ltn 2>/dev/null || lsof -nP -iTCP -sTCP:LISTEN 2>/dev/null`

// Scan menjalankan perintah deteksi di host dan mengembalikan daftar port listen.
func Scan(ctx context.Context, c config.Config, host string) ([]Port, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	opts, dest := config.Connect(c, host)
	args := []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10"}
	args = append(args, opts...)
	args = append(args, dest, remoteCmd)

	var out, errb bytes.Buffer
	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil && out.Len() == 0 {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		if ctx.Err() == context.DeadlineExceeded {
			msg = "timeout"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	ports := Parse(out.String())
	if len(ports) == 0 {
		return nil, fmt.Errorf("tidak ada port listen terdeteksi (ss/netstat/lsof tidak tersedia?)")
	}
	return ports, nil
}

var procRe = regexp.MustCompile(`\(\("([^"]+)"`)

// Parse membaca output ss, netstat, atau lsof dan mengembalikan port unik terurut.
func Parse(output string) []Port {
	byPort := map[int]Port{}
	for _, line := range strings.Split(output, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		var addr, proc string
		switch {
		case strings.Contains(line, "(LISTEN)"): // lsof: COMMAND PID USER FD TYPE DEVICE SIZE NODE NAME (LISTEN)
			if len(f) < 9 {
				continue
			}
			addr, proc = f[len(f)-2], f[0]
		case f[0] == "LISTEN": // ss
			addr = f[3]
			if m := procRe.FindStringSubmatch(line); m != nil {
				proc = m[1]
			}
		case strings.HasPrefix(f[0], "tcp") && len(f) >= 6 && f[5] == "LISTEN": // netstat
			addr = f[3]
			if len(f) >= 7 && strings.Contains(f[6], "/") {
				proc = f[6][strings.Index(f[6], "/")+1:]
			}
		default:
			continue
		}
		i := strings.LastIndex(addr, ":")
		if i < 0 {
			continue
		}
		port, err := strconv.Atoi(addr[i+1:])
		if err != nil || port < 1 || port > 65535 {
			continue
		}
		host := strings.Trim(addr[:i], "[]")
		if host == "" {
			host = "*"
		}
		if old, ok := byPort[port]; ok {
			// satu port bisa muncul untuk IPv4 dan IPv6: gabungkan, pertahankan nama proses
			if old.Process != "" {
				proc = old.Process
			}
			if old.Addr != "127.0.0.1" && old.Addr != "::1" {
				host = old.Addr
			}
		}
		byPort[port] = Port{Port: port, Addr: host, Process: proc}
	}
	ports := make([]Port, 0, len(byPort))
	for _, p := range byPort {
		ports = append(ports, p)
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i].Port < ports[j].Port })
	return ports
}
