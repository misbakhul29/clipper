# ⚙️ Example Configurations - `config.json`

Dokumentasi dan contoh lengkap berkas konfigurasi `config.json` untuk aplikasi **Clipper** (Automated Video Clipper & Shorts Engine).

---

## 📖 Cara Menggunakan Berkas Konfigurasi

Jalankan perintah berikut untuk mengeksekusi Clipper menggunakan berkas `config.json`:

```bash
clipper -config config.json
```

Atau generate berkas `config.json` baru secara interaktif via wizard CLI:

```bash
clipper -i
```

---

## 🏆 1. Rekomendasi Utama: AI Shorts 9:16 + TikTok Karaoke Subtitles (Google Gemini)

Skenario paling umum untuk mengunduh video YouTube, menganalisis klip menarik menggunakan AI Gemini, konversi ke Shorts 9:16 blurred background, dan membakar subtitle animasi karaoke.

```json
{
  "input": "https://www.youtube.com/watch?v=xid1sE8lEec",
  "output_dir": "./output_shorts",
  "mode": "split",
  "strategy": "accurate",
  "shorts": true,
  "shorts_style": "blur",
  "quality": "1080p",
  "cache_dir": "./cache",
  "no_cache": false,
  "concurrency": 0,
  "auto_detect": "ai",
  "translate_lang": "id",
  "burn_subtitles": true,
  "sub_preset": "hormozi",
  "sub_sdh_mode": "strip",
  "sub_emoji": true,
  "sub_style": "karaoke",
  "sub_font_size": 54,
  "use_whisper": false,
  "generate_metadata": true,
  "extract_thumbnail": true,
  "thumbnail_count": 1,
  "hwaccel": "auto",
  "dry_run": false,
  "ai_config": {
    "api_router": "gemini",
    "api_key": "YOUR_GEMINI_API_KEY",
    "model": "gemini-3.6-flash"
  }
}
```

---

## 🎙️ 2. Video Lokal Offline + Local Whisper AI + Custom Font

Gunakan konfigurasi ini untuk memproses video lokal tanpa subtitle YouTube bawaan, menggunakan **Whisper AI** lokal secara offline dan menyuplai font kustom (`.ttf`/`.otf`).

```json
{
  "input": "./my_local_video.mp4",
  "output_dir": "./output_whisper",
  "mode": "split",
  "strategy": "accurate",
  "shorts": true,
  "shorts_style": "blur",
  "cache_dir": "./cache",
  "auto_detect": "ai",
  "translate_lang": "id",
  "burn_subtitles": true,
  "sub_style": "karaoke",
  "sub_font_size": 48,
  "sub_font_path": "./fonts/Montserrat-Bold.ttf",
  "use_whisper": true,
  "ai_config": {
    "api_router": "gemini",
    "api_key": "YOUR_GEMINI_API_KEY",
    "model": "gemini-2.0-flash"
  }
}
```

---

## 📑 3. Pemrosesan Antrean Banyak Video (Batch Queue)

Gunakan `batch_list` untuk memproses daftar banyak URL video dari berkas `urls.txt` secara otomatis dalam satu antrean.

```json
{
  "input": "",
  "batch_list": "./urls.txt",
  "output_dir": "./output_batch",
  "mode": "split",
  "strategy": "accurate",
  "shorts": true,
  "shorts_style": "blur",
  "auto_detect": "ai",
  "translate_lang": "id",
  "burn_subtitles": true,
  "sub_style": "karaoke",
  "sub_font_size": 54,
  "ai_config": {
    "api_router": "gemini",
    "api_key": "YOUR_GEMINI_API_KEY",
    "model": "gemini-3.6-flash"
  }
}
```

---

## ✂️ 4. Pemotongan Timestamps Manual + Watermark PNG

Gunakan konfigurasi ini jika Anda ingin menentukan timestamp segmen awal dan akhir secara manual serta menambahkan logo watermark PNG.

```json
{
  "input": "https://www.youtube.com/watch?v=OKLWm_i8u9g",
  "output_dir": "./output_manual",
  "mode": "split",
  "strategy": "fast",
  "shorts": true,
  "shorts_style": "crop",
  "watermark": "./assets/logo.png",
  "watermark_pos": "top-right",
  "burn_subtitles": true,
  "sub_style": "standard",
  "sub_font_size": 48,
  "segments": [
    {
      "start": "00:00:10",
      "end": "00:00:30",
      "title": "Klip Pertama"
    },
    {
      "start": "00:01:15",
      "end": "00:01:45",
      "title": "Klip Kedua"
    }
  ]
}
```

---

## 🤖 5. Konfigurasi Provider AI Lainnya (DeepSeek / OpenAI / OpenRouter)

### DeepSeek AI
```json
{
  "auto_detect": "ai",
  "ai_config": {
    "api_router": "deepseek",
    "api_key": "YOUR_DEEPSEEK_API_KEY",
    "model": "deepseek-chat"
  }
}
```

### OpenAI (GPT-4o-mini)
```json
{
  "auto_detect": "ai",
  "ai_config": {
    "api_router": "openai",
    "api_key": "YOUR_OPENAI_API_KEY",
    "model": "gpt-4o-mini"
  }
}
```

### OpenRouter
```json
{
  "auto_detect": "ai",
  "ai_config": {
    "api_router": "openrouter",
    "api_key": "YOUR_OPENROUTER_API_KEY",
    "model": "openrouter/free"
  }
}
```

---

## 📝 Referensi Kamus Parameter JSON

| Field JSON | Tipe | Default | Deskripsi |
| :--- | :--- | :--- | :--- |
| `input` | `string` | `""` | Path berkas video lokal atau URL YouTube |
| `batch_list` | `string` | `""` | Path berkas `.txt` berisi daftar URL/video (satu per baris) |
| `output_dir` | `string` | `"."` | Direktori tujuan hasil klip video |
| `output` | `string` | `""` | Nama berkas gabungan (pada mode `merge`) |
| `mode` | `string` | `"split"` | `"split"` (klip terpisah) atau `"merge"` (gabung 1 video) |
| `strategy` | `string` | `"fast"` | `"fast"` (stream copy) atau `"accurate"` (re-encode presisi) |
| `shorts` | `bool` | `false` | Konversi ke format vertikal 9:16 Shorts/TikTok |
| `shorts_style` | `string` | `"crop"` | `"crop"` (center crop), `"blur"` (blurred background), atau `"smart-crop"` |
| `quality` | `string` | `"best"` | Unduhan YouTube (`"best"`, `"1080p"`, `"720p"`, `"480p"`, `"360p"`, `"worst"`) |
| `cache_dir` | `string` | `"./cache"` | Direktori cache video & subtitle per folder |
| `no_cache` | `bool` | `false` | Matikan penggunaan cache |
| `concurrency` | `int` | `0` | Jumlah worker pemrosesan paralel (`0` = jumlah CPU cores) |
| `watermark` | `string` | `""` | Path gambar watermark PNG |
| `watermark_pos` | `string` | `"top-right"` | Posisi watermark (`"top-right"`, `"top-left"`, `"bottom-right"`, `"bottom-left"`, `"center"`) |
| `overlay_text` | `string` | `""` | Teks caption judul overlay |
| `text_pos` | `string` | `"bottom-center"` | Posisi teks overlay |
| `font_size` | `int` | `32` | Ukuran font teks overlay |
| `font_color` | `string` | `"white"` | Warna font teks overlay |
| `auto_detect` | `string` | `""` | `"ai"` (AI highlights), `"silence"`, atau `"scene"` |
| `translate_lang` | `string` | `"id"` | Bahasa sasaran terjemahan judul & subtitle (misal: `"id"`, `"en"`) |
| `burn_subtitles` | `bool` | `false` | Hardcode subtitle permanen pada video |
| `sub_style` | `string` | `"karaoke"` | `"karaoke"` (animasi 2 kata kuning) atau `"standard"` |
| `sub_font_size` | `int` | `48` | Ukuran font subtitle terjemahan |
| `sub_font_path` | `string` | `""` | Path berkas font kustom (`.ttf`/`.otf`) |
| `use_whisper` | `bool` | `false` | Paksa transkripsi offline menggunakan Whisper AI |
| `dry_run` | `bool` | `false` | Pratinjau segmen tanpa merender video |
| `face_tracking` | `bool` | `true` | Pelacak wajah & pembicara aktif pada mode `smart-crop` |
| `pan_duration` | `float` | `0.8` | Durasi pergeseran kamera halus (pan easing) dalam detik |
| `loudnorm` | `bool` | `false` | Normalisasi audio EBU R128 (-14 LUFS, auto-aktif pada Shorts) |
| `loudnorm_i` | `float` | `-14.0` | Target Integrated Loudness dalam LUFS |
| `loudnorm_lra` | `float` | `7.0` | Target Loudness Range dalam LU |
| `loudnorm_tp` | `float` | `-2.0` | Batas maksimum True Peak dalam dBTP |
| `jump_cut` | `bool` | `false` | Smart silence removal / jump-cut pemotong hening di tengah klip |
| `jump_cut_min_silence` | `float` | `1.0` | Ambang hening minimum dalam detik yang akan dipotong |
| `jump_cut_margin` | `float` | `0.2` | Padding wicara aman di sekitar jeda hening dalam detik |
| `jump_cut_noise` | `float` | `-30.0` | Ambang batas kebisingan hening (noise gate) dalam dB |
| `ai_config` | `object` | `{}` | Konfigurasi AI Provider (`api_router`, `api_key`, `model`) |
| `segments` | `array` | `[]` | Daftar segmen manual (`start`, `end`, `title`) |
