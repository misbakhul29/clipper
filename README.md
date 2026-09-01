# Automated Video Clipping System (Golang + FFmpeg)

Sistem pemotong video otomatis berbasis **Go (Golang)** yang terintegrasi dengan **FFmpeg**, **Multi-Provider AI (Google Gemini, DeepSeek, OpenAI, OpenRouter)**, dan **Local Whisper AI**. Sistem ini dirancang untuk memotong video, mendeteksi klip paling menarik/viral secara otomatis, merender format vertikal 9:16 (Shorts/Reels/TikTok), dan menempelkan subtitle animasi interaktif.

## 🚀 Fitur Utama

- 🎤 **TikTok & Instagram Reels Karaoke Subtitles (`-sub-style karaoke`)**: Subtitle animasi cepat per 2–3 kata berlatar kuning menyala + font Impact tebal atau font kustom (`-sub-font-path`).
- 📂 **Per-Video Cache Isolation & Auto-Cleanup (`-clean-cache`)**: Struktur cache berbasis subfolder per video (`./cache/<video_id>/`) untuk mencegah bentrok subtitle, dilengkapi pembersih cache berdasar umur retensi (`-clean-days N`).
- 📑 **Batch Processing Queue (`-batch-list urls.txt`)**: Pemrosesan banyak video sekaligus dari daftar URL/file secara otomatis dalam satu antrean.
- 🔍 **Dry-Run Preview Mode (`-dry-run`)**: Mode simulasi untuk mengecek kalkulasi segmen, hasil auto-detection AI, dan perintah FFmpeg tanpa merender video.
- 🤖 **Multi-Provider AI Highlight Detection**: Dukungan provider AI ganda (**Google Gemini**, **DeepSeek**, **OpenAI**, **OpenRouter**) untuk analisis klip paling viral dari transkrip video.
- 🎙️ **Local Whisper Speech-to-Text (`-use-whisper`)**: Ekstraksi subtitle AI secara otomatis dan offline dari audio video menggunakan Whisper AI.
- 👤 **Smart Subject Motion Auto-Crop (`-shorts-style smart-crop`)**: Pemotongan vertikal 9:16 yang memfokuskan kamera ke subjek bergerak.
- 💬 **Burnt-In Subtitles & Auto Translation (`-burn-subtitles`)**: Menempelkan subtitle terjemahan secara otomatis dan permanen pada video klip/Shorts.
- 🎙️ **Smart Silence & Scene Auto-Detection**: Deteksi otomatis bagian percakapan/suara (`-auto-detect silence`) atau perpindahan adegan (`-auto-detect scene`).
- ⚡ **Parallel Concurrency Engine**: Render banyak klip sekaligus secara paralel via Goroutines worker pool.
- 🎨 **Watermark Image & Text Overlay**: Penambahan logo watermark PNG & caption teks otomatis pada klip.
- 📱 **YouTube Shorts / TikTok / Reels (9:16)**: Format vertikal dengan style *center crop*, *smart crop*, atau *blurred background*.

---

## 📦 Instalasi

### Install Langsung via Go (`go install`)
```bash
go install github.com/misbakhul29/clipper@latest
```
*Pastikan `$GOPATH/bin` (misal: `~/go/bin`) sudah terdaftar di environment variable `PATH` OS Anda agar perintah `clipper` dapat langsung dijalankan dari terminal manapun.*

### Build dari Source Code
```bash
git clone https://github.com/misbakhul29/clipper.git
cd clipper
go build -o bin/clipper .
```

---

## 📖 Dokumentasi & Konfigurasi

- 📑 **[CLI_USAGE.md](CLI_USAGE.md)**: Dokumentasi lengkap baris perintah (*CLI Flags*), opsi Multi-Provider AI, dan contoh skenario penggunaan.
- ⚙️ **[EXAMPLE_CONFIG.md](EXAMPLE_CONFIG.md)**: Contoh lengkap berkas konfigurasi `config.json` dan kamus referensi parameter JSON.

---

## ⚡ Contoh Perintah Cepat

### 1. Render Shorts 9:16 Blur + TikTok Karaoke Subtitles
```bash
clipper -input "https://www.youtube.com/watch?v=xxx" \
  -auto-detect ai \
  -shorts -shorts-style blur \
  -burn-subtitles -sub-style karaoke -sub-font-size 54 \
  -translate-lang id \
  -ai-router gemini -ai-key "YOUR_GEMINI_API_KEY" \
  -outdir ./yt_karaoke_shorts
```

### 2. Dry-Run Simulation (Pratinjau Segmen Tanpa Render Video)
```bash
clipper -input "https://www.youtube.com/watch?v=xxx" -auto-detect silence -shorts -dry-run
```

### 3. Pemrosesan Antrean Banyak Video (Batch Queue)
```bash
clipper -batch-list my_urls.txt -auto-detect ai -shorts -burn-subtitles -sub-style karaoke
```

### 4. Bersihkan Cache yang Berumur Lebih dari 7 Hari
```bash
clipper -clean-cache -clean-days 7
```

### 5. Local Whisper Speech-to-Text Offline
```bash
clipper -input "my_local_video.mp4" -auto-detect ai -use-whisper -shorts -burn-subtitles -sub-style karaoke
```

### 6. Mode Interaktif Wizard
```bash
clipper -i
```
