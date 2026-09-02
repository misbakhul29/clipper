package clipper

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/misbakhul29/clipper/pkg/ai"
	"github.com/misbakhul29/clipper/pkg/detector"
	"github.com/misbakhul29/clipper/pkg/downloader"
	"github.com/misbakhul29/clipper/pkg/transcriber"
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

	if cfg.CleanCache {
		freed, count, err := downloader.CleanCache(cfg.CacheDir, cfg.CleanDays)
		if err != nil {
			return fmt.Errorf("failed to clean cache: %w", err)
		}
		fmt.Printf("Cache cleanup complete: %d items removed, %.2f MB freed\n", count, float64(freed)/(1024*1024))
		return nil
	}

	inputs := cfg.GetBatchInputs()
	if len(inputs) > 1 {
		fmt.Printf("=== Batch Queue Mode Enabled: Processing %d videos ===\n", len(inputs))
		for i, inputPath := range inputs {
			fmt.Printf("\n>>> [Batch %d/%d] Processing video: %s <<<\n", i+1, len(inputs), inputPath)
			subCfg := *cfg
			subCfg.InputFile = inputPath
			subCfg.BatchList = ""
			if err := c.Process(&subCfg); err != nil {
				fmt.Printf("ERROR in batch item %d (%s): %v\n", i+1, inputPath, err)
			}
		}
		fmt.Println("\n=== Batch Queue Execution Completed! ===")
		return nil
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

	var cachedSubEntries []transcriber.SubtitleEntry

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
			cachedSubEntries = subEntries
			cfg.AIConfig.IsShorts = cfg.Shorts
			fmt.Printf("Analyzing %d subtitle entries via AI (%s / %s, is_shorts: %v, target lang: %s)...\n", len(subEntries), cfg.AIConfig.APIRouter, cfg.AIConfig.Model, cfg.Shorts, lang)
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

	ext := filepath.Ext(cfg.InputFile)
	baseName := strings.TrimSuffix(filepath.Base(cfg.InputFile), ext)

	if cfg.DryRun {
		fmt.Printf("\n=== DRY-RUN MODE ENABLED (No video files will be rendered) ===\n")
		fmt.Printf("Input File    : %s\n", cfg.InputFile)
		fmt.Printf("Output Dir    : %s\n", outputDir)
		fmt.Printf("Total Segments: %d\n", len(cfg.Segments))
		fmt.Printf("Mode          : %s, Strategy: %s, Shorts: %v (style: %s)\n", cfg.Mode, cfg.Strategy, cfg.Shorts, cfg.ShortsStyle)
		if cfg.JumpCut {
			minSil := cfg.JumpCutMinSil
			if minSil <= 0 {
				minSil = 1.0
			}
			margin := cfg.JumpCutMargin
			if margin <= 0 {
				margin = 0.2
			}
			fmt.Printf("Jump-Cut      : Enabled (Min Silence: %.1fs, Margin: %.1fs)\n", minSil, margin)
		}
		if cfg.SubFontPath != "" {
			fmt.Printf("Custom Font   : %s\n", cfg.SubFontPath)
		}
		fmt.Println("--------------------------------------------------")
		for i, seg := range cfg.Segments {
			_, _, durSec, _ := CalculateDuration(seg.Start, seg.End)
			outName := fmt.Sprintf("%s_clip_%03d%s", baseName, i+1, ext)
			if seg.Title != "" {
				outName = fmt.Sprintf("%s_%s%s", baseName, sanitizeFilename(seg.Title), ext)
			}
			fmt.Printf("[%d/%d] Segment: %s -> %s (Duration: %.2fs) -> %s\n",
				i+1, len(cfg.Segments), seg.Start, seg.End, durSec, filepath.Join(outputDir, outName))
		}
		fmt.Println("--------------------------------------------------")
		fmt.Println("=== DRY-RUN COMPLETED SUCCESSFULLY ===")
		return nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}

	numWorkers := cfg.Concurrency
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers > len(cfg.Segments) {
		numWorkers = len(cfg.Segments)
	}

	var allSubEntries []transcriber.SubtitleEntry
	if cfg.BurnSubtitles {
		if len(cachedSubEntries) > 0 {
			allSubEntries = cachedSubEntries
			fmt.Printf("Reusing %d cached subtitle entries for burn-in captions!\n", len(allSubEntries))
		} else {
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
		preset := cfg.SubPreset
		if preset == "" {
			preset = "hormozi"
		}
		sdh := cfg.SubSDHMode
		if sdh == "" {
			sdh = "strip"
		}
		fmt.Printf("Burnt-in Subtitles: Enabled (Theme: %s, SDH Narrator: %s)\n", preset, sdh)
	}
	if cfg.Loudnorm {
		targetI := cfg.LoudnormI
		if targetI == 0 {
			targetI = -14.0
		}
		fmt.Printf("Audio Normalization: EBU R128 (Target: %.1f LUFS)\n", targetI)
	}
	if cfg.JumpCut {
		minSil := cfg.JumpCutMinSil
		if minSil <= 0 {
			minSil = 1.0
		}
		margin := cfg.JumpCutMargin
		if margin <= 0 {
			margin = 0.2
		}
		noise := cfg.JumpCutNoise
		if noise == 0 {
			noise = -30.0
		}
		fmt.Printf("Smart Silence Removal (Jump-Cut): Enabled (Min Silence: %.1fs, Margin: %.1fs, Noise: %.1fdB)\n",
			minSil, margin, noise)
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

		var segPath string
		isTemp := false

		if cfg.Mode == ModeMerge {
			segPath = filepath.Join(outputDir, fmt.Sprintf(".temp_seg_%03d%s", i+1, ext))
			isTemp = true
		} else {
			if seg.Title != "" {
				cleanTitle := sanitizeFilename(seg.Title)
				segPath = filepath.Join(outputDir, fmt.Sprintf("%s_%s%s", baseName, cleanTitle, ext))
			} else {
				segPath = filepath.Join(outputDir, fmt.Sprintf("%s_clip_%03d%s", baseName, i+1, ext))
			}
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

	// Worker Pool
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				effectiveInput := cfg.InputFile
				effectiveStart := j.startSec
				effectiveDuration := j.durationSec
				var jumpCutTemp string
				var removedGaps []detector.SilenceGap

				if cfg.JumpCut {
					minSil := cfg.JumpCutMinSil
					if minSil <= 0 {
						minSil = 1.0
					}
					margin := cfg.JumpCutMargin
					if margin <= 0 {
						margin = 0.2
					}
					noise := cfg.JumpCutNoise
					if noise == 0 {
						noise = -30.0
					}

					rawGaps, _ := detector.DetectSilenceGaps(c.runner.FFmpegPath, cfg.InputFile, j.startSec, j.durationSec, noise, minSil)
					keptIntervals, actualRemoved := detector.CalculateJumpCutIntervals(j.durationSec, rawGaps, margin)

					if len(actualRemoved) > 0 && len(keptIntervals) >= 1 {
						var totalRemoved float64
						for _, r := range actualRemoved {
							totalRemoved += r.Duration()
						}
						newDur := j.durationSec - totalRemoved
						fmt.Printf("[%d/%d] Jump-Cut: Excising %d silence pauses (-%.2fs) -> Snappy duration: %.2fs (was %.2fs)\n",
							j.index+1, len(cfg.Segments), len(actualRemoved), totalRemoved, newDur, j.durationSec)

						jumpCutTemp = filepath.Join(outputDir, fmt.Sprintf(".jc_pre_%03d.mp4", j.index+1))
						if jcErr := c.runner.ApplyJumpCut(cfg.InputFile, j.startSec, j.durationSec, keptIntervals, jumpCutTemp); jcErr == nil {
							effectiveInput = jumpCutTemp
							effectiveStart = 0.0
							effectiveDuration = newDur
							removedGaps = actualRemoved
						} else {
							fmt.Printf("[%d/%d] Jump-Cut warning: %v, falling back to original clip\n", j.index+1, len(cfg.Segments), jcErr)
							_ = os.Remove(jumpCutTemp)
							jumpCutTemp = ""
						}
					}
				}

				fmt.Printf("[%d/%d] Cutting segment: %.2fs -> duration %.2fs -> %s\n",
					j.index+1, len(cfg.Segments), effectiveStart, effectiveDuration, j.segPath)
				if cfg.Shorts && cfg.ShortsStyle == "smart-crop" && cfg.FaceTracking {
					fmt.Printf("[%d/%d] Analyzing active speaker & face tracking for smart-crop auto-framing...\n", j.index+1, len(cfg.Segments))
				}

				subPath := ""
				if len(allSubEntries) > 0 {
					sliced := transcriber.SliceSubtitles(allSubEntries, j.startSec, j.startSec+j.durationSec)
					if len(removedGaps) > 0 {
						sliced = transcriber.AdjustSubtitlesForJumpCuts(sliced, removedGaps)
					}
					if len(sliced) > 0 {
						// On-demand token-saving translation for this specific clip segment
						if cfg.TranslateLang != "" && (cfg.AIConfig.APIKey != "" || os.Getenv("OPENROUTER_API_KEY") != "" || os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("DEEPSEEK_API_KEY") != "" || os.Getenv("OPENAI_API_KEY") != "") {
							fmt.Printf("[%d/%d] Translating %d subtitle cues to '%s' via AI (%s / %s)...\n",
								j.index+1, len(cfg.Segments), len(sliced), cfg.TranslateLang, cfg.AIConfig.APIRouter, cfg.AIConfig.Model)
							translated, transErr := ai.TranslateSubtitlesMultiProvider(sliced, cfg.AIConfig, cfg.TranslateLang)
							if transErr != nil {
								fmt.Printf("[AI WARN] Subtitle translation for segment %d failed (%v), using original subtitles.\n", j.index+1, transErr)
							} else if len(translated) > 0 {
								sliced = translated
							}
						}

						tmpSubFile := filepath.Join(outputDir, fmt.Sprintf(".sub_%03d.ass", j.index+1))
						fontName := ""
						if cfg.SubFontPath != "" {
							fontName = strings.TrimSuffix(filepath.Base(cfg.SubFontPath), filepath.Ext(cfg.SubFontPath))
						}

						preset := cfg.SubPreset
						if preset == "" {
							if cfg.SubStyle == "standard" {
								preset = "minimal"
							} else {
								preset = "hormozi"
							}
						}

						sdhMode := cfg.SubSDHMode
						if sdhMode == "" {
							sdhMode = "strip"
						}

						exportErr := transcriber.ExportPresetASS(sliced, tmpSubFile, preset, cfg.SubFontSize, cfg.Shorts, fontName, sdhMode)
						if exportErr == nil {
							subPath = tmpSubFile
						}
					}
				}

				segCfg := *cfg
				segCfg.InputFile = effectiveInput
				err := c.runner.CutSegment(&segCfg, effectiveStart, effectiveDuration, j.segPath, subPath)
				if jumpCutTemp != "" {
					_ = os.Remove(jumpCutTemp)
				}
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
