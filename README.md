# Automated Video Clipping System (Golang + FFmpeg)

Sistem pemotong video otomatis berbasis **Go (Golang)** yang terintegrasi dengan **FFmpeg** & **OpenRouter AI**. Sistem ini mendukung:
- 🤖 **AI Transcript Highlights (`openrouter/free`)** (Deteksi otomatis klip paling menarik/viral dari transkrip subtitle menggunakan LLM via OpenRouter API).
- 🎙️ **Smart Silence & Scene Auto-Detection** (Deteksi otomatis bagian percakapan/suara atau perpindahan adegan tanpa timestamp manual).
- ⚡ **Parallel Concurrency Engine** (Render banyak klip sekaligus secara paralel via Goroutines).
- 🎨 **Watermark Image & Text Overlay** (Penambahan logo watermark PNG & caption teks otomatis).
- 📱 **YouTube Shorts / TikTok / Reels (9:16)** (Format vertikal dengan style *center crop* atau *blurred background*).
- 💾 **Smart Caching System** (`./cache` auto-reuse video YouTube).
- 📽️ **YouTube Video Quality Selector** (Pilihan 1080p, 720p, 480p, 360p, best, worst).

---

## 🤖 AI Transcript Highlights (`-auto-detect ai`)

Fitur kecerdasan buatan untuk menganalisis transkrip subtitle video (YouTube / VTT / SRT) dan memilih segmen-segmen terbaik yang berpotensi viral menggunakan **OpenRouter API** (`openrouter/free` router):

```bash
# 1. Set API Key OpenRouter di Environment Variable (opsional bisa via flag -openrouter-key)
export OPENROUTER_API_KEY="sk-or-v1-..."

# 2. Jalankan Auto-Detect AI untuk memotong klip YouTube & otomatis dikonversi ke Shorts (9:16)
go run ./cmd/clipper -input "https://www.youtube.com/watch?v=xxx" -auto-detect ai -shorts -outdir ./yt_ai_shorts
```

Kamu juga bisa menggunakan model spesifik lainnya via flag `-ai-model`:
```bash
go run ./cmd/clipper -input video.mp4 -auto-detect ai -ai-model "google/gemini-2.0-flash-exp:free" -shorts
```

---

## ⚙️ Generate Konfigurasi JSON (`-init-config` / `-i`)

```bash
# Mode Interaktif Wizard
go run ./cmd/clipper -i

# Generate JSON Config via Flags
go run ./cmd/clipper -init-config my_clips.json -input "https://youtu.be/xxx" -shorts -auto-detect ai
```

Jalankan config yang dibuat:
```bash
go run ./cmd/clipper -config my_clips.json
```

---

## 🚀 Fitur Lainnya

### 1. Smart Silence & Scene Detection (`-auto-detect silence` / `scene`)
```bash
# Otomatis deteksi percakapan & potong klipnya
go run ./cmd/clipper -input video.mp4 -auto-detect silence -outdir ./speech_clips
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
