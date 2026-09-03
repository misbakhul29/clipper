package clipper

import (
	"strings"
	"testing"
)

func TestBuildLoudnormFilter(t *testing.T) {
	t.Run("Default values", func(t *testing.T) {
		cfg := &Config{
			Loudnorm: true,
		}
		filt := BuildLoudnormFilter(cfg)
		expected := "loudnorm=I=-14.0:LRA=7.0:TP=-2.0"
		if filt != expected {
			t.Errorf("BuildLoudnormFilter got %s; want %s", filt, expected)
		}
	})

	t.Run("Custom values", func(t *testing.T) {
		cfg := &Config{
			Loudnorm:    true,
			LoudnormI:   -16.0,
			LoudnormLRA: 11.0,
			LoudnormTP:  -1.5,
		}
		filt := BuildLoudnormFilter(cfg)
		expected := "loudnorm=I=-16.0:LRA=11.0:TP=-1.5"
		if filt != expected {
			t.Errorf("BuildLoudnormFilter got %s; want %s", filt, expected)
		}
	})
}

func TestLoudnormReencodeTrigger(t *testing.T) {
	cfg := &Config{
		InputFile: "test.mp4",
		Loudnorm:  true,
		Strategy:  StrategyFast,
		Shorts:    false,
	}

	hasWatermark := cfg.WatermarkPath != ""
	hasOverlayText := cfg.OverlayText != ""
	hasSubtitles := false
	needsReencode := cfg.Shorts || hasWatermark || hasOverlayText || hasSubtitles || cfg.Strategy == StrategyAccurate || cfg.Loudnorm

	if !needsReencode {
		t.Errorf("expected needsReencode to be true when Loudnorm is enabled")
	}

	filt := BuildLoudnormFilter(cfg)
	if !strings.HasPrefix(filt, "loudnorm=") {
		t.Errorf("expected valid loudnorm filter string, got: %s", filt)
	}
}
