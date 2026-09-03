package ai

import (
	"strings"
	"testing"
)

func TestBuildSocialMetadataPrompts(t *testing.T) {
	transcript := "Ini adalah cuplikan rahasia bagaimana cara mendapatkan uang dan cuan dari YouTube Shorts."
	sysPrompt, userPrompt := BuildSocialMetadataPrompts(transcript, "id", true)

	if !strings.Contains(sysPrompt, "YouTube Shorts") {
		t.Errorf("expected shorts mention in sysPrompt, got: %s", sysPrompt)
	}
	if !strings.Contains(sysPrompt, "Bahasa Indonesia") {
		t.Errorf("expected Indonesian language instruction, got: %s", sysPrompt)
	}
	if !strings.Contains(userPrompt, transcript) {
		t.Errorf("expected transcript in userPrompt")
	}
}

func TestParseSocialMetadataJSON(t *testing.T) {
	t.Run("Standard JSON", func(t *testing.T) {
		raw := `{
			"hook_title": "Rahasia Cuan 100 Juta dari Shorts! 💰",
			"description": "Banyak yang belum tahu trik algoritma ini. Simak sampai tuntas!",
			"call_to_action": "Subscribe untuk tips cuan lainnya!",
			"hashtags": ["shorts", "#cuan", "#bisnis"],
			"virality_score": 92,
			"virality_reason": "Hook awal sangat memikat dan angka spesifik memicu rasa penasaran."
		}`

		meta, err := ParseSocialMetadataJSON(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if meta.HookTitle != "Rahasia Cuan 100 Juta dari Shorts! 💰" {
			t.Errorf("unexpected hook title: %s", meta.HookTitle)
		}
		if meta.ViralityScore != 92 {
			t.Errorf("unexpected virality score: %d", meta.ViralityScore)
		}
		// First hashtag lacked '#' - parser must auto-prefix it
		if meta.Hashtags[0] != "#shorts" {
			t.Errorf("expected #shorts, got: %s", meta.Hashtags[0])
		}
	})

	t.Run("Markdown Wrapped JSON", func(t *testing.T) {
		raw := "```json\n" + `{
			"hook_title": "AI Will Replace Programmers?! 🤖",
			"description": "The debate is hotter than ever.",
			"call_to_action": "Comment your thoughts below!",
			"hashtags": ["#ai", "#coding"],
			"virality_score": 85,
			"virality_reason": "High controversy topic."
		}` + "\n```\nSome trailing explanation from model."

		meta, err := ParseSocialMetadataJSON(raw)
		if err != nil {
			t.Fatalf("unexpected error parsing markdown JSON: %v", err)
		}

		if meta.HookTitle != "AI Will Replace Programmers?! 🤖" {
			t.Errorf("unexpected title: %s", meta.HookTitle)
		}
		if meta.ViralityScore != 85 {
			t.Errorf("unexpected score: %d", meta.ViralityScore)
		}
	})
}

func TestGenerateHeuristicSocialMetadata(t *testing.T) {
	transcript := "Bagaimana cara menghasilkan uang dan cuan melimpah dari bisnis online di tahun 2026."
	meta := GenerateHeuristicSocialMetadata(transcript, "Cuan Melimpah", "id", true)

	if meta.HookTitle != "Cuan Melimpah" {
		t.Errorf("expected clip title, got: %s", meta.HookTitle)
	}
	if !strings.Contains(strings.Join(meta.Hashtags, " "), "#cuan") {
		t.Errorf("expected inferred #cuan tag, got: %v", meta.Hashtags)
	}
	if meta.ViralityScore <= 0 || meta.ViralityScore > 100 {
		t.Errorf("invalid heuristic score: %d", meta.ViralityScore)
	}

	formatted := FormatMetadataText(meta)
	if !strings.Contains(formatted, "VIRAL HOOK TITLE") || !strings.Contains(formatted, "VIRALITY SCORE") {
		t.Errorf("unexpected formatted text: %s", formatted)
	}
}
