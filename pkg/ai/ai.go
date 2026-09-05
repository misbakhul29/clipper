package ai

import (
	"fmt"
	"strings"

	"github.com/misbakhul29/clipper/pkg/transcriber"
)

type AIProviderConfig struct {
	APIRouter      string  `json:"api_router"` // "openrouter", "gemini", "deepseek", "openai" (or "codex")
	APIKey         string  `json:"api_key"`
	Model          string  `json:"model"`
	IsShorts       bool    `json:"is_shorts"`
	TargetDuration float64 `json:"target_duration"` // Desired clip duration in seconds (0 = auto)
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

// RefineAndTranslateSubtitlesMultiProvider cleans and optionally translates subtitle cues using the configured AI provider.
func RefineAndTranslateSubtitlesMultiProvider(entries []transcriber.SubtitleEntry, aiCfg AIProviderConfig, targetLang string, needTranslation bool) ([]transcriber.SubtitleEntry, error) {
	if len(entries) == 0 {
		return entries, nil
	}

	sysPrompt, userPrompt := BuildSubtitleRefineAndTranslatePrompts(entries, targetLang, needTranslation)
	router := strings.ToLower(strings.TrimSpace(aiCfg.APIRouter))

	var content string
	var err error

	if router == "gemini" {
		resolvedKey, resolvedModel, rErr := resolveAPIKeyAndModel(aiCfg.APIKey, "GEMINI_API_KEY", aiCfg.Model, "gemini-3.6-flash", "Gemini")
		if rErr == nil {
			content, err = callGeminiGenerate(resolvedKey, resolvedModel, sysPrompt+"\n\n"+userPrompt)
		} else {
			err = rErr
		}
	} else if router == "deepseek" {
		resolvedKey, resolvedModel, rErr := resolveAPIKeyAndModel(aiCfg.APIKey, "DEEPSEEK_API_KEY", aiCfg.Model, "deepseek-chat", "DeepSeek")
		if rErr == nil {
			content, err = callOpenAICompatibleCompletions(deepSeekEndpoint, resolvedKey, resolvedModel, sysPrompt, userPrompt)
		} else {
			err = rErr
		}
	} else if router == "openai" || router == "codex" {
		resolvedKey, resolvedModel, rErr := resolveAPIKeyAndModel(aiCfg.APIKey, "OPENAI_API_KEY", aiCfg.Model, "gpt-4o-mini", "OpenAI")
		if rErr == nil {
			content, err = callOpenAICompatibleCompletions(openAIEndpoint, resolvedKey, resolvedModel, sysPrompt, userPrompt)
		} else {
			err = rErr
		}
	} else {
		// default OpenRouter
		resolvedKey, resolvedModel, rErr := resolveAPIKeyAndModel(aiCfg.APIKey, "OPENROUTER_API_KEY", aiCfg.Model, "openrouter/free", "OpenRouter")
		if rErr == nil {
			content, err = callOpenAICompatibleCompletions(openRouterEndpoint, resolvedKey, resolvedModel, sysPrompt, userPrompt)
		} else {
			err = rErr
		}
	}

	if err == nil && content != "" {
		if parsed, pErr := ParseSubtitleTranslationJSON(content, entries); pErr == nil && len(parsed) > 0 {
			// Apply final clean pass
			for i := range parsed {
				parsed[i].Text = transcriber.CleanSubtitleGarbage(parsed[i].Text)
			}
			return parsed, nil
		}
	}

	// Deterministic fallback: clean garbage on all entries
	fallbackEntries := make([]transcriber.SubtitleEntry, len(entries))
	copy(fallbackEntries, entries)
	for i := range fallbackEntries {
		fallbackEntries[i].Text = transcriber.CleanSubtitleGarbage(fallbackEntries[i].Text)
	}
	return fallbackEntries, err
}

// TranslateSubtitlesMultiProvider translates subtitle cues to targetLang using the configured AI provider.
func TranslateSubtitlesMultiProvider(entries []transcriber.SubtitleEntry, aiCfg AIProviderConfig, targetLang string) ([]transcriber.SubtitleEntry, error) {
	return RefineAndTranslateSubtitlesMultiProvider(entries, aiCfg, targetLang, true)
}

// AnalyzeHighlightsWithoutSubtitles generates video highlight segments from title and duration without needing subtitles.
func AnalyzeHighlightsWithoutSubtitles(videoTitle string, durationSec float64, aiCfg AIProviderConfig, targetLang string) ([]AIHighlight, error) {
	sysPrompt, userPrompt := BuildMetadataHighlightPrompts(videoTitle, durationSec, aiCfg.TargetDuration, targetLang, aiCfg.IsShorts)

	router := strings.ToLower(strings.TrimSpace(aiCfg.APIRouter))
	var content string
	var err error

	if router == "gemini" {
		resolvedKey, resolvedModel, rErr := resolveAPIKeyAndModel(aiCfg.APIKey, "GEMINI_API_KEY", aiCfg.Model, "gemini-3.6-flash", "Gemini")
		if rErr == nil {
			fullPrompt := fmt.Sprintf("%s\n\n%s", sysPrompt, userPrompt)
			content, err = callGeminiGenerate(resolvedKey, resolvedModel, fullPrompt)
			if err != nil {
				fmt.Printf("[AI ERROR] Gemini highlight analysis failed: %v\n", err)
			}
		} else {
			err = rErr
			fmt.Printf("[AI WARN] Gemini key resolution: %v\n", err)
		}
	} else if router == "deepseek" {
		resolvedKey, resolvedModel, rErr := resolveAPIKeyAndModel(aiCfg.APIKey, "DEEPSEEK_API_KEY", aiCfg.Model, "deepseek-chat", "DeepSeek")
		if rErr == nil {
			content, err = callOpenAICompatibleCompletions(deepSeekEndpoint, resolvedKey, resolvedModel, sysPrompt, userPrompt)
		} else {
			err = rErr
		}
	} else if router == "openai" || router == "codex" {
		resolvedKey, resolvedModel, rErr := resolveAPIKeyAndModel(aiCfg.APIKey, "OPENAI_API_KEY", aiCfg.Model, "gpt-4o-mini", "OpenAI")
		if rErr == nil {
			content, err = callOpenAICompatibleCompletions(openAIEndpoint, resolvedKey, resolvedModel, sysPrompt, userPrompt)
		} else {
			err = rErr
		}
	} else {
		// default OpenRouter
		resolvedKey, resolvedModel, rErr := resolveAPIKeyAndModel(aiCfg.APIKey, "OPENROUTER_API_KEY", aiCfg.Model, "openrouter/free", "OpenRouter")
		if rErr == nil {
			content, err = callOpenAICompatibleCompletions(openRouterEndpoint, resolvedKey, resolvedModel, sysPrompt, userPrompt)
		} else {
			err = rErr
		}
	}

	if err == nil && content != "" {
		if highlights, pErr := ParseAIHighlightsJSON(content); pErr == nil && len(highlights) > 0 {
			return highlights, nil
		} else if pErr != nil {
			fmt.Printf("[AI WARN] Failed to parse AI highlights JSON: %v. Content: %s\n", pErr, content)
		}
	}

	// Graceful fallback: smartly distributed highlights across the video's actual duration (always succeeds!)
	fmt.Printf("[AI INFO] Falling back to heuristic highlight distribution across %.1fs video duration.\n", durationSec)
	return GenerateHeuristicHighlights(durationSec, aiCfg.IsShorts, aiCfg.TargetDuration, targetLang), nil
}

