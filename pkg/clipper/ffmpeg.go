package clipper

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	vEncArgs := selectVideoEncoderArgs(f.FFmpegPath)
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

func isEncoderWorking(ffmpegPath, encoder string) bool {
	cmd := exec.Command(ffmpegPath, "-f", "lavfi", "-i", "color=c=black:s=16x16:d=0.1", "-c:v", encoder, "-f", "null", "-")
	return cmd.Run() == nil
}

func selectVideoEncoderArgs(ffmpegPath string) []string {
	if isEncoderWorking(ffmpegPath, "libx264") {
		return []string{
			"-c:v", "libx264",
			"-crf", "16",
			"-preset", "slow",
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
		}
	}
	if isEncoderWorking(ffmpegPath, "h264_nvenc") {
		return []string{
			"-c:v", "h264_nvenc",
			"-cq", "16",
			"-preset", "p6",
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
		}
	}
	if isEncoderWorking(ffmpegPath, "libopenh264") {
		return []string{
			"-c:v", "libopenh264",
			"-b:v", "20M",
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
		}
	}
	// Fallback to MPEG-4 Ultra-HD Quality Scale (-q:v 1, -b:v 25M, -mbd rd -flags +mv4+aic)
	return []string{
		"-c:v", "mpeg4",
		"-q:v", "1",
		"-b:v", "25M",
		"-mbd", "rd",
		"-flags", "+mv4+aic",
		"-cmp", "2",
		"-subcmp", "2",
		"-movflags", "+faststart",
	}
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
				escapedText := strings.ReplaceAll(cfg.OverlayText, "'", "'\\''")
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
		// Escape single quotes for drawtext
		escapedText := strings.ReplaceAll(cfg.OverlayText, "'", "'\\''")
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

	vEncArgs := selectVideoEncoderArgs(f.FFmpegPath)
	args = append(args, vEncArgs...)
	args = append(args,
		"-c:a", "aac",
		"-b:a", "320k",
		"-ar", "48000",
		outputPath,
	)

	return f.runFFmpeg(args)
}
