package clipper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"clipping/pkg/downloader"
)

// Clipper orchestrates the video processing tasks.
type Clipper struct {
	runner *FFmpegRunner
}

// New creates a new Clipper instance.
func New() (*Clipper, error) {
	runner, err := NewFFmpegRunner("")
	if err != nil {
		return nil, err
	}
	return &Clipper{runner: runner}, nil
}

// Process processes the video according to the provided config.
func (c *Clipper) Process(cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Check if InputFile is a YouTube URL
	if downloader.IsYouTubeURL(cfg.InputFile) {
		fmt.Printf("Detected YouTube URL input: %s\n", cfg.InputFile)
		cacheDir := cfg.CacheDir
		if cacheDir == "" {
			cacheDir = "./cache"
		}
		localPath, err := downloader.DownloadYouTubeVideo(cfg.InputFile, cacheDir, cfg.Quality, cfg.NoCache)
		if err != nil {
			return fmt.Errorf("failed to download YouTube video: %w", err)
		}
		cfg.InputFile = localPath
	}

	if _, err := os.Stat(cfg.InputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", cfg.InputFile)
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = "."
	}

	// Separate output location for Shorts vs non-Shorts clips
	if cfg.Shorts {
		if filepath.Base(filepath.Clean(outputDir)) != "shorts" {
			outputDir = filepath.Join(outputDir, "shorts")
		}
	} else if outputDir == "." {
		outputDir = "clips"
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}

	ext := filepath.Ext(cfg.InputFile)
	baseName := strings.TrimSuffix(filepath.Base(cfg.InputFile), ext)

	var createdFiles []string
	var isTempFile []bool

	fmt.Printf("Starting video processing: %s (%d segments)\n", cfg.InputFile, len(cfg.Segments))
	fmt.Printf("Mode: %s, Strategy: %s, Shorts (9:16): %v (style: %s)\n", cfg.Mode, cfg.Strategy, cfg.Shorts, cfg.ShortsStyle)

	for i, seg := range cfg.Segments {
		startSec, _, durationSec, err := CalculateDuration(seg.Start, seg.End)
		if err != nil {
			return fmt.Errorf("segment %d (%s -> %s) error: %w", i+1, seg.Start, seg.End, err)
		}

		var segFileName string
		title := strings.TrimSpace(seg.Title)
		if title != "" {
			// sanitize title
			title = strings.ReplaceAll(title, " ", "_")
			segFileName = fmt.Sprintf("%s_%s%s", baseName, title, ext)
		} else {
			segFileName = fmt.Sprintf("%s_clip_%03d%s", baseName, i+1, ext)
		}

		segPath := filepath.Join(outputDir, segFileName)

		// If in merge mode, prefix temporary cut clips
		if cfg.Mode == ModeMerge {
			segPath = filepath.Join(outputDir, fmt.Sprintf(".tmp_%s_clip_%03d%s", baseName, i+1, ext))
			isTempFile = append(isTempFile, true)
		} else {
			isTempFile = append(isTempFile, false)
		}

		fmt.Printf("[%d/%d] Cutting segment: %s -> %s (duration: %.2fs) -> %s\n",
			i+1, len(cfg.Segments), seg.Start, seg.End, durationSec, segPath)

		err = c.runner.CutSegment(cfg.InputFile, startSec, durationSec, segPath, cfg.Strategy, cfg.Shorts, cfg.ShortsStyle)
		if err != nil {
			// Clean up any files created so far on failure
			c.cleanupFiles(createdFiles, isTempFile)
			return fmt.Errorf("failed to cut segment %d: %w", i+1, err)
		}

		createdFiles = append(createdFiles, segPath)
	}

	if cfg.Mode == ModeMerge {
		finalOutput := cfg.OutputFile
		if finalOutput == "" {
			finalOutput = filepath.Join(outputDir, fmt.Sprintf("%s_merged%s", baseName, ext))
		} else if !filepath.IsAbs(finalOutput) {
			finalOutput = filepath.Join(outputDir, finalOutput)
		}

		fmt.Printf("Merging %d segments into final video: %s\n", len(createdFiles), finalOutput)
		err := c.runner.MergeSegments(createdFiles, finalOutput)

		// Clean up temporary segment files created for merge
		c.cleanupFiles(createdFiles, isTempFile)

		if err != nil {
			return fmt.Errorf("failed to merge segments: %w", err)
		}
		fmt.Printf("Successfully created merged video: %s\n", finalOutput)
	} else {
		fmt.Printf("Successfully cut %d video segments into directory: %s\n", len(createdFiles), outputDir)
	}

	return nil
}

func (c *Clipper) cleanupFiles(files []string, isTemp []bool) {
	for i, f := range files {
		if i < len(isTemp) && isTemp[i] {
			os.Remove(f)
		}
	}
}
