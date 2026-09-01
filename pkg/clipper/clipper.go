package clipper

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"clipping/pkg/ai"
	"clipping/pkg/detector"
	"clipping/pkg/downloader"
	"clipping/pkg/transcriber"
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

	originalInput := cfg.InputFile

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

	// Auto Detection if segments are empty
	if len(cfg.Segments) == 0 && cfg.AutoDetect != "" {
		fmt.Printf("Auto-detecting segments using '%s' detection mode...\n", cfg.AutoDetect)
		switch cfg.AutoDetect {
		case "ai", "transcript":
			fmt.Printf("Fetching subtitles for AI analysis...\n")
			lang := cfg.TranslateLang
			if lang == "" {
				lang = "id"
			}
			var subEntries []transcriber.SubtitleEntry
			var err error
			if cfg.UseWhisper {
				subEntries, err = transcriber.TranscribeWithWhisper(cfg.InputFile, cfg.CacheDir, lang)
			} else {
				subEntries, err = transcriber.FetchSubtitles(originalInput, cfg.CacheDir, lang)
				if err != nil || len(subEntries) == 0 {
					fmt.Printf("YouTube subtitles unavailable (%v), falling back to local Whisper AI...\n", err)
					subEntries, err = transcriber.TranscribeWithWhisper(cfg.InputFile, cfg.CacheDir, lang)
				}
			}
			if err != nil {
				return fmt.Errorf("failed to fetch or transcribe subtitles: %w", err)
			}
			fmt.Printf("Analyzing %d subtitle entries via AI (%s / %s, target lang: %s)...\n", len(subEntries), cfg.AIConfig.APIRouter, cfg.AIConfig.Model, lang)
			highlights, err := ai.AnalyzeHighlightsMultiProvider(subEntries, cfg.AIConfig, lang)
			if err != nil {
				return fmt.Errorf("AI highlight analysis failed: %w", err)
			}
			for _, h := range highlights {
				cfg.Segments = append(cfg.Segments, Segment{
					Start: h.Start,
					End:   h.End,
					Title: h.Title,
				})
			}
		case "silence":
			detected, err := detector.DetectSilence(c.runner.FFmpegPath, cfg.InputFile, -30, 0.5)
			if err != nil {
				return fmt.Errorf("silence auto detection failed: %w", err)
			}
			for _, d := range detected {
				cfg.Segments = append(cfg.Segments, Segment{
					Start: d.Start,
					End:   d.End,
					Title: d.Title,
				})
			}
		case "scene":
			detected, err := detector.DetectScenes(c.runner.FFmpegPath, cfg.InputFile, 0.3)
			if err != nil {
				return fmt.Errorf("scene auto detection failed: %w", err)
			}
			for _, d := range detected {
				cfg.Segments = append(cfg.Segments, Segment{
					Start: d.Start,
					End:   d.End,
					Title: d.Title,
				})
			}
		default:
			return fmt.Errorf("unrecognized auto_detect mode '%s', expected 'silence', 'scene', or 'ai'", cfg.AutoDetect)
		}
		fmt.Printf("Auto-detected %d segments!\n", len(cfg.Segments))
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

	numWorkers := cfg.Concurrency
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers > len(cfg.Segments) {
		numWorkers = len(cfg.Segments)
	}

	var allSubEntries []transcriber.SubtitleEntry
	if cfg.BurnSubtitles {
		fmt.Printf("Fetching subtitles for burnt-in captions...\n")
		lang := cfg.TranslateLang
		if lang == "" {
			lang = "id"
		}
		var subs []transcriber.SubtitleEntry
		var err error
		if cfg.UseWhisper {
			subs, err = transcriber.TranscribeWithWhisper(cfg.InputFile, cfg.CacheDir, lang)
		} else {
			subs, err = transcriber.FetchSubtitles(originalInput, cfg.CacheDir, lang)
			if err != nil || len(subs) == 0 {
				subs, err = transcriber.TranscribeWithWhisper(cfg.InputFile, cfg.CacheDir, lang)
			}
		}
		if err == nil && len(subs) > 0 {
			allSubEntries = subs
			fmt.Printf("Loaded %d subtitle entries for burn-in captions!\n", len(allSubEntries))
		}
	}

	fmt.Printf("Starting video processing: %s (%d segments, %d workers)\n", cfg.InputFile, len(cfg.Segments), numWorkers)
	fmt.Printf("Mode: %s, Strategy: %s, Shorts (9:16): %v (style: %s)\n", cfg.Mode, cfg.Strategy, cfg.Shorts, cfg.ShortsStyle)
	if cfg.WatermarkPath != "" {
		fmt.Printf("Watermark: %s (pos: %s)\n", cfg.WatermarkPath, cfg.WatermarkPos)
	}
	if cfg.OverlayText != "" {
		fmt.Printf("Overlay Text: '%s' (pos: %s)\n", cfg.OverlayText, cfg.TextPos)
	}
	if cfg.BurnSubtitles {
		fmt.Printf("Burnt-in Subtitles: Enabled\n")
	}

	type job struct {
		index       int
		startSec    float64
		durationSec float64
		segPath     string
		isTemp      bool
	}

	type jobResult struct {
		index   int
		segPath string
		isTemp  bool
		err     error
	}

	jobs := make(chan job, len(cfg.Segments))
	results := make(chan jobResult, len(cfg.Segments))

	// Populate jobs
	for i, seg := range cfg.Segments {
		startSec, _, durationSec, err := CalculateDuration(seg.Start, seg.End)
		if err != nil {
			return fmt.Errorf("segment %d (%s -> %s) error: %w", i+1, seg.Start, seg.End, err)
		}

		var segFileName string
		title := strings.TrimSpace(seg.Title)
		if title != "" {
			title = strings.ReplaceAll(title, " ", "_")
			segFileName = fmt.Sprintf("%s_%s%s", baseName, title, ext)
		} else {
			segFileName = fmt.Sprintf("%s_clip_%03d%s", baseName, i+1, ext)
		}

		segPath := filepath.Join(outputDir, segFileName)
		isTemp := false
		if cfg.Mode == ModeMerge {
			segPath = filepath.Join(outputDir, fmt.Sprintf(".tmp_%s_clip_%03d%s", baseName, i+1, ext))
			isTemp = true
		}

		jobs <- job{
			index:       i,
			startSec:    startSec,
			durationSec: durationSec,
			segPath:     segPath,
			isTemp:      isTemp,
		}
	}
	close(jobs)

	// Launch worker pool
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				fmt.Printf("[%d/%d] Cutting segment: %.2fs -> duration %.2fs -> %s\n",
					j.index+1, len(cfg.Segments), j.startSec, j.durationSec, j.segPath)

				subPath := ""
				if len(allSubEntries) > 0 {
					sliced := transcriber.SliceSubtitles(allSubEntries, j.startSec, j.startSec+j.durationSec)
					if len(sliced) > 0 {
						if cfg.SubStyle == "karaoke" {
							sliced = transcriber.ChunkSubtitlesToWords(sliced, 3)
						}
						tmpSubFile := filepath.Join(outputDir, fmt.Sprintf(".sub_%03d.ass", j.index+1))
						var exportErr error
						if cfg.SubStyle == "karaoke" {
							exportErr = transcriber.ExportKaraokeASS(sliced, tmpSubFile, cfg.SubFontSize, cfg.Shorts)
						} else {
							exportErr = transcriber.ExportASS(sliced, tmpSubFile, cfg.SubFontSize, cfg.Shorts)
						}
						if exportErr == nil {
							subPath = tmpSubFile
						}
					}
				}

				err := c.runner.CutSegment(cfg, j.startSec, j.durationSec, j.segPath, subPath)
				if subPath != "" {
					_ = os.Remove(subPath)
				}
				results <- jobResult{
					index:   j.index,
					segPath: j.segPath,
					isTemp:  j.isTemp,
					err:     err,
				}
			}
		}()
	}

	wg.Wait()
	close(results)

	// Collect results in order
	createdFiles := make([]string, len(cfg.Segments))
	isTempFiles := make([]bool, len(cfg.Segments))

	for r := range results {
		if r.err != nil {
			c.cleanupFiles(createdFiles, isTempFiles)
			return fmt.Errorf("failed cutting segment %d: %w", r.index+1, r.err)
		}
		createdFiles[r.index] = r.segPath
		isTempFiles[r.index] = r.isTemp
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

		c.cleanupFiles(createdFiles, isTempFiles)

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
		if f != "" && i < len(isTemp) && isTemp[i] {
			os.Remove(f)
		}
	}
}
