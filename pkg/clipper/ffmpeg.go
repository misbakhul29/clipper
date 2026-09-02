package clipper

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/misbakhul29/clipper/pkg/detector"
)

// FFmpegRunner handles invoking the ffmpeg executable.
type FFmpegRunner struct {
	FFmpegPath string
}

// NewFFmpegRunner creates a new runner, discovering ffmpeg in system PATH if empty.
func NewFFmpegRunner(customPath string) (*FFmpegRunner, error) {
	path := customPath
	if path == "" {
		p, err := exec.LookPath("ffmpeg")
		if err != nil {
			return nil, fmt.Errorf("ffmpeg not found in system PATH: %w. Please install ffmpeg", err)
		}
		path = p
	}
	return &FFmpegRunner{FFmpegPath: path}, nil
}

// CutSegment trims a single segment from inputPath to outputPath based on config settings and optional subtitle ASS path.
func (f *FFmpegRunner) CutSegment(cfg *Config, startSec, durationSec float64, outputPath string, subPath string) error {
	startStr := FormatSeconds(startSec)
	durStr := FormatSeconds(durationSec)

	hasWatermark := cfg.WatermarkPath != ""
	hasOverlayText := cfg.OverlayText != ""
	hasSubtitles := subPath != ""
	needsReencode := cfg.Shorts || hasWatermark || hasOverlayText || hasSubtitles || cfg.Strategy == StrategyAccurate || cfg.Loudnorm

	if !needsReencode {
		// Fast copy mode without re-encoding
		args := []string{
			"-y",
			"-ss", startStr,
			"-i", cfg.InputFile,
			"-t", durStr,
			"-c", "copy",
			"-avoid_negative_ts", "make_zero",
			outputPath,
		}
		return f.runFFmpeg(args)
	}

	// Re-encoding mode with optional video filters
	args := []string{
		"-y",
		"-ss", startStr,
		"-i", cfg.InputFile,
	}

	if hasWatermark {
		args = append(args, "-i", cfg.WatermarkPath)
	}

	args = append(args, "-t", durStr)

	dynamicCropFilter := ""
	if cfg.Shorts && cfg.ShortsStyle == "smart-crop" && cfg.FaceTracking {
		ft := detector.NewFaceTracker(f.FFmpegPath)
		if cfg.PanDuration > 0 {
			ft.PanDuration = cfg.PanDuration
		}
		if cropFilter, err := ft.TrackFacesInSegment(cfg.InputFile, startSec, durationSec); err == nil && cropFilter != "" {
			dynamicCropFilter = cropFilter
		}
	}

	filterGraph := buildFilterGraph(cfg, hasWatermark, hasOverlayText, subPath, dynamicCropFilter)
	if filterGraph != "" {
		if hasWatermark || cfg.ShortsStyle == "blur" {
			args = append(args, "-filter_complex", filterGraph)
		} else {
			args = append(args, "-vf", filterGraph)
		}
	}

	vEncArgs := selectVideoEncoderArgs(f.FFmpegPath, cfg.HWAccel)
	args = append(args, vEncArgs...)

	if cfg.Loudnorm {
		args = append(args, "-af", BuildLoudnormFilter(cfg))
	}

	args = append(args,
		"-c:a", "aac",
		"-b:a", "320k",
		"-ar", "48000",
		"-avoid_negative_ts", "make_zero",
		outputPath,
	)

	return f.runFFmpeg(args)
}

// HWAccelProfile represents a probed and validated hardware encoder configuration.
type HWAccelProfile struct {
	Name        string   `json:"name"`
	Encoder     string   `json:"encoder"`
	DisplayName string   `json:"display_name"`
	IsHardware  bool     `json:"is_hardware"`
	Args        []string `json:"args"`
}

var (
	hwCacheMu  sync.RWMutex
	hwCacheMap = make(map[string]HWAccelProfile)
)

func isEncoderWorking(ffmpegPath, encoder string) bool {
	cmd := exec.Command(ffmpegPath, "-f", "lavfi", "-i", "color=c=black:s=64x64:d=0.1", "-c:v", encoder, "-f", "null", "-")
	return cmd.Run() == nil
}

// DetectHardwareEncoder probes FFmpeg for available hardware-accelerated video encoders.
func DetectHardwareEncoder(ffmpegPath, preferredMode string) HWAccelProfile {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	preferredMode = strings.ToLower(strings.TrimSpace(preferredMode))
	if preferredMode == "" {
		preferredMode = "auto"
	}

	cacheKey := ffmpegPath + ":" + preferredMode
	hwCacheMu.RLock()
	if prof, ok := hwCacheMap[cacheKey]; ok {
		hwCacheMu.RUnlock()
		return prof
	}
	hwCacheMu.RUnlock()

	hwCacheMu.Lock()
	defer hwCacheMu.Unlock()

	// Double check after lock
	if prof, ok := hwCacheMap[cacheKey]; ok {
		return prof
	}

	profiles := map[string]HWAccelProfile{
		"nvenc": {
			Name:        "nvenc",
			Encoder:     "h264_nvenc",
			DisplayName: "NVIDIA NVENC (h264_nvenc)",
			IsHardware:  true,
			Args: []string{
				"-c:v", "h264_nvenc",
				"-preset", "p6",
				"-tune", "hq",
				"-cq", "18",
				"-pix_fmt", "yuv420p",
				"-movflags", "+faststart",
			},
		},
		"videotoolbox": {
			Name:        "videotoolbox",
			Encoder:     "h264_videotoolbox",
			DisplayName: "Apple Silicon VideoToolbox (h264_videotoolbox)",
			IsHardware:  true,
			Args: []string{
				"-c:v", "h264_videotoolbox",
				"-q:v", "65",
				"-pix_fmt", "yuv420p",
				"-movflags", "+faststart",
			},
		},
		"qsv": {
			Name:        "qsv",
			Encoder:     "h264_qsv",
			DisplayName: "Intel QuickSync (h264_qsv)",
			IsHardware:  true,
			Args: []string{
				"-c:v", "h264_qsv",
				"-preset", "veryfast",
				"-global_quality", "20",
				"-pix_fmt", "yuv420p",
				"-movflags", "+faststart",
			},
		},
		"amf": {
			Name:        "amf",
			Encoder:     "h264_amf",
			DisplayName: "AMD AMF (h264_amf)",
			IsHardware:  true,
			Args: []string{
				"-c:v", "h264_amf",
				"-quality", "quality",
				"-rc", "cqp",
				"-qp_i", "18",
				"-qp_p", "18",
				"-pix_fmt", "yuv420p",
				"-movflags", "+faststart",
			},
		},
		"vaapi": {
			Name:        "vaapi",
			Encoder:     "h264_vaapi",
			DisplayName: "Linux VA-API (h264_vaapi)",
			IsHardware:  true,
			Args: []string{
				"-c:v", "h264_vaapi",
				"-qp", "18",
				"-movflags", "+faststart",
			},
		},
		"libx264": {
			Name:        "libx264",
			Encoder:     "libx264",
			DisplayName: "Software CPU (libx264)",
			IsHardware:  false,
			Args: []string{
				"-c:v", "libx264",
				"-crf", "16",
				"-preset", "slow",
				"-pix_fmt", "yuv420p",
				"-movflags", "+faststart",
			},
		},
		"libopenh264": {
			Name:        "libopenh264",
			Encoder:     "libopenh264",
			DisplayName: "Software CPU (libopenh264)",
			IsHardware:  false,
			Args: []string{
				"-c:v", "libopenh264",
				"-b:v", "20M",
				"-pix_fmt", "yuv420p",
				"-movflags", "+faststart",
			},
		},
		"mpeg4": {
			Name:        "mpeg4",
			Encoder:     "mpeg4",
			DisplayName: "Universal Fallback (mpeg4)",
			IsHardware:  false,
			Args: []string{
				"-c:v", "mpeg4",
				"-q:v", "1",
				"-b:v", "25M",
				"-mbd", "rd",
				"-flags", "+mv4+aic",
				"-cmp", "2",
				"-subcmp", "2",
				"-movflags", "+faststart",
			},
		},
	}

	cpuFallbacks := []string{"libx264", "libopenh264", "mpeg4"}

	// 1. User forced a specific hardware mode
	if prof, ok := profiles[preferredMode]; ok && prof.IsHardware {
		if isEncoderWorking(ffmpegPath, prof.Encoder) {
			hwCacheMap[cacheKey] = prof
			return prof
		}
		// Preferred hardware encoder not functional on host -> fallback to CPU
		fmt.Printf("Hardware encoder '%s' requested but not functional on this host. Falling back to CPU.\n", prof.DisplayName)
	}

	// 2. User forced CPU mode
	if preferredMode == "cpu" || preferredMode == "none" {
		for _, name := range cpuFallbacks {
			if isEncoderWorking(ffmpegPath, profiles[name].Encoder) {
				hwCacheMap[cacheKey] = profiles[name]
				return profiles[name]
			}
		}
		hwCacheMap[cacheKey] = profiles["mpeg4"]
		return profiles["mpeg4"]
	}

	// 3. Auto-detection: probe hardware encoders first in priority order
	hwPriority := []string{"nvenc", "videotoolbox", "qsv", "amf", "vaapi"}
	for _, name := range hwPriority {
		prof := profiles[name]
		if isEncoderWorking(ffmpegPath, prof.Encoder) {
			hwCacheMap[cacheKey] = prof
			return prof
		}
	}

	// 4. Fallback to CPU encoders
	for _, name := range cpuFallbacks {
		prof := profiles[name]
		if isEncoderWorking(ffmpegPath, prof.Encoder) {
			hwCacheMap[cacheKey] = prof
			return prof
		}
	}

	hwCacheMap[cacheKey] = profiles["mpeg4"]
	return profiles["mpeg4"]
}

func selectVideoEncoderArgs(ffmpegPath, hwaccelMode string) []string {
	prof := DetectHardwareEncoder(ffmpegPath, hwaccelMode)
	return prof.Args
}

func buildFilterGraph(cfg *Config, hasWatermark, hasOverlayText bool, subPath string, dynamicCropFilter string) string {
	var filters []string

	// 1. Shorts Aspect Ratio Filter (High Quality Lanczos Resampling)
	if cfg.Shorts {
		if cfg.ShortsStyle == "blur" {
			graph := "[0:v]scale=1080:1920:force_original_aspect_ratio=increase:flags=lanczos,crop=1080:1920,boxblur=20:5[bg];[0:v]scale=1080:1920:force_original_aspect_ratio=decrease:flags=lanczos[fg];[bg][fg]overlay=(W-w)/2:(H-h)/2"

			if subPath != "" {
				escapedSub := strings.ReplaceAll(subPath, "\\", "/")
				escapedSub = strings.ReplaceAll(escapedSub, ":", "\\:")
				graph = fmt.Sprintf("%s,subtitles='%s'", graph, escapedSub)
			}

			if hasOverlayText {
				textPos := getTextPosition(cfg.TextPos)
				fontColor := cfg.FontColor
				if fontColor == "" {
					fontColor = "white"
				}
				fontSize := cfg.FontSize
				if fontSize <= 0 {
					fontSize = 32
				}
				escapedText := escapeDrawtext(cfg.OverlayText)
				drawtext := fmt.Sprintf("drawtext=text='%s':%s:fontsize=%d:fontcolor=%s:box=1:boxcolor=black@0.5:boxborderw=5",
					escapedText, textPos, fontSize, fontColor)
				graph = fmt.Sprintf("%s,%s", graph, drawtext)
			}

			if hasWatermark {
				watermarkOverlay := getWatermarkPosition(cfg.WatermarkPos)
				graph = fmt.Sprintf("%s[vbase];[1:v]scale=150:-1[wm];[vbase][wm]%s", graph, watermarkOverlay)
			}
			return graph
		} else if cfg.ShortsStyle == "smart-crop" {
			// Smart Subject Motion Auto-Crop with Face & Speaker Tracking
			if dynamicCropFilter != "" {
				filters = append(filters, dynamicCropFilter)
			} else {
				filters = append(filters, detector.DefaultCenterCropFilter())
			}
		} else {
			filters = append(filters, "crop=ih*(9/16):ih,scale=1080:1920:flags=lanczos")
		}
	}

	// 2. Burnt-In Subtitles (subtitles filter)
	if subPath != "" {
		escapedSub := strings.ReplaceAll(subPath, "\\", "/")
		escapedSub = strings.ReplaceAll(escapedSub, ":", "\\:")
		filters = append(filters, fmt.Sprintf("subtitles='%s'", escapedSub))
	}

	// 3. Overlay Text (drawtext)
	if hasOverlayText {
		textPos := getTextPosition(cfg.TextPos)
		fontColor := cfg.FontColor
		if fontColor == "" {
			fontColor = "white"
		}
		fontSize := cfg.FontSize
		if fontSize <= 0 {
			fontSize = 32
		}
		escapedText := escapeDrawtext(cfg.OverlayText)
		drawtext := fmt.Sprintf("drawtext=text='%s':%s:fontsize=%d:fontcolor=%s:box=1:boxcolor=black@0.5:boxborderw=5",
			escapedText, textPos, fontSize, fontColor)
		filters = append(filters, drawtext)
	}

	// Simple filter chain string
	simpleChain := strings.Join(filters, ",")

	if hasWatermark {
		wmOverlay := getWatermarkPosition(cfg.WatermarkPos)
		if simpleChain != "" {
			return fmt.Sprintf("[0:v]%s[v0];[1:v]scale=150:-1[wm];[v0][wm]%s", simpleChain, wmOverlay)
		}
		return fmt.Sprintf("[1:v]scale=150:-1[wm];[0:v][wm]%s", wmOverlay)
	}

	return simpleChain
}

func getWatermarkPosition(pos string) string {
	switch strings.ToLower(pos) {
	case "top-left":
		return "overlay=20:20"
	case "bottom-left":
		return "overlay=20:main_h-overlay_h-20"
	case "bottom-right":
		return "overlay=main_w-overlay_w-20:main_h-overlay_h-20"
	case "center":
		return "overlay=(main_w-overlay_w)/2:(main_h-overlay_h)/2"
	default: // "top-right"
		return "overlay=main_w-overlay_w-20:20"
	}
}

func getTextPosition(pos string) string {
	switch strings.ToLower(pos) {
	case "top-left":
		return "x=20:y=20"
	case "top-center":
		return "x=(w-text_w)/2:y=30"
	case "top-right":
		return "x=w-text_w-20:y=20"
	case "center":
		return "x=(w-text_w)/2:y=(h-text_h)/2"
	case "bottom-left":
		return "x=20:y=h-text_h-30"
	case "bottom-right":
		return "x=w-text_w-20:y=h-text_h-30"
	default: // "bottom-center"
		return "x=(w-text_w)/2:y=h-text_h-50"
	}
}

func (f *FFmpegRunner) runFFmpeg(args []string) error {
	cmd := exec.Command(f.FFmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %w\nOutput: %s", err, stderr.String())
	}
	return nil
}

// MergeSegments concatenates multiple video files into a single outputFile using FFmpeg concat demuxer.
func (f *FFmpegRunner) MergeSegments(segmentFiles []string, outputFile string) error {
	if len(segmentFiles) == 0 {
		return fmt.Errorf("no segment files provided for merging")
	}

	// Create temporary file list for concat demuxer
	tmpDir := filepath.Dir(outputFile)
	listFile, err := os.CreateTemp(tmpDir, "concat_*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp concat file list: %w", err)
	}
	listPath := listFile.Name()
	defer os.Remove(listPath)

	var listContent string
	for _, file := range segmentFiles {
		absPath, err := filepath.Abs(file)
		if err != nil {
			absPath = file
		}
		listContent += fmt.Sprintf("file '%s'\n", absPath)
	}

	if _, err := listFile.WriteString(listContent); err != nil {
		listFile.Close()
		return fmt.Errorf("failed to write to concat file list: %w", err)
	}
	listFile.Close()

	// Run ffmpeg concat demuxer
	args := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-c", "copy",
		outputFile,
	}

	cmd := exec.Command(f.FFmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg merge failed: %w\nOutput: %s", err, stderr.String())
	}

	return nil
}

// BuildLoudnormFilter constructs the FFmpeg loudnorm audio filter string.
func BuildLoudnormFilter(cfg *Config) string {
	targetI := cfg.LoudnormI
	if targetI == 0 {
		targetI = -14.0
	}
	targetLRA := cfg.LoudnormLRA
	if targetLRA == 0 {
		targetLRA = 7.0
	}
	targetTP := cfg.LoudnormTP
	if targetTP == 0 {
		targetTP = -2.0
	}
	return fmt.Sprintf("loudnorm=I=%.1f:LRA=%.1f:TP=%.1f", targetI, targetLRA, targetTP)
}

// ApplyJumpCut trims and concatenates kept speech intervals from inputFile to outputPath.
func (f *FFmpegRunner) ApplyJumpCut(inputFile string, startSec, durationSec float64, intervals []detector.KeptInterval, outputPath string) error {
	filterGraph := detector.BuildJumpCutFilter(intervals)
	if filterGraph == "" {
		return fmt.Errorf("empty jump-cut filter graph")
	}

	startStr := FormatSeconds(startSec)
	durStr := FormatSeconds(durationSec)

	args := []string{
		"-y",
		"-ss", startStr,
		"-t", durStr,
		"-i", inputFile,
		"-filter_complex", filterGraph,
		"-map", "[vout]",
		"-map", "[aout]",
	}

	vEncArgs := selectVideoEncoderArgs(f.FFmpegPath, "")
	args = append(args, vEncArgs...)
	args = append(args,
		"-c:a", "aac",
		"-b:a", "320k",
		"-ar", "48000",
		outputPath,
	)

	return f.runFFmpeg(args)
}

func escapeDrawtext(text string) string {
	text = strings.ReplaceAll(text, "\\", "/")
	text = strings.ReplaceAll(text, "'", "'\\''")
	text = strings.ReplaceAll(text, ":", "\\:")
	text = strings.ReplaceAll(text, "%", "％")
	return text
}
