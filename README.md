# Automated Video Clipping System (Golang + FFmpeg)

Sistem pemotong video otomatis berbasis **Go (Golang)** yang terintegrasi dengan **FFmpeg**. Sistem ini mendukung:
- ⚡ **Parallel Concurrency Engine** (Render banyak klip sekaligus secara paralel via Goroutines).
- 🎙️ **Smart Silence & Scene Auto-Detection** (Deteksi otomatis bagian percakapan/suara atau perpindahan adegan tanpa timestamp manual).
- 🎨 **Watermark Image & Text Overlay** (Penambahan logo watermark PNG & caption teks otomatis).
- 📱 **YouTube Shorts / TikTok / Reels (9:16)** (Format vertikal dengan style *center crop* atau *blurred background*).
- 💾 **Smart Caching System** (`./cache` auto-reuse video YouTube).
- 📽️ **YouTube Video Quality Selector** (Pilihan 1080p, 720p, 480p, 360p, best, worst).

---

## 🚀 Fitur Baru & Cara Menggunakannya

### 1. Smart Silence & Scene Detection (`-auto-detect`)
Sistem secara otomatis mendeteksi bagian percakapan/audio yang aktif (menghapus hening) atau pergantian adegan tanpa perlu mengisikan segmen timestamp manual:

```bash
# Otomatis deteksi percakapan & potong klipnya
go run ./cmd/clipper -input video.mp4 -auto-detect silence -outdir ./speech_clips

# Otomatis deteksi percakapan lalu ubah langsung jadi Shorts 9:16
go run ./cmd/clipper -input "https://youtu.be/xxx" -auto-detect silence -shorts -outdir ./shorts_auto
```

---

### 2. Parallel Concurrency Engine (`-concurrency`)
Manfaatkan seluruh core CPU kamu untuk memotong banyak segmen video secara bersamaan (paralel):

```bash
# Menggunakan 4 worker thread paralel
go run ./cmd/clipper -input video.mp4 -segments "00:10-00:25,01:00-01:30,02:15-02:45,04:00-04:30" -concurrency 4
```

---

### 3. Watermark & Text Overlay (`-watermark` & `-text`)
Tambahkan watermark logo dan caption teks judul secara otomatis pada klip hasil eksport:

```bash
# Tambahkan Teks Caption & Logo Watermark
go run ./cmd/clipper -input video.mp4 -segments "00:10-00:30" -text "Eps.69 Highlight" -text-pos bottom-center -watermark logo.png -watermark-pos top-right
```

---

## ⚡ Smart Caching System (`./cache`)

1. **Auto Reuse**: Saat kamu memproses URL YouTube yang sama, sistem **TIDAK AKAN mengunduh ulang** video tersebut.
2. **Instant Processing**: Video di `./cache/` langsung digunakan kembali secara instan (`[CACHE HIT]`).
3. **Pilihan Flag**:
   - `-cache-dir <dir>`: Lokasi penyimpanan cache (default: `./cache`).
   - `-no-cache`: Mengabaikan cache dan memaksa download ulang dari YouTube.

---

## 📽️ Pilihan Kualitas Video YouTube (`-quality`)

- **`best` (Default)**: Mengunduh kualitas tertinggi (hingga 4K/1080p).
- **`1080p`**: Kualitas Maksimal 1080p (Full HD).
- **`720p`**: Kualitas Maksimal 720p (HD).
- **`480p` / `360p`**: Kualitas hemat kuota.

---

## 📂 Generate Konfigurasi JSON (`cmd/genconfig`)

```bash
# Mode Interaktif Wizard (Termasuk prompt Auto-Detect, Watermark, & Concurrency)
go run ./cmd/genconfig -i
```
