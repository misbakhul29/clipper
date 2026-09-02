package clipper

import (
	"os/exec"
	"testing"
)

func TestDetectHardwareEncoder(t *testing.T) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not available in PATH, skipping hardware encoder tests")
	}

	t.Run("Force CPU mode", func(t *testing.T) {
		prof := DetectHardwareEncoder(ffmpegPath, "cpu")
		if prof.IsHardware {
			t.Errorf("expected CPU profile, got hardware: %+v", prof)
		}
		if len(prof.Args) == 0 {
			t.Errorf("expected non-empty encoder args, got empty")
		}
	})

	t.Run("Auto mode returns valid encoder", func(t *testing.T) {
		prof := DetectHardwareEncoder(ffmpegPath, "auto")
		if prof.Encoder == "" {
			t.Errorf("expected non-empty encoder name")
		}
		if len(prof.Args) == 0 {
			t.Errorf("expected non-empty args")
		}
		if prof.DisplayName == "" {
			t.Errorf("expected non-empty display name")
		}
	})

	t.Run("Fallback for unavailable hardware encoder", func(t *testing.T) {
		// videotoolbox is unavailable on Linux, should gracefully fallback
		prof := DetectHardwareEncoder(ffmpegPath, "videotoolbox")
		if prof.Encoder == "" {
			t.Errorf("expected fallback encoder, got empty")
		}
	})

	t.Run("Caching returns cached profile", func(t *testing.T) {
		p1 := DetectHardwareEncoder(ffmpegPath, "cpu")
		p2 := DetectHardwareEncoder(ffmpegPath, "cpu")
		if p1.Encoder != p2.Encoder || p1.DisplayName != p2.DisplayName {
			t.Errorf("expected cached profile to match, got %v vs %v", p1, p2)
		}
	})
}
