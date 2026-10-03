package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type HostCfg struct {
	Address string `toml:"address"`
	User    string `toml:"user,omitempty"`
	Port    int    `toml:"port,omitempty"`
	Key     string `toml:"key,omitempty"`
}

type TunnelCfg struct {
	Name       string `toml:"name"`
	Host       string `toml:"host"`
	Type       string `toml:"type"` // "local" (-L) atau "reverse" (-R)
	Bind       string `toml:"bind,omitempty"`
	LocalPort  int    `toml:"local_port"`
	RemoteHost string `toml:"remote_host,omitempty"`
	RemotePort int    `toml:"remote_port"`
	Autostart  bool   `toml:"autostart"`
}

type Config struct {
	Hosts   map[string]HostCfg `toml:"host"`
	Tunnels []TunnelCfg        `toml:"tunnel"`
}

const configHeader = `# sshtui config
#
# [host.<alias>]  address, user, port, key (opsional)
#   Nilai "host" pada tunnel yang tidak ada di sini akan diteruskan apa adanya
#   ke ssh, jadi alias dari ~/.ssh/config atau "user@host" juga bisa dipakai.
#
# [[tunnel]]
#   type = "local"   : localhost:local_port (Mac) -> remote_host:remote_port (lewat server)
#   type = "reverse" : server:remote_port -> localhost:local_port (Mac)
#   bind             : default 127.0.0.1 (local) / 127.0.0.1 di server (reverse)

`

func Path() string {
	if p := os.Getenv("SSHTUI_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "sshtui", "config.toml")
}

func Default() Config {
	return Config{
		Hosts: map[string]HostCfg{
			"homelab": {Address: "homelab.arry.my.id", User: "arry", Port: 22},
		},
		Tunnels: []TunnelCfg{{
			Name: "DMS Mazda", Host: "homelab", Type: "local",
			LocalPort: 8891, RemoteHost: "localhost", RemotePort: 20128,
		}},
	}
}

func Load(path string) (Config, error) {
	var c Config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		c = Default()
		return c, Save(path, c)
	}
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if c.Hosts == nil {
		c.Hosts = map[string]HostCfg{}
	}
	return c, nil
}

func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(configHeader)
	if err := toml.NewEncoder(&buf).Encode(c); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (t TunnelCfg) BindAddr() string {
	if t.Bind == "" {
		return "127.0.0.1"
	}
	return t.Bind
}

func (t TunnelCfg) RemoteAddr() string {
	if t.RemoteHost == "" {
		return "localhost"
	}
	return t.RemoteHost
}

func (t TunnelCfg) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("nama tidak boleh kosong")
	}
	if t.Host == "" {
		return fmt.Errorf("host tidak boleh kosong")
	}
	if t.Type != "local" && t.Type != "reverse" {
		return fmt.Errorf("type harus local atau reverse")
	}
	if t.LocalPort < 1 || t.LocalPort > 65535 || t.RemotePort < 1 || t.RemotePort > 65535 {
		return fmt.Errorf("port harus 1-65535")
	}
	return nil
}

// Connect mengembalikan opsi koneksi ssh (-p, -i) dan tujuan (user@host) untuk
// sebuah host. Host yang tidak ada di [host.*] diteruskan apa adanya ke ssh.
func Connect(c Config, host string) (opts []string, dest string) {
	dest = host
	h, ok := c.Hosts[host]
	if !ok {
		return nil, dest
	}
	dest = h.Address
	if h.User != "" {
		dest = h.User + "@" + h.Address
	}
	if h.Port != 0 {
		opts = append(opts, "-p", fmt.Sprint(h.Port))
	}
	if h.Key != "" {
		key := h.Key
		if strings.HasPrefix(key, "~/") {
			home, _ := os.UserHomeDir()
			key = filepath.Join(home, key[2:])
		}
		opts = append(opts, "-i", key)
	}
	return opts, dest
}

// SSHArgs menyusun argumen ssh untuk sebuah tunnel.
func SSHArgs(c Config, t TunnelCfg) ([]string, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	args := []string{
		"-N",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-o", "ExitOnForwardFailure=yes",
	}
	opts, dest := Connect(c, t.Host)
	args = append(args, opts...)
	switch t.Type {
	case "local":
		args = append(args, "-L", fmt.Sprintf("%s:%d:%s:%d", t.BindAddr(), t.LocalPort, t.RemoteAddr(), t.RemotePort))
	case "reverse":
		args = append(args, "-R", fmt.Sprintf("%s:%d:localhost:%d", t.BindAddr(), t.RemotePort, t.LocalPort))
	}
	return append(args, dest), nil
}
