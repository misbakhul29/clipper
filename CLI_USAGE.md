# 📖 CLI Command Reference - `clipper`

Dokumentasi lengkap baris perintah (*CLI Flags*) dan contoh penggunaan aplikasi **Clipper** (Automated Video Clipper & Shorts Engine).

---

## 🚀 Kompilasi & Instalasi

```bash
# Build biner lokal di folder proyek (./clipper)
go build ./cmd/clipper

# ATAU Install secara global ke sistem PATH
go install ./cmd/clipper
```

---

## 📋 Daftar Lengkap Flag CLI (`clipper -help`)

| Flag | Tipe | Nilai Default | Deskripsi |
| :--- | :--- | :--- | :--- |
| `-input` | `string` | `""` | Path berkas video lokal atau URL YouTube |
| `-outdir` | `string` | `"."` | Direktori tujuan hasil klip video |
| `-output` | `string` | `""` | Nama berkas keluaran (digunakan pada mode `merge`) |
| `-mode` | `string` | `"split"` | Mode operasi: `"split"` (berkas terpisah) atau `"merge"` (gabung 1 berkas) |
| `-strategy` | `string` | `"fast"` | Strategi pemotongan: `"fast"` (stream copy w/o re-encode) atau `"accurate"` (re-encode presisi) |
| **Shorts (9:16)** | | | |
| `-shorts` | `bool` | `false` | Konversi klip ke format vertikal 9:16 (Shorts/Reels/TikTok) |
| `-shorts-style` | `string` | `"crop"` | Gaya rasio 9:16: `"crop"` (center crop), `"blur"` (latar blurred background), atau `"smart-crop"` (subjek bergerak) |
| **Subtitle & Terjemahan** | | | |
| `-burn-subtitles` / `-subtitles` | `bool` | `false` | Menempelkan (*hardcode*) subtitle terjemahan secara permanen pada video |
| `-sub-style` | `string` | `"karaoke"` | Gaya subtitle: `"karaoke"` (animasi 2-3 kata kuning TikTok) atau `"standard"` (2 baris) |
| `-sub-font-size` | `int` | `48` | Ukuran font subtitle terjemahan |
| `-translate-lang` | `string` | `"id"` | Kode bahasa target terjemahan judul & subtitle (misal: `"id"`, `"en"`) |
| `-use-whisper` | `bool` | `false` | Paksa transkripsi suara offline menggunakan **Whisper AI** lokal |
| **Kecerdasan Buatan (AI) & Auto-Detect** | | | |
| `-auto-detect` | `string` | `""` | Mode deteksi segmen otomatis: `"ai"` (AI highlights), `"silence"`, atau `"scene"` |
| `-ai-router` | `string` | `"openrouter"` | Provider AI: `"openrouter"`, `"gemini"`, `"deepseek"`, `"openai"` (atau `"codex"`) |
| `-ai-key` | `string` | `""` | API Key untuk AI Provider yang dipilih |
| `-openrouter-key` | `string` | `""` | API Key OpenRouter (default mengambil dari env `$OPENROUTER_API_KEY`) |
| `-ai-model` | `string` | `"openrouter/free"` | Nama model AI (misal: `"gemini-2.0-flash"`, `"deepseek-chat"`, `"gpt-4o-mini"`) |
| **YouTube & Kualitas** | | | |
| `-quality` | `string` | `"best"` | Kualitas unduhan YouTube: `"best"`, `"1080p"`, `"720p"`, `"480p"`, `"360p"`, `"worst"` |
| `-cache-dir` | `string` | `"./cache"` | Direktori tempat menyimpan cache video & transkrip YouTube |
| `-no-cache` | `bool` | `false` | Matikan penggunaan cache (paksa unduh ulang) |
| **Kinerja & Watermark** | | | |
| `-concurrency` | `int` | `CPU cores` | Jumlah worker pemrosesan video secara paralel |
| `-watermark` | `string` | `""` | Path gambar logo watermark (PNG) |
| `-watermark-pos` | `string` | `"top-right"` | Posisi watermark: `"top-right"`, `"top-left"`, `"bottom-right"`, `"bottom-left"`, `"center"` |
| `-text` | `string` | `""` | Teks caption judul overlay pada video |
| `-text-pos` | `string` | `"bottom-center"` | Posisi teks overlay: `"bottom-center"`, `"top-center"`, `"center"` |
| `-font-size` | `int` | `32` | Ukuran font teks overlay caption |
| `-font-color` | `string` | `"white"` | Warna font teks overlay (`"white"`, `"yellow"`, `"cyan"`, `"red"`) |
| **Manual Timestamps & Config** | | | |
| `-segments` | `string` | `""` | Timestamp segmen manual dipisahkan koma (misal: `'00:10-00:25,01:00-01:30'`) |
| `-config` | `string` | `""` | Path ke berkas konfigurasi JSON |
| `-init-config` | `string` | `""` | Buat berkas konfigurasi JSON baru (misal: `-init-config config.json`) |
| `-i` / `-interactive` | `bool` | `false` | Jalankan wizard interaktif generator konfigurasi |

---

## 💡 Contoh Penggunaan Skenario Nyata

### 1. Generasi Viral Shorts 9:16 Blur + TikTok Karaoke Subtitles (Rekomendasi Utama 🏆)
```bash
./clipper -input "https://www.youtube.com/watch?v=t7xtO3KqsmM" \
  -auto-detect ai \
  -shorts \
  -shorts-style blur \
  -burn-subtitles \
  -sub-style karaoke \
  -sub-font-size 54 \
  -translate-lang id \
  -openrouter-key "sk-or-v1-..." \
  -outdir ./clips
```

### 2. Video Lokal Offline dengan Transkripsi Whisper AI
```bash
./clipper -input "./my_recording.mp4" \
  -auto-detect ai \
  -use-whisper \
  -shorts \
  -burn-subtitles \
  -outdir ./whisper_shorts
```

### 3. Pemotongan Manual dengan Watermark & Subtitle Standard
```bash
./clipper -input "./podcast.mp4" \
  -segments "00:01:20-00:02:10,00:05:40-00:06:30" \
  -watermark "./logo.png" \
  -watermark-pos top-right \
  -burn-subtitles \
  -sub-style standard \
  -outdir ./manual_clips
```

### 4. Smart Motion Auto-Crop 9:16 (Fokus Subjek Bergerak)
```bash
./clipper -input "https://youtu.be/..." \
  -auto-detect silence \
  -shorts \
  -shorts-style smart-crop \
  -outdir ./smart_crop_clips
```

### 5. Mode Interaktif Wizard
```bash
./clipper -i
```
