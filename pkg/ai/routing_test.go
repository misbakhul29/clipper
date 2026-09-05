package ai

import (
	"testing"
)

func TestResolveTaskConfig_OneToMany(t *testing.T) {
	profiles := []AIProfile{
		{
			ID:     "gemini_acc_1",
			Router: "gemini",
			Model:  "gemini-2.5-flash",
			Key:    "KEY_GEMINI_1",
		},
		{
			ID:     "gemini_acc_2",
			Router: "gemini",
			Model:  "gemini-2.5-pro",
			Key:    "KEY_GEMINI_2",
		},
		{
			ID:     "deepseek_main",
			Router: "deepseek",
			Model:  "deepseek-chat",
			Key:    "KEY_DEEPSEEK",
		},
	}

	// Skenario: gemini_acc_1 digunakan oleh BANYAK task (One-to-Many: segment dan sub_translate)
	routing := AIRoutingModels{
		Segment:      "gemini_acc_1",
		SubTranslate: "gemini_acc_1", // Sharing key yang sama
		Metadata:     "deepseek_main",
	}

	fallback := AIProviderConfig{
		APIRouter: "openrouter",
		Model:     "openrouter/free",
		IsShorts:  true,
	}

	// 1. Resolve Segment
	segCfg := ResolveTaskConfig("segment", profiles, routing, fallback)
	if segCfg.APIRouter != "gemini" || segCfg.APIKey != "KEY_GEMINI_1" || segCfg.Model != "gemini-2.5-flash" {
		t.Errorf("segment config mismatch: %+v", segCfg)
	}

	// 2. Resolve SubTranslate (One-to-Many)
	subCfg := ResolveTaskConfig("sub_translate", profiles, routing, fallback)
	if subCfg.APIRouter != "gemini" || subCfg.APIKey != "KEY_GEMINI_1" || subCfg.Model != "gemini-2.5-flash" {
		t.Errorf("sub_translate config mismatch: %+v", subCfg)
	}

	// 3. Resolve Metadata
	metaCfg := ResolveTaskConfig("metadata", profiles, routing, fallback)
	if metaCfg.APIRouter != "deepseek" || metaCfg.APIKey != "KEY_DEEPSEEK" || metaCfg.Model != "deepseek-chat" {
		t.Errorf("metadata config mismatch: %+v", metaCfg)
	}

	// 4. Fallback if task is unknown or routing is empty
	emptyRouting := AIRoutingModels{}
	fallbackCfg := ResolveTaskConfig("unknown_task", profiles, emptyRouting, fallback)
	if fallbackCfg.APIKey != "KEY_GEMINI_1" { // picks first profile
		t.Errorf("expected first profile as fallback, got %+v", fallbackCfg)
	}
}
