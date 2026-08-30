package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"clipping/pkg/clipper"
)

func main() {
	var (
		fileOut     string
		inputFile   string
		outputDir   string
		outputFile  string
		modeStr     string
		stratStr    string
		isShorts    bool
		shortsStyle string
		quality     string
		segmentsStr string
		interactive bool
	)

	flag.StringVar(&fileOut, "file", "config.json", "Output JSON configuration file path")
	flag.StringVar(&inputFile, "input", "input_video.mp4", "Path to source input video file")
	flag.StringVar(&outputDir, "outdir", "./output_clips", "Output directory for cut videos")
	flag.StringVar(&outputFile, "output", "merged_video.mp4", "Output filename (used in merge mode)")
	flag.StringVar(&modeStr, "mode", "split", "Operation mode: 'split' (separate files) or 'merge' (combined file)")
	flag.StringVar(&stratStr, "strategy", "fast", "Cut strategy: 'fast' (stream copy) or 'accurate' (re-encode)")
	flag.BoolVar(&isShorts, "shorts", false, "Convert cut videos to 9:16 Shorts/Reels/TikTok format")
	flag.StringVar(&shortsStyle, "shorts-style", "crop", "Shorts style: 'crop' (center crop 9:16) or 'blur' (blurred background 9:16)")
	flag.StringVar(&quality, "quality", "best", "YouTube download quality ('best', '1080p', '720p', '480p', '360p', 'worst')")
	flag.StringVar(&segmentsStr, "segments", "00:00:10-00:00:25:clip1,00:01:00-00:01:30:clip2", "Comma-separated segments (format: 'START-END' or 'START-END:TITLE')")
	flag.BoolVar(&interactive, "i", false, "Run in interactive wizard mode")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generator for Video Clipping JSON Configuration Files\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  1. Quick generation with flags:\n")
		fmt.Fprintf(os.Stderr, "     %s -file my_clips.json -input video.mp4 -mode merge -shorts -shorts-style blur -quality 1080p -segments \"00:10-00:25:intro,01:00-01:30:outro\"\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  2. Interactive wizard mode:\n")
		fmt.Fprintf(os.Stderr, "     %s -i\n", os.Args[0])
	}

	flag.Parse()

	if interactive {
		runInteractiveWizard(fileOut)
		return
	}

	// Parse segments from string
	parsedSegments, err := parseSegments(segmentsStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing segments: %v\n", err)
		os.Exit(1)
	}

	cfg := clipper.Config{
		InputFile:   inputFile,
		OutputDir:   outputDir,
		OutputFile:  outputFile,
		Mode:        clipper.Mode(modeStr),
		Strategy:    clipper.CutStrategy(stratStr),
		Shorts:      isShorts,
		ShortsStyle: shortsStyle,
		Quality:     quality,
		Segments:    parsedSegments,
	}

	if err := saveConfig(fileOut, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated config file: %s\n", fileOut)
}

func parseSegments(raw string) ([]clipper.Segment, error) {
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
			// Contains title at the end e.g. "00:10-00:25:intro"
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
	inputFile := promptString(reader, "Input video file path", "sample.mp4")
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

	fmt.Println("\nEnter video segments (press Enter with empty start time when finished):")
	var segments []clipper.Segment
	segIdx := 1
	for {
		fmt.Printf("\n--- Segment #%d ---\n", segIdx)
		start := promptString(reader, "Start time (e.g. 00:00:10 or 10)", "")
		if start == "" {
			if len(segments) == 0 {
				fmt.Println("Warning: At least one segment is required. Please enter start time.")
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

	cfg := clipper.Config{
		InputFile:   inputFile,
		OutputDir:   outputDir,
		OutputFile:  outputFile,
		Mode:        mode,
		Strategy:    strategy,
		Shorts:      isShorts,
		ShortsStyle: shortsStyle,
		Quality:     quality,
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
