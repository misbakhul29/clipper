package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// SocialMetadata represents viral title, description, tags, and score for a clip.
type SocialMetadata struct {
	SegmentIndex   int      `json:"segment_index"`
	StartTime      string   `json:"start_time"`
	EndTime        string   `json:"end_time"`
	DurationSec    float64  `json:"duration_sec"`
	VideoFile      string   `json:"video_file"`
	HookTitle      string   `json:"hook_title"`
	Description    string   `json:"description"`
	CallToAction   string   `json:"call_to_action"`
	Hashtags       []string `json:"hashtags"`
	ViralityScore  int      `json:"virality_score"` // 1-100
	ViralityReason string   `json:"virality_reason"`
}

// BuildSocialMetadataPrompts constructs system and user prompts to generate social media metadata.
func BuildSocialMetadataPrompts(transcript string, targetLang string, isShorts bool) (systemPrompt string, userPrompt string) {
	langName := "Bahasa Indonesia (Indonesian)"
	if targetLang != "" && targetLang != "id" && !strings.HasPrefix(strings.ToLower(targetLang), "ind") {
		langName = targetLang
	}

	formatType := "YouTube Shorts, TikTok, and Instagram Reels (9:16 vertical)"
	if !isShorts {
		formatType = "YouTube video highlight (16:9 landscape)"
	}

	systemPrompt = fmt.Sprintf(`You are an elite viral content strategist and social media growth expert for %s.
Analyze the provided transcript of a short video clip and generate optimized social media metadata to maximize Click-Through-Rate (CTR), audience watch time, and engagement.

All textual output (hook_title, description, call_to_action, virality_reason) MUST be written in %s.

CRITICAL INSTRUCTIONS:
1. "hook_title": A punchy, curiosity-inducing viral headline with high CTR (include 1 relevant emoji). Maximum 70 characters.
2. "description": An engaging 2 to 3 sentence summary highlighting the emotional core, controversy, humor, or lesson of the clip.
3. "call_to_action": A persuasive CTA urging viewers to comment, share, or subscribe/follow.
4. "hashtags": An array of 5 to 8 trending and relevant hashtags (each starting with '#', e.g. ["#shorts", "#fyp", "#viral"]).
5. "virality_score": An integer score from 1 to 100 predicting how likely this clip is to perform well algorithmically.
6. "virality_reason": A 1-2 sentence analytical justification explaining WHY this clip scored this way (e.g. strong opening hook, relatable dilemma, fast pacing).

Output MUST be a strict, valid JSON object with keys:
"hook_title", "description", "call_to_action", "hashtags", "virality_score", "virality_reason"

Do NOT wrap with explanation or conversational text. Return ONLY the raw JSON object.`, formatType, langName)

	userPrompt = fmt.Sprintf("Here is the clip transcript:\n\n%s", transcript)
	return systemPrompt, userPrompt
}

// ParseSocialMetadataJSON parses the AI JSON response into a SocialMetadata struct.
func ParseSocialMetadataJSON(rawJSON string) (*SocialMetadata, error) {
	content := extractJSONObject(rawJSON)

	var meta SocialMetadata
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse social metadata JSON: %w. Content: %s", err, content)
	}

	// Sanitize and normalize fields
	meta.HookTitle = strings.TrimSpace(meta.HookTitle)
	meta.Description = strings.TrimSpace(meta.Description)
	meta.CallToAction = strings.TrimSpace(meta.CallToAction)
	meta.ViralityReason = strings.TrimSpace(meta.ViralityReason)

	if meta.ViralityScore < 1 {
		meta.ViralityScore = 75
	} else if meta.ViralityScore > 100 {
		meta.ViralityScore = 100
	}

	// Ensure hashtags start with #
	var cleanTags []string
	for _, tag := range meta.Hashtags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !strings.HasPrefix(tag, "#") {
			tag = "#" + tag
		}
		cleanTags = append(cleanTags, tag)
	}
	if len(cleanTags) == 0 {
		cleanTags = []string{"#shorts", "#viral", "#trending", "#fyp"}
	}
	meta.Hashtags = cleanTags

	return &meta, nil
}

func extractJSONObject(content string) string {
	content = strings.TrimSpace(content)
	// Strip markdown code block
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

	firstIdx := strings.Index(content, "{")
	if firstIdx != -1 {
		depth := 0
		endIdx := -1
		for i := firstIdx; i < len(content); i++ {
			if content[i] == '{' {
				depth++
			} else if content[i] == '}' {
				depth--
				if depth == 0 {
					endIdx = i
					break
				}
			}
		}
		if endIdx != -1 {
			return content[firstIdx : endIdx+1]
		}
	}
	return content
}

// GenerateSocialMetadataMultiProvider generates social metadata using the configured AI provider.
func GenerateSocialMetadataMultiProvider(transcript string, aiCfg AIProviderConfig, targetLang string, isShorts bool) (*SocialMetadata, error) {
	systemPrompt, userPrompt := BuildSocialMetadataPrompts(transcript, targetLang, isShorts)
	router := strings.ToLower(strings.TrimSpace(aiCfg.APIRouter))

	var rawContent string
	var err error

	switch router {
	case "", "openrouter":
		apiKey, model, resolveErr := resolveAPIKeyAndModel(aiCfg.APIKey, "OPENROUTER_API_KEY", aiCfg.Model, "openrouter/free", "OpenRouter")
		if resolveErr != nil {
			return nil, resolveErr
		}
		rawContent, err = callOpenAICompatibleCompletions("https://openrouter.ai/api/v1/chat/completions", apiKey, model, systemPrompt, userPrompt)

	case "gemini":
		apiKey, model, resolveErr := resolveAPIKeyAndModel(aiCfg.APIKey, "GEMINI_API_KEY", aiCfg.Model, "gemini-2.0-flash", "Gemini")
		if resolveErr != nil {
			return nil, resolveErr
		}
		rawContent, err = callGeminiGenerate(apiKey, model, systemPrompt+"\n\n"+userPrompt)

	case "deepseek":
		apiKey, model, resolveErr := resolveAPIKeyAndModel(aiCfg.APIKey, "DEEPSEEK_API_KEY", aiCfg.Model, "deepseek-chat", "DeepSeek")
		if resolveErr != nil {
			return nil, resolveErr
		}
		rawContent, err = callOpenAICompatibleCompletions("https://api.deepseek.com/chat/completions", apiKey, model, systemPrompt, userPrompt)

	case "openai", "codex":
		apiKey, model, resolveErr := resolveAPIKeyAndModel(aiCfg.APIKey, "OPENAI_API_KEY", aiCfg.Model, "gpt-4o-mini", "OpenAI")
		if resolveErr != nil {
			return nil, resolveErr
		}
		rawContent, err = callOpenAICompatibleCompletions("https://api.openai.com/v1/chat/completions", apiKey, model, systemPrompt, userPrompt)

	default:
		return nil, fmt.Errorf("unsupported AI router '%s'", aiCfg.APIRouter)
	}

	if err != nil {
		return nil, err
	}

	return ParseSocialMetadataJSON(rawContent)
}

// GenerateHeuristicSocialMetadata generates sensible offline fallback social metadata based on keyword inspection.
func GenerateHeuristicSocialMetadata(transcript string, clipTitle string, targetLang string, isShorts bool) *SocialMetadata {
	isIndo := targetLang == "" || targetLang == "id" || strings.HasPrefix(strings.ToLower(targetLang), "ind")

	title := clipTitle
	if title == "" {
		// Take the first punchy clause
		sentences := strings.FieldsFunc(transcript, func(r rune) bool {
			return r == '.' || r == '!' || r == '?' || r == '\n'
		})
		if len(sentences) > 0 {
			title = strings.TrimSpace(sentences[0])
		}
		if title == "" {
			if isIndo {
				title = "Momen Tak Terduga Ini Bikin Kaget! 😱"
			} else {
				title = "You Won't Believe What Happened Here! 😱"
			}
		}
	}

	desc := strings.TrimSpace(transcript)
	if len(desc) > 160 {
		desc = desc[:160] + "..."
	}
	if desc == "" {
		if isIndo {
			desc = "Cuplikan video terbaik dengan pembahasan seru dan menarik."
		} else {
			desc = "The most engaging clip with high energy discussion."
		}
	}

	cta := "Tonton sampai habis & jangan lupa like, share, dan subscribe untuk video seru lainnya! 🔔"
	if !isIndo {
		cta = "Watch till the end & don't forget to like, share, and subscribe for more awesome clips! 🔔"
	}

	// Infer hashtags from keywords
	tags := []string{"#shorts", "#viral", "#trending", "#fyp"}
	lower := strings.ToLower(transcript + " " + title)

	if regexp.MustCompile(`(?i)\b(uang|cuan|duit|dollar|bisnis|kaya|gaji|profit|investasi)\b`).MatchString(lower) {
		tags = append(tags, "#bisnis", "#cuan", "#finansial", "#sukses")
	}
	if regexp.MustCompile(`(?i)\b(ai|teknologi|robot|komputer|coding|program)\b`).MatchString(lower) {
		tags = append(tags, "#ai", "#teknologi", "#tech", "#future")
	}
	if regexp.MustCompile(`(?i)\b(zombi|mati|hantu|darah|seram|horror)\b`).MatchString(lower) {
		tags = append(tags, "#horror", "#cerita", "#alurfilm", "#movie")
	}
	if regexp.MustCompile(`(?i)\b(podcast|wawancara|bicara|cerita)\b`).MatchString(lower) {
		tags = append(tags, "#podcast", "#inspirasi", "#obrolan")
	}

	return &SocialMetadata{
		HookTitle:      title,
		Description:    desc,
		CallToAction:   cta,
		Hashtags:       tags,
		ViralityScore:  82,
		ViralityReason: "High conversational energy with a direct opening hook and strong topic relevance.",
	}
}

// FormatMetadataText formats social metadata into a clean, ready-to-copy text document for creators.
func FormatMetadataText(meta *SocialMetadata) string {
	var sb strings.Builder
	sb.WriteString("================================================================================\n")
	sb.WriteString(fmt.Sprintf("CLIP SOCIAL METADATA: %s\n", meta.VideoFile))
	sb.WriteString(fmt.Sprintf("Segment: #%d (%s -> %s, Duration: %.2fs)\n",
		meta.SegmentIndex, meta.StartTime, meta.EndTime, meta.DurationSec))
	sb.WriteString("================================================================================\n\n")

	sb.WriteString("📌 VIRAL HOOK TITLE:\n")
	sb.WriteString(meta.HookTitle + "\n\n")

	sb.WriteString("📝 DESCRIPTION:\n")
	sb.WriteString(meta.Description + "\n\n")

	sb.WriteString("📣 CALL-TO-ACTION:\n")
	sb.WriteString(meta.CallToAction + "\n\n")

	sb.WriteString("🏷️ HASHTAGS:\n")
	sb.WriteString(strings.Join(meta.Hashtags, " ") + "\n\n")

	sb.WriteString(fmt.Sprintf("🔥 VIRALITY SCORE: %d/100\n", meta.ViralityScore))
	sb.WriteString("💡 WHY IT CAN GO VIRAL:\n")
	sb.WriteString(meta.ViralityReason + "\n\n")
	sb.WriteString("================================================================================\n")

	return sb.String()
}
