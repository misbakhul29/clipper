package detector

import (
	"math"
	"strings"
	"testing"
)

func TestParseSilenceGaps(t *testing.T) {
	sampleOutput := `
[silencedetect @ 0x55ac4638f340] silence_start: 3.123
[silencedetect @ 0x55ac4638f340] silence_end: 6.456 | silence_duration: 3.333
[silencedetect @ 0x55ac4638f340] silence_start: 12.000
[silencedetect @ 0x55ac4638f340] silence_end: 12.400 | silence_duration: 0.400
`
	// minSilence = 1.0 -> the 0.4s silence should be filtered out
	gaps, err := ParseSilenceGaps(sampleOutput, 20.0, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap >= 1.0s, got %d", len(gaps))
	}

	if math.Abs(gaps[0].StartSec-3.123) > 1e-3 || math.Abs(gaps[0].EndSec-6.456) > 1e-3 {
		t.Errorf("unexpected gap: %+v", gaps[0])
	}
}

func TestParseSilenceGaps_LeadingSilenceWithoutStart(t *testing.T) {
	sampleOutput := `
[silencedetect @ 0x55ac4638f340] silence_end: 2.500 | silence_duration: 2.500
[silencedetect @ 0x55ac4638f340] silence_start: 6.000
[silencedetect @ 0x55ac4638f340] silence_end: 9.000 | silence_duration: 3.000
`
	gaps, err := ParseSilenceGaps(sampleOutput, 10.0, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gaps) != 2 {
		t.Fatalf("expected 2 gaps, got %d", len(gaps))
	}
	if math.Abs(gaps[0].StartSec-0.0) > 1e-3 || math.Abs(gaps[0].EndSec-2.5) > 1e-3 {
		t.Errorf("gap 0 mismatch: %+v", gaps[0])
	}
	if math.Abs(gaps[1].StartSec-6.0) > 1e-3 || math.Abs(gaps[1].EndSec-9.0) > 1e-3 {
		t.Errorf("gap 1 mismatch: %+v", gaps[1])
	}
}

func TestParseSilenceGaps_TrailingSilenceWithoutEnd(t *testing.T) {
	sampleOutput := `
[silencedetect @ 0x55ac4638f340] silence_start: 7.000
`
	gaps, err := ParseSilenceGaps(sampleOutput, 10.0, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}
	if math.Abs(gaps[0].StartSec-7.0) > 1e-3 || math.Abs(gaps[0].EndSec-10.0) > 1e-3 {
		t.Errorf("gap mismatch: %+v", gaps[0])
	}
}

func TestCalculateJumpCutIntervals(t *testing.T) {
	t.Run("Silence in the middle with margin", func(t *testing.T) {
		gaps := []SilenceGap{
			{StartSec: 4.0, EndSec: 8.0}, // 4s silence
		}
		// totalDuration = 10s, margin = 0.2s
		// speech 1: 0.0 -> 4.2
		// removed: 4.2 -> 7.8 (3.6s removed)
		// speech 2: 7.8 -> 10.0
		kept, removed := CalculateJumpCutIntervals(10.0, gaps, 0.2)

		if len(kept) != 2 {
			t.Fatalf("expected 2 kept intervals, got %d", len(kept))
		}
		if len(removed) != 1 {
			t.Fatalf("expected 1 removed gap, got %d", len(removed))
		}

		if math.Abs(kept[0].StartSec-0.0) > 1e-3 || math.Abs(kept[0].EndSec-4.2) > 1e-3 {
			t.Errorf("kept[0] mismatch: %+v", kept[0])
		}
		if math.Abs(kept[1].StartSec-7.8) > 1e-3 || math.Abs(kept[1].EndSec-10.0) > 1e-3 {
			t.Errorf("kept[1] mismatch: %+v", kept[1])
		}
		if math.Abs(removed[0].StartSec-4.2) > 1e-3 || math.Abs(removed[0].EndSec-7.8) > 1e-3 {
			t.Errorf("removed[0] mismatch: %+v", removed[0])
		}
	})

	t.Run("No silence gaps", func(t *testing.T) {
		kept, removed := CalculateJumpCutIntervals(15.0, nil, 0.2)
		if len(kept) != 1 || kept[0].EndSec != 15.0 {
			t.Errorf("unexpected kept: %+v", kept)
		}
		if len(removed) != 0 {
			t.Errorf("expected 0 removed, got %d", len(removed))
		}
	})

	t.Run("Silence at start and end", func(t *testing.T) {
		gaps := []SilenceGap{
			{StartSec: 0.0, EndSec: 2.0},
			{StartSec: 8.0, EndSec: 10.0},
		}
		kept, removed := CalculateJumpCutIntervals(10.0, gaps, 0.2)

		// Start cut: 0.0 -> 1.8
		// Middle kept: 1.8 -> 8.2
		// End cut: 8.2 -> 10.0
		if len(kept) != 1 {
			t.Fatalf("expected 1 kept speech interval, got %d", len(kept))
		}
		if math.Abs(kept[0].StartSec-1.8) > 1e-3 || math.Abs(kept[0].EndSec-8.2) > 1e-3 {
			t.Errorf("unexpected kept: %+v", kept[0])
		}
		if len(removed) != 2 {
			t.Fatalf("expected 2 removed gaps, got %d", len(removed))
		}
	})
}

func TestBuildJumpCutFilter(t *testing.T) {
	intervals := []KeptInterval{
		{StartSec: 0.0, EndSec: 3.5},
		{StartSec: 6.0, EndSec: 10.0},
	}

	filter := BuildJumpCutFilter(intervals)
	if !strings.Contains(filter, "trim=start=0.000:end=3.500") {
		t.Errorf("filter missing interval 0: %s", filter)
	}
	if !strings.Contains(filter, "atrim=start=6.000:end=10.000") {
		t.Errorf("filter missing interval 1: %s", filter)
	}
	if !strings.Contains(filter, "concat=n=2:v=1:a=1[vout][aout]") {
		t.Errorf("filter missing concat: %s", filter)
	}
}

func TestBuildJumpCutFilter_SingleInterval(t *testing.T) {
	intervals := []KeptInterval{
		{StartSec: 2.5, EndSec: 10.0},
	}

	filter := BuildJumpCutFilter(intervals)
	if !strings.Contains(filter, "trim=start=2.500:end=10.000,setpts=PTS-STARTPTS[vout]") {
		t.Errorf("filter missing single vout: %s", filter)
	}
	if !strings.Contains(filter, "atrim=start=2.500:end=10.000,asetpts=PTS-STARTPTS[aout]") {
		t.Errorf("filter missing single aout: %s", filter)
	}
	if strings.Contains(filter, "concat") {
		t.Errorf("single interval should not have concat filter: %s", filter)
	}
}

func TestCalculateJumpCutIntervals_AllSilence(t *testing.T) {
	// Entire clip is silence (0.0 to 10.0)
	gaps := []SilenceGap{
		{StartSec: 0.0, EndSec: 10.0},
	}
	kept, removed := CalculateJumpCutIntervals(10.0, gaps, 0.2)

	// Should not remove the entire clip, but preserve it
	if len(kept) != 1 || kept[0].EndSec != 10.0 {
		t.Errorf("expected entire clip to be kept, got: %+v", kept)
	}
	if len(removed) != 0 {
		t.Errorf("expected 0 removed when whole video is silence, got: %+v", removed)
	}
}
