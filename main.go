package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/misbakhul29/clipper/pkg/clipper"
)

func main() {
	var (
		configFile    string
		initConfig    string
		interactive   bool
		inputFile     string
		outputDir     string
		outputFile    string
		modeStr       string
		stratStr      string
		isShorts      bool
		shortsStyle   string
		quality       string
		cacheDir      string
		noCache       bool
		concurrency   int
		watermarkPath string
		watermarkPos  string
		overlayText   string
		textPos       string
		fontSize      int
		fontColor     string
		autoDetect    string
		translateLang string
		burnSubtitles bool
		subStyle      string
		subFontSize   int
		subFontPath   string
		useWhisper    bool
		dryRun        bool
		batchList     string
		cleanCache    bool
		cleanDays     int
		openRouterKey string
		aiModel       string
		segments      string
	)

	flag.StringVar(&configFile, "config", "", "Path to JSON configuration file")
	flag.StringVar(&initConfig, "init-config", "", "Generate a JSON configuration file (e.g. -init-config config.json)")
	flag.BoolVar(&interactive, "i", false, "Run interactive config generator wizard")
	flag.BoolVar(&interactive, "interactive", false, "Run interactive config generator wizard")

	flag.StringVar(&inputFile, "input", "", "Path to source input video file, YouTube URL, or comma-separated URLs")
	flag.StringVar(&outputDir, "outdir", ".", "Output directory for cut videos")
	flag.StringVar(&outputFile, "output", "", "Output filename (used in merge mode)")
	flag.StringVar(&modeStr, "mode", "split", "Operation mode: 'split' (separate files) or 'merge' (combined file)")
	flag.StringVar(&stratStr, "strategy", "fast", "Cut strategy: 'fast' (stream copy) or 'accurate' (re-encode)")
	flag.BoolVar(&isShorts, "shorts", false, "Convert cut videos to 9:16 Shorts/Reels/TikTok format")
	flag.StringVar(&shortsStyle, "shorts-style", "crop", "Shorts aspect ratio style: 'crop' (center crop 9:16), 'blur' (blurred background), or 'smart-crop'")
	flag.StringVar(&quality, "quality", "best", "YouTube download quality ('best', '1080p', '720p', '480p', '360p', 'worst')")
	flag.StringVar(&cacheDir, "cache-dir", "./cache", "Directory for caching downloaded YouTube videos")
	flag.BoolVar(&noCache, "no-cache", false, "Disable YouTube download cache and force re-download")
	flag.IntVar(&concurrency, "concurrency", 0, "Number of parallel workers for rendering clips (default: CPU cores)")
	flag.StringVar(&watermarkPath, "watermark", "", "Path to watermark image file (PNG)")
	flag.StringVar(&watermarkPos, "watermark-pos", "top-right", "Watermark position ('top-right', 'top-left', 'bottom-right', 'bottom-left', 'center')")
	flag.StringVar(&overlayText, "text", "", "Text caption to render on video clips")
	flag.StringVar(&textPos, "text-pos", "bottom-center", "Text caption position ('bottom-center', 'top-center', 'center', 'top-left', 'bottom-left')")
	flag.IntVar(&fontSize, "font-size", 32, "Font size for text caption")
	flag.StringVar(&fontColor, "font-color", "white", "Font color for text caption ('white', 'yellow', 'cyan', 'red')")
	flag.StringVar(&autoDetect, "auto-detect", "", "Smart auto-detection mode for segments ('silence', 'scene', or 'ai')")
	flag.StringVar(&translateLang, "translate-lang", "id", "Target language for AI titles and subtitle translation ('id', 'en', etc.)")
	flag.BoolVar(&burnSubtitles, "burn-subtitles", false, "Hardcode/burn-in subtitles directly onto video clips")
	flag.BoolVar(&burnSubtitles, "subtitles", false, "Hardcode/burn-in subtitles directly onto video clips")
	flag.StringVar(&subStyle, "sub-style", "karaoke", "Subtitle style for burnt-in captions ('karaoke' for TikTok 2-word chunks, or 'standard')")
	flag.IntVar(&subFontSize, "sub-font-size", 48, "Subtitle font size for burnt-in captions")
	flag.StringVar(&subFontPath, "sub-font-path", "", "Custom font file path (.ttf / .otf) for burnt-in captions")
	flag.BoolVar(&useWhisper, "use-whisper", false, "Force local Whisper AI for speech-to-text transcription")
	flag.BoolVar(&dryRun, "dry-run", false, "Analyze segments and preview commands without rendering video files")
	flag.StringVar(&batchList, "batch-list", "", "Path to text file containing video URLs/files (one per line)")
	flag.BoolVar(&cleanCache, "clean-cache", false, "Clean cache directory and exit")
	flag.IntVar(&cleanDays, "clean-days", 0, "Retention threshold in days for cache cleanup (0 = delete all)")
	var aiRouter, aiKey string
	flag.StringVar(&aiRouter, "ai-router", "openrouter", "AI API Provider ('openrouter', 'gemini', 'deepseek', 'openai')")
	flag.StringVar(&aiKey, "ai-key", "", "API Key for selected AI router (e.g. Gemini/DeepSeek/OpenAI key)")
	flag.StringVar(&openRouterKey, "openrouter-key", "", "OpenRouter API Key for AI highlight detection (defaults to $OPENROUTER_API_KEY)")
	flag.StringVar(&aiModel, "ai-model", "openrouter/free", "AI model name (e.g. 'openrouter/free', 'gemini-2.0-flash', 'deepseek-chat', 'gpt-4o-mini')")
	flag.StringVar(&segments, "segments", "", "Comma-separated segment timestamps (e.g. '00:10-00:25,01:00-01:30')")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Automated Video Cutting System in Go (Supports Local Videos, YouTube URLs, Shorts 9:16 & Auto-Detect)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  1. Generate JSON Config via Interactive Wizard:\n")
		fmt.Fprintf(os.Stderr, "     %s -i\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  2. Generate JSON Config via Flags:\n")
		fmt.Fprintf(os.Stderr, "     %s -init-config my_config.json -input video.mp4 -segments \"00:10-00:25\" -shorts\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  3. Process Video using JSON Config:\n")
		fmt.Fprintf(os.Stderr, "     %s -config my_config.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  4. Smart Auto Silence Detection + Parallel Workers:\n")
		fmt.Fprintf(os.Stderr, "     %s -input video.mp4 -auto-detect silence -concurrency 4 -shorts -outdir ./shorts_silence\n", os.Args[0])
	}

	flag.Parse()

	// Handle Interactive Config Generator (-i / -interactive)
	if interactive {
		targetFile := initConfig
		if targetFile == "" {
			targetFile = "config.json"
		}
		runInteractiveWizard(targetFile)
		return
	}

	// Handle Config File Generation via Flag (-init-config config.json)
	if initConfig != "" {
		parsedSegs, err := parseCLISegments(segments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing -segments flag: %v\n", err)
			os.Exit(1)
		}
		genCfg := clipper.Config{
			InputFile:     inputFile,
			OutputDir:     outputDir,
			OutputFile:    outputFile,
			Mode:          clipper.Mode(modeStr),
			Strategy:      clipper.CutStrategy(stratStr),
			Shorts:        isShorts,
			ShortsStyle:   shortsStyle,
			Quality:       quality,
			CacheDir:      cacheDir,
			NoCache:       noCache,
			Concurrency:   concurrency,
			WatermarkPath: watermarkPath,
			WatermarkPos:  watermarkPos,
			OverlayText:   overlayText,
			TextPos:       textPos,
			FontSize:      fontSize,
			FontColor:     fontColor,
			AutoDetect:    autoDetect,
			Segments:      parsedSegs,
		}
		if err := saveConfig(initConfig, genCfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating config file '%s': %v\n", initConfig, err)
			os.Exit(1)
		}
		fmt.Printf("Successfully generated configuration file: %s\n", initConfig)
		return
	}

	var cfg clipper.Config

	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading config file '%s': %v\n", configFile, err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing JSON config file '%s': %v\n", configFile, err)
			os.Exit(1)
		}
	}

	// CLI flags override or supply missing values
	// Override config values ONLY if flags were explicitly passed on CLI
	if isFlagPassed("input") {
		cfg.InputFile = inputFile
	}
	if isFlagPassed("outdir") || cfg.OutputDir == "" {
		cfg.OutputDir = outputDir
	}
	if isFlagPassed("output") {
		cfg.OutputFile = outputFile
	}
	if isFlagPassed("mode") {
		cfg.Mode = clipper.Mode(modeStr)
	}
	if isFlagPassed("strategy") {
		cfg.Strategy = clipper.CutStrategy(stratStr)
	}
	if isFlagPassed("shorts") {
		cfg.Shorts = isShorts
	}
	if isFlagPassed("shorts-style") {
		cfg.ShortsStyle = shortsStyle
	}
	if isFlagPassed("quality") {
		cfg.Quality = quality
	}
	if isFlagPassed("cache-dir") {
		cfg.CacheDir = cacheDir
	}
	if isFlagPassed("no-cache") {
		cfg.NoCache = noCache
	}
	if isFlagPassed("concurrency") {
		cfg.Concurrency = concurrency
	}
	if isFlagPassed("watermark") {
		cfg.WatermarkPath = watermarkPath
	}
	if isFlagPassed("watermark-pos") {
		cfg.WatermarkPos = watermarkPos
	}
	if isFlagPassed("text") {
		cfg.OverlayText = overlayText
	}
	if isFlagPassed("text-pos") {
		cfg.TextPos = textPos
	}
	if isFlagPassed("font-size") {
		cfg.FontSize = fontSize
	}
	if isFlagPassed("font-color") {
		cfg.FontColor = fontColor
	}
	if isFlagPassed("auto-detect") {
		cfg.AutoDetect = autoDetect
	}
	if isFlagPassed("translate-lang") {
		cfg.TranslateLang = translateLang
	}
	if isFlagPassed("burn-subtitles") || isFlagPassed("subtitles") {
		cfg.BurnSubtitles = burnSubtitles
	}
	if isFlagPassed("sub-style") {
		cfg.SubStyle = subStyle
	}
	if isFlagPassed("sub-font-size") {
		cfg.SubFontSize = subFontSize
	}
	if isFlagPassed("sub-font-path") {
		cfg.SubFontPath = subFontPath
	}
	if isFlagPassed("use-whisper") {
		cfg.UseWhisper = useWhisper
	}
	if isFlagPassed("dry-run") {
		cfg.DryRun = dryRun
	}
	if isFlagPassed("batch-list") {
		cfg.BatchList = batchList
	}
	if isFlagPassed("clean-cache") {
		cfg.CleanCache = cleanCache
	}
	if isFlagPassed("clean-days") {
		cfg.CleanDays = cleanDays
	}
	if isFlagPassed("ai-router") {
		cfg.AIConfig.APIRouter = aiRouter
	}
	if isFlagPassed("ai-key") {
		cfg.AIConfig.APIKey = aiKey
	}
	if isFlagPassed("openrouter-key") {
		cfg.AIConfig.APIKey = openRouterKey
		cfg.OpenRouterKey = openRouterKey
	}
	if isFlagPassed("ai-model") {
		cfg.AIConfig.Model = aiModel
		cfg.AIModel = aiModel
	}

	// Parse CLI segments string if provided
	if segments != "" {
		parsedSegs, err := parseCLISegments(segments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing -segments flag: %v\n", err)
			os.Exit(1)
		}
		cfg.Segments = append(cfg.Segments, parsedSegs...)
	}

	if !cfg.CleanCache && cfg.InputFile == "" && cfg.BatchList == "" {
		flag.Usage()
		os.Exit(1)
	}

	app, err := clipper.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization error: %v\n", err)
		os.Exit(1)
	}

	if err := app.Process(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Processing failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

func parseCLISegments(raw string) ([]clipper.Segment, error) {
	pairs := strings.Split(raw, ",")
	var result []clipper.Segment
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		title := ""
		timePart := pair
		if strings.Contains(pair, ":") && (strings.Count(pair, ":") > 4 || strings.LastIndex(pair, ":") > strings.LastIndex(pair, "-")) {
			idx := strings.LastIndex(pair, ":")
			timePart = pair[:idx]
			title = pair[idx+1:]
		}

		parts := strings.Split(timePart, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid segment pair '%s', expected format 'START-END' or 'START-END:TITLE'", pair)
		}
		result = append(result, clipper.Segment{
			Start: strings.TrimSpace(parts[0]),
			End:   strings.TrimSpace(parts[1]),
			Title: strings.TrimSpace(title),
		})
	}
	return result, nil
}

func runInteractiveWizard(defaultFile string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("=== Interactive Config Generator Wizard ===")

	fileOut := promptString(reader, "Output config filename", defaultFile)
	inputFile := promptString(reader, "Input video file path or YouTube URL", "https://www.youtube.com/watch?v=sample")
	outputDir := promptString(reader, "Output directory for clips", "./output_clips")

	fmt.Println("\nMode options:")
	fmt.Println("  1. split (Cut into individual clip files)")
	fmt.Println("  2. merge (Cut and merge into single video)")
	modeChoice := promptString(reader, "Select mode (1 or 2)", "1")
	mode := clipper.ModeSplit
	if modeChoice == "2" || strings.ToLower(modeChoice) == "merge" {
		mode = clipper.ModeMerge
	}

	outputFile := ""
	if mode == clipper.ModeMerge {
		outputFile = promptString(reader, "Merged output video filename", "merged_highlight.mp4")
	}

	fmt.Println("\nCutting strategy options:")
	fmt.Println("  1. fast     (Stream copy, super fast without re-encoding)")
	fmt.Println("  2. accurate (Re-encode for frame-accurate cuts)")
	stratChoice := promptString(reader, "Select strategy (1 or 2)", "1")
	strategy := clipper.StrategyFast
	if stratChoice == "2" || strings.ToLower(stratChoice) == "accurate" {
		strategy = clipper.StrategyAccurate
	}

	shortsChoice := promptString(reader, "\nConvert to 9:16 Shorts/Reels format? (y/n)", "n")
	isShorts := strings.ToLower(shortsChoice) == "y" || strings.ToLower(shortsChoice) == "yes"
	shortsStyle := "crop"
	if isShorts {
		fmt.Println("Shorts aspect ratio style:")
		fmt.Println("  1. crop (Center crop 9:16)")
		fmt.Println("  2. blur (Blurred top/bottom background 9:16)")
		styleChoice := promptString(reader, "Select style (1 or 2)", "1")
		if styleChoice == "2" || strings.ToLower(styleChoice) == "blur" {
			shortsStyle = "blur"
		}
	}

	quality := promptString(reader, "\nYouTube Video Download Quality (best, 1080p, 720p, 480p, 360p, worst)", "best")
	autoDetect := promptString(reader, "\nAuto Detection Mode (press Enter to skip, or enter 'silence' / 'scene')", "")

	var segments []clipper.Segment
	if autoDetect == "" {
		fmt.Println("\nEnter video segments (press Enter with empty start time when finished):")
		segIdx := 1
		for {
			fmt.Printf("\n--- Segment #%d ---\n", segIdx)
			start := promptString(reader, "Start time (e.g. 00:00:10 or 10)", "")
			if start == "" {
				if len(segments) == 0 {
					fmt.Println("Warning: At least one segment or auto-detect is required. Please enter start time.")
					continue
				}
				break
			}
			end := promptString(reader, "End time (e.g. 00:00:25 or 25)", "")
			title := promptString(reader, "Segment title/label (optional)", "")

			segments = append(segments, clipper.Segment{
				Start: start,
				End:   end,
				Title: title,
			})
			segIdx++
		}
	}

	cfg := clipper.Config{
		InputFile:   inputFile,
		OutputDir:   outputDir,
		OutputFile:  outputFile,
		Mode:        mode,
		Strategy:    strategy,
		Shorts:      isShorts,
		ShortsStyle: shortsStyle,
		Quality:     quality,
		AutoDetect:  autoDetect,
		Segments:    segments,
	}

	if err := saveConfig(fileOut, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\nError saving config: %v\n", err)
		return
	}

	fmt.Printf("\nSuccessfully created configuration file: %s\n", fileOut)
}

func promptString(reader *bufio.Reader, prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func saveConfig(filePath string, cfg clipper.Config) error {
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
