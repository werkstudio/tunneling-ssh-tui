# sshtui

TUI modern dan intuitif untuk mengelola SSH tunnel (Go + Bubble Tea).

## Fitur Utama

- **Responsive Split-Pane Layout**: Tampilan dua panel berdampingan otomatis pada terminal lebar (≥ 110 kolom) dan tata letak vertikal adaptif pada terminal kompak.
- **Connection Inspector**:
  - Visual diagram alur rute koneksi (`Local <-> SSH Host <-> Remote Target`).
  - Pemantauan latensi RTT host secara berkala via native Go TCP probe.
  - Verifikasi status listening port lokal secara real-time.
  - Informasi proses SSH (PID, uptime, jumlah reconnect, waktu start).
- **Scrollable Live Logs**: Viewport log koneksi SSH real-time yang dapat di-scroll dengan indikator persentase.
- **Pencarian & Filter Cepat**: Filter daftar tunnel secara instan hanya dengan menekan `/`.
- **Integrasi Clipboard**: Salin URL lokal (`http://localhost:<port>`) ke clipboard dengan satu tombol (`c`).
- **Deteksi Port Remote**: Deteksi port yang sedang listening di remote host (`ctrl+d`) lengkap dengan animasi spinner.
- **Help Modal**: Akses cepat ke panduan tombol kapan saja dengan menekan `?`.

## Build & Jalankan

```bash
make build && ./bin/sshtui
make install    # salin ke ~/.local/bin
make test
```

## Struktur Proyek

```
cmd/sshtui/        entrypoint + CLI (list, scan, up, path)
internal/config/   load/save TOML, penyusun argumen ssh
internal/tunnel/   manager proses ssh -N + telemetri & health engine
internal/detect/   deteksi port listen di server (ss / netstat / lsof)
internal/tui/      antarmuka Bubble Tea responsif & modal komponen
```

## Konfigurasi

Lokasi konfigurasi default: `~/.config/sshtui/config.toml` (dapat di-override menggunakan environment variable `SSHTUI_CONFIG`).

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

## Tombol Pintasan (Keybindings)

### Navigasi & Kontrol Utama

| Tombol | Fungsi |
|---|---|
| `↑` `↓` / `j` `k` | Pilih tunnel dalam daftar |
| `enter` / `space` | Start / Stop tunnel terpilih |
| `r` | Restart proses tunnel terpilih |
| `a` / `x` | Start semua / Stop semua tunnel |
| `/` | Buka filter / pencarian instan nama & host (`esc` untuk batal/reset) |
| `c` | Salin URL tunnel (`http://localhost:<port>`) ke clipboard sistem |
| `o` | Buka `http://localhost:<port>` di browser default |
| `?` | Tampilkan / sembunyikan modal bantuan pintasan |
| `PgUp` / `PgDn` (atau `J` / `K`) | Scroll viewport log inspeksi koneksi ke atas / ke bawah |
| `n` / `e` / `d` | Tambah / Edit / Hapus tunnel |
| `q` | Keluar dari aplikasi (semua tunnel otomatis dimatikan) |

### Formulir & Deteksi Port

| Tombol | Fungsi |
|---|---|
| `tab` / `shift+tab` | Pindah ke field berikutnya / sebelumnya |
| `space` / `←` `→` | Ubah pilihan opsi toggle (tipe tunnel / autostart) |
| `ctrl+d` | **Deteksi port remote** di server via SSH (disertai animasi spinner) |
| `ctrl+s` | Simpan perubahan konfigurasi tunnel |
| `esc` | Batal / tutup formulir |

## Split-Pane Layout & Inspeksi Koneksi

sshtui mengadopsi antarmuka modern yang secara cerdas menyesuaikan dengan dimensi terminal Anda:

1. **Panel Kiri (Tunnel List)**:
   - Menampilkan status setiap tunnel:
     - `● RUNNING` (hijau)
     - `○ STOPPED` (abu-abu)
     - `▲ ERROR` (merah)
     - `⏳ STARTING` (kuning)
   - Indikator latensi ping RTT ke host server.
   - Indikator autostart, binding port lokal, dan target remote.

2. **Panel Kanan (Connection Inspector & Logs)**:
   - **Diagram Alur Rute**: Menunjukkan visual alur koneksi:
     - Local (-L): `127.0.0.1:8891 ──────► [homelab:22] ──────► localhost:20128`
     - Reverse (-R): `127.0.0.1:20128 ◄────── [homelab:22] ◄────── 127.0.0.1:8891`
   - **Local Listener Status**: Memverifikasi apakah port lokal aktif mendengar koneksi TCP.
   - **Host Latency Check**: Melakukan ping RTT ke SSH server secara berkala.
   - **Scrollable Live Logs**: Log realtime proses `ssh` dengan navigasi scrolling keyboard `PgUp`/`PgDn` atau `J`/`K` serta indikator posisi scroll `[xx%]`.

3. **Responsif**: Pada terminal dengan lebar < 110 kolom, antarmuka otomatis beralih ke layout tumpuk (stacked vertical) untuk menjaga kenyamanan pembacaan.

## Deteksi Port Remote

Dari formulir penambahan/pengeditan tunnel:
1. Isi kolom **Host** (alias, nama host dari `~/.ssh/config`, atau `user@host`).
2. Tekan `ctrl+d`: sshtui akan menjalankan pemindaian otomatis di server remote melalui SSH (`ss -ltnp`, fallback `netstat`, `lsof`) dengan indikator animasi spinner.
3. Daftar port yang sedang listening akan muncul dalam dialog interaktif.
4. Pilih salah satu port untuk otomatis melengkapi Port Remote, Remote Host, Port Lokal, dan Nama tunnel.

Pemindaian port juga dapat dijalankan langsung via CLI tanpa TUI:
```bash
sshtui scan homelab
```

## Catatan Teknis

- Perintah `ssh` dieksekusi dengan opsi `BatchMode=yes`; autentikasi remote harus menggunakan SSH key atau SSH agent.
- Pengaturan bind default adalah `127.0.0.1`.
- Clipboard didukung di macOS (`pbcopy`), Linux (`wl-copy`, `xclip`, `xsel`), dan Windows (`clip.exe`).
