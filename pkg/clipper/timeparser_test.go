package clipper

import (
	"testing"
)

func TestParseTimeSeconds(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		wantErr  bool
	}{
		{"10", 10.0, false},
		{"10.5", 10.5, false},
		{"01:30", 90.0, false},
		{"00:01:30", 90.0, false},
		{"01:02:03", 3723.0, false},
		{"01:02:03.500", 3723.5, false},
		{"invalid", 0, true},
		{"10:60:70:80", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseTimeSeconds(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseTimeSeconds(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("ParseTimeSeconds(%q) = %v, expected %v", tt.input, got, tt.expected)
		}
	}
}

func TestFormatSeconds(t *testing.T) {
	formatted := FormatSeconds(90.5)
	expected := "00:01:30.500"
	if formatted != expected {
		t.Errorf("FormatSeconds(90.5) = %q, expected %q", formatted, expected)
	}
}

func TestCalculateDuration(t *testing.T) {
	startSec, endSec, duration, err := CalculateDuration("00:01:00", "00:02:30.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if startSec != 60.0 || endSec != 150.5 || duration != 90.5 {
		t.Errorf("got start=%.1f end=%.1f dur=%.1f; want start=60 end=150.5 dur=90.5", startSec, endSec, duration)
	}

	_, _, _, err = CalculateDuration("00:02:00", "00:01:00")
	if err == nil {
		t.Errorf("expected error when start > end, got nil")
	}
}
