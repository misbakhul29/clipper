package transcriber

import (
	"fmt"
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
func FetchSubtitles(inputStr, outputDir string) ([]SubtitleEntry, error) {
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
	binPath := findYtDlpBinary()
	if binPath == "" {
		return nil, fmt.Errorf("yt-dlp binary required to fetch YouTube subtitles")
	}

	subTemplate := filepath.Join(outputDir, "sub_%(id)s")
	args := []string{
		"--write-auto-sub",
		"--write-sub",
		"--sub-lang", "en,id,en-orig",
		"--sub-format", "vtt",
		"--skip-download",
		"-o", subTemplate,
		inputStr,
	}

	cmd := exec.Command(binPath, args...)
	_ = cmd.Run()

	// Look for generated VTT files in outputDir
	matches, _ := filepath.Glob(filepath.Join(outputDir, "sub_*.vtt"))
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
