# Automated Video Clipping System (Golang + FFmpeg)

Sistem pemotong video otomatis berbasis **Go (Golang)** yang terintegrasi dengan **FFmpeg**. Sistem ini mendukung:
- ⚡ **Parallel Concurrency Engine** (Render banyak klip sekaligus secara paralel via Goroutines).
- 🎙️ **Smart Silence & Scene Auto-Detection** (Deteksi otomatis bagian percakapan/suara atau perpindahan adegan tanpa timestamp manual).
- 🎨 **Watermark Image & Text Overlay** (Penambahan logo watermark PNG & caption teks otomatis).
- 📱 **YouTube Shorts / TikTok / Reels (9:16)** (Format vertikal dengan style *center crop* atau *blurred background*).
- 💾 **Smart Caching System** (`./cache` auto-reuse video YouTube).
- 📽️ **YouTube Video Quality Selector** (Pilihan 1080p, 720p, 480p, 360p, best, worst).
- ⚙️ **Unified Config Generator** (Generate file konfigurasi JSON via interaktif `-i` atau flag `-init-config`).

---

## 📂 Struktur Project (`cmd/` & `pkg/`)

- [cmd/clipper/main.go](file:///home/misbakhulmunir/Dokumen/projects/clipping/cmd/clipper/main.go): Single executable CLI untuk memotong, mengunduh/menggabungkan video, serta meng-generate konfigurasi JSON.
- [pkg/clipper](file:///home/misbakhulmunir/Dokumen/projects/clipping/pkg/clipper/clipper.go): Core library pemotong video & FFmpeg wrapper.
- [pkg/detector](file:///home/misbakhulmunir/Dokumen/projects/clipping/pkg/detector/detector.go): Smart Silence & Scene Detection module.
- [pkg/downloader](file:///home/misbakhulmunir/Dokumen/projects/clipping/pkg/downloader/youtube.go): Module pengunduh otomatis YouTube video.

---

## ⚙️ Generate Konfigurasi JSON (`-init-config` / `-i`)

Kamu bisa membuat file konfigurasi JSON secara otomatis langsung via perintah `clipper`:

```bash
# 1. Mode Interaktif Wizard (-i)
go run ./cmd/clipper -i

# 2. Generate JSON Config via Flags (-init-config)
go run ./cmd/clipper -init-config my_clips.json -input "https://youtu.be/xxx" -shorts -quality 1080p -segments "00:10-00:25:intro"
```

Jalankan config yang dibuat:
```bash
go run ./cmd/clipper -config my_clips.json
```

---

## 🚀 Fitur & Perintah Lainnya

### 1. Smart Silence & Scene Detection (`-auto-detect`)
```bash
# Otomatis deteksi percakapan & potong klipnya
go run ./cmd/clipper -input video.mp4 -auto-detect silence -outdir ./speech_clips

# Otomatis deteksi percakapan lalu ubah langsung jadi Shorts 9:16
go run ./cmd/clipper -input "https://youtu.be/xxx" -auto-detect silence -shorts -outdir ./shorts_auto
```

### 2. Parallel Concurrency Engine (`-concurrency`)
```bash
# Menggunakan 4 worker thread paralel
go run ./cmd/clipper -input video.mp4 -segments "00:10-00:25,01:00-01:30" -concurrency 4
```

### 3. Watermark & Text Overlay (`-watermark` & `-text`)
```bash
# Tambahkan Teks Caption & Logo Watermark
go run ./cmd/clipper -input video.mp4 -segments "00:10-00:30" -text "Eps.69 Highlight" -text-pos bottom-center -watermark logo.png -watermark-pos top-right
```
