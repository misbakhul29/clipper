# Automated Video Clipping System (Golang + FFmpeg)

Sistem pemotong video otomatis berbasis **Go (Golang)** yang terintegrasi dengan **FFmpeg** & **OpenRouter AI**. Sistem ini mendukung:
- 💬 **Burnt-In Subtitles & Auto Translation (`-burn-subtitles` / `-subtitles`)** (Menempelkan subtitle terjemahan secara otomatis dan permanen pada video klip/Shorts).
- 🤖 **AI Transcript Highlights (`openrouter/free`)** (Deteksi otomatis klip paling menarik/viral dari transkrip subtitle menggunakan LLM via OpenRouter API).
- 🎙️ **Smart Silence & Scene Auto-Detection** (Deteksi otomatis bagian percakapan/suara atau perpindahan adegan tanpa timestamp manual).
- ⚡ **Parallel Concurrency Engine** (Render banyak klip sekaligus secara paralel via Goroutines).
- 🎨 **Watermark Image & Text Overlay** (Penambahan logo watermark PNG & caption teks otomatis).
- 📱 **YouTube Shorts / TikTok / Reels (9:16)** (Format vertikal dengan style *center crop* atau *blurred background*).
- 💾 **Smart Caching System** (`./cache` auto-reuse video YouTube).
- 📽️ **YouTube Video Quality Selector** (Pilihan 1080p, 720p, 480p, 360p, best, worst).

---

## 💬 Burnt-In Subtitles (`-burn-subtitles`)

Tambahkan subtitle terjemahan yang menempel secara permanen (*hardcoded captions*) pada video Shorts / klip:

```bash
# Render Shorts (9:16 Blur) + Auto-Detect AI + Subtitle Terjemahan Bahasa Indonesia Nempel di Video!
go run ./cmd/clipper -input "https://www.youtube.com/watch?v=xxx" -auto-detect ai -shorts -shorts-style blur -burn-subtitles -translate-lang id -outdir ./yt_subtitled_shorts
```

---

## 🤖 AI Transcript Highlights (`-auto-detect ai`)

Fitur kecerdasan buatan untuk menganalisis transkrip subtitle video (YouTube / VTT / SRT) dan memilih segmen-segmen terbaik yang berpotensi viral menggunakan **OpenRouter API** (`openrouter/free` router):

```bash
# Set API Key OpenRouter di Environment Variable (opsional bisa via flag -openrouter-key)
export OPENROUTER_API_KEY="sk-or-v1-..."

# Jalankan Auto-Detect AI
go run ./cmd/clipper -input "https://www.youtube.com/watch?v=xxx" -auto-detect ai -shorts -burn-subtitles
```

---

## ⚙️ Generate Konfigurasi JSON (`-init-config` / `-i`)

```bash
# Mode Interaktif Wizard
go run ./cmd/clipper -i

# Generate JSON Config via Flags
go run ./cmd/clipper -init-config my_clips.json -input "https://youtu.be/xxx" -shorts -auto-detect ai -burn-subtitles
```

Jalankan config yang dibuat:
```bash
go run ./cmd/clipper -config my_clips.json
```
