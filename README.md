# sshtui

TUI modern untuk mengelola SSH tunnel (Go + Bubble Tea).

## Build & jalankan

```bash
make build && ./bin/sshtui
make install    # salin ke ~/.local/bin
make test
```

## Struktur

```
cmd/sshtui/        entrypoint + CLI (list, scan, up, path)
internal/config/   load/save TOML, penyusun argumen ssh
internal/tunnel/   manager proses ssh -N + auto-reconnect
internal/detect/   deteksi port listen di server (ss / netstat / lsof)
internal/tui/      antarmuka Bubble Tea
```

## Config

`~/.config/sshtui/config.toml` (override dengan env `SSHTUI_CONFIG`).

```toml
[host.homelab]
address = "homelab.arry.my.id"
user = "arry"
port = 22
key = "~/.ssh/id_ed25519"   # opsional

[[tunnel]]
name = "DMS Mazda"
host = "homelab"        # alias [host.*], alias ~/.ssh/config, atau user@host
type = "local"          # local (-L) | reverse (-R)
local_port = 8891
remote_host = "localhost"
remote_port = 20128
autostart = false
```

## Tombol

| Tombol | Fungsi |
|---|---|
| ↑↓ / jk | pilih tunnel |
| enter / space | start / stop |
| r | restart |
| a / x | start semua / stop semua |
| o | buka `http://localhost:<port>` |
| n / e / d | tambah / edit / hapus |
| q | keluar (tunnel ikut berhenti) |

Di form: `tab` pindah field, `space`/`←→` ubah pilihan, **`ctrl+d` deteksi port remote**, `ctrl+s` simpan, `esc` batal.

## Deteksi port remote

Dari form, isi **Host** lalu tekan `ctrl+d`: sshtui menjalankan `ss -ltnp` (fallback `netstat`, `lsof`) di server lewat ssh dan menampilkan daftar port. Pilih satu untuk mengisi Port remote, Remote host, dan (jika kosong) Port lokal serta Nama.

Tanpa TUI: `sshtui scan homelab`. Nama proses hanya terlihat untuk proses milik user ssh (atau jika root).

## Catatan

- ssh dijalankan dengan `BatchMode=yes`; autentikasi harus lewat key/agent.
- Bind default `127.0.0.1`.
