package transcriber

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetSubtitlePreset(t *testing.T) {
	t.Run("Hormozi preset", func(t *testing.T) {
		p := GetSubtitlePreset("hormozi", true)
		if p.Name != "hormozi" {
			t.Errorf("expected hormozi, got %s", p.Name)
		}
		if p.PrimaryColor != "&H0000FFFF" {
			t.Errorf("expected yellow &H0000FFFF, got %s", p.PrimaryColor)
		}
		if !strings.Contains(p.TransformTag, "fscx115") {
			t.Errorf("expected bounce transform tag, got %s", p.TransformTag)
		}
		if p.MaxWords != 2 {
			t.Errorf("expected 2 words, got %d", p.MaxWords)
		}
	})

	t.Run("Minimal and Devon alias", func(t *testing.T) {
		p1 := GetSubtitlePreset("minimal", true)
		p2 := GetSubtitlePreset("devon", true)
		if p1.Name != "minimal" || p2.Name != "minimal" {
			t.Errorf("expected minimal for both, got %s and %s", p1.Name, p2.Name)
		}
		if p1.PrimaryColor != "&H00FFFFFF" {
			t.Errorf("expected white primary color, got %s", p1.PrimaryColor)
		}
	})

	t.Run("Neon preset", func(t *testing.T) {
		p := GetSubtitlePreset("neon", true)
		if p.Name != "neon" {
			t.Errorf("expected neon, got %s", p.Name)
		}
		if p.PrimaryColor != "&H00FFFF00" {
			t.Errorf("expected cyan &H00FFFF00, got %s", p.PrimaryColor)
		}
		if !strings.Contains(p.TransformTag, "blur") {
			t.Errorf("expected blur tag, got %s", p.TransformTag)
		}
	})

	t.Run("Cinematic preset", func(t *testing.T) {
		p := GetSubtitlePreset("cinematic", false)
		if p.Name != "cinematic" {
			t.Errorf("expected cinematic, got %s", p.Name)
		}
		if p.FontName != "Georgia" {
			t.Errorf("expected Georgia font, got %s", p.FontName)
		}
		if p.MaxWords != 7 {
			t.Errorf("expected 7 max words, got %d", p.MaxWords)
		}
	})
}

func TestExportPresetASS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipper_preset_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	entries := []SubtitleEntry{
		{Start: "0:00:01.00", End: "0:00:04.00", Text: "Hello this is a viral clip for tiktok"},
	}

	outHormozi := filepath.Join(tmpDir, "hormozi.ass")
	if err := ExportPresetASS(entries, outHormozi, "hormozi", 60, true, "Montserrat-Bold"); err != nil {
		t.Fatalf("ExportPresetASS hormozi failed: %v", err)
	}

	data, err := os.ReadFile(outHormozi)
	if err != nil {
		t.Fatalf("failed to read exported ASS: %v", err)
	}
	content := string(data)

	// Check custom font and size override
	if !strings.Contains(content, "Montserrat-Bold") {
		t.Errorf("expected Montserrat-Bold in ASS header, content: %s", content)
	}
	if !strings.Contains(content, ",60,") {
		t.Errorf("expected font size 60 in ASS header, content: %s", content)
	}
	// Check pop-in animation tag
	if !strings.Contains(content, `{\fscx115\fscy115`) {
		t.Errorf("expected pop-in bounce tag in dialogue, content: %s", content)
	}
	// Check 2-word micro-chunking
	if !strings.Contains(content, "HELLO THIS") {
		t.Errorf("expected 2-word chunk 'HELLO THIS', content: %s", content)
	}
}
