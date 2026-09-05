package clipper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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
		detected, subs, err := c.DetectSegmentsWithSubs(cfg, originalInput)
		if err != nil {
			return err
		}
		cfg.Segments = append(cfg.Segments, detected...)
		if len(subs) > 0 {
			cachedSubEntries = subs
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
	isTargetLang := true
	targetLang := cfg.TranslateLang
	if targetLang == "" {
		targetLang = "id"
	}

	if cfg.Subtitles || cfg.GenerateMetadata {
		if len(cachedSubEntries) > 0 {
			allSubEntries = cachedSubEntries
			fmt.Printf("Reusing %d cached subtitle entries for video clips!\n", len(allSubEntries))
		} else {
			fmt.Printf("Checking subtitles for video clips (target lang: %s)...\n", targetLang)
			if cfg.UseWhisper {
				subs, err := transcriber.TranscribeWithWhisper(cfg.InputFile, cfg.CacheDir, targetLang)
				if err == nil && len(subs) > 0 {
					allSubEntries = subs
					isTargetLang = true
					fmt.Printf("Transcribed %d subtitle entries using Whisper!\n", len(allSubEntries))
				}
			} else {
				subs, matched, err := transcriber.FetchSubtitlesWithLangInfo(originalInput, cfg.CacheDir, targetLang)
				if err == nil && len(subs) > 0 {
					allSubEntries = subs
					isTargetLang = matched
					if isTargetLang {
						fmt.Printf("Loaded %d subtitle entries matching target language '%s'!\n", len(allSubEntries), targetLang)
					} else {
						fmt.Printf("Loaded %d fallback subtitle entries (will translate & refine per clip segment to '%s')\n", len(allSubEntries), targetLang)
					}
				}
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
	if cfg.Subtitles {
		preset := cfg.SubPreset
		if preset == "" {
			preset = "hormozi"
		}
		sdh := cfg.SubSDHMode
		if sdh == "" {
			sdh = "strip"
		}
		emojiStatus := "Enabled"
		if !cfg.SubEmoji {
			emojiStatus = "Disabled"
		}
		fmt.Printf("Burnt-in Subtitles: Enabled (Theme: %s, SDH: %s, Emojis: %s)\n", preset, sdh, emojiStatus)
	}
	if cfg.GenerateMetadata {
		fmt.Println("Social Metadata: Enabled (metadata.json & .txt companion)")
	}
	if cfg.ExtractThumbnail {
		tCount := cfg.ThumbnailCount
		if tCount <= 0 {
			tCount = 1
		}
		fmt.Printf("Thumbnail Extractor: Enabled (Hook cover & clean frames, count: %d)\n", tCount)
	}
	hwProf := DetectHardwareEncoder(c.runner.FFmpegPath, cfg.HWAccel)
	if hwProf.IsHardware {
		fmt.Printf("Hardware Acceleration: %s [5x-10x speedup]\n", hwProf.DisplayName)
	} else {
		fmt.Printf("Hardware Acceleration: Software CPU (%s)\n", hwProf.Encoder)
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
		seg         Segment
		startSec    float64
		durationSec float64
		segPath     string
		isTemp      bool
	}

	type jobResult struct {
		index   int
		segPath string
		isTemp  bool
		meta    *ai.SocialMetadata
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
			seg:         seg,
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
				var segSubEntries []transcriber.SubtitleEntry
				if len(j.seg.Subtitles) > 0 {
					// Use custom segment subtitle cues configured in studio!
					for _, cue := range j.seg.Subtitles {
						segSubEntries = append(segSubEntries, transcriber.SubtitleEntry{
							Start: ai.FormatSecondsToTime(cue.Start),
							End:   ai.FormatSecondsToTime(cue.End),
							Text:  cue.Text,
						})
					}
				} else if len(allSubEntries) > 0 {
					segSubEntries = transcriber.SliceSubtitles(allSubEntries, j.startSec, j.startSec+j.durationSec)
					if len(removedGaps) > 0 {
						segSubEntries = transcriber.AdjustSubtitlesForJumpCuts(segSubEntries, removedGaps)
					}
				} else if cfg.Subtitles {
					// Skenario 3: No YouTube subtitles available at all -> Run STT per clip segment!
					fmt.Printf("[%d/%d] Generating subtitles via per-clip Speech-To-Text (Whisper)...\n", j.index+1, len(cfg.Segments))
					clipSubs, sttErr := transcriber.TranscribeWithWhisper(effectiveInput, cfg.CacheDir, targetLang)
					if sttErr == nil && len(clipSubs) > 0 {
						segSubEntries = clipSubs
					} else if sttErr != nil {
						fmt.Printf("[%d/%d] Per-clip STT warning: %v\n", j.index+1, len(cfg.Segments), sttErr)
					}
				}

				if cfg.Subtitles && len(segSubEntries) > 0 {
					sliced := segSubEntries
					sdhMode := cfg.SubSDHMode
					if sdhMode == "" {
						sdhMode = "strip"
					}
					sliced = transcriber.FilterSDHEntries(sliced, sdhMode)

					// Deterministic cleaning first
					for sIdx := range sliced {
						sliced[sIdx].Text = transcriber.CleanSubtitleGarbage(sliced[sIdx].Text)
					}

					if len(sliced) > 0 {
						// AI Refine and Translate:
						// If isTargetLang is false, translates to targetLang + cleans narrator/garbage.
						// If isTargetLang is true, cleans narrator/garbage and polishes spoken dialogue.
						subAI := cfg.GetAITaskConfig("sub_translate")
						needAI := len(j.seg.Subtitles) == 0 && (subAI.APIKey != "" || os.Getenv("OPENROUTER_API_KEY") != "" || os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("DEEPSEEK_API_KEY") != "" || os.Getenv("OPENAI_API_KEY") != "")
						if needAI {
							needTranslation := !isTargetLang
							actionLabel := "Polishing & cleaning"
							if needTranslation {
								actionLabel = fmt.Sprintf("Translating to '%s' & cleaning", targetLang)
							}
							fmt.Printf("[%d/%d] %s %d subtitle cues via AI (%s / %s)...\n",
								j.index+1, len(cfg.Segments), actionLabel, len(sliced), subAI.APIRouter, subAI.Model)
							refined, refErr := ai.RefineAndTranslateSubtitlesMultiProvider(sliced, subAI, targetLang, needTranslation)
							if refErr != nil {
								fmt.Printf("[AI WARN] Subtitle processing for segment %d (%v), using cleaned local subtitles.\n", j.index+1, refErr)
							} else if len(refined) > 0 {
								sliced = refined
							}
						}

						tmpSubFile := filepath.Join(outputDir, fmt.Sprintf(".sub_%03d.ass", j.index+1))
						fontName := ""
						if cfg.SubFontPath != "" {
							fontName = strings.TrimSuffix(filepath.Base(cfg.SubFontPath), filepath.Ext(cfg.SubFontPath))
						}

						preset := cfg.SubPreset
						if j.seg.SubPreset != "" {
							preset = j.seg.SubPreset
						}
						if preset == "" {
							if cfg.SubStyle == "standard" {
								preset = "minimal"
							} else {
								preset = "hormozi"
							}
						}

						subFontSize := cfg.SubFontSize
						if j.seg.SubFontSize > 0 {
							subFontSize = j.seg.SubFontSize
						}

						subPosition := j.seg.SubPosition
						if subPosition == "" {
							subPosition = "bottom"
						}

						exportErr := transcriber.ExportPresetASSWithPosition(sliced, tmpSubFile, preset, subFontSize, cfg.Shorts, fontName, sdhMode, cfg.SubEmoji, subPosition)
						if exportErr == nil {
							subPath = tmpSubFile
						}
					}
				}

				segCfg := *cfg
				segCfg.InputFile = effectiveInput
				if numWorkers > 1 {
					segCfg.ShowProgress = false
				}
				err := c.runner.CutSegment(&segCfg, effectiveStart, effectiveDuration, j.segPath, subPath)
				if numWorkers > 1 && err == nil {
					fmt.Printf("[%d/%d] Finished rendering: %s\n", j.index+1, len(cfg.Segments), j.segPath)
				}
				if jumpCutTemp != "" {
					_ = os.Remove(jumpCutTemp)
				}
				if subPath != "" {
					_ = os.Remove(subPath)
				}
				var segMeta *ai.SocialMetadata
				if err == nil && cfg.GenerateMetadata && !j.isTemp {
					var transcriptBuilder strings.Builder
					for _, entry := range segSubEntries {
						cleanText, _ := transcriber.ExtractSDHAndSpeech(entry.Text)
						if cleanText != "" {
							transcriptBuilder.WriteString(cleanText)
							transcriptBuilder.WriteString(" ")
						}
					}
					clipTranscript := strings.TrimSpace(transcriptBuilder.String())

					metaAI := cfg.GetAITaskConfig("metadata")
					if metaAI.APIKey != "" || os.Getenv("OPENROUTER_API_KEY") != "" || os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("DEEPSEEK_API_KEY") != "" || os.Getenv("OPENAI_API_KEY") != "" {
						aiMeta, aiErr := ai.GenerateSocialMetadataMultiProvider(clipTranscript, metaAI, cfg.TranslateLang, cfg.Shorts)
						if aiErr == nil && aiMeta != nil {
							segMeta = aiMeta
						}
					}
					if segMeta == nil {
						segMeta = ai.GenerateHeuristicSocialMetadata(clipTranscript, j.seg.Title, cfg.TranslateLang, cfg.Shorts)
					}

					segMeta.SegmentIndex = j.index + 1
					segMeta.StartTime = j.seg.Start
					segMeta.EndTime = j.seg.End
					segMeta.DurationSec = effectiveDuration
					segMeta.VideoFile = filepath.Base(j.segPath)

					baseWithoutExt := strings.TrimSuffix(j.segPath, filepath.Ext(j.segPath))
					metaJSONPath := baseWithoutExt + "_metadata.json"
					metaTXTPath := baseWithoutExt + "_metadata.txt"

					if jsonData, mErr := json.MarshalIndent(segMeta, "", "  "); mErr == nil {
						_ = os.WriteFile(metaJSONPath, jsonData, 0644)
					}
					_ = os.WriteFile(metaTXTPath, []byte(ai.FormatMetadataText(segMeta)), 0644)
				}

				if err == nil && cfg.ExtractThumbnail && !j.isTemp {
					thumbCount := cfg.ThumbnailCount
					if thumbCount <= 0 {
						thumbCount = 1
					}
					if thumbCount > 3 {
						thumbCount = 3
					}
					bestTimes, bErr := detector.FindBestHookFrames(c.runner.FFmpegPath, j.segPath, effectiveDuration, thumbCount)
					if bErr == nil && len(bestTimes) > 0 {
						baseWithoutExt := strings.TrimSuffix(j.segPath, filepath.Ext(j.segPath))
						cleanThumbPath := baseWithoutExt + "_thumb_clean.jpg"
						hookThumbPath := baseWithoutExt + "_thumb_hook.jpg"

						titleText := ""
						if segMeta != nil && segMeta.HookTitle != "" {
							titleText = segMeta.HookTitle
						} else if j.seg.Title != "" {
							titleText = j.seg.Title
						} else if len(segSubEntries) > 0 {
							firstSentence, _ := transcriber.ExtractSDHAndSpeech(segSubEntries[0].Text)
							titleText = strings.TrimSpace(firstSentence)
						}

						// 1. Primary clean frame
						_ = detector.ExtractThumbnail(c.runner.FFmpegPath, j.segPath, bestTimes[0], cleanThumbPath)

						// 2. Hook cover with title overlay
						_ = detector.ExtractThumbnailWithHook(c.runner.FFmpegPath, j.segPath, bestTimes[0], titleText, cfg.Shorts, hookThumbPath)

						// 3. Optional alternative frames
						for tIdx := 1; tIdx < len(bestTimes); tIdx++ {
							altThumbPath := fmt.Sprintf("%s_thumb_%d.jpg", baseWithoutExt, tIdx+1)
							_ = detector.ExtractThumbnail(c.runner.FFmpegPath, j.segPath, bestTimes[tIdx], altThumbPath)
						}
						fmt.Printf("[%d/%d] Extracted cover thumbnail: %s (best expression at %.2fs)\n",
							j.index+1, len(cfg.Segments), filepath.Base(hookThumbPath), bestTimes[0])
					}
				}

				results <- jobResult{
					index:   j.index,
					segPath: j.segPath,
					isTemp:  j.isTemp,
					meta:    segMeta,
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
	var allMetas []*ai.SocialMetadata

	for r := range results {
		if r.err != nil {
			c.cleanupFiles(createdFiles, isTempFiles)
			return fmt.Errorf("failed cutting segment %d: %w", r.index+1, r.err)
		}
		createdFiles[r.index] = r.segPath
		isTempFiles[r.index] = r.isTemp
		if r.meta != nil {
			allMetas = append(allMetas, r.meta)
		}
	}

	if cfg.GenerateMetadata && len(allMetas) > 0 {
		sort.Slice(allMetas, func(i, j int) bool {
			return allMetas[i].SegmentIndex < allMetas[j].SegmentIndex
		})
		summaryFile := filepath.Join(outputDir, "metadata.json")
		if sumData, mErr := json.MarshalIndent(allMetas, "", "  "); mErr == nil {
			_ = os.WriteFile(summaryFile, sumData, 0644)
			fmt.Printf("Generated social metadata companion files in: %s\n", outputDir)
		}
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

// DetectSegments performs automated segment detection (ai, silence, or scene) on the input media and returns the detected segments.
func (c *Clipper) DetectSegments(cfg *Config) ([]Segment, error) {
	segs, _, err := c.DetectSegmentsWithSubs(cfg, cfg.InputFile)
	return segs, err
}

// DetectSegmentsWithSubs performs automated segment detection and returns any cached subtitles retrieved during detection.
func (c *Clipper) DetectSegmentsWithSubs(cfg *Config, originalInput string) ([]Segment, []transcriber.SubtitleEntry, error) {
	if cfg.InputFile == "" {
		return nil, nil, fmt.Errorf("input file is required for segment detection")
	}

	if originalInput == "" {
		originalInput = cfg.InputFile
	}

	// If InputFile is a YouTube URL, ensure it is downloaded to local cache first
	if downloader.IsYouTubeURL(cfg.InputFile) {
		cacheDir := cfg.CacheDir
		if cacheDir == "" {
			cacheDir = "./cache"
		}
		localPath, err := downloader.DownloadYouTubeVideo(cfg.InputFile, cacheDir, cfg.Quality, cfg.NoCache)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to download YouTube video: %w", err)
		}
		cfg.InputFile = localPath
	}

	if _, err := os.Stat(cfg.InputFile); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("input file does not exist: %s", cfg.InputFile)
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.AutoDetect))
	if mode == "" {
		mode = "ai"
	}

	var segments []Segment
	var cachedSubs []transcriber.SubtitleEntry

	switch mode {
	case "ai", "transcript":
		lang := cfg.TranslateLang
		if lang == "" {
			lang = "id"
		}
		var subEntries []transcriber.SubtitleEntry

		// If user explicitly enabled Whisper, try it
		if cfg.UseWhisper {
			subEntries, _ = transcriber.TranscribeWithWhisper(cfg.InputFile, cfg.CacheDir, lang)
		} else {
			// Check if subtitles are already cached or readily available on YouTube
			subEntries, _ = transcriber.FetchSubtitles(originalInput, cfg.CacheDir, lang)
			if len(subEntries) == 0 {
				subEntries, _ = transcriber.FetchSubtitles(originalInput, cfg.CacheDir, "en")
			}
		}

		segAI := cfg.GetAITaskConfig("segment")

		// If subtitles were retrieved, analyze highlights from transcript
		if len(subEntries) > 0 {
			cachedSubs = subEntries
			highlights, err := ai.AnalyzeHighlightsMultiProvider(subEntries, segAI, lang)
			if err != nil {
				fmt.Printf("[AI WARN] Transcript analysis failed: %v. Falling back to metadata...\n", err)
			} else if len(highlights) > 0 {
				for _, h := range highlights {
					segments = append(segments, Segment{
						Start: h.Start,
						End:   h.End,
						Title: h.Title,
					})
				}
				return segments, cachedSubs, nil
			}
		}

		// NO SUBTITLES REQUIRED: Generate smart segment list directly from video duration & metadata!
		durationSec, _ := c.runner.GetVideoDuration(cfg.InputFile)
		if durationSec <= 0 {
			durationSec = 180.0
		}
		title := filepath.Base(cfg.InputFile)
		if originalInput != "" && !strings.HasPrefix(originalInput, "/") {
			title = originalInput
		}

		highlights, aiErr := ai.AnalyzeHighlightsWithoutSubtitles(title, durationSec, segAI, lang)
		if aiErr == nil && len(highlights) > 0 {
			for _, h := range highlights {
				segments = append(segments, Segment{
					Start: h.Start,
					End:   h.End,
					Title: h.Title,
				})
			}
			return segments, nil, nil
		}
		for _, h := range highlights {
			segments = append(segments, Segment{
				Start: h.Start,
				End:   h.End,
				Title: h.Title,
			})
		}
	case "silence":
		detected, err := detector.DetectSilence(c.runner.FFmpegPath, cfg.InputFile, -30, 0.5)
		if err != nil {
			return nil, nil, fmt.Errorf("silence auto detection failed: %w", err)
		}
		for _, d := range detected {
			segments = append(segments, Segment{
				Start: d.Start,
				End:   d.End,
				Title: d.Title,
			})
		}
	case "scene":
		detected, err := detector.DetectScenes(c.runner.FFmpegPath, cfg.InputFile, 0.3)
		if err != nil {
			return nil, nil, fmt.Errorf("scene auto detection failed: %w", err)
		}
		for _, d := range detected {
			segments = append(segments, Segment{
				Start: d.Start,
				End:   d.End,
				Title: d.Title,
			})
		}
	default:
		return nil, nil, fmt.Errorf("unrecognized auto_detect mode '%s', expected 'silence', 'scene', or 'ai'", mode)
	}

	return segments, cachedSubs, nil
}
