package clipper

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// CutSegment trims a single segment from inputPath to outputPath.
func (f *FFmpegRunner) CutSegment(inputPath string, startSec, durationSec float64, outputPath string, strategy CutStrategy, isShorts bool, shortsStyle string) error {
	startStr := FormatSeconds(startSec)
	durStr := FormatSeconds(durationSec)

	var args []string

	if isShorts {
		// Video filter required for 9:16 Shorts format
		args = []string{
			"-y",
			"-ss", startStr,
			"-i", inputPath,
			"-t", durStr,
		}

		if shortsStyle == "blur" {
			filterComplex := "[0:v]scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920,boxblur=20:5[bg];[0:v]scale=1080:1920:force_original_aspect_ratio=decrease[fg];[bg][fg]overlay=(W-w)/2:(H-h)/2"
			args = append(args, "-filter_complex", filterComplex)
		} else {
			// Center crop 9:16 (default)
			args = append(args, "-vf", "crop=ih*(9/16):ih,scale=1080:1920")
		}

		args = append(args,
			"-c:v", "mpeg4",
			"-c:a", "aac",
			"-avoid_negative_ts", "make_zero",
			outputPath,
		)
	} else if strategy == StrategyFast {
		// Fast copy mode: placing -ss before -i for fast seek, -to/-t after -i
		args = []string{
			"-y",
			"-ss", startStr,
			"-i", inputPath,
			"-t", durStr,
			"-c", "copy",
			"-avoid_negative_ts", "make_zero",
			outputPath,
		}
	} else {
		// Accurate mode: re-encode for frame accuracy
		args = []string{
			"-y",
			"-ss", startStr,
			"-i", inputPath,
			"-t", durStr,
			"-c:v", "mpeg4",
			"-c:a", "aac",
			"-avoid_negative_ts", "make_zero",
			outputPath,
		}
	}

	cmd := exec.Command(f.FFmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg cut failed: %w\nOutput: %s", err, stderr.String())
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
