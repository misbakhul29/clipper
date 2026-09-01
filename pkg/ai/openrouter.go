package ai

import (
	"fmt"

	"github.com/misbakhul29/clipper/pkg/transcriber"
)

const openRouterEndpoint = "https://openrouter.ai/api/v1/chat/completions"

// AnalyzeHighlights sends timestamped transcript entries to OpenRouter API to select top engaging highlight clips.
func AnalyzeHighlights(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string, isShorts bool) ([]AIHighlight, error) {
	resolvedKey, resolvedModel, err := resolveAPIKeyAndModel(apiKey, "OPENROUTER_API_KEY", model, "openrouter/free", "OpenRouter")
	if err != nil {
		return nil, err
	}

	highlights, err := callOpenAICompatibleAPI(openRouterEndpoint, entries, resolvedKey, resolvedModel, targetLang, isShorts)
	if err != nil && resolvedModel != "openrouter/free" {
		fmt.Printf("[AI WARN] Model '%s' failed: %v. Retrying with fallback model 'openrouter/free'...\n", resolvedModel, err)
		return callOpenAICompatibleAPI(openRouterEndpoint, entries, resolvedKey, "openrouter/free", targetLang, isShorts)
	}
	return highlights, err
}

// TranslateSubtitlesOpenRouter translates subtitle entries via OpenRouter API.
func TranslateSubtitlesOpenRouter(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string) ([]transcriber.SubtitleEntry, error) {
	resolvedKey, resolvedModel, err := resolveAPIKeyAndModel(apiKey, "OPENROUTER_API_KEY", model, "openrouter/free", "OpenRouter")
	if err != nil {
		return entries, err
	}

	translated, err := callOpenAICompatibleTranslation(openRouterEndpoint, entries, resolvedKey, resolvedModel, targetLang)
	if err != nil && resolvedModel != "openrouter/free" {
		fmt.Printf("[AI WARN] Translation model '%s' failed: %v. Retrying with fallback model 'openrouter/free'...\n", resolvedModel, err)
		return callOpenAICompatibleTranslation(openRouterEndpoint, entries, resolvedKey, "openrouter/free", targetLang)
	}
	return translated, err
}

