package ai

import (
	"github.com/misbakhul29/clipper/pkg/transcriber"
)

const deepSeekEndpoint = "https://api.deepseek.com/chat/completions"

// AnalyzeHighlightsDeepSeek sends transcript entries to DeepSeek API (OpenAI-compatible format).
func AnalyzeHighlightsDeepSeek(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string, isShorts bool) ([]AIHighlight, error) {
	resolvedKey, resolvedModel, err := resolveAPIKeyAndModel(apiKey, "DEEPSEEK_API_KEY", model, "deepseek-chat", "DeepSeek")
	if err != nil {
		return nil, err
	}

	return callOpenAICompatibleAPI(deepSeekEndpoint, entries, resolvedKey, resolvedModel, targetLang, isShorts)
}

// TranslateSubtitlesDeepSeek translates subtitle entries via DeepSeek API.
func TranslateSubtitlesDeepSeek(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string) ([]transcriber.SubtitleEntry, error) {
	resolvedKey, resolvedModel, err := resolveAPIKeyAndModel(apiKey, "DEEPSEEK_API_KEY", model, "deepseek-chat", "DeepSeek")
	if err != nil {
		return entries, err
	}

	return callOpenAICompatibleTranslation(deepSeekEndpoint, entries, resolvedKey, resolvedModel, targetLang)
}

