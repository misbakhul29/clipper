package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"clipping/pkg/clipper"
)

func main() {
	var (
		configFile  string
		inputFile   string
		outputDir   string
		outputFile  string
		modeStr     string
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
		segments      string
	)

	flag.StringVar(&configFile, "config", "", "Path to JSON configuration file")
	flag.StringVar(&inputFile, "input", "", "Path to source input video file or YouTube URL")
	flag.StringVar(&outputDir, "outdir", ".", "Output directory for cut videos")
	flag.StringVar(&outputFile, "output", "", "Output filename (used in merge mode)")
	flag.StringVar(&modeStr, "mode", "split", "Operation mode: 'split' (separate files) or 'merge' (combined file)")
	flag.StringVar(&stratStr, "strategy", "fast", "Cut strategy: 'fast' (stream copy) or 'accurate' (re-encode)")
	flag.BoolVar(&isShorts, "shorts", false, "Convert cut videos to 9:16 Shorts/Reels/TikTok format")
	flag.StringVar(&shortsStyle, "shorts-style", "crop", "Shorts aspect ratio style: 'crop' (center crop 9:16) or 'blur' (blurred background 9:16)")
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
	flag.StringVar(&autoDetect, "auto-detect", "", "Smart auto-detection mode for segments ('silence' or 'scene')")
	flag.StringVar(&segments, "segments", "", "Comma-separated segment timestamps (e.g. '00:10-00:25,01:00-01:30')")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Automated Video Cutting System in Go (Supports Local Videos, YouTube URLs, Shorts 9:16 & Auto-Detect)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  1. Using JSON config file:\n")
		fmt.Fprintf(os.Stderr, "     %s -config examples/segments.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  2. Smart Auto Silence Detection + Parallel Workers:\n")
		fmt.Fprintf(os.Stderr, "     %s -input video.mp4 -auto-detect silence -concurrency 4 -shorts -outdir ./shorts_silence\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  3. Adding Text Caption and Watermark Image:\n")
		fmt.Fprintf(os.Stderr, "     %s -input sample.mp4 -segments \"00:05-00:15\" -text \"My Channel\" -watermark logo.png\n", os.Args[0])
	}

	flag.Parse()

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
	if inputFile != "" {
		cfg.InputFile = inputFile
	}
	if outputDir != "" && cfg.OutputDir == "" {
		cfg.OutputDir = outputDir
	}
	if outputFile != "" {
		cfg.OutputFile = outputFile
	}
	if modeStr != "" && cfg.Mode == "" {
		cfg.Mode = clipper.Mode(modeStr)
	}
	if stratStr != "" && cfg.Strategy == "" {
		cfg.Strategy = clipper.CutStrategy(stratStr)
	}
	if isShorts {
		cfg.Shorts = true
	}
	if shortsStyle != "" {
		cfg.ShortsStyle = shortsStyle
	}
	if quality != "" {
		cfg.Quality = quality
	}
	if cacheDir != "" && cfg.CacheDir == "" {
		cfg.CacheDir = cacheDir
	}
	if noCache {
		cfg.NoCache = true
	}
	if concurrency > 0 {
		cfg.Concurrency = concurrency
	}
	if watermarkPath != "" {
		cfg.WatermarkPath = watermarkPath
	}
	if watermarkPos != "" {
		cfg.WatermarkPos = watermarkPos
	}
	if overlayText != "" {
		cfg.OverlayText = overlayText
	}
	if textPos != "" {
		cfg.TextPos = textPos
	}
	if fontSize > 0 {
		cfg.FontSize = fontSize
	}
	if fontColor != "" {
		cfg.FontColor = fontColor
	}
	if autoDetect != "" {
		cfg.AutoDetect = autoDetect
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

	if cfg.InputFile == "" || (len(cfg.Segments) == 0 && cfg.AutoDetect == "") {
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
		parts := strings.Split(pair, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid segment pair '%s', expected format 'START-END'", pair)
		}
		result = append(result, clipper.Segment{
			Start: strings.TrimSpace(parts[0]),
			End:   strings.TrimSpace(parts[1]),
		})
	}
	return result, nil
}
