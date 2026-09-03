package detector

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ThumbnailCandidate represents a candidate video frame evaluated for thumbnail quality.
type ThumbnailCandidate struct {
	TimeSec float64 `json:"time_sec"`
	Score   float64 `json:"score"`
	HasFace bool    `json:"has_face"`
}

// FindBestHookFrames scans the opening 3–5 seconds of videoPath, detects prominent facial expressions
// using embedded cascade classifier, and returns the top timestamps suitable for cover art.
func FindBestHookFrames(ffmpegPath, videoPath string, clipDuration float64, count int) ([]float64, error) {
	if count <= 0 {
		count = 1
	}
	if count > 3 {
		count = 3
	}

	if ffmpegPath == "" {
		if p, err := exec.LookPath("ffmpeg"); err == nil {
			ffmpegPath = p
		} else {
			return defaultFallbackTimes(clipDuration, count), fmt.Errorf("ffmpeg not found: %w", err)
		}
	}

	if clipDuration <= 0 {
		clipDuration = 5.0
	}

	windowDuration := math.Min(5.0, clipDuration)
	if windowDuration < 1.0 {
		windowDuration = clipDuration
	}

	tmpDir, err := os.MkdirTemp("", "clipper_thumb_*")
	if err != nil {
		return defaultFallbackTimes(clipDuration, count), err
	}
	defer os.RemoveAll(tmpDir)

	sampleFPS := 2.0
	if windowDuration < 2.5 {
		sampleFPS = 4.0
	}

	outPattern := filepath.Join(tmpDir, "sample_%04d.jpg")
	args := []string{
		"-y",
		"-ss", "0.000",
		"-t", fmt.Sprintf("%.3f", windowDuration),
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%.2f,scale=480:-2", sampleFPS),
		"-q:v", "3",
		outPattern,
	}

	cmd := exec.Command(ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return defaultFallbackTimes(clipDuration, count), nil
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "sample_*.jpg"))
	if err != nil || len(files) == 0 {
		return defaultFallbackTimes(clipDuration, count), nil
	}
	sort.Strings(files)

	cls, _ := InitFaceClassifier()

	var candidates []ThumbnailCandidate
	for i, fPath := range files {
		timeOffset := float64(i) / sampleFPS
		if timeOffset > clipDuration {
			continue
		}

		var score float64
		var hasFace bool

		if cls != nil {
			det, found := detectPrimaryFace(cls, fPath)
			if found {
				hasFace = true
				// Score based on confidence and face size
				score = float64(det.Score) * (1.0 + det.Scale/80.0)
			}
		}

		if !hasFace {
			// Fallback: prefer frames slightly after the start (1.0s - 2.0s) to avoid black opening fade
			score = 1.0 - math.Abs(timeOffset-1.5)/10.0
		}

		candidates = append(candidates, ThumbnailCandidate{
			TimeSec: timeOffset,
			Score:   score,
			HasFace: hasFace,
		})
	}

	if len(candidates) == 0 {
		return defaultFallbackTimes(clipDuration, count), nil
	}

	// Sort candidates by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Select top candidates with time separation (at least 0.5s apart)
	var selected []float64
	for _, c := range candidates {
		tooClose := false
		for _, s := range selected {
			if math.Abs(c.TimeSec-s) < 0.5 {
				tooClose = true
				break
			}
		}
		if !tooClose {
			selected = append(selected, c.TimeSec)
			if len(selected) >= count {
				break
			}
		}
	}

	if len(selected) == 0 {
		return defaultFallbackTimes(clipDuration, count), nil
	}

	return selected, nil
}

func defaultFallbackTimes(clipDuration float64, count int) []float64 {
	base := []float64{0.5, 1.5, 2.5}
	var res []float64
	for _, t := range base {
		if t < clipDuration {
			res = append(res, t)
		}
		if len(res) >= count {
			break
		}
	}
	if len(res) == 0 {
		res = append(res, 0.0)
	}
	return res
}

// ExtractThumbnail extracts a single high-quality frame from videoPath at timeSec.
func ExtractThumbnail(ffmpegPath, videoPath string, timeSec float64, outputPath string) error {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", timeSec),
		"-i", videoPath,
		"-frames:v", "1",
		"-q:v", "2",
		outputPath,
	}
	cmd := exec.Command(ffmpegPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed extracting thumbnail: %w (output: %s)", err, string(out))
	}
	return nil
}

// ExtractThumbnailWithHook extracts a high-quality frame and overlays a viral hook title cover banner.
func ExtractThumbnailWithHook(ffmpegPath, videoPath string, timeSec float64, hookTitle string, isShorts bool, outputPath string) error {
	hookTitle = strings.TrimSpace(hookTitle)
	if hookTitle == "" {
		return ExtractThumbnail(ffmpegPath, videoPath, timeSec, outputPath)
	}

	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	tmpDir := filepath.Dir(outputPath)
	tmpAssFile, err := os.CreateTemp(tmpDir, ".thumb_overlay_*.ass")
	if err != nil {
		return ExtractThumbnail(ffmpegPath, videoPath, timeSec, outputPath)
	}
	tmpAss := tmpAssFile.Name()
	_ = tmpAssFile.Close()
	defer os.Remove(tmpAss)

	assContent := BuildThumbnailASS(hookTitle, isShorts)

	if err := os.WriteFile(tmpAss, []byte(assContent), 0644); err != nil {
		return ExtractThumbnail(ffmpegPath, videoPath, timeSec, outputPath)
	}

	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", timeSec),
		"-i", videoPath,
		"-frames:v", "1",
		"-vf", fmt.Sprintf("ass=%s", tmpAss),
		"-q:v", "2",
		outputPath,
	}

	cmd := exec.Command(ffmpegPath, args...)
	if _, err := cmd.CombinedOutput(); err != nil {
		// Fallback to clean frame if ASS filter fails
		return ExtractThumbnail(ffmpegPath, videoPath, timeSec, outputPath)
	}

	return nil
}

// BuildThumbnailASS generates ASS subtitle script specifically optimized for thumbnail hook covers.
func BuildThumbnailASS(hookTitle string, isShorts bool) string {
	playResX := 1920
	playResY := 1080
	fontSize := 64
	marginV := 180
	alignment := 8 // Top-Center

	if isShorts {
		playResX = 1080
		playResY = 1920
		fontSize = 68
		marginV = 450 // Positioned in upper-third to avoid being obscured by TikTok/Reels UI icons
		alignment = 8
	}

	// Sanitize ASS special characters to prevent broken markup tags
	hookTitle = strings.ReplaceAll(hookTitle, "{", "(")
	hookTitle = strings.ReplaceAll(hookTitle, "}", ")")
	hookTitle = strings.ReplaceAll(hookTitle, "\\", "/")

	// Auto-wrap long titles into 2 or 3 lines with \N
	wrapped := wrapTitleText(hookTitle, 4)

	var sb strings.Builder
	sb.WriteString("[Script Info]\n")
	sb.WriteString("ScriptType: v4.00+\n")
	sb.WriteString(fmt.Sprintf("PlayResX: %d\n", playResX))
	sb.WriteString(fmt.Sprintf("PlayResY: %d\n", playResY))
	sb.WriteString("ScaledBorderAndShadow: yes\n\n")

	sb.WriteString("[V4+ Styles]\n")
	sb.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	// Impact font, vibrant yellow fill (&H0000FFFF), thick 6.0 black outline, 3.0 shadow, upper-third alignment
	sb.WriteString(fmt.Sprintf("Style: HookCover,Impact,%d,&H0000FFFF,&H0000FFFF,&H00000000,&H90000000,-1,0,0,0,100,100,1,0,1,6.0,3.0,%d,60,60,%d,1\n\n",
		fontSize, alignment, marginV))

	sb.WriteString("[Events]\n")
	sb.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	sb.WriteString(fmt.Sprintf("Dialogue: 0,0:00:00.00,0:00:10.00,HookCover,,0,0,0,,%s\n", wrapped))

	return sb.String()
}

func wrapTitleText(title string, wordsPerLine int) string {
	if wordsPerLine <= 0 {
		wordsPerLine = 4
	}
	words := strings.Fields(title)
	if len(words) <= wordsPerLine {
		return title
	}

	var lines []string
	for i := 0; i < len(words); i += wordsPerLine {
		end := i + wordsPerLine
		if end > len(words) {
			end = len(words)
		}
		lines = append(lines, strings.Join(words[i:end], " "))
	}
	return strings.Join(lines, "\\N")
}
