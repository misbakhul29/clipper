package ai

import (
	"os"
	"strings"
	"testing"

	"github.com/misbakhul29/clipper/pkg/transcriber"
)

func TestParseAIHighlightsJSON(t *testing.T) {
	rawJSON := `[
  {"start": "00:01:10", "end": "00:01:45", "title": "funny_joke"},
  {"start": "00:05:00", "end": "00:05:30", "title": "key_tip"}
]`

	highlights, err := ParseAIHighlightsJSON(rawJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(highlights) != 2 {
		t.Fatalf("expected 2 highlights, got %d", len(highlights))
	}

	if highlights[0].Start != "00:01:10" || highlights[0].Title != "funny_joke" {
		t.Errorf("highlight 0 mismatch: %+v", highlights[0])
	}
}

func TestParseAIHighlightsJSON_MarkdownWrapped(t *testing.T) {
	markdownJSON := "```json\n" + `[
  {"start": "00:02:00", "end": "00:02:25", "title": "intro_highlight"}
]` + "\n```"

	highlights, err := ParseAIHighlightsJSON(markdownJSON)
	if err != nil {
		t.Fatalf("unexpected error parsing markdown wrapped JSON: %v", err)
	}

	if len(highlights) != 1 || highlights[0].Title != "intro_highlight" {
		t.Errorf("highlight mismatch: %+v", highlights)
	}
}

func TestParseAIHighlightsJSON_Truncated(t *testing.T) {
	truncatedJSON := `[
  {"start": "00:01:00", "end": "00:01:30", "title": "item1"},
  {"start": "00:02:00", "end": "00:02:30", "title": "item2"},
  {"start": "00:03:00", "end": "00:03:30", "title": "item3"`

	highlights, err := ParseAIHighlightsJSON(truncatedJSON)
	if err != nil {
		t.Fatalf("unexpected error parsing truncated JSON: %v", err)
	}

	if len(highlights) != 2 {
		t.Fatalf("expected 2 repaired highlights, got %d", len(highlights))
	}
}

func TestParseAIHighlightsJSON_NoisyTags(t *testing.T) {
	noisyJSON := `[{"start": "00:01:00", "end": "00:01:30", "title": "item1"}]</arg_value></tool_call>[{"start": "00:01:00", "end": "00:01:30", "title": "item1"}]`

	highlights, err := ParseAIHighlightsJSON(noisyJSON)
	if err != nil {
		t.Fatalf("unexpected error parsing noisy JSON: %v", err)
	}

	if len(highlights) != 1 {
		t.Fatalf("expected 1 highlight, got %d", len(highlights))
	}
}

func TestBuildHighlightPrompts_Shorts(t *testing.T) {
	entries := []transcriber.SubtitleEntry{
		{Start: "00:00:01", End: "00:00:05", Text: "Halo semuanya"},
		{Start: "00:00:06", End: "00:00:10", Text: "selamat datang di video ini"},
	}

	sysPrompt, userPrompt := BuildHighlightPrompts(entries, "id", true)
	if !strings.Contains(sysPrompt, "YouTube Shorts & TikTok") {
		t.Errorf("expected YouTube Shorts & TikTok in sysPrompt, got: %s", sysPrompt)
	}
	if !strings.Contains(sysPrompt, "between 20 seconds and 60 seconds") {
		t.Errorf("expected shorts duration rule in sysPrompt, got: %s", sysPrompt)
	}
	if !strings.Contains(sysPrompt, "Bahasa Indonesia") {
		t.Errorf("expected Indonesian language instruction in sysPrompt, got: %s", sysPrompt)
	}
	if !strings.Contains(userPrompt, "Halo semuanya") {
		t.Errorf("expected transcript text in userPrompt, got: %s", userPrompt)
	}
}

func TestBuildHighlightPrompts_NonShorts(t *testing.T) {
	entries := []transcriber.SubtitleEntry{
		{Start: "00:00:01", End: "00:00:05", Text: "Halo semuanya"},
		{Start: "00:00:06", End: "00:00:10", Text: "selamat datang di video ini"},
	}

	sysPrompt, _ := BuildHighlightPrompts(entries, "en", false)
	if !strings.Contains(sysPrompt, "YouTube video highlights and compilations") {
		t.Errorf("expected long-form highlights in sysPrompt, got: %s", sysPrompt)
	}
	if !strings.Contains(sysPrompt, "between 1 minute (60 seconds) and 5 minutes (300 seconds)") {
		t.Errorf("expected 1 to 5 min duration rule in sysPrompt, got: %s", sysPrompt)
	}
}

func TestResolveAPIKeyAndModel(t *testing.T) {
	// 1. Direct key provided
	key, model, err := resolveAPIKeyAndModel("my-key", "CUSTOM_ENV_TEST", "", "default-model", "TestProvider")
	if err != nil || key != "my-key" || model != "default-model" {
		t.Errorf("expected direct key and default model, got key=%s, model=%s, err=%v", key, model, err)
	}

	// 2. Env var fallback
	os.Setenv("CUSTOM_ENV_TEST", "env-secret-key")
	defer os.Unsetenv("CUSTOM_ENV_TEST")

	key, model, err = resolveAPIKeyAndModel("", "CUSTOM_ENV_TEST", "custom-model", "default-model", "TestProvider")
	if err != nil || key != "env-secret-key" || model != "custom-model" {
		t.Errorf("expected env key and custom model, got key=%s, model=%s, err=%v", key, model, err)
	}

	// 3. Missing key error
	_, _, err = resolveAPIKeyAndModel("", "NON_EXISTENT_ENV_KEY", "", "default-model", "TestProvider")
	if err == nil {
		t.Error("expected error when no key or env var is present")
	}
}

func TestBuildSubtitleTranslationPrompts(t *testing.T) {
	entries := []transcriber.SubtitleEntry{
		{Start: "00:00:01", End: "00:00:03", Text: "Hello world"},
		{Start: "00:00:04", End: "00:00:07", Text: "Welcome to our coding show"},
	}

	sysPrompt, userPrompt := BuildSubtitleTranslationPrompts(entries, "id")
	if !strings.Contains(sysPrompt, "Bahasa Indonesia") {
		t.Errorf("expected Indonesian language instruction in sysPrompt, got: %s", sysPrompt)
	}
	if !strings.Contains(userPrompt, "Hello world") {
		t.Errorf("expected subtitle text in userPrompt, got: %s", userPrompt)
	}
}

func TestParseSubtitleTranslationJSON(t *testing.T) {
	entries := []transcriber.SubtitleEntry{
		{Start: "00:00:01.00", End: "00:00:03.00", Text: "Hello world"},
		{Start: "00:00:04.00", End: "00:00:07.00", Text: "Welcome to our coding show"},
	}

	// 1. Array of objects
	jsonOutput := `[
		{"id": 1, "text": "Halo dunia"},
		{"id": 2, "text": "Selamat datang di acara koding kami"}
	]`
	translated, err := ParseSubtitleTranslationJSON(jsonOutput, entries)
	if err != nil {
		t.Fatalf("unexpected error parsing translation: %v", err)
	}
	if len(translated) != 2 || translated[0].Text != "Halo dunia" || translated[0].Start != "00:00:01.00" {
		t.Errorf("translation object mismatch: %+v", translated)
	}

	// 2. Array of strings fallback wrapped in markdown
	stringArrayOutput := "```json\n" + `[
		"Halo dunia string",
		"Selamat datang string"
	]` + "\n```"
	translatedStr, err := ParseSubtitleTranslationJSON(stringArrayOutput, entries)
	if err != nil {
		t.Fatalf("unexpected error parsing string array translation: %v", err)
	}
	if len(translatedStr) != 2 || translatedStr[0].Text != "Halo dunia string" || translatedStr[1].Text != "Selamat datang string" {
		t.Errorf("translation string array mismatch: %+v", translatedStr)
	}
}

func TestBuildGeminiAudioSTTPrompt(t *testing.T) {
	sys, user := BuildGeminiAudioSTTPrompt("id")
	if !strings.Contains(sys, "Bahasa Indonesia") {
		t.Errorf("expected Bahasa Indonesia rule in prompt, got: %s", sys)
	}
	if !strings.Contains(user, "transcribe this audio") {
		t.Errorf("expected user prompt, got: %s", user)
	}
}

func TestParseGeminiAudioSTTResponse(t *testing.T) {
	raw := "```json\n" + `[
		{"start": 1.25, "end": 3.50, "text": "Halo teman-teman semua!"},
		{"start": 3.80, "end": 6.20, "text": "Hari ini kita bahas video clipping."}
	]` + "\n```"

	cues, err := ParseGeminiAudioSTTResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cues) != 2 {
		t.Fatalf("expected 2 cues, got %d", len(cues))
	}
	if cues[0].Start != 1.25 || cues[0].End != 3.50 || cues[0].Text != "Halo teman-teman semua!" {
		t.Errorf("unexpected cue 0: %+v", cues[0])
	}
}


