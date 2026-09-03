# Clipper — Minimalist AI Video Clipper & Shorts Engine (Golang + FFmpeg)

Sistem pemotong video dan generator Shorts otomatis berbasis **Go (Golang)** dan **FFmpeg**, ditenagai **Google Gemini AI Audio Speech-to-Text** dan **Multi-Provider AI Highlight Detection**.

---

## ⚡ 3 Cara Utama Menggunakan Clipper

Clipper didesain sangat intuitif dengan 3 perintah utama:

### 1. 🌐 Web Studio Dashboard (`clipper serve`)
Jalankan studio visual lengkap dengan video preview player, AI highlight detector, in-queue custom subtitle editor, dan rendering engine:
```bash
clipper serve
# atau tentukan port kustom
clipper serve :8080
# atau gunakan shortcut
clipper -s
```
Buka **http://localhost:8000** di browser Anda.

---

### 2. ⚙️ Eksekusi via Berkas Konfigurasi (`clipper config`)
Jalankan proses pemotongan dan rendering video langsung dari berkas JSON:
```bash
clipper config config.json
# atau gunakan shortcut
clipper -c segments.json
# atau langsung sebut nama filenya
clipper my_project.json
```

---

### 3. 📝 Buat Berkas Konfigurasi Baru (`clipper init`)
Buat template `config.json` siap pakai atau jalankan panduan interaktif:
```bash
# Membuat template config.json siap pakai
clipper init
clipper init my_config.json

# Menjalankan panduan interaktif (wizard)
clipper -i
clipper init --wizard
```

---

## 🚀 Fitur Unggulan

- 🌐 **Interactive Web Studio**: HTML5 video trimmer, pratinjau real-time, import/export `config.json`.
- 🎙️ **Gemini Multimodal Speech-to-Text (STT)**: Transkripsi audio langsung ke subtitle akurat per-detik via Google Gemini API tanpa beban CPU/RAM.
- 🎨 **Viral Subtitle Theme Presets**:
  - `hormozi` (Pop-in bounce kuning viral)
  - `minimal` (Clean white ala Ali Abdaal / Devon)
  - `neon` (Electric cyan glowing magenta)
  - `cinematic` (Soft ivory classic)
- 👤 **Smart-Crop (Active Speaker Tracking)**: Konversi otomatis ke vertikal 9:16 dengan pelacak wajah dan panning kamera halus.
- ✂️ **Smart Silence Removal (Jump-Cut)**: Memotong jeda hening di tengah video secara otomatis.
- 🔊 **EBU R128 Audio Normalization**: Penyetaraan volume standar broadcast (-14 LUFS).
- 📋 **AI Social Metadata & Thumbnail Extractor**: Menghasilkan judul hook, deskripsi, hashtags, dan cover JPG otomatis.
- ⚡ **Hardware Acceleration Auto-Detect**: Dukungan otomatis untuk NVIDIA NVENC, Apple VideoToolbox, Intel QSV, dan CPU.

---

## 📦 Instalasi

### Install via Go
```bash
go install github.com/misbakhul29/clipper@latest
```

### Build dari Source Code
```bash
git clone https://github.com/misbakhul29/clipper.git
cd clipper
go build -o bin/clipper .
```

---

## 📖 Referensi Contoh `config.json`

Lihat [examples/segments.json](examples/segments.json) untuk contoh lengkap seluruh opsi konfigurasi.
