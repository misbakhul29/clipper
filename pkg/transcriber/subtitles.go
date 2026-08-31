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
)

// SubtitleEntry represents a timestamped text entry from a subtitle file.
type SubtitleEntry struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Text  string `json:"text"`
}

var vttTimeRegex = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}[\.\,]\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2}[\.\,]\d{3})`)

// ParseVTT parses VTT or SRT subtitle file content into a slice of SubtitleEntry.
func ParseVTT(content string) ([]SubtitleEntry, error) {
	lines := strings.Split(content, "\n")
	var entries []SubtitleEntry

	var currentStart, currentEnd string
	var currentText []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for timestamp line
		if match := vttTimeRegex.FindStringSubmatch(line); len(match) > 2 {
			// Save previous entry if exists
			if currentStart != "" && len(currentText) > 0 {
				fullText := strings.TrimSpace(strings.Join(currentText, " "))
				if fullText != "" {
					entries = append(entries, SubtitleEntry{
						Start: normalizeTimestamp(currentStart),
						End:   normalizeTimestamp(currentEnd),
						Text:  cleanSubtitleText(fullText),
					})
				}
				currentText = nil
			}

			currentStart = match[1]
			currentEnd = match[2]
			continue
		}

		if line == "" || strings.HasPrefix(line, "WEBVTT") || strings.HasPrefix(line, "NOTE") || isNumeric(line) {
			continue
		}

		if currentStart != "" {
			currentText = append(currentText, line)
		}
	}

	// Save last entry
	if currentStart != "" && len(currentText) > 0 {
		fullText := strings.TrimSpace(strings.Join(currentText, " "))
		if fullText != "" {
			entries = append(entries, SubtitleEntry{
				Start: normalizeTimestamp(currentStart),
				End:   normalizeTimestamp(currentEnd),
				Text:  cleanSubtitleText(fullText),
			})
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no subtitle entries found in content")
	}

	return entries, nil
}

// FetchSubtitles attempts to retrieve subtitles for a YouTube URL or local file path.
func FetchSubtitles(inputStr, outputDir, lang string) ([]SubtitleEntry, error) {
	if outputDir == "" {
		outputDir = "."
	}

	// Case 1: Input is already a local .vtt or .srt file
	if (strings.HasSuffix(inputStr, ".vtt") || strings.HasSuffix(inputStr, ".srt")) && fileExists(inputStr) {
		data, err := os.ReadFile(inputStr)
		if err != nil {
			return nil, err
		}
		return ParseVTT(string(data))
	}

	// Case 2: YouTube URL download subtitles via yt-dlp
	if lang != "" {
		matches, _ := filepath.Glob(filepath.Join(outputDir, fmt.Sprintf("*%s*.vtt", lang)))
		if len(matches) > 0 {
			data, err := os.ReadFile(matches[0])
			if err == nil && len(data) > 0 {
				return ParseVTT(string(data))
			}
		}
	} else {
		matches, _ := filepath.Glob(filepath.Join(outputDir, "sub_*.vtt"))
		if len(matches) > 0 {
			data, err := os.ReadFile(matches[0])
			if err == nil && len(data) > 0 {
				return ParseVTT(string(data))
			}
		}
	}

	binPath := findYtDlpBinary()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp binary required to fetch YouTube subtitles")
	}

	subTemplate := filepath.Join(outputDir, "sub_%(id)s")
	subLangArg := "id,en,ko,all"
	if lang != "" {
		subLangArg = fmt.Sprintf("%s,id,en,ko,all", lang)
	}

	args := []string{
		"--write-auto-sub",
		"--write-sub",
		"--sub-lang", subLangArg,
		"--sub-format", "vtt",
		"--skip-download",
		"-o", subTemplate,
		inputStr,
	}

	cmd := exec.Command(binPath, args...)
	_ = cmd.Run()

	var matches []string
	if lang != "" {
		matches, _ = filepath.Glob(filepath.Join(outputDir, fmt.Sprintf("*%s*.vtt", lang)))
	}
	if len(matches) == 0 {
		matches, _ = filepath.Glob(filepath.Join(outputDir, "sub_*.vtt"))
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("failed to download subtitles from YouTube input: %s", inputStr)
	}

	// Use first match
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read downloaded subtitle file '%s': %w", matches[0], err)
	}

	return ParseVTT(string(data))
}

func findYtDlpBinary() string {
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path
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
	if fontSize <= 0 {
		fontSize = 48
	}

	marginV := 160
	if isShorts {
		marginV = 420 // Position subtitle higher up on 9:16 Shorts frame
	}

	var sb strings.Builder
	sb.WriteString("[Script Info]\n")
	sb.WriteString("ScriptType: v4.00+\n")
	sb.WriteString("PlayResX: 1080\n")
	sb.WriteString("PlayResY: 1920\n")
	sb.WriteString("ScaledBorderAndShadow: yes\n\n")

	sb.WriteString("[V4+ Styles]\n")
	sb.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	sb.WriteString(fmt.Sprintf("Style: Default,Arial,%d,&H00FFFFFF,&H00000000,&H00000000,&H80000000,-1,0,0,0,100,100,0,0,1,4,1,2,40,40,%d,1\n\n", fontSize, marginV))

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

// ExportKaraokeASS exports subtitle entries into TikTok-style ASS format with yellow active word styling and thick outline.
func ExportKaraokeASS(entries []SubtitleEntry, outputPath string, fontSize int, isShorts bool) error {
	if fontSize <= 0 {
		fontSize = 54
	}

	marginV := 180
	if isShorts {
		marginV = 440 // Centered in lower third of 9:16 frame
	}

	var sb strings.Builder
	sb.WriteString("[Script Info]\n")
	sb.WriteString("ScriptType: v4.00+\n")
	sb.WriteString("PlayResX: 1080\n")
	sb.WriteString("PlayResY: 1920\n")
	sb.WriteString("ScaledBorderAndShadow: yes\n\n")

	sb.WriteString("[V4+ Styles]\n")
	sb.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	// Style: Vibrant Yellow (&H0000FFFF) text with thick black outline (&H00000000) and shadow
	sb.WriteString(fmt.Sprintf("Style: Default,Impact,%d,&H0000FFFF,&H00000000,&H00000000,&H90000000,-1,0,0,0,105,105,1,0,1,5,2,2,40,40,%d,1\n\n", fontSize, marginV))

	sb.WriteString("[Events]\n")
	sb.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	for _, entry := range entries {
		upperText := strings.ToUpper(strings.TrimSpace(entry.Text))
		cleanText := strings.ReplaceAll(upperText, "\n", " ")
		sb.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", entry.Start, entry.End, cleanText))
	}

	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}
