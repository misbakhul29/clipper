# Automated Video Clipping System (Golang + FFmpeg)

Sistem pemotong video otomatis berbasis **Go (Golang)** yang terintegrasi dengan **FFmpeg**. Sistem ini memproses potongan video berdasarkan timestamp `start` dan `end`, mendukung **YouTube URL**, **Smart Caching System**, pemisahan folder otomatis, pemeliharaan kualitas video, serta konversi otomatis ke format **YouTube Shorts / TikTok / Reels (9:16)**.

---

## ⚡ Smart Caching System (`./cache`)

Sistem secara otomatis mengimplementasikan **Smart Caching** untuk menghemat kuota internet dan mempercepat proses clipping:

1. **Auto Reuse**: Saat kamu memproses URL YouTube yang sama (meski dengan timestamp atau mode Shorts yang berbeda), sistem **TIDAK AKAN mengunduh ulang** video tersebut.
2. **Instant Processing**: Video yang telah tersimpan di direktori `./cache/` langsung digunakan kembali secara instan (`[CACHE HIT]`).
3. **Pilihan Flag**:
   - `-cache-dir <dir>`: Menentukan direktori lokasi penyimpanan cache (default: `./cache`).
   - `-no-cache`: Mengabaikan cache dan memaksa download ulang dari YouTube.

---

## 📽️ Pilihan Kualitas Video YouTube (`-quality`)

- **`best` (Default)**: Mengunduh resolusi kualitas tertinggi yang tersedia (hingga 4K/2K/1080p).
- **`1080p`**: Memilih resolusi Maksimal 1080p (Full HD).
- **`720p`**: Memilih resolusi Maksimal 720p (HD).
- **`480p`**: Memilih resolusi 480p.
- **`360p`**: Memilih resolusi 360p.
- **`worst`**: Mengunduh resolusi paling rendah (hemat kuota).

---

## 🚀 Cara Menggunakan

### 1. Memotong Video YouTube dengan Smart Cache & Shorts (9:16)
```bash
# Perintah 1: Mengunduh 1080p dan menyimpannya di cache (Pertama kali)
go run ./cmd/clipper -input "https://www.youtube.com/watch?v=t7xtO3KqsmM" -quality 1080p -segments "00:22-01:45" -outdir ./yt_clips

# Perintah 2: Eksekusi ulang URL sama untuk Shorts -> INSTAN via [CACHE HIT] tanpa download ulang!
go run ./cmd/clipper -input "https://www.youtube.com/watch?v=t7xtO3KqsmM" -segments "00:22-01:45" -shorts -shorts-style blur -outdir ./yt_clips
```

### 2. Memaksa Download Ulang Tanpa Cache (`-no-cache`)
```bash
go run ./cmd/clipper -input "https://www.youtube.com/watch?v=t7xtO3KqsmM" -no-cache -segments "00:10-00:30"
```

### 3. Generate Konfigurasi JSON (`cmd/genconfig`)
```bash
# Mode Interaktif Wizard (Termasuk prompt cache)
go run ./cmd/genconfig -i

# Via CLI Flags dengan Cache & Quality
go run ./cmd/genconfig -file my_clips.json -input "https://youtu.be/xxx" -quality 1080p -shorts -segments "00:10-00:25:intro"
```

Jalankan config yang dibuat:
```bash
go run ./cmd/clipper -config my_clips.json
```
