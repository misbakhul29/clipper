package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/misbakhul29/clipper/pkg/clipper"
	"github.com/misbakhul29/clipper/pkg/web"
)

const Version = "v1.36.0"

func printUsage() {
	fmt.Printf(`CLIPPER %s — Minimalist AI Video Clipper & Shorts Engine

USAGE:
  clipper <command> [arguments]

COMMANDS:
  serve, -s, -serve [port]       Launch the Web Studio UI dashboard (default: :8000)
                                 Examples:
                                   clipper serve
                                   clipper serve :8080
                                   clipper -s 3000

  config, -c, -config <file>     Run video clipping directly using a JSON config file
                                 Examples:
                                   clipper config config.json
                                   clipper -c segments.json
                                   clipper my_project.json

  init, -i, -init [filename]     Create a starter config.json template or run interactive wizard
                                 Examples:
                                   clipper init
                                   clipper init custom.json
                                   clipper -i

  version, -v, -version          Display current version information
  help, -h, --help               Show this help message

WEB STUDIO & CONFIGURATION:
  All rendering parameters (Shorts 9:16, Subtitles, AI Highlights, Audio Normalization,
  Silence Removal, Overlays, Watermark, and Hardware Acceleration) are managed visually
  in the Web Studio or through a config.json file.
`, Version)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		return
	}

	cmd := strings.ToLower(strings.TrimSpace(args[0]))
	switch cmd {
	case "help", "-h", "--help", "-help":
		printUsage()
		return

	case "version", "-v", "--version", "-version":
		fmt.Printf("Clipper %s\n", Version)
		return

	case "serve", "-s", "--serve", "-serve":
		serveAddr := ":8000"
		if len(args) > 1 {
			addr := args[1]
			if !strings.HasPrefix(addr, ":") && !strings.Contains(addr, ":") {
				addr = ":" + addr
			}
			serveAddr = addr
		}
		srv := web.NewServer(serveAddr, &clipper.Config{})
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Web Studio error: %v\n", err)
			os.Exit(1)
		}
		return

	case "init", "-i", "--init", "-init", "-interactive", "--interactive":
		targetFile := "config.json"
		if len(args) > 1 {
			targetFile = args[1]
		}
		if cmd == "-i" || cmd == "-interactive" || cmd == "--interactive" || (len(args) > 1 && args[1] == "--wizard") {
			runInteractiveWizard(targetFile)
			return
		}
		if err := generateSampleConfigFile(targetFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✨ Created starter configuration template: %s\n", targetFile)
		fmt.Printf("Edit %s or run:\n  clipper config %s\n", targetFile, targetFile)
		return

	case "config", "-c", "--config", "-config":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: Config file path required.\nUsage: clipper config <path/to/config.json>")
			os.Exit(1)
		}
		runConfig(args[1])
		return

	default:
		// If user passes a flag or file directly: e.g. `clipper config.json` or `clipper -serve=:8000`
		if strings.HasPrefix(cmd, "-s=") || strings.HasPrefix(cmd, "--serve=") || strings.HasPrefix(cmd, "-serve=") {
			parts := strings.Split(cmd, "=")
			serveAddr := parts[1]
			if !strings.HasPrefix(serveAddr, ":") && !strings.Contains(serveAddr, ":") {
				serveAddr = ":" + serveAddr
			}
			srv := web.NewServer(serveAddr, &clipper.Config{})
			if err := srv.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Web Studio error: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if strings.HasPrefix(cmd, "-c=") || strings.HasPrefix(cmd, "--config=") || strings.HasPrefix(cmd, "-config=") {
			parts := strings.Split(cmd, "=")
			runConfig(parts[1])
			return
		}
		if strings.HasSuffix(cmd, ".json") {
			runConfig(args[0])
			return
		}

		fmt.Fprintf(os.Stderr, "Unknown command or flag '%s'\n\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func runConfig(cfgPath string) {
	cfg, err := clipper.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration '%s': %v\n", cfgPath, err)
		os.Exit(1)
	}

	app, err := clipper.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization error: %v\n", err)
		os.Exit(1)
	}

	if err := app.Process(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Processing failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

func generateSampleConfigFile(filePath string) error {
	sample := clipper.Config{
		InputFile:        "https://www.youtube.com/watch?v=sample_video",
		OutputDir:        "./clips",
		OutputFile:       "merged_highlight.mp4",
		Mode:             clipper.ModeSplit,
		Strategy:         clipper.StrategyFast,
		Shorts:           true,
		ShortsStyle:      "blur",
		Quality:          "1080p",
		AutoDetect:       "ai",
		TargetDuration:   30,
		TranslateLang:    "id",
		Subtitles:        true,
		SubPreset:        "hormozi",
		SubFontSize:      48,
		SubEmoji:         true,
		SubSDHMode:       "strip",
		Loudnorm:         true,
		JumpCut:          true,
		GenerateMetadata: true,
		ExtractThumbnail: true,
		ThumbnailCount:   1,
		HWAccel:          "auto",
		ShowProgress:     true,
		FaceTracking:     true,
		Segments: []clipper.Segment{
			{
				Start: "00:00:10",
				End:   "00:00:40",
				Title: "Highlight 1 - Hook",
			},
			{
				Start: "00:01:20",
				End:   "00:01:50",
				Title: "Highlight 2 - Peak Moment",
			},
		},
	}
	return saveConfig(filePath, sample)
}

func runInteractiveWizard(defaultFile string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("=== Interactive Config Generator Wizard ===")

	fileOut := promptString(reader, "Output config filename", defaultFile)
	inputFile := promptString(reader, "Input video file path or YouTube URL", "https://www.youtube.com/watch?v=sample")
	outputDir := promptString(reader, "Output directory for clips", "./clips")

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
		fmt.Println("  3. smart-crop (Active speaker & face tracking)")
		styleChoice := promptString(reader, "Select style (1, 2, or 3)", "1")
		if styleChoice == "2" || strings.ToLower(styleChoice) == "blur" {
			shortsStyle = "blur"
		} else if styleChoice == "3" || strings.ToLower(styleChoice) == "smart-crop" {
			shortsStyle = "smart-crop"
		}
	}

	loudnormChoice := promptString(reader, "\nEnable EBU R128 audio normalization (-14 LUFS)? (y/n)", "y")
	loudnormEnabled := strings.ToLower(loudnormChoice) == "y" || strings.ToLower(loudnormChoice) == "yes"

	jumpCutChoice := promptString(reader, "\nEnable Smart Silence Removal (Jump-Cut) inside clips? (y/n)", "n")
	jumpCutEnabled := strings.ToLower(jumpCutChoice) == "y" || strings.ToLower(jumpCutChoice) == "yes"

	subChoice := promptString(reader, "\nBurn subtitles directly into video? (y/n)", "n")
	burnSubs := strings.ToLower(subChoice) == "y" || strings.ToLower(subChoice) == "yes"
	subPresetChoice := "hormozi"
	subSDHChoice := "strip"
	subEmojiChoice := true
	if burnSubs {
		fmt.Println("Subtitle Theme Presets:")
		fmt.Println("  1. hormozi   (Viral yellow, bold outline, pop-in bounce animation)")
		fmt.Println("  2. minimal   (Clean crisp white, subtle border - Ali Abdaal / Devon style)")
		fmt.Println("  3. neon      (Electric cyan, glowing magenta blur)")
		fmt.Println("  4. cinematic (Soft ivory, wide spacing, classic subtitle)")
		presetInput := promptString(reader, "Select preset (1-4 or name)", "1")
		switch strings.ToLower(presetInput) {
		case "2", "minimal", "devon":
			subPresetChoice = "minimal"
		case "3", "neon":
			subPresetChoice = "neon"
		case "4", "cinematic":
			subPresetChoice = "cinematic"
		default:
			subPresetChoice = "hormozi"
		}

		fmt.Println("\nSilent Narrator & SDH Context Handling [...] :")
		fmt.Println("  1. strip   (Remove [...] to keep speech snappy & clean - recommended)")
		fmt.Println("  2. top-box (Render [...] as elegant static context box at top)")
		fmt.Println("  3. keep    (Keep [...] as-is)")
		sdhInput := promptString(reader, "Select SDH mode (1-3 or name)", "1")
		subSDHChoice = "strip"
		switch strings.ToLower(sdhInput) {
		case "2", "top-box", "box":
			subSDHChoice = "top-box"
		case "3", "keep":
			subSDHChoice = "keep"
		default:
			subSDHChoice = "strip"
		}

		emojiChoice := promptString(reader, "\nAuto-inject contextual emojis (e.g. 💰, 🔥, 💡) into subtitles? (y/n)", "y")
		subEmojiChoice = strings.ToLower(emojiChoice) == "y" || strings.ToLower(emojiChoice) == "yes"
	}

	metaChoice := promptString(reader, "\nGenerate companion social media metadata (metadata.json & .txt)? (y/n)", "y")
	generateMeta := strings.ToLower(metaChoice) == "y" || strings.ToLower(metaChoice) == "yes"

	thumbChoice := promptString(reader, "\nExtract cover thumbnail and hook frame (.jpg)? (y/n)", "y")
	extractThumbnails := strings.ToLower(thumbChoice) == "y" || strings.ToLower(thumbChoice) == "yes"

	hwChoice := promptString(reader, "\nHardware Acceleration mode (auto, nvenc, videotoolbox, qsv, vaapi, amf, cpu)", "auto")
	quality := promptString(reader, "\nYouTube Video Download Quality (best, 1080p, 720p, 480p, 360p, worst)", "best")
	autoDetect := promptString(reader, "\nAuto Detection Mode (press Enter to skip, or enter 'ai' / 'silence' / 'scene')", "")

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
		InputFile:        inputFile,
		OutputDir:        outputDir,
		OutputFile:       outputFile,
		Mode:             mode,
		Strategy:         strategy,
		Shorts:           isShorts,
		ShortsStyle:      shortsStyle,
		Quality:          quality,
		AutoDetect:       autoDetect,
		Loudnorm:         loudnormEnabled,
		JumpCut:          jumpCutEnabled,
		Subtitles:        burnSubs,
		SubPreset:        subPresetChoice,
		SubSDHMode:       subSDHChoice,
		SubEmoji:         subEmojiChoice,
		GenerateMetadata: generateMeta,
		ExtractThumbnail: extractThumbnails,
		ThumbnailCount:   1,
		HWAccel:          hwChoice,
		ShowProgress:     true,
		FaceTracking:     true,
		Segments:         segments,
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

// parseCLISegments parses a comma-separated string of video segment definitions.
// Formats supported per segment:
//   - "START-END" (e.g., "00:00-00:02")
//   - "START-END:TITLE" (e.g., "00:05-00:25:Epic Moment")
func parseCLISegments(raw string) ([]clipper.Segment, error) {
	pairs := strings.Split(raw, ",")
	var result []clipper.Segment
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		dashIdx := strings.Index(pair, "-")
		if dashIdx == -1 {
			return nil, fmt.Errorf("invalid segment format '%s', expected 'START-END' or 'START-END:TITLE'", pair)
		}

		start := strings.TrimSpace(pair[:dashIdx])
		rest := strings.TrimSpace(pair[dashIdx+1:])
		if start == "" || rest == "" {
			return nil, fmt.Errorf("invalid segment '%s': start and end must not be empty", pair)
		}

		var end, title string
		parts := strings.Split(rest, ":")
		switch len(parts) {
		case 1:
			end = parts[0]
		case 2:
			if isTimeDigits(parts[1]) {
				end = rest
			} else {
				end = parts[0]
				title = parts[1]
			}
		case 3:
			if isTimeDigits(parts[2]) {
				end = rest
			} else {
				end = parts[0] + ":" + parts[1]
				title = parts[2]
			}
		default:
			if isTimeDigits(parts[1]) && isTimeDigits(parts[2]) {
				end = parts[0] + ":" + parts[1] + ":" + parts[2]
				title = strings.Join(parts[3:], ":")
			} else if isTimeDigits(parts[1]) {
				end = parts[0] + ":" + parts[1]
				title = strings.Join(parts[2:], ":")
			} else {
				end = parts[0]
				title = strings.Join(parts[1:], ":")
			}
		}

		result = append(result, clipper.Segment{
			Start: start,
			End:   strings.TrimSpace(end),
			Title: strings.TrimSpace(title),
		})
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid segments found in '%s'", raw)
	}
	return result, nil
}

func isTimeDigits(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, ch := range s {
		if (ch < '0' || ch > '9') && ch != '.' {
			return false
		}
	}
	return true
}

