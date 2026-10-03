# AGENTS.md — Guidance for AI Coding Agents

Panduan ini ditujukan bagi AI coding agent atau pengembang yang akan membaca, memelihara, dan meneruskan pengembangan di repositori **`sshtui`**.

---

## 1. Ikhtisar Repositori & Filosofi Desain

**`sshtui`** adalah Terminal User Interface (TUI) modern berbasis bahasa pemrograman **Go** untuk mengelola SSH port forwarding (local `-L` dan reverse `-R` tunnel) secara intuitif, informatif, dan andal.

### Prinsip Utama:
1. **Native Go First**: Seluruh telemetri koneksi (pengukuran latensi RTT host dan verifikasi port listening lokal) menggunakan library standar Go (`net.DialTimeout`), bukan mengeksekusi subprocess shell (`ping`, `lsof`, `nc`) secara liar.
2. **Responsif & Zero UI Stutter**: Antarmuka Bubble Tea tidak boleh mengalami freeze/blocking. Pembacaan jaringan di dalam render loop (`View()`) wajib dilindungi oleh in-memory cache dengan Time-To-Live (TTL) atau dijalankan via asynchronous `tea.Cmd`.
3. **Zero Breaking Changes**: Skema konfigurasi pada `~/.config/sshtui/config.toml` harus selalu backward-compatible.
4. **Unified Keyboard Navigation**: Navigasi kursor utama selalu terfokus pada daftar tunnel (`↑↓` atau `jk`), sementara viewport log di panel samping dapat digulir langsung dengan `PgUp/PgDn` atau `J/K` tanpa perlu modal pane switching yang merepotkan.

---

## 2. Struktur Proyek & Tanggung Jawab Paket

```text
sshtui/
├── cmd/
│   └── sshtui/
│       └── main.go           # CLI entrypoint, penanganan subkomando (list, scan, up, path) & inisialisasi TUI
├── internal/
│   ├── config/
│   │   └── config.go         # Parsing & penyimpanan file TOML, resolusi host SSH, dan penyusunan argumen ssh
│   ├── detect/
│   │   ├── detect.go         # Pemindaian port listening di server remote via SSH (ss -ltnp, netstat, lsof)
│   │   └── detect_test.go    # Pengujian unit parser deteksi port
│   ├── tunnel/
│   │   ├── tunnel.go         # Runtime pengelola proses ssh -N, auto-reconnect backoff, capturing error stderr
│   │   ├── telemetry.go      # Mesin telemetri native: ProbeHost (RTT latency) & CheckLocalPort dengan in-memory TTL cache
│   │   └── telemetry_test.go # Pengujian unit telemetri dan status snapshot
│   └── tui/
│       ├── tui.go            # Model utama Bubble Tea, render split-pane responsif, form handler, event loop
│       ├── helpers.go        # Helper OS clipboard (pbcopy/wl-copy/xclip/clip.exe) & formatter diagram rute ASCII
│       ├── helpers_test.go   # Pengujian unit helpers & isolasi clipboard runner mock
│       ├── modal.go          # Komponen overlay Help Modal popup (?)
│       ├── filter_test.go    # Pengujian unit sistem filter/pencarian instan (/)
│       └── split_test.go     # Pengujian unit layout split-pane & inspektor koneksi
├── docs/
│   └── superpowers/          # Dokumen arsitektur, spesifikasi teknis, dan rencana kerja
├── Makefile                  # Target otomasi build, install, test, dan clean
├── go.mod / go.sum           # Dependensi Go
└── README.md                 # Dokumentasi pengguna publik
```

---

## 3. Alur Kerja Build & Pengujian

### Perintah Esensial:
```bash
# 1. Kompilasi binary ke ./bin/sshtui
make build

# 2. Jalankan static analysis / linter
go vet ./...

# 3. Jalankan seluruh unit test suite
go test -v ./...

# 4. Jalankan test tanpa cache (verifikasi penuh)
go test -count=1 ./...

# 5. Pasang binary ke ~/.local/bin
make install
```

> [!NOTE]
> Unit test di `internal/tunnel/telemetry_test.go` membuat TCP listener loopback (`127.0.0.1:0`) untuk memverifikasi socket dialing secara riil. Pastikan lingkungan eksekusi memiliki izin loopback socket (`BypassSandbox: true` jika berjalan dalam sandbox ketat).

---

## 4. Konvensi Kode & Detail Arsitektur

### 4.1 State Management & Concurrency
- `tunnel.Runtime` mengelola siklus hidup proses SSH latar belakang.
- Mutasi status (`Status`, `since`, `lastError`, `retryCount`, `nextRetryAt`, `logs`) wajib dilindungi oleh `r.mu sync.Mutex`.
- Pembacaan data oleh UI dilakukan secara atomik melalui method `r.Snapshot() -> tunnel.Snapshot`.
- `tunnel.Manager` mengelola kumpulan runtime per nama tunnel secara thread-safe menggunakan mutex tersendiri.

### 4.2 Telemetri & Caching
- **ProbeHost (`internal/tunnel/telemetry.go`)**:
  - Mengukur latensi RTT ke host port SSH via TCP dial dengan timeout 2 detik.
  - Memiliki in-memory cache `latencyMap` dengan TTL 5 detik (`cacheExpiry`).
- **CheckLocalPort (`internal/tunnel/telemetry.go`)**:
  - Memverifikasi apakah port lokal aktif mendengar koneksi dengan timeout 200 milidetik.
  - Memiliki in-memory cache `portCheckMap` dengan TTL 1 detik (`portCheckTTL`) untuk mencegah overhead socket berlebih saat Bubble Tea merender puluhan frame per detik.

### 4.3 Layout Responsif di `internal/tui/tui.go`
- Ambang batas responsif adalah **lebar terminal 110 kolom**:
  - `m.width >= 110`: Mengaktifkan **Split-Pane Layout** (Kolom kiri 45% untuk daftar tunnel & filter, kolom kanan 55% untuk inspektor koneksi & live log viewport).
  - `m.width < 110`: Beralih otomatis ke **Stacked Vertical Layout** bertumpuk agar teks tidak terpotong canggung.
- Truncation defensif: Semua string dinamis (path config di header, diagram rute di kartu inspektor) wajib dibatasi lebarnya menggunakan fungsi `truncate(str, maxLen)` agar tidak menyebabkan *line-wrapping artifact*.

### 4.4 Clipboard Integration
- Gunakan abstraksi `clipboardRunner` di `internal/tui/helpers.go`.
- Jangan pernah memanggil perintah clipboard sistem secara langsung di dalam unit test tanpa melalui mock runner, agar tidak mencemari clipboard mesin pengembang saat menjalankan `go test`.

---

## 5. Fitur Penting & Pemetaan Tombol

| Tombol | Konteks | Fungsi |
|---|---|---|
| `↑` `↓` / `j` `k` | List | Memilih tunnel |
| `enter` / `space` | List | Toggle Start / Stop tunnel |
| `r` | List | Restart proses SSH tunnel terpilih |
| `a` / `x` | List | Start semua / Stop semua |
| `/` | List | Buka baris filter/pencarian instan |
| `c` | List | Salin URL lokal (`http://localhost:<port>`) ke clipboard |
| `o` | List | Buka URL di browser default |
| `?` | List | Buka / tutup Help Modal popup |
| `PgUp` / `PgDn` (atau `J` / `K`) | List | Scroll riwayat log realtime |
| `n` / `e` / `d` | List | Tambah / Edit / Hapus konfigurasi tunnel |
| `ctrl+d` | Form | Deteksi port remote otomatis di server lewat SSH |
| `ctrl+s` | Form | Simpan konfigurasi |
| `esc` | Modal/Form/Filter | Batal / tutup modal / reset filter |
| `q` | List | Keluar aplikasi & hentikan semua tunnel aktif |

---

## 6. Rekomendasi Pengembangan Lanjutan (Roadmap)

Jika Anda ditugaskan untuk menambahkan fitur baru atau melanjutkan refactoring, perhatikan area berikut:

1. **Async Telemetry via `tea.Cmd`**:
   - Saat ini probe telemetri memanfaatkan cache TTL 1-5 detik di `View()`. Untuk arsitektur Bubble Tea yang sepenuhnya asinkron murni, probe dapat dijadwalkan secara periodik melalui `tea.Cmd` dan disimpan ke dalam model field, sehingga `View()` murni bertindak sebagai pure function tanpa I/O.
2. **Template Layanan / Badges**:
   - Menambahkan pengenalan otomatis port populer (misal `3306` = MySQL, `5432` = Postgres, `6379` = Redis, `80/443` = HTTP/HTTPS) untuk menampilkan ikon/badge tipe layanan pada tabel tunnel.
3. **Live Config Reloading**:
   - Mendeteksi perubahan file konfigurasi `~/.config/sshtui/config.toml` secara otomatis (file watcher) tanpa harus me-restart aplikasi.
4. **Ekspor & Impor SSH Config**:
   - Sinkronisasi dua arah dengan blok `Host` pada `~/.ssh/config`.
