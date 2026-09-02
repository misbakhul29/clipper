package transcriber

import (
	"testing"

	"github.com/misbakhul29/clipper/pkg/detector"
)

func TestAdjustSubtitlesForJumpCuts(t *testing.T) {
	// Silence removed from 5.0s to 8.0s (3.0s cut)
	gaps := []detector.SilenceGap{
		{StartSec: 5.0, EndSec: 8.0},
	}

	entries := []SubtitleEntry{
		{Start: "0:00:01.00", End: "0:00:03.00", Text: "Hello before silence"},
		{Start: "0:00:05.50", End: "0:00:07.00", Text: "Cough during silence"}, // should be discarded
		{Start: "0:00:10.00", End: "0:00:13.00", Text: "After silence"},          // 10 - 3 = 7, 13 - 3 = 10
	}

	adjusted := AdjustSubtitlesForJumpCuts(entries, gaps)

	if len(adjusted) != 2 {
		t.Fatalf("expected 2 entries after jumpcut, got %d", len(adjusted))
	}

	// First cue unchanged
	if adjusted[0].Start != "0:00:01.00" || adjusted[0].End != "0:00:03.00" {
		t.Errorf("unexpected cue 0: %+v", adjusted[0])
	}

	// Third cue shifted by 3 seconds: 0:00:07.00 to 0:00:10.00
	if adjusted[1].Start != "0:00:07.00" || adjusted[1].End != "0:00:10.00" {
		t.Errorf("unexpected cue 1 (should be shifted by -3s): %+v", adjusted[1])
	}
}
