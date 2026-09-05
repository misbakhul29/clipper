package clipper

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/misbakhul29/clipper/pkg/ai"
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

// SubtitleCue represents an individual timestamped subtitle text entry inside a clip.
type SubtitleCue struct {
	Start float64 `json:"start"` // start offset in seconds relative to segment (e.g. 1.2)
	End   float64 `json:"end"`   // end offset in seconds relative to segment (e.g. 3.5)
	Text  string  `json:"text"`  // caption text
}

// Segment represents a video cut specification.
type Segment struct {
	Start       string        `json:"start"` // e.g. "00:00:10", "10", "01:15.5"
	End         string        `json:"end"`   // e.g. "00:00:25", "25", "01:30.0"
	Title       string        `json:"title,omitempty"`
	Subtitles   []SubtitleCue `json:"subtitles,omitempty"`     // Custom cue list for this segment
	SubPreset   string        `json:"sub_preset,omitempty"`   // Override preset per segment
	SubFontSize int           `json:"sub_font_size,omitempty"` // Override font size per segment
	SubPosition string        `json:"sub_position,omitempty"`  // "bottom", "middle", "top"
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
	AutoDetect     string              `json:"auto_detect"`     // Auto detection mode: "silence", "scene", or "ai"
	TargetDuration float64             `json:"target_duration"` // Desired segment clip duration in seconds (0 = auto)
	TranslateLang  string              `json:"translate_lang"`  // Target language for subtitle translation (e.g. "id", "en")
	Subtitles     bool                `json:"subtitles"`     // Include captions/subtitles on video clips
	SubStyle      string              `json:"sub_style"`     // Subtitle style: 'karaoke' or 'standard'
	SubPreset     string              `json:"sub_preset"`    // Viral subtitle theme preset: 'hormozi', 'minimal', 'devon', 'neon', 'cinematic'
	SubSDHMode    string              `json:"sub_sdh_mode"`  // Handling for silent narrator & SDH brackets: 'strip', 'top-box', 'keep'
	SubEmoji      bool                `json:"sub_emoji"`     // Auto-inject contextual emojis based on keywords into subtitle cues
	SubFontSize   int                 `json:"sub_font_size"` // Subtitle font size for burnt-in captions (default: 48)
	SubFontPath   string              `json:"sub_font_path"`// Path to custom font file (.ttf / .otf) for burnt-in captions
	UseWhisper       bool                `json:"use_whisper"`       // Force local Whisper AI for speech-to-text transcription
	GenerateMetadata bool                `json:"generate_metadata"` // Generate companion social metadata (metadata.json / .txt) for clips
	ExtractThumbnail bool                `json:"extract_thumbnail"` // Extract high-resolution cover thumbnail & hook frame (.jpg)
	ThumbnailCount   int                 `json:"thumbnail_count"`   // Number of candidate thumbnails to extract (1 to 3, default: 1)
	HWAccel          string              `json:"hwaccel"`           // Hardware acceleration mode: 'auto', 'nvenc', 'videotoolbox', 'qsv', 'vaapi', 'amf', 'cpu'
	ShowProgress     bool                `json:"show_progress"`     // Display interactive terminal progress bar during rendering (default: true)
	AIConfig         ai.AIProviderConfig `json:"ai_config,omitempty"`        // Multi-provider AI config
	AIConfigs        []ai.AIProfile      `json:"ai_configs,omitempty"`     // Multi-AI account profiles list
	RoutingModels    ai.AIRoutingModels  `json:"routing_models,omitempty"` // Task-to-Profile ID routing mapping
	OpenRouterKey string              `json:"openrouter_key"`// OpenRouter API Key (legacy fallback)
	AIModel       string              `json:"ai_model"`     // AI model name (legacy fallback)
	DryRun        bool                `json:"dry_run"`      // Dry-run mode: analyze & preview commands without rendering video
	BatchList     string              `json:"batch_list"`    // Path to text file containing list of video URLs/files (one per line)
	CleanCache    bool                `json:"clean_cache"`   // Clean cache directory
	CleanDays     int                 `json:"clean_days"`    // Delete cache files older than N days (0 = clean all)
	FaceTracking  bool                `json:"face_tracking"` // Dynamic active speaker / face tracking for smart-crop
	PanDuration   float64             `json:"pan_duration"`  // Duration of camera pan interpolation in seconds (default: 0.8)
	Loudnorm      bool                `json:"loudnorm"`      // EBU R128 audio normalization (-af loudnorm)
	LoudnormI     float64             `json:"loudnorm_i"`    // Integrated loudness target in LUFS (default: -14)
	LoudnormLRA   float64             `json:"loudnorm_lra"`  // Loudness range target in LU (default: 7)
	LoudnormTP    float64             `json:"loudnorm_tp"`   // Maximum true peak in dBTP (default: -2)
	JumpCut       bool                `json:"jump_cut"`      // Smart silence removal & snappy jump-cuts inside clips
	JumpCutMinSil float64             `json:"jump_cut_min_silence"` // Minimum silence pause to cut in seconds (default: 1.0)
	JumpCutMargin float64             `json:"jump_cut_margin"` // Padding margin around speech in seconds (default: 0.2)
	JumpCutNoise  float64             `json:"jump_cut_noise"`  // Silence noise gate threshold in dB (default: -30.0)
	Segments      []Segment           `json:"segments"`
}

// DefaultConfig returns a Config initialized with system-wide canonical defaults.
func DefaultConfig() Config {
	return Config{
		OutputDir:        "./clips",
		OutputFile:       "merged_highlight.mp4",
		Mode:             ModeSplit,
		Strategy:         StrategyFast,
		Shorts:           true,
		ShortsStyle:      "blur",
		Quality:          "1080p",
		CacheDir:         "./cache",
		Concurrency:      runtime.NumCPU(),
		AutoDetect:       "ai",
		TargetDuration:   30,
		TranslateLang:    "id",
		Subtitles:        true,
		SubPreset:        "hormozi",
		SubSDHMode:       "strip",
		SubEmoji:         true,
		SubFontSize:      48,
		GenerateMetadata: true,
		ExtractThumbnail: true,
		ThumbnailCount:   1,
		HWAccel:          "auto",
		ShowProgress:     true,
		FaceTracking:     true,
		PanDuration:      0.8,
		Loudnorm:         true,
		LoudnormI:        -14,
		LoudnormLRA:      7,
		LoudnormTP:       -2,
		JumpCut:          true,
		JumpCutMinSil:    1.0,
		JumpCutMargin:    0.2,
		JumpCutNoise:     -30.0,
	}
}

// NewSampleConfig returns a populated sample Config for starter templates and config generation.
// This serves as the single source of truth for CLI templates, initialization, and tests.
func NewSampleConfig() Config {
	cfg := DefaultConfig()
	cfg.InputFile = "https://www.youtube.com/watch?v=sample_video"
	cfg.AIConfigs = []ai.AIProfile{
		{
			ID:     "gemini_acc_1",
			Router: "gemini",
			Model:  "gemini-2.5-flash",
			Key:    "YOUR_GEMINI_API_KEY_1",
		},
		{
			ID:     "gemini_acc_2",
			Router: "gemini",
			Model:  "gemini-2.5-flash",
			Key:    "YOUR_GEMINI_API_KEY_2",
		},
		{
			ID:     "deepseek_main",
			Router: "deepseek",
			Model:  "deepseek-chat",
			Key:    "YOUR_DEEPSEEK_API_KEY",
		},
	}
	cfg.RoutingModels = ai.AIRoutingModels{
		Segment:      "gemini_acc_1",
		SubTranslate: "gemini_acc_1",
		Metadata:     "deepseek_main",
	}
	cfg.Segments = []Segment{
		{
			Start: "00:00:10",
			End:   "00:00:40",
			Title: "Highlight 1 - Hook",
		},
		{
			Start: "00:01:20",
			End:   "00:01:50",
			Title: "Highlight 2 - Peak Moment",
		},
	}
	return cfg
}

// GetAITaskConfig resolves the effective AIProviderConfig for a given task (e.g. "segment", "sub_translate", "metadata").
func (c *Config) GetAITaskConfig(task string) ai.AIProviderConfig {
	fallback := c.AIConfig
	fallback.IsShorts = c.Shorts
	if c.TargetDuration > 0 {
		fallback.TargetDuration = c.TargetDuration
	}
	return ai.ResolveTaskConfig(task, c.AIConfigs, c.RoutingModels, fallback)
}

// MarshalJSON customizes serialization to omit empty ai_config when ai_configs is used.
func (c Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	aux := struct {
		Alias
		AIConfig *ai.AIProviderConfig `json:"ai_config,omitempty"`
	}{
		Alias: Alias(c),
	}
	if c.AIConfig.APIRouter != "" || c.AIConfig.APIKey != "" || c.AIConfig.Model != "" {
		aux.AIConfig = &c.AIConfig
	}
	return json.Marshal(aux)
}

// UnmarshalJSON implements custom unmarshaling to support "subtitles", "subtitle", "burn_subtitles", and "burn_subtitle" config keys,
// as well as flexible parsing for single vs array ai_config.
func (c *Config) UnmarshalJSON(data []byte) error {
	type Alias Config
	aux := &struct {
		Subtitles     *bool           `json:"subtitles"`
		Subtitle      *bool           `json:"subtitle"`
		BurnSubtitles *bool           `json:"burn_subtitles"`
		BurnSubtitle  *bool           `json:"burn_subtitle"`
		RawAIConfig   json.RawMessage `json:"ai_config"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if aux.Subtitles != nil {
		c.Subtitles = *aux.Subtitles
	} else if aux.Subtitle != nil {
		c.Subtitles = *aux.Subtitle
	} else if aux.BurnSubtitles != nil {
		c.Subtitles = *aux.BurnSubtitles
	} else if aux.BurnSubtitle != nil {
		c.Subtitles = *aux.BurnSubtitle
	}

	// Handle ai_config if provided as array []AIProfile
	if len(aux.RawAIConfig) > 0 {
		trimmed := strings.TrimSpace(string(aux.RawAIConfig))
		if strings.HasPrefix(trimmed, "[") {
			var profiles []ai.AIProfile
			if err := json.Unmarshal(aux.RawAIConfig, &profiles); err == nil && len(profiles) > 0 {
				c.AIConfigs = append(c.AIConfigs, profiles...)
			}
		} else if strings.HasPrefix(trimmed, "{") {
			var single ai.AIProviderConfig
			if err := json.Unmarshal(aux.RawAIConfig, &single); err == nil {
				c.AIConfig = single
			}
		}
	}

	return nil
}

// LoadConfig reads and parses a Config from a JSON file.
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", filePath, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON in config file '%s': %w", filePath, err)
	}

	return &cfg, nil
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
	c.AIConfig.IsShorts = c.Shorts
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
