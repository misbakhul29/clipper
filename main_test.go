package main

import (
	"testing"
)

func TestParseCLISegments(t *testing.T) {
	t.Run("Standard timestamps without titles", func(t *testing.T) {
		raw := "00:00-00:02, 01:15-01:30"
		segs, err := parseCLISegments(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(segs) != 2 {
			t.Fatalf("expected 2 segments, got %d", len(segs))
		}
		if segs[0].Start != "00:00" || segs[0].End != "00:02" || segs[0].Title != "" {
			t.Errorf("unexpected seg 0: %+v", segs[0])
		}
		if segs[1].Start != "01:15" || segs[1].End != "01:30" || segs[1].Title != "" {
			t.Errorf("unexpected seg 1: %+v", segs[1])
		}
	})

	t.Run("Timestamps with titles", func(t *testing.T) {
		raw := "00:05-00:25:Epic Moment, 01:00-01:45:Ending Speech"
		segs, err := parseCLISegments(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(segs) != 2 {
			t.Fatalf("expected 2 segments, got %d", len(segs))
		}
		if segs[0].Title != "Epic Moment" || segs[0].Start != "00:05" || segs[0].End != "00:25" {
			t.Errorf("unexpected seg 0: %+v", segs[0])
		}
		if segs[1].Title != "Ending Speech" || segs[1].Start != "01:00" || segs[1].End != "01:45" {
			t.Errorf("unexpected seg 1: %+v", segs[1])
		}
	})
}
