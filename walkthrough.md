# Walkthrough: TUI UI/UX & Connection Enhancements

## Ringkasan Perubahan

Peningkatan menyeluruh pada antarmuka pengguna (TUI), telemetri koneksi, dan pengalaman navigasi untuk `sshtui`:

1. **Responsive Split-Pane Layout**:
   - Terminal lebar (≥ 110 kolom): Tampilan berdampingan otomatis (kolom kiri ~45% untuk daftar tunnel, kolom kanan ~55% untuk inspektor koneksi & log).
   - Terminal kompak (< 110 kolom): Tata letak vertikal bertumpuk adaptif tanpa teks terpotong canggung.
   - Header dinamis: Menampilkan judul, pill ringkasan status (`● X Aktif · ○ Y Berhenti · ▲ Z Error`), serta path konfigurasi yang disingkat cerdas (`~`) dan dipotong rapi (`…`).

2. **Real-Time Connection Inspector & Telemetri**:
   - **Diagram Alur Rute**: Menampilkan diagram visual panah koneksi (misal `127.0.0.1:8891 ──────► [homelab.my.id:22] ──────► localhost:20128`).
   - **Native Go Host Latency Probe**: Pemeriksaan TCP round-trip latency (RTT ping) berkala ke remote SSH host dengan pewarnaan latensi (<50ms hijau, 50-150ms cyan, 150-300ms kuning, >300ms oranye).
   - **Local Listener Check**: Verifikasi real-time apakah port lokal benar-benar sedang listening dan siap menerima trafik.
   - **Status Proses & Lifecycle**: PID, uptime, waktu start, dan hitungan auto-reconnect.

3. **Scrollable Live Logs**:
   - Panel log proses SSH real-time dengan viewport mandiri.
   - Navigasi scrolling keyboard (`PgUp`/`PgDn` atau `J`/`K`) dilengkapi indikator posisi scroll `[xx%]`.

4. **Pencarian & Filter Cepat (`/`)**:
   - Filter instan daftar tunnel berdasarkan nama dan host tanpa lag.
   - Tekan `esc` untuk mereset pencarian.

5. **Integrasi Clipboard (`c`)**:
   - Salin URL lokal (`http://localhost:<port>`) langsung ke clipboard sistem (mendukung macOS `pbcopy`, Linux `wl-copy`/`xclip`/`xsel`, dan Windows `clip.exe`).

6. **Modal Bantuan Pintasan (`?`)**:
   - Overlay modal pop-up yang menyajikan cheatsheet tombol pintas kapan saja.

7. **Deteksi Port Remote dengan Animasi Spinner (`ctrl+d`)**:
   - Pemindaian port listening di server remote melalui SSH (`ss -ltnp`, `netstat`, `lsof`).
   - Dilengkapi animasi spinner braille halus (`⠋ ⠙ ⠹ ⠸`) saat proses scan berlangsung.

---

## Perbandingan Visual Layout (ASCII Diagram)

### 1. Split-Pane Layout (Terminal Lebar ≥ 110 Kolom)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ sshtui SSH tunnel manager         [ ● 1 Aktif  ·  ○ 1 Berhenti  ·  ▲ 0 Error ]     ~/.config/... │
├───────────────────────────────────┬──────────────────────────────────────────────────────────────┤
│ DAFTAR TUNNEL                     │ INSPEKTOR KONEKSI: DMS Mazda                                 │
│                                   │                                                              │
│ ► ● [LOCAL] DMS Mazda             │ Rute Koneksi:                                                │
│     127.0.0.1:8891                │ 127.0.0.1:8891  ──────►  [homelab:22]  ──────►  local:20128  │
│     homelab (28ms)                │                                                              │
│                                   │ Latensi Host  : 28ms (homelab.arry.my.id:22)                 │
│   ○ [LOCAL] 9router               │ Port Lokal    : 127.0.0.1:8891 (Listening ●)                 │
│     127.0.0.1:20128               │ Status Proses : PID 14201 | Uptime: 14m32s                   │
│     homelab                       │                                                              │
│                                   ├──────────────────────────────────────────────────────────────┤
│                                   │ LIVE LOGS (DMS Mazda)               [PgUp/PgDn scroll 100%] │
│                                   │ 14:10:02 [DMS Mazda] connected to homelab.arry.my.id         │
│                                   │ 14:10:03 [DMS Mazda] local forwarding listening on :8891     │
├───────────────────────────────────┴──────────────────────────────────────────────────────────────┤
│ enter/space start/stop · / filter · c copy url · o browser · ? bantuan · q keluar                │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 2. Stacked Layout (Terminal Kompak < 110 Kolom)

```text
┌─────────────────────────────────────────────────────────────────┐
│ sshtui SSH tunnel manager  [ ● 1 Aktif · ○ 1 Berhenti ]         │
├─────────────────────────────────────────────────────────────────┤
│ DAFTAR TUNNEL                                                   │
│ ► ● [LOCAL] DMS Mazda    127.0.0.1:8891 (28ms)                  │
│   ○ [LOCAL] 9router      127.0.0.1:20128                        │
├─────────────────────────────────────────────────────────────────┤
│ INSPEKTOR KONEKSI: DMS Mazda                                    │
│ 127.0.0.1:8891  ──────►  [homelab:22]  ──────►  local:20128     │
│ Latensi: 28ms  ·  Port Lokal: 127.0.0.1:8891 (Listening ●)      │
├─────────────────────────────────────────────────────────────────┤
│ LIVE LOGS (DMS Mazda)                   [PgUp/PgDn scroll 100%] │
│ 14:10:02 [DMS Mazda] connected to homelab.arry.my.id            │
├─────────────────────────────────────────────────────────────────┤
│ enter/space start/stop · / filter · ? bantuan · q keluar        │
└─────────────────────────────────────────────────────────────────┘
```

### 3. Modal Bantuan Pintasan (`?`)

```text
┌─────────────────────────────────────────────────────────────────┐
│                     [ BANTUAN PINTASAN SSHTUI ]                 │
│                                                                 │
│   ↑ / ↓ atau j / k        Navigasi daftar tunnel                │
│   enter / space           Start / stop tunnel                   │
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

## Tabel Lengkap Tombol Pintasan

| Tombol | Konteks | Aksi |
|---|---|---|
| `↑` `↓` / `j` `k` | Daftar Utama | Navigasi pemilihan tunnel |
| `enter` / `space` | Daftar Utama | Menjalankan atau menghentikan tunnel yang dipilih |
| `r` | Daftar Utama | Me-restart proses tunnel terpilih |
| `a` / `x` | Daftar Utama | Menjalankan semua atau menghentikan semua tunnel |
| `/` | Daftar Utama | Membuka baris pencarian/filter instan |
| `c` | Daftar Utama | Menyalin URL lokal (`http://localhost:<port>`) ke clipboard |
| `o` | Daftar Utama | Membuka URL lokal di browser web bawaan |
| `?` | Daftar Utama | Membuka atau menutup pop-up modal bantuan pintasan |
| `PgUp` / `PgDn` (atau `J` / `K`) | Daftar Utama | Menggulir log live di panel kanan |
| `n` / `e` / `d` | Daftar Utama | Membuka formulir tambah / edit / hapus tunnel |
| `q` | Daftar Utama | Keluar dari aplikasi (mematikan semua tunnel) |
| `tab` / `shift+tab` | Formulir | Pindah ke input field berikutnya / sebelumnya |
| `space` / `←` `→` | Formulir | Mengubah pilihan toggle tipe tunnel dan autostart |
| `ctrl+d` | Formulir | Memulai pemindaian port remote otomatis via SSH |
| `ctrl+s` | Formulir | Menyimpan konfigurasi tunnel ke file |
| `esc` | Formulir / Modal / Filter | Membatalkan / menutup modal / mereset filter |

---

## Hasil Verifikasi & Pengujian

### 1. Static Analysis & Linter
```bash
go vet ./...
```
*Hasil*: `0 warning`, bersih tanpa kendala.

### 2. Rangkaian Pengujian Unit (Unit Test Suite)
```bash
go test -count=1 ./...
```
```text
?       sshtui/cmd/sshtui       [no test files]
?       sshtui/internal/config  [no test files]
ok      sshtui/internal/detect  0.378s
ok      sshtui/internal/tui     2.816s
ok      sshtui/internal/tunnel  0.626s
```
*Hasil*: 100% tes lulus di seluruh paket (`detect`, `tui`, `tunnel`).

### 3. Kompilasi & Pembuatan Binary
```bash
make build
```
```text
go build -o bin/sshtui ./cmd/sshtui
```
*Hasil*: Binary `./bin/sshtui` berhasil dibuat dan siap digunakan.

### 4. Eksekusi Perintah CLI
- `./bin/sshtui -h`: Menampilkan panduan CLI dan tabel fitur baru secara rapi.
- `./bin/sshtui path`: Mengembalikan lokasi path konfigurasi `~/.config/sshtui/config.toml`.
- `./bin/sshtui list`: Menampilkan ringkasan konfigurasi tunnel yang aktif.
