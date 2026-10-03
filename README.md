# sshtui 🚀

> **TUI modern, informatif, dan intuitif untuk mengelola SSH Port Forwarding (Go + Bubble Tea).**

`sshtui` menyederhanakan manajemen SSH tunnel (Local Forward `-L` dan Reverse Forward `-R`) dengan antarmuka terminal bergaya split-pane yang kaya telemetri koneksi, pemantauan latensi real-time, log live yang dapat di-scroll, dan pendeteksian port otomatis di server remote.

---

## ✨ Fitur Unggulan

- 🖥️ **Responsive Split-Pane Interface**:
  - Otomatis menampilkan tampilan dua kolom berdampingan pada terminal lebar (≥ 110 kolom): kolom kiri untuk daftar tunnel & filter, kolom kanan untuk inspektor koneksi & live log.
  - Beralih adaptif ke mode tumpuk vertikal (*graceful stacked fallback*) pada terminal yang lebih sempit (< 110 kolom).
- 🔍 **Real-Time Connection Inspector**:
  - **Diagram Alur Rute**: Menampilkan visual arah lalu lintas (`Local Port ──────► [SSH Server] ──────► Remote Target`).
  - **Host Ping RTT Latency**: Memeriksa latensi TCP ke remote SSH server secara berkala dengan pewarnaan intuitif (<50ms hijau, 50-150ms cyan, 150-300ms kuning, >300ms oranye).
  - **Local Listener Verification**: Memverifikasi secara riil apakah socket port lokal di komputer Anda sudah siap menerima koneksi aplikasi.
  - **Proses & Lifecycle**: Melacak status proses (`RUNNING`, `STOPPED`, `RECONNECTING`, `ERROR`), uptime, countdown reconnect otomatis, dan detail error spesifik.
- 📜 **Scrollable Live Logs**:
  - Panel log stderr SSH langsung dengan viewport tersendiri.
  - Navigasi scrolling keyboard (`PgUp`/`PgDn` atau `J`/`K`) dilengkapi indikator posisi persentase `[xx%]`.
- ⚡ **Pencarian & Filter Instan (`/`)**:
  - Filter cepat daftar tunnel berdasarkan nama atau host secara realtime tanpa jeda.
- 📋 **Integrasi Clipboard Sistem (`c`)**:
  - Salin URL lokal (`http://localhost:<port>`) langsung ke clipboard dengan satu tombol (mendukung macOS `pbcopy`, Linux `wl-copy`/`xclip`/`xsel`, dan Windows `clip.exe`).
- 📡 **Deteksi Port Remote Otomatis (`ctrl+d`)**:
  - Pindai port yang sedang listening di remote host lewat SSH (`ss -ltnp`, `netstat`, `lsof`) disertai animasi spinner braille halus (`⠋ ⠙ ⠹ ⠸`).
- ❓ **Help Modal Popup (`?`)**:
  - Cheatsheet tombol pintasan interaktif yang bisa dimunculkan dan ditutup kapan saja.

---

## 📸 Tampilan Visual

### 1. Split-Pane Layout (Lebar Terminal ≥ 110 Kolom)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ sshtui SSH tunnel manager         [ ● 2 Aktif  ·  ○ 1 Berhenti  ·  ▲ 0 Error ]     ~/.config/... │
├───────────────────────────────────┬──────────────────────────────────────────────────────────────┤
│ DAFTAR TUNNEL (/ filter)          │ INSPEKTOR KONEKSI: DMS Mazda                                 │
│                                   │                                                              │
│ ► ● [LOCAL] DMS Mazda             │ Rute Koneksi:                                                │
│     127.0.0.1:8891                │ 127.0.0.1:8891  ──────►  [homelab:22]  ──────►  local:20128  │
│     homelab (28ms)          ↻     │                                                              │
│                                   │ Latensi Host  : 28ms (homelab.arry.my.id:22)                 │
│   ● [LOCAL] Postgres Staging      │ Port Lokal    : 127.0.0.1:8891 (Listening ●)                 │
│     127.0.0.1:5433                │ Status Proses : Berjalan normal (Uptime: 18m 42s)            │
│     homelab (28ms)                │ Kredensial    : user: arry | key: ~/.ssh/id_ed25519          │
│                                   ├──────────────────────────────────────────────────────────────┤
│   ○ [REV]   Webhook Test          │ LIVE LOGS (DMS Mazda)               [PgUp/PgDn scroll 100%] │
│     127.0.0.1:3000                │ 14:10:02 [DMS Mazda] connected to homelab.arry.my.id         │
│     vps-sg                        │ 14:10:03 [DMS Mazda] local forwarding listening on :8891     │
├───────────────────────────────────┴──────────────────────────────────────────────────────────────┤
│ enter/space start/stop · / filter · c copy url · o browser · ? bantuan · q keluar                │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 2. Modal Bantuan Pintasan (`?`)

```text
┌─────────────────────────────────────────────────────────────────┐
│                     [ BANTUAN PINTASAN SSHTUI ]                 │
│                                                                 │
│   ↑ / ↓ atau j / k        Navigasi daftar tunnel                │
│   enter / space           Start / stop tunnel terpilih          │
│   r                       Restart proses tunnel                 │
│   a / x                   Start semua / stop semua              │
│   /                       Filter / cari nama & host             │
│   c                       Salin URL ke clipboard                │
│   o                       Buka URL di web browser               │
│   PgUp / PgDn (J / K)     Scroll viewport log                   │
│   n / e / d               Tambah / edit / hapus tunnel          │
│   ctrl+d (di form)        Deteksi port remote via SSH           │
│   q                       Keluar aplikasi                       │
│                                                                 │
│                 Tekan '?' atau 'esc' untuk menutup              │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🛠️ Panduan Instalasi (How to Install)

`sshtui` dapat dipasang di berbagai sistem operasi (**macOS**, **Linux**, dan **Windows**). Pastikan Anda memiliki **OpenSSH Client (`ssh`)** dan **Go 1.22+** terpasang di sistem Anda.

---

### Cara 1: Universal via `go install` (Semua OS)

Jika Anda sudah memiliki Go di mesin Anda, cara tercepat adalah memasang binary langsung:

```bash
go install github.com/werkstudio/tunneling-ssh-tui/cmd/sshtui@latest
```

> **Catatan**: Pastikan direktori bin Go (`$HOME/go/bin` atau `$GOPATH/bin`) sudah terdaftar di `PATH` shell Anda.

---

### Cara 2: Instalasi per Sistem Operasi (Build dari Sumber)

#### 🍏 macOS (Apple Silicon & Intel)

macOS sudah dilengkapi OpenSSH bawaan dan clipboard tool `pbcopy`.

1. **Clone repository:**
   ```bash
   git clone https://github.com/werkstudio/tunneling-ssh-tui.git
   cd tunneling-ssh-tui
   ```

2. **Kompilasi dan pasang:**
   ```bash
   # Kompilasi binary ke ./bin/sshtui
   make build

   # Pasang binary ke ~/.local/bin/sshtui
   make install
   ```

3. **Pastikan `~/.local/bin` ada di PATH:**
   Jika belum ada, tambahkan ke `~/.zshrc` (atau `~/.bash_profile`):
   ```bash
   echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
   source ~/.zshrc
   ```

---

#### 🐧 Linux (Ubuntu, Debian, Fedora, Arch Linux)

Pada Linux, pastikan utility clipboard (`xclip` untuk X11 atau `wl-clipboard` untuk Wayland) terpasang agar fitur salin URL (`c`) berfungsi optimal.

1. **Pasang Dependensi Dasar:**
   - **Ubuntu / Debian**:
     ```bash
     sudo apt update
     sudo apt install -y git golang openssh-client xclip # atau: wl-clipboard (Wayland)
     ```
   - **Fedora / RHEL**:
     ```bash
     sudo dnf install -y git golang openssh-clients xclip # atau: wl-clipboard
     ```
   - **Arch Linux**:
     ```bash
     sudo pacman -S git go openssh xclip # atau: wl-clipboard
     ```

2. **Clone & Pasang:**
   ```bash
   git clone https://github.com/werkstudio/tunneling-ssh-tui.git
   cd tunneling-ssh-tui
   make build

   # Opsi A: Pasang untuk user saat ini (~/.local/bin)
   make install

   # Opsi B: Pasang system-wide (untuk seluruh pengguna)
   sudo install -m 755 bin/sshtui /usr/local/bin/
   ```

---

#### 🪟 Windows

Ada dua cara menjalankan `sshtui` di Windows:

##### Opsi A: Menggunakan WSL (Windows Subsystem for Linux) — *Direkomendasikan*
1. Buka terminal WSL (Ubuntu/Debian).
2. Ikuti langkah instalasi untuk **Linux** di atas.

##### Opsi B: Native Windows (PowerShell / Command Prompt)
1. Pastikan **Go** dan **OpenSSH Client** (bawaan Windows Settings -> Optional Features) sudah aktif.
2. Buka PowerShell dan jalankan:
   ```powershell
   git clone https://github.com/werkstudio/tunneling-ssh-tui.git
   cd tunneling-ssh-tui
   go build -o sshtui.exe ./cmd/sshtui
   ```
3. Pindahkan `sshtui.exe` ke direktori yang terdaftar di environment variable `PATH` (misalnya `C:\Program Files\sshtui\` atau folder tools pribadi Anda).

---

### Verifikasi Instalasi

Jalankan perintah berikut di terminal untuk memastikan `sshtui` sudah terpasang dan dapat dipanggil dengan benar:

```bash
sshtui path
sshtui list
```

Untuk meluncurkan antarmuka visual TUI:
```bash
sshtui
```


---

## ⚙️ Konfigurasi

Lokasi konfigurasi default:
- `~/.config/sshtui/config.toml` (dapat di-override via environment variable `SSHTUI_CONFIG`).

### Contoh File `config.toml`:

```toml
# Definisi Host Remote (Opsional jika sudah ada di ~/.ssh/config)
[host.homelab]
address = "homelab.arry.my.id"
user = "arry"
port = 22
key = "~/.ssh/id_ed25519"   # opsional

[host.vps-sg]
address = "139.180.200.10"
user = "root"
port = 2222

# Definisi Tunnel
[[tunnel]]
name = "DMS Mazda"
host = "homelab"        # alias [host.*], alias ~/.ssh/config, atau user@host
type = "local"          # local (-L) | reverse (-R)
bind = "127.0.0.1"      # kosong = 127.0.0.1
local_port = 8891
remote_host = "localhost"
remote_port = 20128
autostart = true

[[tunnel]]
name = "Postgres Staging"
host = "homelab"
type = "local"
local_port = 5433
remote_host = "10.0.0.15"
remote_port = 5432
autostart = false

[[tunnel]]
name = "Webhook Test"
host = "vps-sg"
type = "reverse"        # remote port di-forward ke port lokal kita
local_port = 3000
remote_port = 8080
autostart = false
```

---

## ⌨️ Daftar Tombol Pintasan (Keybindings)

### Navigasi & Kontrol Utama
| Tombol | Fungsi |
|---|---|
| `↑` `↓` / `j` `k` | Pindah kursor pilihan tunnel |
| `enter` / `space` | Jalankan (Start) atau Hentikan (Stop) tunnel terpilih |
| `r` | Restart tunnel terpilih |
| `a` / `x` | Start semua tunnel / Stop semua tunnel |
| `/` | Buka bar pencarian / filter instan (`esc` untuk batal) |
| `c` | Salin URL (`http://localhost:<port>`) ke clipboard sistem |
| `o` | Buka URL di browser web bawaan |
| `?` | Buka / tutup modal bantuan pintasan |
| `PgUp` / `PgDn` (atau `J` / `K`) | Gulir riwayat log di panel kanan |
| `n` | Tambah tunnel baru |
| `e` | Edit tunnel terpilih |
| `d` | Hapus tunnel terpilih (dengan konfirmasi) |
| `q` atau `ctrl+c` | Keluar dari aplikasi (semua tunnel otomatis dimatikan) |

### Formulir Tambah / Edit Tunnel
| Tombol | Fungsi |
|---|---|
| `tab` / `shift+tab` | Pindah ke field berikutnya / sebelumnya |
| `space` / `←` `→` | Ubah pilihan opsi toggle (Tipe / Autostart) |
| `ctrl+d` | **Deteksi port remote** di server via SSH otomatis |
| `ctrl+s` | Simpan konfigurasi |
| `esc` | Batal dan kembali ke daftar utama |

---

## 💻 Penggunaan Mode CLI (Headless)

Selain antarmuka TUI interaktif, `sshtui` juga menyediakan subkomando CLI:

```bash
# Menampilkan daftar konfigurasi tunnel
sshtui list

# Memindai port listening di server remote tanpa membuka TUI
sshtui scan homelab

# Menjalankan tunnel tertentu di foreground (Ctrl+C untuk berhenti)
sshtui up "DMS Mazda"

# Menjalankan semua tunnel sekaligus di foreground
sshtui up --all

# Menampilkan lokasi file konfigurasi yang sedang aktif
sshtui path
```

---

## 📁 Struktur Repositori

```text
cmd/sshtui/        # Entrypoint CLI dan loader TUI
internal/config/   # Parser TOML, penyusun argumen SSH, dan resolusi host
internal/tunnel/   # Runtime proses SSH -N, auto-reconnect, dan mesin telemetri RTT
internal/detect/   # Deteksi port remote (ss, netstat, lsof)
internal/tui/      # Antarmuka Bubble Tea, layout split-pane, inspector & modal
docs/              # Spesifikasi desain teknis dan panduan implementasi
AGENTS.md          # Panduan arsitektur untuk AI coding agents
```

---

## 📄 Lisensi

Didistribusikan di bawah lisensi MIT. Silakan gunakan dan kembangkan sesuai kebutuhan Anda.
