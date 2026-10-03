package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"sshtui/internal/config"
	"sshtui/internal/detect"
	"sshtui/internal/tui"
	"sshtui/internal/tunnel"
)

const usage = `sshtui — SSH tunnel manager

Pemakaian:
  sshtui                 buka TUI
  sshtui list            tampilkan tunnel di config
  sshtui scan <host>     deteksi port yang listen di server (alias [host.*], alias ssh, atau user@host)
  sshtui up <nama>...    jalankan tunnel di foreground (Ctrl+C untuk berhenti)
  sshtui up --all        jalankan semua tunnel
  sshtui path            tampilkan lokasi config

Config: ~/.config/sshtui/config.toml (override dengan env SSHTUI_CONFIG)
`

func main() {
	path := config.Path()
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) == 0 {
		runTUI(cfg, path)
		return
	}
	switch args[0] {
	case "list":
		for _, t := range cfg.Tunnels {
			auto := ""
			if t.Autostart {
				auto = " [autostart]"
			}
			fmt.Printf("%-20s %-8s localhost:%d <-> %s:%s:%d%s\n", t.Name, t.Type, t.LocalPort, t.Host, t.RemoteAddr(), t.RemotePort, auto)
		}
	case "path":
		fmt.Println(path)
	case "scan":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "pemakaian: sshtui scan <host>")
			os.Exit(1)
		}
		runScan(cfg, args[1])
	case "up":
		runUp(cfg, args[1:])
	default:
		fmt.Print(usage)
	}
}

func runTUI(cfg config.Config, path string) {
	mgr := tunnel.NewManager()
	for _, t := range cfg.Tunnels {
		if t.Autostart {
			mgr.StartTunnel(cfg, t)
		}
	}
	if err := tui.Run(cfg, path, mgr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	mgr.StopAll()
}

func runScan(cfg config.Config, host string) {
	ports, err := detect.Scan(context.Background(), cfg, host)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan gagal:", err)
		os.Exit(1)
	}
	fmt.Printf("%-7s %-16s %s\n", "PORT", "BIND", "PROSES")
	for _, p := range ports {
		proc := p.Process
		if proc == "" {
			proc = "-"
		}
		fmt.Printf("%-7d %-16s %s\n", p.Port, p.Addr, proc)
	}
}

func runUp(cfg config.Config, names []string) {
	var sel []config.TunnelCfg
	for _, t := range cfg.Tunnels {
		for _, n := range names {
			if n == "--all" || strings.EqualFold(n, t.Name) {
				sel = append(sel, t)
				break
			}
		}
	}
	if len(sel) == 0 {
		fmt.Fprintln(os.Stderr, "tunnel tidak ditemukan; lihat `sshtui list`")
		os.Exit(1)
	}
	mgr := tunnel.NewManager()
	for _, t := range sel {
		name := t.Name
		mgr.Get(name).Echo = func(l string) { fmt.Printf("[%s] %s\n", name, l) }
		if err := mgr.StartTunnel(cfg, t); err != nil {
			fmt.Fprintln(os.Stderr, name+":", err)
			os.Exit(1)
		}
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	mgr.StopAll()
}
