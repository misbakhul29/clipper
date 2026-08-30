package clipper

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseTimeSeconds parses time string into float64 seconds.
// Accepted formats:
// - "SS" or "SS.mmm" (e.g. "90", "12.5")
// - "MM:SS" or "MM:SS.mmm" (e.g. "01:30", "05:12.4")
// - "HH:MM:SS" or "HH:MM:SS.mmm" (e.g. "01:02:03", "00:01:15.500")
func ParseTimeSeconds(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty time string")
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		sec, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid seconds format '%s': %w", s, err)
		}
		return sec, nil
	case 2:
		min, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes in '%s': %w", s, err)
		}
		sec, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid seconds in '%s': %w", s, err)
		}
		return min*60 + sec, nil
	case 3:
		hr, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hours in '%s': %w", s, err)
		}
		min, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes in '%s': %w", s, err)
		}
		sec, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid seconds in '%s': %w", s, err)
		}
		return hr*3600 + min*60 + sec, nil
	default:
		return 0, fmt.Errorf("unrecognized time format '%s'", s)
	}
}

// FormatSeconds formats float64 seconds into HH:MM:SS.mmm string suitable for FFmpeg.
func FormatSeconds(totalSec float64) string {
	if totalSec < 0 {
		totalSec = 0
	}
	hours := int(totalSec) / 3600
	minutes := (int(totalSec) % 3600) / 60
	seconds := totalSec - float64(hours*3600+minutes*60)
	return fmt.Sprintf("%02d:%02d:%06.3f", hours, minutes, seconds)
}

// CalculateDuration calculates duration in seconds between startStr and endStr.
func CalculateDuration(startStr, endStr string) (float64, float64, float64, error) {
	startSec, err := ParseTimeSeconds(startStr)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid start time '%s': %w", startStr, err)
	}
	endSec, err := ParseTimeSeconds(endStr)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid end time '%s': %w", endStr, err)
	}

	if endSec <= startSec {
		return 0, 0, 0, fmt.Errorf("end time (%s -> %.2fs) must be greater than start time (%s -> %.2fs)", endStr, endSec, startStr, startSec)
	}

	durationSec := endSec - startSec
	return startSec, endSec, durationSec, nil
}
