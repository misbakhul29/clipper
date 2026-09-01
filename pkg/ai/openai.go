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

	"github.com/misbakhul29/clipper/pkg/transcriber"
)

// AnalyzeHighlightsOpenAI sends transcript entries to OpenAI API.
func AnalyzeHighlightsOpenAI(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string) ([]AIHighlight, error) {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key required. Set OPENAI_API_KEY env var or set ai_config.api_key")
	}

	if model == "" {
		model = "gpt-4o-mini"
	}

	return callOpenAICompatibleAPI("https://api.openai.com/v1/chat/completions", entries, apiKey, model, targetLang)
}

func callOpenAICompatibleAPI(endpoint string, entries []transcriber.SubtitleEntry, apiKey, model, targetLang string) ([]AIHighlight, error) {
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
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned HTTP status %d: %s", resp.StatusCode, string(respBytes))
	}

	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(respBytes, &openRouterResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if openRouterResp.Error != nil && openRouterResp.Error.Message != "" {
		return nil, fmt.Errorf("API error: %s", openRouterResp.Error.Message)
	}

	if len(openRouterResp.Choices) == 0 {
		return nil, fmt.Errorf("API returned no choices in response")
	}

	aiContent := openRouterResp.Choices[0].Message.Content
	return ParseAIHighlightsJSON(aiContent)
}
