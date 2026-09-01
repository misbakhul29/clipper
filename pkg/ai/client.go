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

// OpenAICompatibleRequest models the payload for OpenAI / OpenRouter / DeepSeek Chat Completions API.
type OpenAICompatibleRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	IncludeReasoning bool      `json:"include_reasoning,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAICompatibleResponse models the response from OpenAI-compatible Chat Completions API.
type OpenAICompatibleResponse struct {
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

// Type aliases for backwards-compatibility
type OpenRouterRequest = OpenAICompatibleRequest
type OpenRouterResponse = OpenAICompatibleResponse

// callOpenAICompatibleAPI sends highlight detection prompts to any OpenAI-compatible completions endpoint.
func callOpenAICompatibleAPI(endpoint string, entries []transcriber.SubtitleEntry, apiKey, model, targetLang string) ([]AIHighlight, error) {
	systemPrompt, userPrompt := BuildHighlightPrompts(entries, targetLang)

	reqBody := OpenAICompatibleRequest{
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
		return nil, fmt.Errorf("API request to %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response from %s: %w", endpoint, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API (%s) returned HTTP %d: %s", endpoint, resp.StatusCode, string(respBytes))
	}

	var chatResp OpenAICompatibleResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal API response: %w", err)
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return nil, fmt.Errorf("API error (%s): %s", endpoint, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("API (%s) returned no choices in response", endpoint)
	}

	aiContent := chatResp.Choices[0].Message.Content
	return ParseAIHighlightsJSON(aiContent)
}

// resolveAPIKeyAndModel resolves API key from argument or environment variable, and applies default model if empty.
func resolveAPIKeyAndModel(apiKey, envVar, model, defaultModel, providerName string) (string, string, error) {
	if apiKey == "" {
		apiKey = os.Getenv(envVar)
	}
	if apiKey == "" {
		return "", "", fmt.Errorf("%s API key required. Set %s env var or set ai_config.api_key", providerName, envVar)
	}
	if strings.TrimSpace(model) == "" {
		model = defaultModel
	}
	return apiKey, model, nil
}
