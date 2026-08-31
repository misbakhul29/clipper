package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"clipping/pkg/transcriber"
)

// AIHighlight represents a highlight clip recommended by the AI.
type AIHighlight struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Title string `json:"title"`
}

type OpenRouterRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	IncludeReasoning bool      `json:"include_reasoning"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Error   *AIError `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type AIError struct {
	Message string `json:"message"`
}

// AnalyzeHighlights sends timestamped transcript entries to OpenRouter API to select top engaging highlight clips.
func AnalyzeHighlights(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string) ([]AIHighlight, error) {
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key required. Set OPENROUTER_API_KEY env var or pass -openrouter-key")
	}

	if model == "" {
		model = "openrouter/free"
	}

	// Compact/group subtitle entries to prevent LLM prompt bloat and API timeout
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

	systemPrompt := fmt.Sprintf(`You are an expert video editor AI for YouTube Shorts & TikTok.
Analyze the provided timestamped video transcript and identify 2 to 5 most engaging, funny, viral, or key highlight moments.

CRITICAL RULES FOR CLIP DURATION & FORMAT:
1. Each clip MUST be between 20 seconds and 60 seconds long.
2. NEVER output clips shorter than 15 seconds. Ensure the start and end timestamps cover a full, complete conversation or funny scene.
3. The "start" and "end" timestamps must be exact strings formatted as "HH:MM:SS" or "MM:SS".
%s

Your output MUST be a strict JSON array of objects with keys: "start", "end", and "title" (short descriptive label).
Do NOT include any markdown codeblocks or explanation. Return ONLY the raw JSON array.`, langInstruction)

	userPrompt := fmt.Sprintf("Here is the timestamped transcript:\n\n%s", sb.String())

	reqBody := OpenRouterRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenRouter request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenRouter API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenRouter API returned HTTP status %d: %s", resp.StatusCode, string(respBytes))
	}

	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(respBytes, &openRouterResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenRouter response: %w", err)
	}

	if openRouterResp.Error != nil && openRouterResp.Error.Message != "" {
		return nil, fmt.Errorf("OpenRouter API error: %s", openRouterResp.Error.Message)
	}

	if len(openRouterResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenRouter API returned no choices in response")
	}

	aiContent := openRouterResp.Choices[0].Message.Content
	return ParseAIHighlightsJSON(aiContent)
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
