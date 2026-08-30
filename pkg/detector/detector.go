package detector

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	silenceStartRegex = regexp.MustCompile(`silence_start:\s*([\d\.]+)`)
	silenceEndRegex   = regexp.MustCompile(`silence_end:\s*([\d\.]+)`)
	scenePtsTimeRegex = regexp.MustCompile(`pts_time:\s*([\d\.]+)`)
)

// DetectedSegment represents a detected video interval.
type DetectedSegment struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Title string `json:"title,omitempty"`
}

// DetectSilence analyzes video audio and detects active/spoken audio segments (non-silent intervals).
func DetectSilence(ffmpegPath, inputFile string, noiseDb float64, minDurationSec float64) ([]DetectedSegment, error) {
	if ffmpegPath == "" {
		p, err := exec.LookPath("ffmpeg")
		if err != nil {
			return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
		}
		ffmpegPath = p
	}

	if noiseDb == 0 {
		noiseDb = -30
	}
	if minDurationSec <= 0 {
		minDurationSec = 0.5
	}

	filterArg := fmt.Sprintf("silencedetect=noise=%.1fdB:d=%.2f", noiseDb, minDurationSec)
	args := []string{
		"-i", inputFile,
		"-af", filterArg,
		"-f", "null",
		"-",
	}

	cmd := exec.Command(ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run()
	output := stderr.String()

	return ParseSilenceOutput(output, minDurationSec)
}

// ParseSilenceOutput parses FFmpeg silencedetect output log into active non-silent segments.
func ParseSilenceOutput(output string, minSegmentDuration float64) ([]DetectedSegment, error) {
	lines := strings.Split(output, "\n")
	var silenceStarts []float64
	var silenceEnds []float64

	for _, line := range lines {
		if match := silenceStartRegex.FindStringSubmatch(line); len(match) > 1 {
			val, err := strconv.ParseFloat(match[1], 64)
			if err == nil {
				silenceStarts = append(silenceStarts, val)
			}
		}
		if match := silenceEndRegex.FindStringSubmatch(line); len(match) > 1 {
			val, err := strconv.ParseFloat(match[1], 64)
			if err == nil {
				silenceEnds = append(silenceEnds, val)
			}
		}
	}

	var activeSegments []DetectedSegment
	lastPos := 0.0

	for i := 0; i < len(silenceStarts); i++ {
		startSil := silenceStarts[i]
		if startSil-lastPos >= minSegmentDuration {
			activeSegments = append(activeSegments, DetectedSegment{
				Start: formatSeconds(lastPos),
				End:   formatSeconds(startSil),
				Title: fmt.Sprintf("speech_%03d", len(activeSegments)+1),
			})
		}
		if i < len(silenceEnds) {
			lastPos = silenceEnds[i]
		}
	}

	if len(activeSegments) == 0 && len(silenceStarts) == 0 {
		return nil, fmt.Errorf("no silence detected in audio stream")
	}

	return activeSegments, nil
}

// DetectScenes analyzes video frames and detects visual scene changes (cuts).
func DetectScenes(ffmpegPath, inputFile string, threshold float64) ([]DetectedSegment, error) {
	if ffmpegPath == "" {
		p, err := exec.LookPath("ffmpeg")
		if err != nil {
			return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
		}
		ffmpegPath = p
	}

	if threshold <= 0 {
		threshold = 0.3
	}

	filterArg := fmt.Sprintf("select='gt(scene,%.2f)',metadata=print", threshold)
	args := []string{
		"-i", inputFile,
		"-filter_complex", filterArg,
		"-f", "null",
		"-",
	}

	cmd := exec.Command(ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run()
	output := stderr.String()

	return ParseSceneOutput(output)
}

// ParseSceneOutput parses FFmpeg scene metadata print output into scene segments.
func ParseSceneOutput(output string) ([]DetectedSegment, error) {
	matches := scenePtsTimeRegex.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no scene changes detected with threshold")
	}

	var timestamps []float64
	timestamps = append(timestamps, 0.0)

	for _, match := range matches {
		if len(match) > 1 {
			val, err := strconv.ParseFloat(match[1], 64)
			if err == nil && val > 0 {
				timestamps = append(timestamps, val)
			}
		}
	}

	var segments []DetectedSegment
	for i := 0; i < len(timestamps)-1; i++ {
		start := timestamps[i]
		end := timestamps[i+1]
		if end-start >= 1.0 {
			segments = append(segments, DetectedSegment{
				Start: formatSeconds(start),
				End:   formatSeconds(end),
				Title: fmt.Sprintf("scene_%03d", len(segments)+1),
			})
		}
	}

	return segments, nil
}

func formatSeconds(totalSec float64) string {
	if totalSec < 0 {
		totalSec = 0
	}
	hours := int(totalSec) / 3600
	minutes := (int(totalSec) % 3600) / 60
	seconds := totalSec - float64(hours*3600+minutes*60)
	return fmt.Sprintf("%02d:%02d:%06.3f", hours, minutes, seconds)
}
