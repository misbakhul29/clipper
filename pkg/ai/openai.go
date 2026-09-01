package ai

import (
	"github.com/misbakhul29/clipper/pkg/transcriber"
)

const openAIEndpoint = "https://api.openai.com/v1/chat/completions"

// AnalyzeHighlightsOpenAI sends transcript entries to OpenAI API.
func AnalyzeHighlightsOpenAI(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string, isShorts bool) ([]AIHighlight, error) {
	resolvedKey, resolvedModel, err := resolveAPIKeyAndModel(apiKey, "OPENAI_API_KEY", model, "gpt-4o-mini", "OpenAI")
	if err != nil {
		return nil, err
	}

	return callOpenAICompatibleAPI(openAIEndpoint, entries, resolvedKey, resolvedModel, targetLang, isShorts)
}

// TranslateSubtitlesOpenAI translates subtitle entries via OpenAI API.
func TranslateSubtitlesOpenAI(entries []transcriber.SubtitleEntry, apiKey, model, targetLang string) ([]transcriber.SubtitleEntry, error) {
	resolvedKey, resolvedModel, err := resolveAPIKeyAndModel(apiKey, "OPENAI_API_KEY", model, "gpt-4o-mini", "OpenAI")
	if err != nil {
		return entries, err
	}

	return callOpenAICompatibleTranslation(openAIEndpoint, entries, resolvedKey, resolvedModel, targetLang)
}

