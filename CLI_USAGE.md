# 📖 Panduan CLI & Perintah - `clipper`

Aplikasi **Clipper** dirancang dengan antarmuka baris perintah (CLI) yang bersih, terstruktur, dan terpusat pada 3 alur utama: **Web Studio (`serve`)**, **Eksekusi Konfigurasi (`config`)**, dan **Inisialisasi Proyek (`init`)**.

---

## 🚀 Perintah Utama (Commands)

| Perintah | Alias | Deskripsi |
| :--- | :--- | :--- |
| `clipper serve [port]` | `clipper -s`, `clipper -serve` | Menjalankan server Web Studio lokal interaktif (default port `:8000`) |
| `clipper config <file.json>` | `clipper -c`, `clipper -config`, `clipper <file.json>` | Menjalankan pemrosesan & rendering klip video langsung dari berkas JSON |
| `clipper init [filename]` | `clipper -i`, `clipper -init`, `clipper init` | Membuat berkas template konfigurasi siap pakai atau wizard interaktif |
| `clipper version` | `clipper -v`, `clipper -version` | Menampilkan versi aplikasi |
| `clipper help` | `clipper -h`, `clipper --help` | Menampilkan ringkasan panduan bantuan |

---

## 1. 🌐 Web Studio (`clipper serve`)

Menjalankan antarmuka grafis Web Studio di browser lokal:
```bash
# Menjalankan di port default :8000
clipper serve

# Menjalankan di port tertentu
clipper serve :8080
clipper -s 3000
```
Fitur Web Studio:
- **Pratinjau Video & Audio Range Scrubbing**: Menggunakan HTML5 video player terintegrasi.
- **Smart Segment Generator**: AI Highlights, Deteksi Suara (Silence), atau Transisi Adegan (Scene).
- **In-Queue Custom Subtitle Studio**: Editor cue teks subtitle dengan Google Gemini Multimodal Audio Speech-to-Text.
- **Impor / Ekspor `config.json`**: Sinkronisasi 100% antara konfigurasi berkas dan antarmuka grafis.

---

## 2. ⚙️ Eksekusi Berkas Konfigurasi (`clipper config`)

Seluruh parameter pemotongan video (Shorts 9:16, Subtitle Presets, Audio Normalization, Jump-Cut, Watermark, Overlay Text, Concurrency) dikelola rapi melalui berkas `config.json`:
```bash
# Menjalankan konfigurasi
clipper config config.json

# Menggunakan alias pendek
clipper -c segments.json

# Langsung menjalankan berkas JSON
clipper my_project.json
```

---

## 3. 📝 Buat Berkas Konfigurasi Baru (`clipper init`)

Untuk membuat berkas konfigurasi baru secara cepat:
```bash
# Membuat template config.json di direktori saat ini
clipper init

# Membuat template dengan nama berkas tertentu
clipper init my_config.json

# Menjalankan panduan interaktif berbasis tanya-jawab di terminal (wizard)
clipper -i
clipper init --wizard
```

---

## ⚙️ Struktur Berkas `config.json`

Contoh berkas konfigurasi lengkap:
```json
{
  "input": "https://www.youtube.com/watch?v=sample_video",
  "output_dir": "./clips",
  "output": "merged_highlight.mp4",
  "mode": "split",
  "strategy": "fast",
  "shorts": true,
  "shorts_style": "smart-crop",
  "quality": "1080p",
  "subtitles": true,
  "sub_preset": "hormozi",
  "sub_sdh_mode": "strip",
  "sub_emoji": true,
  "sub_font_size": 48,
  "loudnorm": true,
  "jump_cut": true,
  "jump_cut_min_silence": 1.0,
  "jump_cut_margin": 0.2,
  "jump_cut_noise": -30.0,
  "generate_metadata": true,
  "extract_thumbnail": true,
  "thumbnail_count": 1,
  "hwaccel": "auto",
  "watermark": "",
  "watermark_pos": "top-right",
  "overlay_text": "",
  "text_pos": "bottom-center",
  "font_size": 32,
  "font_color": "white",
  "auto_detect": "ai",
  "target_duration": 30,
  "translate_lang": "id",
  "ai_config": {
    "api_router": "gemini",
    "api_key": "YOUR_GEMINI_API_KEY",
    "model": "gemini-2.5-flash"
  },
  "segments": [
    {
      "start": "00:00:10",
      "end": "00:00:40",
      "title": "Highlight 1"
    }
  ]
}
```
