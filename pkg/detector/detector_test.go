package detector

import (
	"testing"
)

func TestParseSilenceOutput(t *testing.T) {
	mockOutput := `
[silencedetect @ 0x55d14e1a00] silence_start: 12.500
[silencedetect @ 0x55d14e1a00] silence_end: 15.200 | silence_duration: 2.700
[silencedetect @ 0x55d14e1a00] silence_start: 45.000
[silencedetect @ 0x55d14e1a00] silence_end: 48.100 | silence_duration: 3.100
`

	segments, err := ParseSilenceOutput(mockOutput, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(segments) != 2 {
		t.Fatalf("expected 2 active segments, got %d", len(segments))
	}

	if segments[0].Start != "00:00:00.000" || segments[0].End != "00:00:12.500" {
		t.Errorf("segment 0 mismatch: start=%s end=%s", segments[0].Start, segments[0].End)
	}
	if segments[1].Start != "00:00:15.200" || segments[1].End != "00:00:45.000" {
		t.Errorf("segment 1 mismatch: start=%s end=%s", segments[1].Start, segments[1].End)
	}
}

func TestParseSceneOutput(t *testing.T) {
	mockOutput := `
[Parsed_metadata_1 @ 0x55d14e2b00] pts_time:10.500
[Parsed_metadata_1 @ 0x55d14e2b00] pts_time:25.000
`

	segments, err := ParseSceneOutput(mockOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(segments) != 2 {
		t.Fatalf("expected 2 scene segments, got %d", len(segments))
	}

	if segments[0].Start != "00:00:00.000" || segments[0].End != "00:00:10.500" {
		t.Errorf("scene segment 0 mismatch: start=%s end=%s", segments[0].Start, segments[0].End)
	}
}
