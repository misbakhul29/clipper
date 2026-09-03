package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/misbakhul29/clipper/pkg/transcriber"
)

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inline_data,omitempty"`
}

type GeminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"` // base64 encoded data
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	Error      *GeminiError      `json:"error,omitempty"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// NormalizeGeminiModel sanitizes or aliases outdated/invalid model names to official Google Gemini models.
func NormalizeGeminiModel(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" || strings.HasPrefix(m, "gemini-3") || m == "default" {
		return "gemini-2.5-flash"
	}
	return m
}

// callGeminiGenerate sends text generation prompt to Google Gemini REST API.
func callGeminiGenerate(apiKey, model, prompt string) (string, error) {
	model = NormalizeGeminiModel(model)
	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Gemini request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create http request for Gemini: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Gemini API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errResp := fmt.Sprintf("Gemini API returned HTTP status %d: %s", resp.StatusCode, string(respBytes))
		fmt.Printf("[AI ERROR] %s (model: %s)\n", errResp, model)
		return "", fmt.Errorf("%s", errResp)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal Gemini response: %w", err)
	}

	if geminiResp.Error != nil && geminiResp.Error.Message != "" {
		return "", fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini API returned no text candidate in response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// callGeminiAudioSTT sends audio bytes and transcription prompt to Google Gemini Multimodal REST API.
func callGeminiAudioSTT(apiKey, model, prompt string, mimeType string, audioBase64 string) (string, error) {
	model = NormalizeGeminiModel(model)
	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						InlineData: &GeminiInlineData{
							MimeType: mimeType,
							Data:     audioBase64,
						},
					},
					{
						Text: prompt,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Gemini Audio request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create http request for Gemini: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Gemini API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errResp := fmt.Sprintf("Gemini API returned HTTP status %d: %s", resp.StatusCode, string(respBytes))
		fmt.Printf("[AI ERROR] %s (model: %s)\n", errResp, model)
		return "", fmt.Errorf("%s", errResp)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal Gemini response: %w", err)
	}

	if geminiResp.Error != nil && geminiResp.Error.Message != "" {
		return "", fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini API returned no text candidate in response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// TranscribeAudioGemini transcribes audio data to time-aligned subtitle cues using Gemini Audio API.
func TranscribeAudioGemini(apiKey, model, targetLang string, audioBytes []byte, mimeType string) ([]AudioSubtitleCue, error) {
	resolvedKey, resolvedModel, err := resolveAPIKeyAndModel(apiKey, "GEMINI_API_KEY", model, "gemini-2.5-flash", "Gemini")
	if err != nil {
		return nil, err
	}

	if mimeType == "" {
		mimeType = "audio/mp3"
	}

	encoded := base64.StdEncoding.EncodeToString(audioBytes)
	sysPrompt, userPrompt := BuildGeminiAudioSTTPrompt(targetLang)
	fullPrompt := sysPrompt + "\n\n" + userPrompt

	aiContent, err := callGeminiAudioSTT(resolvedKey, resolvedModel, fullPrompt, mimeType, encoded)
	if err != nil {
		return nil, err
	}

	return ParseGeminiAudioSTTResponse(aiContent)
}

// AnalyzeHighlightsGemini sends transcript entries to Google Gemini REST API.
func AnalyzeHighlightsGemini(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string, isShorts bool) ([]AIHighlight, error) {
	resolvedKey, resolvedModel, err := resolveAPIKeyAndModel(apiKey, "GEMINI_API_KEY", model, "gemini-2.5-flash", "Gemini")
	if err != nil {
		return nil, err
	}

	systemPrompt, userPrompt := BuildHighlightPrompts(entries, targetLang, isShorts)
	aiContent, err := callGeminiGenerate(resolvedKey, resolvedModel, systemPrompt+"\n\n"+userPrompt)
	if err != nil {
		return nil, err
	}
	return ParseAIHighlightsJSON(aiContent)
}

// TranslateSubtitlesGemini translates subtitle entries into targetLang using Google Gemini API.
func TranslateSubtitlesGemini(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string) ([]transcriber.SubtitleEntry, error) {
	if len(entries) == 0 || targetLang == "" {
		return entries, nil
	}
	resolvedKey, resolvedModel, err := resolveAPIKeyAndModel(apiKey, "GEMINI_API_KEY", model, "gemini-2.5-flash", "Gemini")
	if err != nil {
		return entries, err
	}

	systemPrompt, userPrompt := BuildSubtitleTranslationPrompts(entries, targetLang)
	aiContent, err := callGeminiGenerate(resolvedKey, resolvedModel, systemPrompt+"\n\n"+userPrompt)
	if err != nil {
		return entries, err
	}
	return ParseSubtitleTranslationJSON(aiContent, entries)
}

