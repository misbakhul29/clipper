package ai

import (
	"fmt"
	"os"

	"github.com/misbakhul29/clipper/pkg/transcriber"
)

// AnalyzeHighlightsDeepSeek sends transcript entries to DeepSeek API (OpenAI-compatible format).
func AnalyzeHighlightsDeepSeek(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string) ([]AIHighlight, error) {
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("DeepSeek API key required. Set DEEPSEEK_API_KEY env var or set ai_config.api_key")
	}

	if model == "" {
		model = "deepseek-chat"
	}

	return callOpenAICompatibleAPI("https://api.deepseek.com/chat/completions", entries, apiKey, model, targetLang)
}
