package ai

import (
	"strings"
	"testing"

	"github.com/misbakhul29/clipper/pkg/transcriber"
)

func TestBuildSubtitleRefineAndTranslatePrompts(t *testing.T) {
	entries := []transcriber.SubtitleEntry{
		{Start: "00:00:01.000", End: "00:00:03.000", Text: "[Musik] Narrator: Halo guys! ><+="},
		{Start: "00:00:03.500", End: "00:00:06.000", Text: "(Laughter) Selamat datang kembali."},
	}

	// Test 1: With Translation
	sysPrompt, userPrompt := BuildSubtitleRefineAndTranslatePrompts(entries, "id", true)
	if !strings.Contains(sysPrompt, "Clean, refine, and translate") {
		t.Errorf("expected translation prompt, got %s", sysPrompt)
	}
	if !strings.Contains(userPrompt, "Halo guys!") {
		t.Errorf("user prompt should have pre-cleaned text, got %s", userPrompt)
	}

	// Test 2: Without Translation (Refine & Clean only)
	sysPromptClean, _ := BuildSubtitleRefineAndTranslatePrompts(entries, "id", false)
	if !strings.Contains(sysPromptClean, "Clean, polish, and fix grammar") {
		t.Errorf("expected polish & clean prompt, got %s", sysPromptClean)
	}
}

func TestParseSubtitleRefineJSON(t *testing.T) {
	entries := []transcriber.SubtitleEntry{
		{Start: "00:00:01.000", End: "00:00:03.000", Text: "Original 1"},
		{Start: "00:00:03.500", End: "00:00:06.000", Text: "Original 2"},
	}

	rawJSON := `[
		{"id": 1, "text": "Halo semuanya!"},
		{"id": 2, "text": "Selamat datang kembali."}
	]`

	parsed, err := ParseSubtitleTranslationJSON(rawJSON, entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 items, got %d", len(parsed))
	}
	if parsed[0].Text != "Halo semuanya!" {
		t.Errorf("parsed[0] = %q, want 'Halo semuanya!'", parsed[0].Text)
	}
	if parsed[1].Text != "Selamat datang kembali." {
		t.Errorf("parsed[1] = %q, want 'Selamat datang kembali.'", parsed[1].Text)
	}
}
