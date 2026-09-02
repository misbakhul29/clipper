package detector

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// SilenceGap represents an interval of silence detected in an audio track.
type SilenceGap struct {
	StartSec float64 `json:"start_sec"`
	EndSec   float64 `json:"end_sec"`
}

// Duration returns the length of the silence gap in seconds.
func (s SilenceGap) Duration() float64 {
	return s.EndSec - s.StartSec
}

// KeptInterval represents an active speech interval to be preserved during jump-cutting.
type KeptInterval struct {
	StartSec float64 `json:"start_sec"`
	EndSec   float64 `json:"end_sec"`
}

// Duration returns the length of the kept interval in seconds.
func (k KeptInterval) Duration() float64 {
	return k.EndSec - k.StartSec
}

// DetectSilenceGaps analyzes the segment [startSec, startSec+durationSec] of inputFile
// and returns all silence gaps with duration >= minSilence.
func DetectSilenceGaps(ffmpegPath, inputFile string, startSec, durationSec, noiseDb, minSilence float64) ([]SilenceGap, error) {
	if ffmpegPath == "" {
		p, err := exec.LookPath("ffmpeg")
		if err != nil {
			return nil, fmt.Errorf("ffmpeg not found in system PATH: %w", err)
		}
		ffmpegPath = p
	}

	if noiseDb == 0 {
		noiseDb = -30.0
	}
	if minSilence <= 0 {
		minSilence = 1.0
	}

	filterArg := fmt.Sprintf("silencedetect=noise=%.1fdB:d=%.2f", noiseDb, minSilence)
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", startSec),
		"-t", fmt.Sprintf("%.3f", durationSec),
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

	return ParseSilenceGaps(output, durationSec, minSilence)
}

var (
	silStartRe = regexp.MustCompile(`silence_start:\s*([\d\.]+)`)
	silEndRe   = regexp.MustCompile(`silence_end:\s*([\d\.]+)`)
)

// ParseSilenceGaps extracts SilenceGap intervals from FFmpeg silencedetect stderr output.
func ParseSilenceGaps(output string, totalDuration, minSilence float64) ([]SilenceGap, error) {
	lines := strings.Split(output, "\n")
	var gaps []SilenceGap

	inSilence := false
	currentStart := 0.0

	for _, line := range lines {
		if m := silStartRe.FindStringSubmatch(line); len(m) > 1 {
			if val, err := strconv.ParseFloat(m[1], 64); err == nil {
				currentStart = val
				inSilence = true
			}
		} else if m := silEndRe.FindStringSubmatch(line); len(m) > 1 {
			if val, err := strconv.ParseFloat(m[1], 64); err == nil {
				en := val
				if en > totalDuration {
					en = totalDuration
				}
				st := currentStart
				if !inSilence {
					// Audio started in silence without a preceding silence_start log
					st = 0.0
				}
				dur := en - st
				if dur >= minSilence && en > st {
					gaps = append(gaps, SilenceGap{
						StartSec: st,
						EndSec:   en,
					})
				}
				inSilence = false
			}
		}
	}

	// If audio ended while still in silence
	if inSilence {
		en := totalDuration
		dur := en - currentStart
		if dur >= minSilence && en > currentStart {
			gaps = append(gaps, SilenceGap{
				StartSec: currentStart,
				EndSec:   en,
			})
		}
	}

	return gaps, nil
}

// CalculateJumpCutIntervals computes the intervals of active speech to preserve,
// padding each speech segment with marginSec to prevent clipping the ends of words.
func CalculateJumpCutIntervals(totalDuration float64, gaps []SilenceGap, marginSec float64) ([]KeptInterval, []SilenceGap) {
	if totalDuration <= 0 {
		return nil, nil
	}
	if len(gaps) == 0 {
		return []KeptInterval{{StartSec: 0.0, EndSec: totalDuration}}, nil
	}
	if marginSec < 0 {
		marginSec = 0.0
	}

	var kept []KeptInterval
	var removed []SilenceGap
	currentPos := 0.0

	for _, g := range gaps {
		// Calculate cut boundary with margins
		cutStart := g.StartSec + marginSec
		cutEnd := g.EndSec - marginSec

		// If gap is at the very beginning of the clip, don't add margin before speech
		if g.StartSec <= 0.05 {
			cutStart = 0.0
		}
		// If gap is at the very end of the clip, cut until the end
		if g.EndSec >= totalDuration-0.05 {
			cutEnd = totalDuration
		}

		if cutEnd > cutStart {
			// There is an actual gap to remove after subtracting speech margin
			if cutStart > currentPos {
				kept = append(kept, KeptInterval{
					StartSec: currentPos,
					EndSec:   cutStart,
				})
			}
			removed = append(removed, SilenceGap{
				StartSec: cutStart,
				EndSec:   cutEnd,
			})
			currentPos = cutEnd
		}
	}

	if currentPos < totalDuration {
		kept = append(kept, KeptInterval{
			StartSec: currentPos,
			EndSec:   totalDuration,
		})
	}

	if len(kept) == 0 {
		kept = append(kept, KeptInterval{StartSec: 0.0, EndSec: totalDuration})
	}

	return kept, removed
}

// BuildJumpCutFilter generates an FFmpeg filter_complex chain of trim/atrim + concat
// that joins the kept speech intervals seamlessly.
func BuildJumpCutFilter(intervals []KeptInterval) string {
	if len(intervals) <= 1 {
		return ""
	}

	var parts []string
	var inputs []string

	for i, in := range intervals {
		vLabel := fmt.Sprintf("v%d", i)
		aLabel := fmt.Sprintf("a%d", i)

		parts = append(parts, fmt.Sprintf("[0:v]trim=start=%.3f:end=%.3f,setpts=PTS-STARTPTS[%s]", in.StartSec, in.EndSec, vLabel))
		parts = append(parts, fmt.Sprintf("[0:a]atrim=start=%.3f:end=%.3f,asetpts=PTS-STARTPTS[%s]", in.StartSec, in.EndSec, aLabel))
		inputs = append(inputs, fmt.Sprintf("[%s][%s]", vLabel, aLabel))
	}

	concatPart := fmt.Sprintf("%sconcat=n=%d:v=1:a=1[vout][aout]", strings.Join(inputs, ""), len(intervals))
	parts = append(parts, concatPart)

	return strings.Join(parts, ";")
}
