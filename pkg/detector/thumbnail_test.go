package detector

import (
	"strings"
	"testing"
)

func TestBuildThumbnailASS(t *testing.T) {
	t.Run("Shorts 9:16 Canvas", func(t *testing.T) {
		ass := BuildThumbnailASS("Rahasia Viral Sukses 100 Juta dari YouTube", true)

		if !strings.Contains(ass, "PlayResX: 1080") || !strings.Contains(ass, "PlayResY: 1920") {
			t.Errorf("expected 1080x1920 canvas for shorts thumbnail, got:\n%s", ass)
		}
		if !strings.Contains(ass, "HookCover") {
			t.Errorf("expected HookCover style, got:\n%s", ass)
		}
		// Check that long title wrapped with \N
		if !strings.Contains(ass, "\\N") {
			t.Errorf("expected line wrap \\N for long title, got:\n%s", ass)
		}
	})

	t.Run("Landscape 16:9 Canvas", func(t *testing.T) {
		ass := BuildThumbnailASS("Short Title", false)

		if !strings.Contains(ass, "PlayResX: 1920") || !strings.Contains(ass, "PlayResY: 1080") {
			t.Errorf("expected 1920x1080 canvas for landscape thumbnail, got:\n%s", ass)
		}
		if !strings.Contains(ass, "Short Title") {
			t.Errorf("expected Short Title in dialogue text")
		}
	})

	t.Run("Sanitize Brackets and Backslashes", func(t *testing.T) {
		ass := BuildThumbnailASS("Rahasia {Keren} AC\\DC", false)
		if strings.Contains(ass, "{Keren}") {
			t.Errorf("expected curly braces to be replaced with parentheses, got:\n%s", ass)
		}
		if !strings.Contains(ass, "(Keren)") {
			t.Errorf("expected (Keren) in dialogue, got:\n%s", ass)
		}
		if strings.Contains(ass, "AC\\DC") {
			t.Errorf("expected backslash to be replaced with slash, got:\n%s", ass)
		}
	})
}

func TestWrapTitleText(t *testing.T) {
	cases := []struct {
		input        string
		wordsPerLine int
		expected     string
	}{
		{"Hello World", 4, "Hello World"},
		{"Satu Dua Tiga Empat Lima Enam Tujuh", 4, "Satu Dua Tiga Empat\\NLima Enam Tujuh"},
		{"Single", 2, "Single"},
	}

	for _, c := range cases {
		got := wrapTitleText(c.input, c.wordsPerLine)
		if got != c.expected {
			t.Errorf("wrapTitleText(%q, %d) = %q, want %q", c.input, c.wordsPerLine, got, c.expected)
		}
	}
}

func TestDefaultFallbackTimes(t *testing.T) {
	t1 := defaultFallbackTimes(10.0, 1)
	if len(t1) != 1 || t1[0] != 0.5 {
		t.Errorf("expected [0.5], got: %v", t1)
	}

	t3 := defaultFallbackTimes(10.0, 3)
	if len(t3) != 3 || t3[1] != 1.5 || t3[2] != 2.5 {
		t.Errorf("expected [0.5, 1.5, 2.5], got: %v", t3)
	}

	tShort := defaultFallbackTimes(1.0, 3)
	if len(tShort) != 1 || tShort[0] != 0.5 {
		t.Errorf("expected 1 fallback time for 1.0s clip, got: %v", tShort)
	}
}
