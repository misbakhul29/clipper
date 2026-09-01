package ai

import (
	"fmt"
	"strings"

	"github.com/misbakhul29/clipper/pkg/transcriber"
)

type AIProviderConfig struct {
	APIRouter string `json:"api_router"` // "openrouter", "gemini", "deepseek", "openai" (or "codex")
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	IsShorts  bool   `json:"is_shorts"`
}

// AnalyzeHighlightsMultiProvider analyzes timestamped transcript entries using the configured AI provider.
func AnalyzeHighlightsMultiProvider(entries []transcriber.SubtitleEntry, aiCfg AIProviderConfig, targetLang string) ([]AIHighlight, error) {
	router := strings.ToLower(strings.TrimSpace(aiCfg.APIRouter))
	if router == "" || router == "openrouter" {
		return AnalyzeHighlights(entries, aiCfg.APIKey, aiCfg.Model, targetLang, aiCfg.IsShorts)
	} else if router == "gemini" {
		return AnalyzeHighlightsGemini(entries, aiCfg.APIKey, aiCfg.Model, targetLang, aiCfg.IsShorts)
	} else if router == "deepseek" {
		return AnalyzeHighlightsDeepSeek(entries, aiCfg.APIKey, aiCfg.Model, targetLang, aiCfg.IsShorts)
	} else if router == "openai" || router == "codex" {
		return AnalyzeHighlightsOpenAI(entries, aiCfg.APIKey, aiCfg.Model, targetLang, aiCfg.IsShorts)
	}
	return nil, fmt.Errorf("unsupported api_router '%s', must be 'openrouter', 'gemini', 'deepseek', or 'openai'", aiCfg.APIRouter)
}
