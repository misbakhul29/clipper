package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/misbakhul29/clipper/pkg/transcriber"
)

// AIHighlight represents a highlight clip recommended by the AI.
type AIHighlight struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Title string `json:"title"`
}

// BuildHighlightPrompts constructs system and user prompts with compacted timestamped transcripts and appropriate duration rules.
func BuildHighlightPrompts(entries []transcriber.SubtitleEntry, targetLang string, isShorts bool) (systemPrompt string, userPrompt string) {
	groupedEntries := groupSubtitleEntries(entries, 15.0)

	var sb strings.Builder
	for _, entry := range groupedEntries {
		sb.WriteString(fmt.Sprintf("[%s -> %s] %s\n", entry.Start, entry.End, entry.Text))
	}

	langInstruction := ""
	if targetLang == "id" || strings.HasPrefix(strings.ToLower(targetLang), "ind") {
		langInstruction = "The 'title' field MUST be written in Bahasa Indonesia (Indonesian)."
	} else if targetLang != "" {
		langInstruction = fmt.Sprintf("The 'title' field MUST be written in %s language.", targetLang)
	}

	var roleDesc string
	var durationRules string

	if isShorts {
		roleDesc = "You are an expert video editor AI for YouTube Shorts & TikTok."
		durationRules = `CRITICAL RULES FOR CLIP DURATION & FORMAT:
1. Each clip MUST be between 20 seconds and 60 seconds long.
2. NEVER output clips shorter than 15 seconds. Ensure the start and end timestamps cover a full, complete conversation or funny scene.`
	} else {
		roleDesc = "You are an expert video editor AI for YouTube video highlights and compilations."
		durationRules = `CRITICAL RULES FOR CLIP DURATION & FORMAT:
1. Each clip MUST be between 1 minute (60 seconds) and 5 minutes (300 seconds) long.
2. NEVER output clips shorter than 60 seconds (1 minute). Ensure the start and end timestamps cover a full, complete topic, scene, or in-depth discussion.`
	}

	systemPrompt = fmt.Sprintf(`%s
Analyze the provided timestamped video transcript and identify 2 to 5 most engaging, funny, viral, or key highlight moments.

%s
3. The "start" and "end" timestamps must be exact strings formatted as "HH:MM:SS" or "MM:SS".
%s

Your output MUST be a strict JSON array of objects with keys: "start", "end", and "title" (short descriptive label).
Do NOT include any markdown codeblocks or explanation. Return ONLY the raw JSON array.`, roleDesc, durationRules, langInstruction)

	userPrompt = fmt.Sprintf("Here is the timestamped transcript:\n\n%s", sb.String())
	return systemPrompt, userPrompt
}

// SubtitleCueItem is used for structured translation prompt payloads.
type SubtitleCueItem struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

// BuildSubtitleTranslationPrompts constructs prompts to translate subtitle cues to targetLang.
func BuildSubtitleTranslationPrompts(entries []transcriber.SubtitleEntry, targetLang string) (systemPrompt string, userPrompt string) {
	langName := targetLang
	if targetLang == "id" || strings.HasPrefix(strings.ToLower(targetLang), "ind") {
		langName = "Bahasa Indonesia (Indonesian)"
	}

	var items []SubtitleCueItem
	for i, e := range entries {
		items = append(items, SubtitleCueItem{
			ID:   i + 1,
			Text: e.Text,
		})
	}

	itemsJSON, _ := json.MarshalIndent(items, "", "  ")

	systemPrompt = fmt.Sprintf(`You are an expert video subtitle and closed-caption translator.
Translate the provided video subtitle cues into %s.

CRITICAL TRANSLATION RULES:
1. Preserve the natural spoken conversational tone, humor, slang, and context of the video.
2. Keep subtitle lines concise and easily readable for fast video playback.
3. You MUST return EXACTLY %d translated items matching the input IDs from 1 to %d in order.
4. Output MUST be a strict JSON array of objects with keys "id" (number) and "text" (translated string).
Example:
[
  {"id": 1, "text": "Terjemahan baris 1"},
  {"id": 2, "text": "Terjemahan baris 2"}
]
Do NOT include markdown formatting, codeblocks, or extra explanation. Return ONLY the raw JSON array.`, langName, len(entries), len(entries))

	userPrompt = fmt.Sprintf("Here are the %d subtitle cues to translate:\n\n%s", len(entries), string(itemsJSON))
	return systemPrompt, userPrompt
}

// ParseSubtitleTranslationJSON parses translated subtitle cues and maps them back to original entries preserving timestamps.
func ParseSubtitleTranslationJSON(content string, originalEntries []transcriber.SubtitleEntry) ([]transcriber.SubtitleEntry, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		idx := strings.Index(content, "\n")
		if idx != -1 {
			content = content[idx+1:]
		}
		if lastIdx := strings.LastIndex(content, "```"); lastIdx != -1 {
			content = content[:lastIdx]
		}
		content = strings.TrimSpace(content)
	}

	firstIdx := strings.Index(content, "[")
	if firstIdx != -1 {
		depth := 0
		endIdx := -1
		for i := firstIdx; i < len(content); i++ {
			if content[i] == '[' {
				depth++
			} else if content[i] == ']' {
				depth--
				if depth == 0 {
					endIdx = i
					break
				}
			}
		}
		if endIdx != -1 {
			content = content[firstIdx : endIdx+1]
		}
	}

	content = repairTruncatedJSON(content)

	// Try parsing as []SubtitleCueItem
	var cueItems []SubtitleCueItem
	if err := json.Unmarshal([]byte(content), &cueItems); err == nil && len(cueItems) > 0 {
		out := make([]transcriber.SubtitleEntry, len(originalEntries))
		copy(out, originalEntries)

		for _, item := range cueItems {
			idx := item.ID - 1
			if idx >= 0 && idx < len(out) && strings.TrimSpace(item.Text) != "" {
				out[idx].Text = strings.TrimSpace(item.Text)
			}
		}
		return out, nil
	}

	// Fallback 1: Try parsing as []string
	var stringItems []string
	if err := json.Unmarshal([]byte(content), &stringItems); err == nil && len(stringItems) > 0 {
		out := make([]transcriber.SubtitleEntry, len(originalEntries))
		copy(out, originalEntries)
		for i, text := range stringItems {
			if i < len(out) && strings.TrimSpace(text) != "" {
				out[i].Text = strings.TrimSpace(text)
			}
		}
		return out, nil
	}

	return originalEntries, fmt.Errorf("failed to parse subtitle translation response: %s", content)
}

// ParseAIHighlightsJSON parses raw JSON string or markdown-wrapped JSON string into []AIHighlight.
func ParseAIHighlightsJSON(content string) ([]AIHighlight, error) {
	content = strings.TrimSpace(content)
	// Strip markdown ```json ... ``` if present
	if strings.HasPrefix(content, "```") {
		idx := strings.Index(content, "\n")
		if idx != -1 {
			content = content[idx+1:]
		}
		if lastIdx := strings.LastIndex(content, "```"); lastIdx != -1 {
			content = content[:lastIdx]
		}
		content = strings.TrimSpace(content)
	}

	// Extract the first valid JSON array [...] using bracket matching to ignore any trailing LLM noise or tool XML tags
	firstIdx := strings.Index(content, "[")
	if firstIdx != -1 {
		depth := 0
		endIdx := -1
		for i := firstIdx; i < len(content); i++ {
			if content[i] == '[' {
				depth++
			} else if content[i] == ']' {
				depth--
				if depth == 0 {
					endIdx = i
					break
				}
			}
		}
		if endIdx != -1 {
			content = content[firstIdx : endIdx+1]
		}
	}

	content = repairTruncatedJSON(content)

	var highlights []AIHighlight
	if err := json.Unmarshal([]byte(content), &highlights); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON response: %w. Content: %s", err, content)
	}

	if len(highlights) == 0 {
		return nil, fmt.Errorf("AI returned an empty list of highlights")
	}

	return highlights, nil
}

func repairTruncatedJSON(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "[") && !strings.HasSuffix(content, "]") {
		lastClose := strings.LastIndex(content, "}")
		if lastClose != -1 {
			return content[:lastClose+1] + "]"
		}
	}
	return content
}

func groupSubtitleEntries(entries []transcriber.SubtitleEntry, groupWindowSec float64) []transcriber.SubtitleEntry {
	if len(entries) == 0 {
		return entries
	}
	var grouped []transcriber.SubtitleEntry

	var currentStart, currentEnd string
	var currentTexts []string

	for i, entry := range entries {
		if i == 0 {
			currentStart = entry.Start
			currentEnd = entry.End
			currentTexts = append(currentTexts, entry.Text)
			continue
		}

		startSec := parseTimestampToSeconds(entry.Start)
		currStartSec := parseTimestampToSeconds(currentStart)

		if startSec-currStartSec < groupWindowSec {
			currentEnd = entry.End
			if !containsText(currentTexts, entry.Text) {
				currentTexts = append(currentTexts, entry.Text)
			}
		} else {
			grouped = append(grouped, transcriber.SubtitleEntry{
				Start: currentStart,
				End:   currentEnd,
				Text:  strings.Join(currentTexts, " "),
			})
			currentStart = entry.Start
			currentEnd = entry.End
			currentTexts = []string{entry.Text}
		}
	}

	if currentStart != "" && len(currentTexts) > 0 {
		grouped = append(grouped, transcriber.SubtitleEntry{
			Start: currentStart,
			End:   currentEnd,
			Text:  strings.Join(currentTexts, " "),
		})
	}

	return grouped
}

func parseTimestampToSeconds(ts string) float64 {
	parts := strings.Split(ts, ":")
	if len(parts) == 3 {
		var h, m, s float64
		fmt.Sscanf(parts[0], "%f", &h)
		fmt.Sscanf(parts[1], "%f", &m)
		fmt.Sscanf(parts[2], "%f", &s)
		return h*3600 + m*60 + s
	} else if len(parts) == 2 {
		var m, s float64
		fmt.Sscanf(parts[0], "%f", &m)
		fmt.Sscanf(parts[1], "%f", &s)
		return m*60 + s
	}
	var s float64
	fmt.Sscanf(ts, "%f", &s)
	return s
}

func containsText(slice []string, text string) bool {
	for _, item := range slice {
		if item == text {
			return true
		}
	}
	return false
}

// BuildMetadataHighlightPrompts constructs system and user prompts to generate video segments from metadata and duration without subtitles.
func BuildMetadataHighlightPrompts(videoTitle string, durationSec float64, targetDuration float64, targetLang string, isShorts bool) (systemPrompt string, userPrompt string) {
	durStr := FormatSecondsToTime(durationSec)
	langInstruction := ""
	if targetLang == "id" || strings.HasPrefix(strings.ToLower(targetLang), "ind") {
		langInstruction = "The 'title' field MUST be written in Bahasa Indonesia (Indonesian)."
	} else if targetLang != "" {
		langInstruction = fmt.Sprintf("The 'title' field MUST be written in %s language.", targetLang)
	}

	var clipRules string
	if targetDuration > 0 {
		clipRules = fmt.Sprintf("Each clip should be approximately %.0f seconds long (±15 seconds).", targetDuration)
	} else if isShorts {
		clipRules = "Each clip should be between 25 seconds and 55 seconds long, optimized for Shorts/Reels/TikTok."
	} else {
		clipRules = "Each clip should be between 1 minute and 4 minutes long, optimized for YouTube video highlights."
	}

	systemPrompt = fmt.Sprintf(`You are an expert video editor and social media viral content strategist.
Given a video's title and total duration, propose 2 to 5 high-impact viral segment timestamps (start, end, and title).

CRITICAL RULES:
1. %s
2. The first segment MUST capture an opening hook starting at "00:00".
3. The "start" and "end" timestamps must be formatted as "MM:SS" or "HH:MM:SS" and MUST NOT exceed the total duration (%s).
4. %s

Respond ONLY with a valid JSON array of objects with keys: "start", "end", "title".
No markdown, no explanation, only raw JSON.`, clipRules, durStr, langInstruction)

	userPrompt = fmt.Sprintf("Video Title: %s\nTotal Duration: %s (%.0f seconds)", videoTitle, durStr, durationSec)
	return systemPrompt, userPrompt
}

// GenerateHeuristicHighlights produces intelligently spaced segment intervals across a video's duration without requiring subtitles.
func GenerateHeuristicHighlights(durationSec float64, isShorts bool, targetDuration float64, targetLang string) []AIHighlight {
	if durationSec <= 10 {
		return []AIHighlight{
			{Start: "00:00", End: FormatSecondsToTime(durationSec), Title: "Full Video Highlight"},
		}
	}

	clipLength := targetDuration
	if clipLength <= 0 {
		if isShorts {
			clipLength = 35.0
		} else {
			if durationSec <= 180 {
				clipLength = 45.0
			} else if durationSec <= 600 {
				clipLength = 90.0 // 1.5 min
			} else if durationSec <= 1800 {
				clipLength = 150.0 // 2.5 min
			} else {
				clipLength = 240.0 // 4 min
			}
		}
	}

	if durationSec < clipLength {
		return []AIHighlight{
			{Start: "00:00", End: FormatSecondsToTime(durationSec), Title: "Opening Highlight"},
		}
	}

	isIndo := targetLang == "id" || strings.HasPrefix(strings.ToLower(targetLang), "ind")
	tHook := "Opening Viral Hook"
	tInsight := "Key Core Insight"
	tClimax := "Peak Climax Moment"
	tOutro := "Actionable Conclusion"
	if isIndo {
		tHook = "Hook Pembuka Menarik"
		tInsight = "Poin Inti & Pembahasan Utama"
		tClimax = "Momen Puncak & Klimaks"
		tOutro = "Kesimpulan & Penutup"
	}

	var highlights []AIHighlight
	// 1. Opening Hook
	hookEnd := clipLength
	if hookEnd > durationSec {
		hookEnd = durationSec
	}
	highlights = append(highlights, AIHighlight{
		Start: "00:00",
		End:   FormatSecondsToTime(hookEnd),
		Title: tHook,
	})

	// 2. Middle highlight
	if durationSec > clipLength*2.5 {
		midStart := durationSec * 0.35
		midEnd := midStart + clipLength
		if midEnd < durationSec {
			highlights = append(highlights, AIHighlight{
				Start: FormatSecondsToTime(midStart),
				End:   FormatSecondsToTime(midEnd),
				Title: tInsight,
			})
		}
	}

	// 3. Climax
	if durationSec > clipLength*4.0 {
		climaxStart := durationSec * 0.65
		climaxEnd := climaxStart + clipLength
		if climaxEnd < durationSec {
			highlights = append(highlights, AIHighlight{
				Start: FormatSecondsToTime(climaxStart),
				End:   FormatSecondsToTime(climaxEnd),
				Title: tClimax,
			})
		}
	}

	// 4. Conclusion / CTA
	if durationSec > clipLength*2.0 {
		finalEnd := durationSec
		finalStart := durationSec - clipLength
		if finalStart > hookEnd+10 {
			highlights = append(highlights, AIHighlight{
				Start: FormatSecondsToTime(finalStart),
				End:   FormatSecondsToTime(finalEnd),
				Title: tOutro,
			})
		}
	}

	return highlights
}

// FormatSecondsToTime formats float seconds into MM:SS or HH:MM:SS.
func FormatSecondsToTime(sec float64) string {
	totalSec := int(sec)
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// AudioSubtitleCue represents a single timestamped subtitle cue from Speech-to-Text.
type AudioSubtitleCue struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// BuildGeminiAudioSTTPrompt constructs the prompt for Gemini Audio transcription.
func BuildGeminiAudioSTTPrompt(targetLang string) (systemPrompt string, userPrompt string) {
	langRule := ""
	if targetLang == "id" || strings.HasPrefix(strings.ToLower(targetLang), "ind") {
		langRule = "Transcribe the audio accurately in Bahasa Indonesia."
	} else if targetLang != "" {
		langRule = fmt.Sprintf("Transcribe the audio accurately in %s language.", targetLang)
	} else {
		langRule = "Transcribe the audio in its original spoken language."
	}

	systemPrompt = fmt.Sprintf(`You are an expert speech-to-text audio transcriber.
%s
Listen carefully to the audio file provided and generate precise, time-aligned subtitle cues.
Split sentences into natural subtitle phrases (each cue around 1.0 to 4.0 seconds long).
The timestamps (start and end in seconds, float format) MUST be relative to the beginning of the audio clip (starting at 0.0).

OUTPUT FORMAT:
Output ONLY a valid JSON array of objects with the following schema:
[
  {
    "start": 0.0,
    "end": 2.5,
    "text": "Transcribed speech phrase here"
  }
]
Do not wrap in explanations. Output pure JSON only.`, langRule)

	userPrompt = "Please transcribe this audio clip into precise timestamped subtitle cues JSON."
	return systemPrompt, userPrompt
}

// ParseGeminiAudioSTTResponse parses Gemini STT output into a list of AudioSubtitleCue.
func ParseGeminiAudioSTTResponse(responseText string) ([]AudioSubtitleCue, error) {
	clean := strings.TrimSpace(responseText)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	startIdx := strings.Index(clean, "[")
	endIdx := strings.LastIndex(clean, "]")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return nil, fmt.Errorf("no valid JSON array found in Gemini Audio response: %s", responseText)
	}
	jsonStr := clean[startIdx : endIdx+1]

	var cues []AudioSubtitleCue
	if err := json.Unmarshal([]byte(jsonStr), &cues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Gemini Audio cues: %w", err)
	}

	return cues, nil
}

