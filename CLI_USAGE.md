# 📖 CLI Command Reference - `clipper`

Dokumentasi lengkap baris perintah (*CLI Flags*), konfigurasi **Multi-Provider AI**, dan contoh penggunaan aplikasi **Clipper** (Automated Video Clipper & Shorts Engine).

---

## 🚀 Kompilasi & Instalasi

### 1. Install Langsung dari Repositori Remote GitHub
```bash
go install github.com/misbakhul29/clipper@latest
```

### 2. Build Lokal dari Source Code
```bash
# Build biner lokal di folder proyek (./bin/clipper)
go build -o bin/clipper .

# ATAU Install lokal ke sistem PATH (~/go/bin/clipper)
go install .
```

---

## 📋 Daftar Lengkap Flag CLI (`clipper -help`)

| Flag | Tipe | Nilai Default | Deskripsi |
| :--- | :--- | :--- | :--- |
| `-input` | `string` | `""` | Path berkas video lokal, URL YouTube, atau beberapa URL dipisahkan koma |
| `-outdir` | `string` | `"."` | Direktori tujuan hasil klip video |
| `-output` | `string` | `""` | Nama berkas keluaran (digunakan pada mode `merge`) |
| `-mode` | `string` | `"split"` | Mode operasi: `"split"` (berkas terpisah) atau `"merge"` (gabung 1 berkas) |
| `-strategy` | `string` | `"fast"` | Strategi pemotongan: `"fast"` (stream copy w/o re-encode) atau `"accurate"` (re-encode presisi) |
| **Shorts (9:16) & Face Tracking** | | | |
| `-shorts` | `bool` | `false` | Konversi klip ke format vertikal 9:16 (Shorts/Reels/TikTok) |
| `-shorts-style` | `string` | `"crop"` | Gaya rasio 9:16: `"crop"` (center crop), `"blur"` (latar blur), atau `"smart-crop"` (pelacak wajah aktif) |
| `-face-tracking` | `bool` | `true` | Aktifkan dynamic active speaker / face tracking auto-framing pada mode `smart-crop` |
| `-pan-duration` | `float` | `0.8` | Durasi transisi pergerakan kamera halus (*camera pan easing*) dalam detik |
| **Subtitle & Terjemahan** | | | |
| `-burn-subtitles` / `-subtitles` | `bool` | `false` | Menempelkan (*hardcode*) subtitle terjemahan secara permanen pada video |
| `-sub-style` | `string` | `"karaoke"` | Gaya subtitle: `"karaoke"` (animasi 2-3 kata kuning TikTok) atau `"standard"` (2 baris) |
| `-sub-font-size` | `int` | `48` | Ukuran font subtitle terjemahan |
| `-sub-font-path` | `string` | `""` | Path berkas font kustom (`.ttf`/`.otf`) untuk caption subtitle |
| `-translate-lang` | `string` | `"id"` | Kode bahasa target terjemahan judul & subtitle (misal: `"id"`, `"en"`) |
| `-use-whisper` | `bool` | `false` | Paksa transkripsi suara offline menggunakan **Whisper AI** lokal |
| **Kecerdasan Buatan (AI) & Auto-Detect** | | | |
| `-auto-detect` | `string` | `""` | Mode deteksi segmen otomatis: `"ai"` (AI highlights), `"silence"`, atau `"scene"` |
| `-ai-router` | `string` | `"openrouter"` | Provider AI: `"gemini"`, `"deepseek"`, `"openai"`, atau `"openrouter"` |
| `-ai-key` | `string` | `""` | API Key untuk AI Provider yang dipilih (atau via Environment Variables) |
| `-openrouter-key` | `string` | `""` | API Key OpenRouter (fallback legacy dari env `$OPENROUTER_API_KEY`) |
| `-ai-model` | `string` | `"openrouter/free"` | Nama model AI (misal: `"gemini-3.6-flash"`, `"deepseek-chat"`, `"gpt-4o-mini"`) |
| **Pembersihan Cache & Batch Processing** | | | |
| `-clean-cache` | `bool` | `false` | Bersihkan direktori cache dan keluar |
| `-clean-days` | `int` | `0` | Ambang batas umur cache dalam hari (`0` = hapus seluruh cache) |
| `-batch-list` | `string` | `""` | Path berkas teks berisi daftar URL/file video (satu per baris) untuk antrean otomatis |
| `-dry-run` | `bool` | `false` | Jalankan simulasi pratinjau segmen & perintah FFmpeg tanpa merender video |
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
| **Normalisasi Audio (EBU R128 / Loudnorm)** | | | |
| `-loudnorm` | `bool` | `false` | Standarisasi volume audio ke standar EBU R128 (-14 LUFS, auto-aktif pada `-shorts`) |
| `-loudnorm-i` | `float` | `-14.0` | Target Integrated Loudness dalam LUFS (standar YouTube/TikTok: `-14.0`) |
| `-loudnorm-lra` | `float` | `7.0` | Target rentang loudness (Loudness Range) dalam LU |
| `-loudnorm-tp` | `float` | `-2.0` | Batas maksimum True Peak dalam dBTP untuk mencegah distorsi |
| **Manual Timestamps & Config** | | | |
| `-segments` | `string` | `""` | Timestamp segmen manual dipisahkan koma (misal: `'00:10-00:25,01:00-01:30'`) |
| `-config` | `string` | `""` | Path ke berkas konfigurasi JSON |
| `-init-config` | `string` | `""` | Buat berkas konfigurasi JSON baru (misal: `-init-config config.json`) |
| `-i` / `-interactive` | `bool` | `false` | Jalankan wizard interaktif generator konfigurasi |

---

## 🤖 Konfigurasi Multi-Provider AI & Auto-Detect

Clipper mendukung integrasi **Multi-Provider AI Router** yang dapat disesuaikan dengan kebutuhan Anda. Anda dapat memasukkan API Key melalui flag `-ai-key` atau menggunakan **Environment Variable** OS.

| Provider (`-ai-router`) | Environment Variable Fallback | Model Default | Contoh Model Populer |
| :--- | :--- | :--- | :--- |
| `gemini` | `GEMINI_API_KEY` | `gemini-2.0-flash` | `gemini-3.6-flash`, `gemini-1.5-pro` |
| `deepseek` | `DEEPSEEK_API_KEY` | `deepseek-chat` | `deepseek-chat`, `deepseek-reasoner` |
| `openai` | `OPENAI_API_KEY` | `gpt-4o-mini` | `gpt-4o-mini`, `gpt-4o` |
| `openrouter` | `OPENROUTER_API_KEY` | `openrouter/free` | `openrouter/free`, `anthropic/claude-3.5-sonnet` |

---

### Contoh Konfigurasi Provider AI

#### 1. Google Gemini (Rekomendasi Utama 🚀)
```bash
# Perintah CLI
clipper -input "https://www.youtube.com/watch?v=xxx" \
  -auto-detect ai \
  -ai-router gemini \
  -ai-key "AIzaSyC-np-3N_..." \
  -ai-model "gemini-3.6-flash" \
  -shorts -burn-subtitles -sub-style karaoke
```

*Contoh `config.json`:*
```json
{
  "auto_detect": "ai",
  "ai_config": {
    "api_router": "gemini",
    "api_key": "AIzaSyC-np-3N_...",
    "model": "gemini-3.6-flash"
  }
}
```

#### 2. DeepSeek AI
```bash
# Perintah CLI
export DEEPSEEK_API_KEY="sk-..."
clipper -input "https://www.youtube.com/watch?v=xxx" \
  -auto-detect ai \
  -ai-router deepseek \
  -ai-model "deepseek-chat" \
  -shorts -burn-subtitles
```

*Contoh `config.json`:*
```json
{
  "auto_detect": "ai",
  "ai_config": {
    "api_router": "deepseek",
    "api_key": "sk-...",
    "model": "deepseek-chat"
  }
}
```

#### 3. OpenAI (GPT-4o / GPT-4o-mini)
```bash
# Perintah CLI
export OPENAI_API_KEY="sk-proj-..."
clipper -input "https://www.youtube.com/watch?v=xxx" \
  -auto-detect ai \
  -ai-router openai \
  -ai-model "gpt-4o-mini" \
  -shorts -burn-subtitles
```

#### 4. OpenRouter (Multi-LLM Aggregator)
```bash
# Perintah CLI
export OPENROUTER_API_KEY="sk-or-v1-..."
clipper -input "https://www.youtube.com/watch?v=xxx" \
  -auto-detect ai \
  -ai-router openrouter \
  -ai-model "openrouter/free" \
  -shorts -burn-subtitles
```

---

## 💡 Contoh Penggunaan Skenario Nyata

### 1. Generasi Viral Shorts 9:16 Blur + TikTok Karaoke Subtitles (Google Gemini)
```bash
clipper -input "https://www.youtube.com/watch?v=t7xtO3KqsmM" \
  -auto-detect ai \
  -shorts \
  -shorts-style blur \
  -burn-subtitles \
  -sub-style karaoke \
  -sub-font-size 54 \
  -translate-lang id \
  -ai-router gemini -ai-key "AIzaSy..." \
  -outdir ./clips
```

### 2. Mode Simulasi Dry-Run (Pratinjau Segmen & Perintah FFmpeg)
```bash
clipper -input "https://www.youtube.com/watch?v=xxx" -auto-detect silence -shorts -dry-run
```

### 3. Pemrosesan Antrean Banyak Video (Batch Queue)
```bash
clipper -batch-list my_urls.txt -auto-detect ai -shorts -burn-subtitles -sub-style karaoke
```

### 4. Membersihkan Cache Video & Subtitle
```bash
# Hapus seluruh cache
clipper -clean-cache

# Hapus cache yang umurnya lebih dari 7 hari
clipper -clean-cache -clean-days 7
```

### 5. Video Lokal Offline dengan Transkripsi Whisper AI & Custom Font
```bash
clipper -input "./my_recording.mp4" \
  -auto-detect ai \
  -use-whisper \
  -shorts \
  -burn-subtitles \
  -sub-font-path "./fonts/Montserrat-Bold.ttf" \
  -outdir ./whisper_shorts
```

### 6. Mode Interaktif Wizard
```bash
clipper -i
```

### 7. Vertical Shorts dengan Dynamic Speaker Auto-Framing (Face Tracking)
```bash
clipper -input "https://www.youtube.com/watch?v=xxx" \
  -auto-detect ai \
  -shorts -shorts-style smart-crop \
  -pan-duration 0.8 \
  -burn-subtitles -sub-style karaoke \
  -outdir ./smart_crop_shorts
```

### 8. Auto Audio Normalization (EBU R128 -14 LUFS)
```bash
clipper -input "podcast.mp4" \
  -segments "00:10-00:40" \
  -loudnorm -loudnorm-i -14 \
  -outdir ./normalized_clips
```
