package ai

import (
	"testing"
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
