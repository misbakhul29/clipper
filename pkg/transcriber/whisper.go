package transcriber

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/misbakhul29/clipper/pkg/downloader"
)

// TranscribeWithWhisper uses local Whisper CLI to transcribe video/audio into []SubtitleEntry
func TranscribeWithWhisper(videoPath, cacheDir, lang string) ([]SubtitleEntry, error) {
	whisperBin, err := exec.LookPath("whisper")
	if err != nil {
		whisperBin, err = exec.LookPath("whisper-cpp")
		if err != nil {
			return nil, fmt.Errorf("local Whisper binary not found on PATH. Please install 'whisper' (pip install openai-whisper) or 'whisper-cpp'")
		}
	}

	videoCacheDir := downloader.GetVideoCacheDir(cacheDir, videoPath)
	if err := os.MkdirAll(videoCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	baseName := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	wavPath := filepath.Join(videoCacheDir, fmt.Sprintf("%s_whisper.wav", baseName))
	vttPath := filepath.Join(videoCacheDir, fmt.Sprintf("%s_whisper.vtt", baseName))

	if fileExists(vttPath) {
		fmt.Printf("[CACHE HIT] Found cached Whisper VTT subtitle: %s\n", vttPath)
		data, err := os.ReadFile(vttPath)
		if err == nil {
			return ParseVTT(string(data))
		}
	}

	// 1. Extract 16kHz Mono WAV using FFmpeg
	ffmpegBin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found on PATH: %w", err)
	}

	fmt.Printf("Extracting audio for Whisper transcription: %s -> %s\n", videoPath, wavPath)
	cmdWav := exec.Command(ffmpegBin, "-y", "-i", videoPath, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", wavPath)
	if out, err := cmdWav.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to extract audio for Whisper: %w. Output: %s", err, string(out))
	}
	defer os.Remove(wavPath)

	// 2. Run Whisper CLI
	fmt.Printf("Transcribing audio via local Whisper AI (%s, lang: %s)...\n", whisperBin, lang)
	args := []string{
		wavPath,
		"--output_format", "vtt",
		"--output_dir", videoCacheDir,
	}
	if lang != "" {
		args = append(args, "--language", lang)
	}

	cmdWhisper := exec.Command(whisperBin, args...)
	if out, err := cmdWhisper.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("Whisper transcription failed: %w. Output: %s", err, string(out))
	}

	// Rename output .vtt if needed
	producedVTT := filepath.Join(videoCacheDir, fmt.Sprintf("%s_whisper.vtt", baseName))
	if !fileExists(producedVTT) {
		// Look for any .vtt produced in videoCacheDir matching baseName
		matches, _ := filepath.Glob(filepath.Join(videoCacheDir, "*.vtt"))
		if len(matches) > 0 {
			producedVTT = matches[0]
		} else {
			return nil, fmt.Errorf("Whisper succeeded but VTT output file was not found")
		}
	}

	data, err := os.ReadFile(producedVTT)
	if err != nil {
		return nil, fmt.Errorf("failed to read produced VTT file: %w", err)
	}
	return ParseVTT(string(data))
}
