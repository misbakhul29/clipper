package ui

import (
	"testing"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		sec      float64
		expected string
	}{
		{0.0, "00:00"},
		{5.4, "00:05"},
		{59.6, "01:00"},
		{65.0, "01:05"},
		{3599.0, "59:59"},
		{-1.0, "--:--"},
	}

	for _, c := range cases {
		got := FormatDuration(c.sec)
		if got != c.expected {
			t.Errorf("FormatDuration(%f) = %q, want %q", c.sec, got, c.expected)
		}
	}
}

func TestProgressBar(t *testing.T) {
	pb := NewProgressBar("Test Task", 100.0)
	if pb.Total != 100.0 {
		t.Errorf("expected total 100.0, got %f", pb.Total)
	}

	// Update should not panic
	pb.Update(50.0, "2.5x", "00:10")
	if pb.Current != 50.0 {
		t.Errorf("expected current 50.0, got %f", pb.Current)
	}

	// Finish should mark finished
	pb.Finish("Completed!")
	if !pb.finished {
		t.Errorf("expected finished = true")
	}

	// Double finish should not panic
	pb.Finish("Again")
}
