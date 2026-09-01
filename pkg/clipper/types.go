package clipper

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"clipping/pkg/ai"
)

func sanitizeFilename(s string) string {
	var builder strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			builder.WriteRune(r)
		} else if r == ' ' {
			builder.WriteRune('_')
		}
	}
	return builder.String()
}

// Mode specifies how the cut segments should be saved.
type Mode string

const (
	ModeSplit Mode = "split"
	ModeMerge Mode = "merge"
)

// CutStrategy specifies whether FFmpeg uses stream copy or re-encoding.
type CutStrategy string

const (
	StrategyFast     CutStrategy = "fast"     // -c copy (fast, may align to nearest keyframe)
	StrategyAccurate CutStrategy = "accurate" // re-encode for frame-accurate cuts
)

// Segment represents a video cut specification.
type Segment struct {
	Start string `json:"start"` // e.g. "00:00:10", "10", "01:15.5"
	End   string `json:"end"`   // e.g. "00:00:25", "25", "01:30.0"
	Title string `json:"title,omitempty"`
}

// Config holds options for the video cutting job.
type Config struct {
	InputFile     string              `json:"input"`
	OutputDir     string              `json:"output_dir"`
	OutputFile    string              `json:"output"`       // Used for merge mode or prefix
	Mode          Mode                `json:"mode"`         // "split" or "merge"
	Strategy      CutStrategy         `json:"strategy"`     // "fast" or "accurate"
	Shorts        bool                `json:"shorts"`       // Convert to 9:16 Shorts format
	ShortsStyle   string              `json:"shorts_style"` // "crop" (center 9:16) or "blur" (blurred background 9:16)
	Quality       string              `json:"quality"`      // YouTube download quality e.g. "best", "1080p", "720p", "480p", "360p", "worst"
	CacheDir      string              `json:"cache_dir"`    // Directory to cache downloaded YouTube videos (default: "./cache")
	NoCache       bool                `json:"no_cache"`     // Disable YouTube download cache and force re-download
	Concurrency   int                 `json:"concurrency"`  // Number of parallel workers for rendering clips (default: NumCPU)
	WatermarkPath string              `json:"watermark"`    // Path to watermark image (PNG)
	WatermarkPos  string              `json:"watermark_pos"`// Position: top-left, top-right, bottom-left, bottom-right, center
	OverlayText   string              `json:"overlay_text"` // Text caption to render on video
	TextPos       string              `json:"text_pos"`     // Position for overlay text
	FontSize      int                 `json:"font_size"`    // Font size for overlay text
	FontColor     string              `json:"font_color"`   // Font color for overlay text (e.g. "white", "yellow")
	AutoDetect    string              `json:"auto_detect"`  // Auto detection mode: "silence", "scene", or "ai"
	TranslateLang string              `json:"translate_lang"`// Target language for subtitle translation (e.g. "id", "en")
	BurnSubtitles bool                `json:"burn_subtitles"`// Hardcode/burn-in subtitles directly onto video clips
	SubStyle      string              `json:"sub_style"`    // Subtitle style: "karaoke" (TikTok 2-word chunks) or "standard"
	SubFontSize   int                 `json:"sub_font_size"` // Subtitle font size for burnt-in captions (default: 48)
	SubFontPath   string              `json:"sub_font_path"`// Path to custom font file (.ttf / .otf) for burnt-in captions
	UseWhisper    bool                `json:"use_whisper"`   // Force local Whisper AI for speech-to-text transcription
	AIConfig      ai.AIProviderConfig `json:"ai_config"`    // Multi-provider AI config
	OpenRouterKey string              `json:"openrouter_key"`// OpenRouter API Key (legacy fallback)
	AIModel       string              `json:"ai_model"`     // AI model name (legacy fallback)
	DryRun        bool                `json:"dry_run"`      // Dry-run mode: analyze & preview commands without rendering video
	BatchList     string              `json:"batch_list"`    // Path to text file containing list of video URLs/files (one per line)
	CleanCache    bool                `json:"clean_cache"`   // Clean cache directory
	CleanDays     int                 `json:"clean_days"`    // Delete cache files older than N days (0 = clean all)
	Segments      []Segment           `json:"segments"`
}

// GetBatchInputs parses multiple input URLs or file paths from BatchList or comma-separated InputFile.
func (c *Config) GetBatchInputs() []string {
	var inputs []string
	if c.BatchList != "" {
		if data, err := os.ReadFile(c.BatchList); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, l := range lines {
				l = strings.TrimSpace(l)
				if l != "" && !strings.HasPrefix(l, "#") {
					inputs = append(inputs, l)
				}
			}
		}
	}
	if len(inputs) == 0 && strings.Contains(c.InputFile, ",") {
		for _, part := range strings.Split(c.InputFile, ",") {
			p := strings.TrimSpace(part)
			if p != "" {
				inputs = append(inputs, p)
			}
		}
	}
	if len(inputs) == 0 && c.InputFile != "" {
		inputs = append(inputs, c.InputFile)
	}
	return inputs
}

// Validate checks the configuration for missing or invalid parameters.
func (c *Config) Validate() error {
	if c.CleanCache {
		return nil
	}
	if c.InputFile == "" && c.BatchList == "" {
		return fmt.Errorf("input video file path or -batch-list is required")
	}
	if len(c.Segments) == 0 && c.AutoDetect == "" {
		return fmt.Errorf("either segments or auto_detect must be provided")
	}
	if c.Mode == "" {
		c.Mode = ModeSplit
	}
	if c.Mode != ModeSplit && c.Mode != ModeMerge {
		return fmt.Errorf("invalid mode '%s', must be 'split' or 'merge'", c.Mode)
	}
	if c.Strategy == "" {
		c.Strategy = StrategyFast
	}
	if c.Strategy != StrategyFast && c.Strategy != StrategyAccurate {
		return fmt.Errorf("invalid strategy '%s', must be 'fast' or 'accurate'", c.Strategy)
	}
	if c.ShortsStyle == "" {
		c.ShortsStyle = "crop"
	}
	if c.ShortsStyle != "crop" && c.ShortsStyle != "blur" && c.ShortsStyle != "smart-crop" {
		return fmt.Errorf("invalid shorts_style '%s', must be 'crop', 'blur', or 'smart-crop'", c.ShortsStyle)
	}
	if c.Quality == "" {
		c.Quality = "best"
	}
	if c.CacheDir == "" {
		c.CacheDir = "./cache"
	}

	// Validate / Sync AIConfig
	if c.AIConfig.APIRouter == "" {
		c.AIConfig.APIRouter = "openrouter"
	}
	if c.AIConfig.APIKey == "" && c.OpenRouterKey != "" {
		c.AIConfig.APIKey = c.OpenRouterKey
	}
	if c.AIConfig.Model == "" && c.AIModel != "" {
		c.AIConfig.Model = c.AIModel
	}
	if c.AIConfig.Model == "" {
		router := strings.ToLower(c.AIConfig.APIRouter)
		if router == "gemini" {
			c.AIConfig.Model = "gemini-2.0-flash"
		} else if router == "deepseek" {
			c.AIConfig.Model = "deepseek-chat"
		} else if router == "openai" || router == "codex" {
			c.AIConfig.Model = "gpt-4o-mini"
		} else {
			c.AIConfig.Model = "openrouter/free"
		}
	}
	// Sync back flat fields for compatibility
	c.OpenRouterKey = c.AIConfig.APIKey
	c.AIModel = c.AIConfig.Model
	if c.Concurrency <= 0 {
		c.Concurrency = runtime.NumCPU()
	}
	if c.WatermarkPos == "" {
		c.WatermarkPos = "top-right"
	}
	if c.TextPos == "" {
		c.TextPos = "bottom-center"
	}
	if c.FontSize <= 0 {
		c.FontSize = 32
	}
	if c.SubFontSize <= 0 {
		c.SubFontSize = 48
	}
	if c.SubStyle == "" {
		c.SubStyle = "karaoke"
	}
	if c.FontColor == "" {
		c.FontColor = "white"
	}
	return nil
}
