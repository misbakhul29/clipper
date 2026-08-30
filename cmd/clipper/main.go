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
		stratStr    string
		isShorts    bool
		shortsStyle string
		quality     string
		cacheDir    string
		noCache     bool
		segments    string
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
	flag.StringVar(&segments, "segments", "", "Comma-separated segment timestamps (e.g. '00:10-00:25,01:00-01:30')")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Automated Video Cutting System in Go (Supports Local Videos, YouTube URLs & Shorts 9:16)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  1. Using JSON config file:\n")
		fmt.Fprintf(os.Stderr, "     %s -config examples/segments.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  2. Cutting YouTube video directly into Shorts format (1080p):\n")
		fmt.Fprintf(os.Stderr, "     %s -input \"https://www.youtube.com/watch?v=dQw4w9WgXcQ\" -quality 1080p -segments \"00:10-00:25\" -shorts -shorts-style blur -outdir ./yt_shorts\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  3. Using CLI flags (Merge Mode):\n")
		fmt.Fprintf(os.Stderr, "     %s -input sample.mp4 -segments \"00:05-00:15,00:30-00:45\" -mode merge -output final.mp4\n", os.Args[0])
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

	// Parse CLI segments string if provided
	if segments != "" {
		parsedSegs, err := parseCLISegments(segments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing -segments flag: %v\n", err)
			os.Exit(1)
		}
		cfg.Segments = append(cfg.Segments, parsedSegs...)
	}

	if cfg.InputFile == "" || len(cfg.Segments) == 0 {
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
