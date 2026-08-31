# Automated Video Clipping System (Golang + FFmpeg)

Sistem pemotong video otomatis berbasis **Go (Golang)** yang terintegrasi dengan **FFmpeg**, **OpenRouter AI**, dan **Local Whisper AI**. Sistem ini mendukung:

- 🎤 **TikTok & Instagram Reels Karaoke Subtitles (`-sub-style karaoke`)** (Subtitle animasi cepat per 2–3 kata berlatar kuning menyala + font Impact tebal yang viral di media sosial).
- 🎙️ **Local Whisper Speech-to-Text (`-use-whisper`)** (Ekstraksi subtitle AI secara otomatis dan offline dari audio video menggunakan Whisper AI).
- 👤 **Smart Subject Motion Auto-Crop (`-shorts-style smart-crop`)** (Pemotongan vertikal 9:16 yang memfokuskan kamera ke subjek bergerak).
- 💬 **Burnt-In Subtitles & Auto Translation (`-burn-subtitles`)** (Menempelkan subtitle terjemahan secara otomatis dan permanen pada video klip/Shorts dengan kustomisasi `-sub-font-size`).
- 🤖 **AI Transcript Highlights (`openrouter/free`)** (Deteksi otomatis klip paling menarik/viral dari transkrip subtitle menggunakan LLM via OpenRouter API).
- 🎙️ **Smart Silence & Scene Auto-Detection** (Deteksi otomatis bagian percakapan/suara atau perpindahan adegan tanpa timestamp manual).
- ⚡ **Parallel Concurrency Engine** (Render banyak klip sekaligus secara paralel via Goroutines).
- 🎨 **Watermark Image & Text Overlay** (Penambahan logo watermark PNG & caption teks otomatis).
- 📱 **YouTube Shorts / TikTok / Reels (9:16)** (Format vertikal dengan style *center crop*, *smart crop*, atau *blurred background*).
- 💾 **Smart Caching System** (`./cache` auto-reuse video YouTube & transkrip Whisper).

---

## 🎤 TikTok Karaoke Subtitles (`-sub-style karaoke`)

```bash
# Render Shorts (9:16 Blur) + AI Auto-Detect + TikTok Animated Karaoke Subtitles (Kuning Menyala, 54pt)
go run ./cmd/clipper -input "https://www.youtube.com/watch?v=xxx" -auto-detect ai -shorts -shorts-style blur -burn-subtitles -sub-style karaoke -sub-font-size 54 -translate-lang id -outdir ./yt_karaoke_shorts
```

---

## 🎙️ Local Whisper Speech-to-Text (`-use-whisper`)

Jika video lokal / YouTube tidak memiliki subtitle CC bawaan, gunakan `-use-whisper` untuk transkripsi offline otomatis:

```bash
go run ./cmd/clipper -input "my_local_video.mp4" -auto-detect ai -use-whisper -shorts -burn-subtitles -sub-style karaoke
```

---

## ⚙️ Generate Konfigurasi JSON (`-init-config` / `-i`)

```bash
# Mode Interaktif Wizard
go run ./cmd/clipper -i
```
