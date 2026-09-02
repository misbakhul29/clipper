package transcriber

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/misbakhul29/clipper/pkg/detector"
	"github.com/misbakhul29/clipper/pkg/downloader"
)

// SubtitleEntry represents a timestamped text entry from a subtitle file.
type SubtitleEntry struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Text  string `json:"text"`
}

var vttTimeRegex = regexp.MustCompile(`((?:\d{1,2}:)?\d{2}:\d{2}[\.\,]\d{3})\s*-->\s*((?:\d{1,2}:)?\d{2}:\d{2}[\.\,]\d{3})`)
var sdhBracketRegex = regexp.MustCompile(`(?s)\[(.*?)\]`)

// ExtractSDHAndSpeech separates silent narrator context / sound effect descriptions inside brackets [...]
// from actual spoken dialogue text.
func ExtractSDHAndSpeech(text string) (speechText string, sdhText string) {
	matches := sdhBracketRegex.FindAllStringSubmatch(text, -1)
	var sdhParts []string
	for _, m := range matches {
		if len(m) >= 2 {
			clean := strings.TrimSpace(m[1])
			if clean != "" {
				sdhParts = append(sdhParts, clean)
			}
		}
	}
	sdhText = strings.Join(sdhParts, " | ")

	// Strip brackets to get spoken speech
	speech := sdhBracketRegex.ReplaceAllString(text, "")
	speech = strings.TrimSpace(speech)
	speech = strings.ReplaceAll(speech, "\n", " ")
	for strings.Contains(speech, "  ") {
		speech = strings.ReplaceAll(speech, "  ", " ")
	}

	return speech, sdhText
}

// FilterSDHEntries filters or cleans subtitle entries according to sdhMode ("strip", "top-box", "keep").
// In "strip" mode, it removes [...] brackets and drops any cues that become empty.
func FilterSDHEntries(entries []SubtitleEntry, sdhMode string) []SubtitleEntry {
	if sdhMode != "strip" {
		return entries
	}
	var cleaned []SubtitleEntry
	for _, entry := range entries {
		speech, _ := ExtractSDHAndSpeech(entry.Text)
		if speech != "" {
			entry.Text = speech
			cleaned = append(cleaned, entry)
		}
	}
	return cleaned
}

// ParseVTT parses VTT or SRT subtitle file content into a slice of SubtitleEntry.
func ParseVTT(content string) ([]SubtitleEntry, error) {
	lines := strings.Split(content, "\n")
	var entries []SubtitleEntry

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || line == "WEBVTT" || strings.HasPrefix(line, "NOTE") || isNumeric(line) {
			continue
		}

		matches := vttTimeRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			start := normalizeTimestamp(matches[1])
			end := normalizeTimestamp(matches[2])

			// Read sub text until empty line or next timestamp
			var textLines []string
			for j := i + 1; j < len(lines); j++ {
				subLine := strings.TrimSpace(lines[j])
				if subLine == "" || vttTimeRegex.MatchString(subLine) {
					i = j - 1
					break
				}
				textLines = append(textLines, subLine)
				i = j
			}

			fullText := cleanSubtitleText(strings.Join(textLines, " "))
			if fullText != "" {
				entries = append(entries, SubtitleEntry{
					Start: start,
					End:   end,
					Text:  fullText,
				})
			}
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no subtitle entries found in content")
	}

	return entries, nil
}

// findMatchingSubtitleFile searches dir for a .vtt file matching target language code precisely.
// It avoids false positive substring matches (e.g. video ID 'xid1sE8lEec' matching '*id*.vtt' for Russian '.ru.vtt').
func findMatchingSubtitleFile(dir, lang string) string {
	matches, err := filepath.Glob(filepath.Join(dir, "*.vtt"))
	if err != nil || len(matches) == 0 {
		return ""
	}

	targetLang := strings.ToLower(strings.TrimSpace(lang))

	// 1. Precise check for requested language (e.g. .id.vtt, .id-orig.vtt, .id-auto.vtt)
	if targetLang != "" {
		for _, m := range matches {
			base := strings.ToLower(filepath.Base(m))
			parts := strings.Split(base, ".")
			if len(parts) >= 3 {
				fileLang := parts[len(parts)-2]
				if fileLang == targetLang || strings.HasPrefix(fileLang, targetLang+"-") || strings.HasPrefix(fileLang, targetLang+"_") {
					return m
				}
			}
		}
	}

	// 2. Fallback check for English if requested language was not found
	if targetLang != "en" {
		for _, m := range matches {
			base := strings.ToLower(filepath.Base(m))
			parts := strings.Split(base, ".")
			if len(parts) >= 3 {
				fileLang := parts[len(parts)-2]
				if fileLang == "en" || strings.HasPrefix(fileLang, "en-") || strings.HasPrefix(fileLang, "en_") {
					return m
				}
			}
		}
	}

	return ""
}

// findAnySubtitleFile returns the first available .vtt subtitle file found in dir.
func findAnySubtitleFile(dir string) string {
	matches, err := filepath.Glob(filepath.Join(dir, "*.vtt"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	// Prefer orig or base language if available
	for _, m := range matches {
		base := strings.ToLower(filepath.Base(m))
		if strings.Contains(base, "-orig.") || strings.Contains(base, ".orig.") {
			return m
		}
	}
	return matches[0]
}

// FetchSubtitles attempts to retrieve subtitles for a YouTube URL or local file path.
func FetchSubtitles(inputStr, outputDir, lang string) ([]SubtitleEntry, error) {
	if outputDir == "" {
		outputDir = "./cache"
	}

	// Case 1: Input is already a local .vtt or .srt file
	if (strings.HasSuffix(inputStr, ".vtt") || strings.HasSuffix(inputStr, ".srt")) && fileExists(inputStr) {
		data, err := os.ReadFile(inputStr)
		if err != nil {
			return nil, err
		}
		return ParseVTT(string(data))
	}

	videoCacheDir := downloader.GetVideoCacheDir(outputDir, inputStr)
	if err := os.MkdirAll(videoCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create subtitle cache dir: %w", err)
	}

	// Case 2: Check cache for existing matching subtitle file (exact, en fallback, or any cached sub)
	if match := findMatchingSubtitleFile(videoCacheDir, lang); match != "" {
		data, err := os.ReadFile(match)
		if err == nil && len(data) > 0 {
			return ParseVTT(string(data))
		}
	}
	if match := findAnySubtitleFile(videoCacheDir); match != "" {
		data, err := os.ReadFile(match)
		if err == nil && len(data) > 0 {
			return ParseVTT(string(data))
		}
	}

	binPath := findYtDlpBinary()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp binary required to fetch YouTube subtitles")
	}

	subTemplate := filepath.Join(videoCacheDir, "sub_%(id)s")

	// Attempt 1: Try requested target language
	if lang != "" {
		args := []string{
			"--write-auto-sub",
			"--write-sub",
			"--sub-lang", fmt.Sprintf("%s,%s-*", lang, lang),
			"--sub-format", "vtt",
			"--skip-download",
			"-o", subTemplate,
			inputStr,
		}
		cmd := exec.Command(binPath, args...)
		_ = cmd.Run()

		if match := findMatchingSubtitleFile(videoCacheDir, lang); match != "" {
			data, err := os.ReadFile(match)
			if err == nil && len(data) > 0 {
				return ParseVTT(string(data))
			}
		}
	}

	// Attempt 2: Try original video track (.*-orig, e.g. ko-orig, ja-orig, en-orig) to avoid 429 machine-translation errors
	argsOrig := []string{
		"--write-auto-sub",
		"--write-sub",
		"--sub-lang", ".*-orig,orig",
		"--sub-format", "vtt",
		"--skip-download",
		"-o", subTemplate,
		inputStr,
	}
	cmdOrig := exec.Command(binPath, argsOrig...)
	_ = cmdOrig.Run()

	if match := findAnySubtitleFile(videoCacheDir); match != "" {
		data, err := os.ReadFile(match)
		if err == nil && len(data) > 0 {
			return ParseVTT(string(data))
		}
	}

	// Attempt 3: Try English or fallback
	argsFallback := []string{
		"--write-auto-sub",
		"--write-sub",
		"--sub-lang", "en,en-*",
		"--sub-format", "vtt",
		"--skip-download",
		"-o", subTemplate,
		inputStr,
	}
	cmdFallback := exec.Command(binPath, argsFallback...)
	_ = cmdFallback.Run()

	match := findMatchingSubtitleFile(videoCacheDir, lang)
	if match == "" {
		match = findAnySubtitleFile(videoCacheDir)
	}
	if match == "" {
		return nil, fmt.Errorf("no matching subtitle file found for language '%s' (input: %s)", lang, inputStr)
	}

	data, err := os.ReadFile(match)
	if err != nil {
		return nil, fmt.Errorf("failed to read downloaded subtitle file '%s': %w", match, err)
	}

	return ParseVTT(string(data))
}

func findYtDlpBinary() string {
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path
	}
	cacheDir, err := os.UserCacheDir()
	if err == nil {
		binName := "yt-dlp"
		if runtime.GOOS == "windows" {
			binName = "yt-dlp.exe"
		}
		userBin := filepath.Join(cacheDir, "clipper", "bin", binName)
		if fileExists(userBin) {
			return userBin
		}
	}
	localBin := filepath.Join("bin", "yt-dlp")
	if runtime.GOOS == "windows" {
		localBin = filepath.Join("bin", "yt-dlp.exe")
	}
	if fileExists(localBin) {
		abs, _ := filepath.Abs(localBin)
		return abs
	}
	return ""
}

func normalizeTimestamp(ts string) string {
	return strings.ReplaceAll(ts, ",", ".")
}

func cleanSubtitleText(text string) string {
	// Strip HTML tags like <c>, </c>, <b>, etc.
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(text, "")
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SliceSubtitles extracts subtitle entries that overlap with the segment time interval [startSec, endSec]
// and shifts timestamps to be relative to 00:00:00.00.
func SliceSubtitles(entries []SubtitleEntry, startSec, endSec float64) []SubtitleEntry {
	var sliced []SubtitleEntry

	for _, entry := range entries {
		eStart := parseTimestampToSec(entry.Start)
		eEnd := parseTimestampToSec(entry.End)

		// Check overlap with interval [startSec, endSec]
		if eEnd > startSec && eStart < endSec {
			relStart := eStart - startSec
			if relStart < 0 {
				relStart = 0
			}
			relEnd := eEnd - startSec
			if relEnd > (endSec - startSec) {
				relEnd = endSec - startSec
			}

			sliced = append(sliced, SubtitleEntry{
				Start: formatASSTime(relStart),
				End:   formatASSTime(relEnd),
				Text:  entry.Text,
			})
		}
	}

	return sliced
}

// ExportASS exports subtitle entries into an Advanced SubStation Alpha (.ass) format file for FFmpeg burn-in.
func ExportASS(entries []SubtitleEntry, outputPath string, fontSize int, isShorts bool) error {
	return ExportASSWithFont(entries, outputPath, fontSize, isShorts, "Arial")
}

// ExportASSWithFont exports subtitle entries into ASS format with custom font name.
func ExportASSWithFont(entries []SubtitleEntry, outputPath string, fontSize int, isShorts bool, fontName string) error {
	if fontSize <= 0 {
		fontSize = 48
	}
	if fontName == "" {
		fontName = "Arial"
	}

	marginV := 160
	if isShorts {
		marginV = 420 // Position subtitle higher up on 9:16 Shorts frame
	}

	var sb strings.Builder
	playResX := 1920
	playResY := 1080
	if isShorts {
		playResX = 1080
		playResY = 1920
	}
	sb.WriteString(fmt.Sprintf("PlayResX: %d\n", playResX))
	sb.WriteString(fmt.Sprintf("PlayResY: %d\n", playResY))
	sb.WriteString("ScaledBorderAndShadow: yes\n\n")

	sb.WriteString("[V4+ Styles]\n")
	sb.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	sb.WriteString(fmt.Sprintf("Style: Default,%s,%d,&H00FFFFFF,&H00000000,&H00000000,&H80000000,-1,0,0,0,100,100,0,0,1,4,1,2,40,40,%d,1\n\n", fontName, fontSize, marginV))

	sb.WriteString("[Events]\n")
	sb.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	for _, entry := range entries {
		cleanText := strings.ReplaceAll(entry.Text, "\n", "\\N")
		sb.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", entry.Start, entry.End, cleanText))
	}

	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}

func parseTimestampToSec(ts string) float64 {
	ts = strings.ReplaceAll(ts, ",", ".")
	parts := strings.Split(ts, ":")
	if len(parts) == 3 {
		var h, m, s float64
		fmt.Sscanf(parts[0], "%f", &h)
		fmt.Sscanf(parts[1], "%f", &m)
		fmt.Sscanf(parts[2], "%f", &s)
		return h*3600 + m*60 + s
	} else if len(parts) == 2 {
		var m, s float64
		fmt.Sscanf(parts[0], "%f", &m)
		fmt.Sscanf(parts[1], "%f", &s)
		return m*60 + s
	}
	var s float64
	fmt.Sscanf(ts, "%f", &s)
	return s
}

func formatASSTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	totalCentisecs := int(math.Round(sec * 100))
	cs := totalCentisecs % 100
	totalSecs := totalCentisecs / 100
	secs := totalSecs % 60
	totalMins := totalSecs / 60
	mins := totalMins % 60
	hours := totalMins / 60
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, mins, secs, cs)
}

// ChunkSubtitlesToWords splits long subtitle entries into rapid 2-3 word micro-chunks with interpolated timestamps for TikTok-style animated captions.
func ChunkSubtitlesToWords(entries []SubtitleEntry, maxWords int) []SubtitleEntry {
	if maxWords <= 0 {
		maxWords = 3
	}

	var chunked []SubtitleEntry
	for _, entry := range entries {
		words := strings.Fields(entry.Text)
		if len(words) <= maxWords {
			chunked = append(chunked, entry)
			continue
		}

		startSec := parseTimestampToSec(entry.Start)
		endSec := parseTimestampToSec(entry.End)
		totalDur := endSec - startSec
		if totalDur <= 0 {
			chunked = append(chunked, entry)
			continue
		}
		timePerWord := totalDur / float64(len(words))

		for i := 0; i < len(words); i += maxWords {
			endIdx := i + maxWords
			if endIdx > len(words) {
				endIdx = len(words)
			}
			chunkWords := words[i:endIdx]

			cStartSec := startSec + float64(i)*timePerWord
			cEndSec := startSec + float64(endIdx)*timePerWord

			chunked = append(chunked, SubtitleEntry{
				Start: formatASSTime(cStartSec),
				End:   formatASSTime(cEndSec),
				Text:  strings.Join(chunkWords, " "),
			})
		}
	}

	return chunked
}

// SubtitlePreset defines visual, timing, and animation parameters for ASS subtitles.
type SubtitlePreset struct {
	Name           string
	FontName       string
	FontSize       int
	PrimaryColor   string
	SecondaryColor string
	OutlineColor   string
	BackColor      string
	Bold           int
	Outline        float64
	Shadow         float64
	BorderStyle    int
	Spacing        float64
	ScaleX         int
	ScaleY         int
	MarginV        int
	TransformTag   string
	MaxWords       int
	Uppercase      bool
}

// GetSubtitlePreset returns the preset theme configuration for ASS subtitle rendering.
func GetSubtitlePreset(name string, isShorts bool) SubtitlePreset {
	norm := strings.ToLower(strings.TrimSpace(name))
	marginV := 160
	if isShorts {
		marginV = 440 // Centered in lower third of 9:16 frame
	}

	switch norm {
	case "minimal", "devon":
		fontSize := 46
		if !isShorts {
			fontSize = 36
			marginV = 120
		}
		return SubtitlePreset{
			Name:           "minimal",
			FontName:       "Arial",
			FontSize:       fontSize,
			PrimaryColor:   "&H00FFFFFF", // Crisp white
			SecondaryColor: "&H00FFFFFF",
			OutlineColor:   "&H00111111", // Subtle clean dark outline
			BackColor:      "&H80000000",
			Bold:           -1,
			Outline:        2.0,
			Shadow:         1.0,
			BorderStyle:    1,
			Spacing:        0.5,
			ScaleX:         100,
			ScaleY:         100,
			MarginV:        marginV,
			TransformTag:   "",
			MaxWords:       3,
			Uppercase:      false,
		}

	case "neon":
		fontSize := 54
		if !isShorts {
			fontSize = 42
			marginV = 140
		}
		return SubtitlePreset{
			Name:           "neon",
			FontName:       "Impact",
			FontSize:       fontSize,
			PrimaryColor:   "&H00FFFF00", // Electric Cyan (&HAABBGGRR: Blue FF, Green FF)
			SecondaryColor: "&H00FFFF00",
			OutlineColor:   "&H008000FF", // Neon Violet/Magenta glow
			BackColor:      "&H00000000",
			Bold:           -1,
			Outline:        4.5,
			Shadow:         2.0,
			BorderStyle:    1,
			Spacing:        1.0,
			ScaleX:         105,
			ScaleY:         105,
			MarginV:        marginV,
			TransformTag:   `{\blur3}`,
			MaxWords:       3,
			Uppercase:      true,
		}

	case "cinematic":
		fontSize := 38
		if isShorts {
			fontSize = 42
		} else {
			marginV = 100
		}
		return SubtitlePreset{
			Name:           "cinematic",
			FontName:       "Georgia",
			FontSize:       fontSize,
			PrimaryColor:   "&H00F0F0F0", // Soft Ivory White
			SecondaryColor: "&H00F0F0F0",
			OutlineColor:   "&H00151515", // Gentle filmic shadow
			BackColor:      "&H90000000",
			Bold:           0,
			Outline:        1.5,
			Shadow:         1.2,
			BorderStyle:    1,
			Spacing:        1.5,
			ScaleX:         100,
			ScaleY:         100,
			MarginV:        marginV,
			TransformTag:   "",
			MaxWords:       7,
			Uppercase:      false,
		}

	case "hormozi", "default", "":
		fallthrough
	default:
		fontSize := 56
		if !isShorts {
			fontSize = 44
			marginV = 160
		}
		return SubtitlePreset{
			Name:           "hormozi",
			FontName:       "Impact",
			FontSize:       fontSize,
			PrimaryColor:   "&H0000FFFF", // Vibrant Neon Yellow (&HAABBGGRR: Green FF, Red FF)
			SecondaryColor: "&H0000FF55", // Lime Accent
			OutlineColor:   "&H00000000", // Thick black outline
			BackColor:      "&H90000000",
			Bold:           -1,
			Outline:        5.0,
			Shadow:         2.5,
			BorderStyle:    1,
			Spacing:        1.0,
			ScaleX:         105,
			ScaleY:         105,
			MarginV:        marginV,
			TransformTag:   `{\fscx115\fscy115\t(0,100,\fscx105\fscy105)}`, // Pop-in bounce animation
			MaxWords:       2,
			Uppercase:      true,
		}
	}
}

// ExportPresetASS exports subtitle entries using a named visual preset with optional custom font, size, and SDH separation.
func ExportPresetASS(entries []SubtitleEntry, outputPath string, presetName string, fontSize int, isShorts bool, customFont string, sdhMode string) error {
	preset := GetSubtitlePreset(presetName, isShorts)

	if fontSize > 0 {
		preset.FontSize = fontSize
	}
	if customFont != "" {
		preset.FontName = customFont
	}

	sdhMode = strings.ToLower(strings.TrimSpace(sdhMode))
	if sdhMode == "" {
		sdhMode = "strip"
	}

	var speechEntries []SubtitleEntry
	var sdhEntries []SubtitleEntry

	for _, entry := range entries {
		switch sdhMode {
		case "top-box":
			speech, sdh := ExtractSDHAndSpeech(entry.Text)
			if speech != "" {
				speechEntries = append(speechEntries, SubtitleEntry{
					Start: entry.Start,
					End:   entry.End,
					Text:  speech,
				})
			}
			if sdh != "" {
				sdhEntries = append(sdhEntries, SubtitleEntry{
					Start: entry.Start,
					End:   entry.End,
					Text:  sdh,
				})
			}
		case "keep":
			speechEntries = append(speechEntries, entry)
		case "strip":
			fallthrough
		default:
			speech, _ := ExtractSDHAndSpeech(entry.Text)
			if speech != "" {
				speechEntries = append(speechEntries, SubtitleEntry{
					Start: entry.Start,
					End:   entry.End,
					Text:  speech,
				})
			}
		}
	}

	// Micro-chunk words if preset defines MaxWords
	var formattedSpeech []SubtitleEntry
	if preset.MaxWords > 0 {
		formattedSpeech = ChunkSubtitlesToWords(speechEntries, preset.MaxWords)
	} else {
		formattedSpeech = speechEntries
	}

	if len(formattedSpeech) == 0 && len(sdhEntries) == 0 {
		return fmt.Errorf("no dialogue or narrator entries to render")
	}

	var sb strings.Builder
	playResX := 1920
	playResY := 1080
	if isShorts {
		playResX = 1080
		playResY = 1920
	}
	sb.WriteString("[Script Info]\n")
	sb.WriteString("ScriptType: v4.00+\n")
	sb.WriteString(fmt.Sprintf("PlayResX: %d\n", playResX))
	sb.WriteString(fmt.Sprintf("PlayResY: %d\n", playResY))
	sb.WriteString("ScaledBorderAndShadow: yes\n\n")

	sb.WriteString("[V4+ Styles]\n")
	sb.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")

	// Primary dialogue style
	sb.WriteString(fmt.Sprintf("Style: Default,%s,%d,%s,%s,%s,%s,%d,0,0,0,%d,%d,%.1f,0,%d,%.1f,%.1f,2,40,40,%d,1\n",
		preset.FontName,
		preset.FontSize,
		preset.PrimaryColor,
		preset.SecondaryColor,
		preset.OutlineColor,
		preset.BackColor,
		preset.Bold,
		preset.ScaleX,
		preset.ScaleY,
		preset.Spacing,
		preset.BorderStyle,
		preset.Outline,
		preset.Shadow,
		preset.MarginV,
	))

	// Narrator context style (Top-Center Alignment 8, Italic, soft white)
	if sdhMode == "top-box" && len(sdhEntries) > 0 {
		narratorFont := "Arial"
		narratorSize := 32
		narratorMarginV := 80
		if isShorts {
			narratorSize = 36
			narratorMarginV = 120
		}
		sb.WriteString(fmt.Sprintf("Style: Narrator,%s,%d,&H00F0F0F0,&H00000000,&H00000000,&H90000000,-1,-1,0,0,100,100,0.5,0,1,2.0,1.2,8,60,60,%d,1\n",
			narratorFont, narratorSize, narratorMarginV))
	}
	sb.WriteString("\n")

	sb.WriteString("[Events]\n")
	sb.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	// Write static Narrator cues if top-box enabled
	for _, sdh := range sdhEntries {
		cleanText := strings.TrimSpace(sdh.Text)
		cleanText = strings.ReplaceAll(cleanText, "\n", " ")
		sb.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Narrator,,0,0,0,,[%s]\n", sdh.Start, sdh.End, cleanText))
	}

	// Write spoken dialogue cues
	for _, entry := range formattedSpeech {
		text := strings.TrimSpace(entry.Text)
		if preset.Uppercase {
			text = strings.ToUpper(text)
		}
		text = strings.ReplaceAll(text, "\n", " ")

		if preset.TransformTag != "" {
			text = preset.TransformTag + text
		}

		sb.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", entry.Start, entry.End, text))
	}

	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}

// ExportKaraokeASS exports subtitle entries into TikTok-style ASS format with yellow active word styling and thick outline.
func ExportKaraokeASS(entries []SubtitleEntry, outputPath string, fontSize int, isShorts bool) error {
	return ExportKaraokeASSWithFont(entries, outputPath, fontSize, isShorts, "Impact")
}

// ExportKaraokeASSWithFont exports subtitle entries into TikTok-style ASS format with custom font name.
func ExportKaraokeASSWithFont(entries []SubtitleEntry, outputPath string, fontSize int, isShorts bool, fontName string) error {
	return ExportPresetASS(entries, outputPath, "hormozi", fontSize, isShorts, fontName, "strip")
}

// AdjustSubtitlesForJumpCuts shifts and filters subtitle entries after silence intervals have been excised.
func AdjustSubtitlesForJumpCuts(entries []SubtitleEntry, gaps []detector.SilenceGap) []SubtitleEntry {
	if len(gaps) == 0 || len(entries) == 0 {
		return entries
	}

	var adjusted []SubtitleEntry

	for _, entry := range entries {
		tStart := parseTimestampToSec(entry.Start)
		tEnd := parseTimestampToSec(entry.End)

		newStart, _ := mapTimeAfterJumpCuts(tStart, gaps)
		newEnd, _ := mapTimeAfterJumpCuts(tEnd, gaps)

		// If the entry has a positive duration after cut
		if newEnd-newStart >= 0.08 {
			adjusted = append(adjusted, SubtitleEntry{
				Start: formatASSTime(newStart),
				End:   formatASSTime(newEnd),
				Text:  entry.Text,
			})
		}
	}

	return adjusted
}

func mapTimeAfterJumpCuts(t float64, gaps []detector.SilenceGap) (float64, bool) {
	shift := 0.0
	for _, g := range gaps {
		if t >= g.EndSec {
			shift += g.Duration()
		} else if t > g.StartSec && t < g.EndSec {
			return g.StartSec - shift, true
		}
	}
	return t - shift, false
}
