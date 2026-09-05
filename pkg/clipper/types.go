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

// ShortsConfig holds 9:16 vertical formatting and camera framing options.
type ShortsConfig struct {
	Enabled      bool    `json:"enabled"`                 // Convert to 9:16 Shorts format
	Style        string  `json:"style,omitempty"`        // "crop" (center 9:16), "blur" (blurred 9:16), or "smart-crop"
	FaceTracking bool    `json:"face_tracking,omitempty"` // Dynamic active speaker / face tracking for smart-crop
	PanDuration  float64 `json:"pan_duration,omitempty"`  // Duration of camera pan interpolation in seconds (default: 0.8)
}

// SubtitleConfig holds viral subtitle burning, typography, styling, and SDH options.
type SubtitleConfig struct {
	Enabled       bool   `json:"enabled"`                  // Include captions/subtitles on video clips
	Preset        string `json:"preset,omitempty"`         // Viral subtitle theme preset: 'hormozi', 'minimal', 'devon', 'neon', 'cinematic'
	Style         string `json:"style,omitempty"`          // Subtitle style: 'karaoke' or 'standard'
	SDHMode       string `json:"sdh_mode,omitempty"`       // Handling for silent narrator & SDH brackets: 'strip', 'top-box', 'keep'
	Emoji         bool   `json:"emoji,omitempty"`          // Auto-inject contextual emojis based on keywords
	FontSize      int    `json:"font_size,omitempty"`      // Subtitle font size for burnt-in captions (default: 48)
	FontPath      string `json:"font_path,omitempty"`      // Path to custom font file (.ttf / .otf)
	TranslateLang string `json:"translate_lang,omitempty"` // Target language for subtitle translation (e.g. "id", "en")
	UseWhisper    bool   `json:"use_whisper,omitempty"`    // Force local Whisper AI for speech-to-text transcription
}

// LoudnormConfig holds EBU R128 audio normalization parameters.
type LoudnormConfig struct {
	Enabled bool    `json:"enabled"`       // Enable EBU R128 audio normalization (-af loudnorm)
	I       float64 `json:"i,omitempty"`   // Integrated loudness target in LUFS (default: -14)
	LRA     float64 `json:"lra,omitempty"` // Loudness range target in LU (default: 7)
	TP      float64 `json:"tp,omitempty"`  // Maximum true peak in dBTP (default: -2)
}

// JumpCutConfig holds silence detection and jump-cut removal parameters.
type JumpCutConfig struct {
	Enabled    bool    `json:"enabled"`               // Smart silence removal & snappy jump-cuts inside clips
	MinSilence float64 `json:"min_silence,omitempty"` // Minimum silence pause to cut in seconds (default: 1.0)
	Margin     float64 `json:"margin,omitempty"`      // Padding margin around speech in seconds (default: 0.2)
	Noise      float64 `json:"noise,omitempty"`       // Silence noise gate threshold in dB (default: -30.0)
}

// AudioConfig groups audio processing options.
type AudioConfig struct {
	Loudnorm LoudnormConfig `json:"loudnorm,omitempty"`
	JumpCut  JumpCutConfig  `json:"jump_cut,omitempty"`
}

// WatermarkConfig holds watermark branding settings.
type WatermarkConfig struct {
	Path     string `json:"path,omitempty"`     // Path to watermark image (PNG)
	Position string `json:"position,omitempty"` // Position: top-left, top-right, bottom-left, bottom-right, center
}

// OverlayTextConfig holds overlay text caption settings.
type OverlayTextConfig struct {
	Text      string `json:"text,omitempty"`       // Text caption to render on video
	Position  string `json:"position,omitempty"`   // Position for overlay text
	FontSize  int    `json:"font_size,omitempty"`  // Font size for overlay text
	FontColor string `json:"font_color,omitempty"` // Font color for overlay text (e.g. "white", "yellow")
}

// BrandingConfig groups visual branding and watermark options.
type BrandingConfig struct {
	Watermark   WatermarkConfig   `json:"watermark,omitempty"`
	OverlayText OverlayTextConfig `json:"overlay_text,omitempty"`
}

// SocialConfig groups social metadata generation and thumbnail extraction.
type SocialConfig struct {
	GenerateMetadata bool `json:"generate_metadata,omitempty"` // Generate companion social metadata (metadata.json / .txt)
	ExtractThumbnail bool `json:"extract_thumbnail,omitempty"` // Extract high-resolution cover thumbnail & hook frame (.jpg)
	ThumbnailCount   int  `json:"thumbnail_count,omitempty"`   // Number of candidate thumbnails to extract (1 to 3, default: 1)
}

// Config holds options for the video cutting job.
type Config struct {
	InputFile     string              `json:"input"`
	OutputDir     string              `json:"output_dir"`
	OutputFile    string              `json:"output"`       // Used for merge mode or prefix
	Mode          Mode                `json:"mode"`         // "split" or "merge"
	Strategy      CutStrategy         `json:"strategy"`     // "fast" or "accurate"
	Quality       string              `json:"quality"`      // YouTube download quality e.g. "best", "1080p", "720p", "480p", "360p", "worst"
	CacheDir      string              `json:"cache_dir"`    // Directory to cache downloaded YouTube videos (default: "./cache")
	NoCache       bool                `json:"no_cache"`     // Disable YouTube download cache and force re-download
	Concurrency   int                 `json:"concurrency"`  // Number of parallel workers for rendering clips (default: NumCPU)
	AutoDetect     string              `json:"auto_detect"`     // Auto detection mode: "silence", "scene", or "ai"
	TargetDuration float64             `json:"target_duration"` // Desired segment clip duration in seconds (0 = auto)
	HWAccel          string              `json:"hwaccel"`           // Hardware acceleration mode: 'auto', 'nvenc', 'videotoolbox', 'qsv', 'vaapi', 'amf', 'cpu'
	ShowProgress     bool                `json:"show_progress"`     // Display interactive terminal progress bar during rendering (default: true)
	DryRun        bool                `json:"dry_run"`      // Dry-run mode: analyze & preview commands without rendering video
	BatchList     string              `json:"batch_list"`    // Path to text file containing list of video URLs/files (one per line)
	CleanCache    bool                `json:"clean_cache"`   // Clean cache directory
	CleanDays     int                 `json:"clean_days"`    // Delete cache files older than N days (0 = clean all)

	// Structured Child Configurations (handled via custom MarshalJSON / UnmarshalJSON)
	ShortsConfig   ShortsConfig   `json:"-"`
	SubtitleConfig SubtitleConfig `json:"-"`
	AudioConfig    AudioConfig    `json:"-"`
	BrandingConfig BrandingConfig `json:"-"`
	SocialConfig   SocialConfig   `json:"-"`

	// Flat Compatibility Fields
	Shorts        bool                `json:"-"`
	ShortsStyle   string              `json:"-"`
	FaceTracking  bool                `json:"-"`
	PanDuration   float64             `json:"-"`
	Subtitles     bool                `json:"-"`
	SubStyle      string              `json:"-"`
	SubPreset     string              `json:"-"`
	SubSDHMode    string              `json:"-"`
	SubEmoji      bool                `json:"-"`
	SubFontSize   int                 `json:"-"`
	SubFontPath   string              `json:"-"`
	TranslateLang string              `json:"-"`
	UseWhisper    bool                `json:"-"`
	Loudnorm      bool                `json:"-"`
	LoudnormI     float64             `json:"-"`
	LoudnormLRA   float64             `json:"-"`
	LoudnormTP    float64             `json:"-"`
	JumpCut       bool                `json:"-"`
	JumpCutMinSil float64             `json:"-"`
	JumpCutMargin float64             `json:"-"`
	JumpCutNoise  float64             `json:"-"`
	WatermarkPath string              `json:"-"`
	WatermarkPos  string              `json:"-"`
	OverlayText   string              `json:"-"`
	TextPos       string              `json:"-"`
	FontSize      int                 `json:"-"`
	FontColor     string              `json:"-"`
	GenerateMetadata bool             `json:"-"`
	ExtractThumbnail bool             `json:"-"`
	ThumbnailCount   int              `json:"-"`

	// Multi-AI account configurations & routing
	AIConfig         ai.AIProviderConfig `json:"ai_config,omitempty"`
	AIConfigs        []ai.AIProfile      `json:"ai_configs,omitempty"`
	RoutingModels    ai.AIRoutingModels  `json:"routing_models,omitempty"`
	OpenRouterKey string              `json:"openrouter_key,omitempty"`
	AIModel       string              `json:"ai_model,omitempty"`

	Segments      []Segment           `json:"segments"`
}

// Sync ensures all flat fields and nested child structs remain completely synchronized.
func (c *Config) Sync() {
	// Shorts sync
	if c.ShortsConfig.Enabled || c.ShortsConfig.Style != "" || c.ShortsConfig.FaceTracking || c.ShortsConfig.PanDuration > 0 {
		c.Shorts = c.ShortsConfig.Enabled
		c.ShortsStyle = c.ShortsConfig.Style
		c.FaceTracking = c.ShortsConfig.FaceTracking
		c.PanDuration = c.ShortsConfig.PanDuration
	} else if c.Shorts || c.ShortsStyle != "" || c.FaceTracking || c.PanDuration > 0 {
		c.ShortsConfig.Enabled = c.Shorts
		c.ShortsConfig.Style = c.ShortsStyle
		c.ShortsConfig.FaceTracking = c.FaceTracking
		c.ShortsConfig.PanDuration = c.PanDuration
	}

	// Subtitles sync
	if c.SubtitleConfig.Enabled || c.SubtitleConfig.Preset != "" || c.SubtitleConfig.Style != "" || c.SubtitleConfig.SDHMode != "" || c.SubtitleConfig.Emoji || c.SubtitleConfig.FontSize > 0 || c.SubtitleConfig.FontPath != "" || c.SubtitleConfig.TranslateLang != "" || c.SubtitleConfig.UseWhisper {
		c.Subtitles = c.SubtitleConfig.Enabled
		c.SubPreset = c.SubtitleConfig.Preset
		c.SubStyle = c.SubtitleConfig.Style
		c.SubSDHMode = c.SubtitleConfig.SDHMode
		c.SubEmoji = c.SubtitleConfig.Emoji
		c.SubFontSize = c.SubtitleConfig.FontSize
		c.SubFontPath = c.SubtitleConfig.FontPath
		c.TranslateLang = c.SubtitleConfig.TranslateLang
		c.UseWhisper = c.SubtitleConfig.UseWhisper
	} else if c.Subtitles || c.SubPreset != "" || c.SubStyle != "" || c.SubSDHMode != "" || c.SubEmoji || c.SubFontSize > 0 || c.SubFontPath != "" || c.TranslateLang != "" || c.UseWhisper {
		c.SubtitleConfig.Enabled = c.Subtitles
		c.SubtitleConfig.Preset = c.SubPreset
		c.SubtitleConfig.Style = c.SubStyle
		c.SubtitleConfig.SDHMode = c.SubSDHMode
		c.SubtitleConfig.Emoji = c.SubEmoji
		c.SubtitleConfig.FontSize = c.SubFontSize
		c.SubtitleConfig.FontPath = c.SubFontPath
		c.SubtitleConfig.TranslateLang = c.TranslateLang
		c.SubtitleConfig.UseWhisper = c.UseWhisper
	}

	// Audio loudnorm sync
	if c.AudioConfig.Loudnorm.Enabled || c.AudioConfig.Loudnorm.I != 0 || c.AudioConfig.Loudnorm.LRA != 0 || c.AudioConfig.Loudnorm.TP != 0 {
		c.Loudnorm = c.AudioConfig.Loudnorm.Enabled
		c.LoudnormI = c.AudioConfig.Loudnorm.I
		c.LoudnormLRA = c.AudioConfig.Loudnorm.LRA
		c.LoudnormTP = c.AudioConfig.Loudnorm.TP
	} else if c.Loudnorm || c.LoudnormI != 0 || c.LoudnormLRA != 0 || c.LoudnormTP != 0 {
		c.AudioConfig.Loudnorm.Enabled = c.Loudnorm
		c.AudioConfig.Loudnorm.I = c.LoudnormI
		c.AudioConfig.Loudnorm.LRA = c.LoudnormLRA
		c.AudioConfig.Loudnorm.TP = c.LoudnormTP
	}

	// Audio jump-cut sync
	if c.AudioConfig.JumpCut.Enabled || c.AudioConfig.JumpCut.MinSilence > 0 || c.AudioConfig.JumpCut.Margin > 0 || c.AudioConfig.JumpCut.Noise != 0 {
		c.JumpCut = c.AudioConfig.JumpCut.Enabled
		c.JumpCutMinSil = c.AudioConfig.JumpCut.MinSilence
		c.JumpCutMargin = c.AudioConfig.JumpCut.Margin
		c.JumpCutNoise = c.AudioConfig.JumpCut.Noise
	} else if c.JumpCut || c.JumpCutMinSil > 0 || c.JumpCutMargin > 0 || c.JumpCutNoise != 0 {
		c.AudioConfig.JumpCut.Enabled = c.JumpCut
		c.AudioConfig.JumpCut.MinSilence = c.JumpCutMinSil
		c.AudioConfig.JumpCut.Margin = c.JumpCutMargin
		c.AudioConfig.JumpCut.Noise = c.JumpCutNoise
	}

	// Branding sync
	if c.BrandingConfig.Watermark.Path != "" || c.BrandingConfig.Watermark.Position != "" {
		c.WatermarkPath = c.BrandingConfig.Watermark.Path
		c.WatermarkPos = c.BrandingConfig.Watermark.Position
	} else if c.WatermarkPath != "" || c.WatermarkPos != "" {
		c.BrandingConfig.Watermark.Path = c.WatermarkPath
		c.BrandingConfig.Watermark.Position = c.WatermarkPos
	}
	if c.BrandingConfig.OverlayText.Text != "" || c.BrandingConfig.OverlayText.Position != "" || c.BrandingConfig.OverlayText.FontSize > 0 || c.BrandingConfig.OverlayText.FontColor != "" {
		c.OverlayText = c.BrandingConfig.OverlayText.Text
		c.TextPos = c.BrandingConfig.OverlayText.Position
		c.FontSize = c.BrandingConfig.OverlayText.FontSize
		c.FontColor = c.BrandingConfig.OverlayText.FontColor
	} else if c.OverlayText != "" || c.TextPos != "" || c.FontSize > 0 || c.FontColor != "" {
		c.BrandingConfig.OverlayText.Text = c.OverlayText
		c.BrandingConfig.OverlayText.Position = c.TextPos
		c.BrandingConfig.OverlayText.FontSize = c.FontSize
		c.BrandingConfig.OverlayText.FontColor = c.FontColor
	}

	// Social sync
	if c.SocialConfig.GenerateMetadata || c.SocialConfig.ExtractThumbnail || c.SocialConfig.ThumbnailCount > 0 {
		c.GenerateMetadata = c.SocialConfig.GenerateMetadata
		c.ExtractThumbnail = c.SocialConfig.ExtractThumbnail
		c.ThumbnailCount = c.SocialConfig.ThumbnailCount
	} else if c.GenerateMetadata || c.ExtractThumbnail || c.ThumbnailCount > 0 {
		c.SocialConfig.GenerateMetadata = c.GenerateMetadata
		c.SocialConfig.ExtractThumbnail = c.ExtractThumbnail
		c.SocialConfig.ThumbnailCount = c.ThumbnailCount
	}
}

// DefaultConfig returns a Config initialized with system-wide canonical defaults.
func DefaultConfig() Config {
	cfg := Config{
		OutputDir:      "./clips",
		OutputFile:     "merged_highlight.mp4",
		Mode:           ModeSplit,
		Strategy:       StrategyFast,
		Quality:        "1080p",
		CacheDir:       "./cache",
		Concurrency:    runtime.NumCPU(),
		AutoDetect:     "ai",
		TargetDuration: 30,
		HWAccel:        "auto",
		ShowProgress:   true,

		ShortsConfig: ShortsConfig{
			Enabled:      true,
			Style:        "blur",
			FaceTracking: true,
			PanDuration:  0.8,
		},
		SubtitleConfig: SubtitleConfig{
			Enabled:       true,
			Preset:        "hormozi",
			Style:         "karaoke",
			SDHMode:       "strip",
			Emoji:         true,
			FontSize:      48,
			TranslateLang: "id",
		},
		AudioConfig: AudioConfig{
			Loudnorm: LoudnormConfig{
				Enabled: true,
				I:       -14,
				LRA:     7,
				TP:      -2,
			},
			JumpCut: JumpCutConfig{
				Enabled:    true,
				MinSilence: 1.0,
				Margin:     0.2,
				Noise:      -30.0,
			},
		},
		SocialConfig: SocialConfig{
			GenerateMetadata: true,
			ExtractThumbnail: true,
			ThumbnailCount:   1,
		},
	}
	cfg.Sync()
	return cfg
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
	cfg.Sync()
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

// MarshalJSON customizes serialization to output clean structured grouping.
func (c Config) MarshalJSON() ([]byte, error) {
	cCopy := c
	cCopy.Sync()

	type Alias Config
	aux := struct {
		Alias
		Shorts    ShortsConfig         `json:"shorts"`
		Subtitles SubtitleConfig       `json:"subtitles"`
		Audio     AudioConfig          `json:"audio"`
		Branding  *BrandingConfig      `json:"branding,omitempty"`
		Social    *SocialConfig        `json:"social,omitempty"`
		AIConfig  *ai.AIProviderConfig `json:"ai_config,omitempty"`
	}{
		Alias:     Alias(cCopy),
		Shorts:    cCopy.ShortsConfig,
		Subtitles: cCopy.SubtitleConfig,
		Audio:     cCopy.AudioConfig,
	}
	if cCopy.BrandingConfig.Watermark.Path != "" || cCopy.BrandingConfig.OverlayText.Text != "" {
		aux.Branding = &cCopy.BrandingConfig
	}
	if cCopy.SocialConfig.GenerateMetadata || cCopy.SocialConfig.ExtractThumbnail || cCopy.SocialConfig.ThumbnailCount > 0 {
		aux.Social = &cCopy.SocialConfig
	}
	if cCopy.AIConfig.APIRouter != "" || cCopy.AIConfig.APIKey != "" || cCopy.AIConfig.Model != "" {
		aux.AIConfig = &cCopy.AIConfig
	}
	return json.Marshal(aux)
}

// UnmarshalJSON implements custom unmarshaling to support both structured child objects
// AND legacy flat config keys for 100% backward compatibility.
func (c *Config) UnmarshalJSON(data []byte) error {
	type Alias Config
	aux := &struct {
		// Polymorphic child configs
		RawShorts    json.RawMessage `json:"shorts"`
		RawSubtitles json.RawMessage `json:"subtitles"`
		RawSubtitle  json.RawMessage `json:"subtitle"`
		RawBurnSubs  json.RawMessage `json:"burn_subtitles"`
		RawBurnSub   json.RawMessage `json:"burn_subtitle"`
		RawAudio     json.RawMessage `json:"audio"`
		RawBranding  json.RawMessage `json:"branding"`
		RawSocial    json.RawMessage `json:"social"`

		// Flat legacy fields
		ShortsStyle      *string  `json:"shorts_style"`
		FaceTracking     *bool    `json:"face_tracking"`
		PanDuration      *float64 `json:"pan_duration"`
		SubStyle         *string  `json:"sub_style"`
		SubPreset        *string  `json:"sub_preset"`
		SubSDHMode       *string  `json:"sub_sdh_mode"`
		SubEmoji         *bool    `json:"sub_emoji"`
		SubFontSize      *int     `json:"sub_font_size"`
		SubFontPath      *string  `json:"sub_font_path"`
		TranslateLang    *string  `json:"translate_lang"`
		UseWhisper       *bool    `json:"use_whisper"`
		Loudnorm         *bool    `json:"loudnorm"`
		LoudnormI        *float64 `json:"loudnorm_i"`
		LoudnormLRA      *float64 `json:"loudnorm_lra"`
		LoudnormTP       *float64 `json:"loudnorm_tp"`
		JumpCut          *bool    `json:"jump_cut"`
		JumpCutMinSil    *float64 `json:"jump_cut_min_silence"`
		JumpCutMargin    *float64 `json:"jump_cut_margin"`
		JumpCutNoise     *float64 `json:"jump_cut_noise"`
		WatermarkPath    *string  `json:"watermark"`
		WatermarkPos     *string  `json:"watermark_pos"`
		OverlayText      *string  `json:"overlay_text"`
		TextPos          *string  `json:"text_pos"`
		FontSize         *int     `json:"font_size"`
		FontColor        *string  `json:"font_color"`
		GenerateMetadata *bool    `json:"generate_metadata"`
		ExtractThumbnail *bool    `json:"extract_thumbnail"`
		ThumbnailCount   *int     `json:"thumbnail_count"`

		RawAIConfig      json.RawMessage `json:"ai_config"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// 1. Shorts parsing
	if len(aux.RawShorts) > 0 {
		trimmed := strings.TrimSpace(string(aux.RawShorts))
		if trimmed == "true" || trimmed == "false" {
			c.Shorts = trimmed == "true"
			c.ShortsConfig.Enabled = c.Shorts
		} else if strings.HasPrefix(trimmed, "{") {
			var sc ShortsConfig
			if err := json.Unmarshal(aux.RawShorts, &sc); err == nil {
				c.ShortsConfig = sc
				c.Shorts = sc.Enabled
				c.ShortsStyle = sc.Style
				c.FaceTracking = sc.FaceTracking
				c.PanDuration = sc.PanDuration
			}
		}
	}
	if aux.ShortsStyle != nil {
		c.ShortsStyle = *aux.ShortsStyle
		c.ShortsConfig.Style = *aux.ShortsStyle
	}
	if aux.FaceTracking != nil {
		c.FaceTracking = *aux.FaceTracking
		c.ShortsConfig.FaceTracking = *aux.FaceTracking
	}
	if aux.PanDuration != nil {
		c.PanDuration = *aux.PanDuration
		c.ShortsConfig.PanDuration = *aux.PanDuration
	}

	// 2. Subtitles parsing
	subRaw := aux.RawSubtitles
	if len(subRaw) == 0 {
		subRaw = aux.RawSubtitle
	}
	if len(subRaw) == 0 {
		subRaw = aux.RawBurnSubs
	}
	if len(subRaw) == 0 {
		subRaw = aux.RawBurnSub
	}
	if len(subRaw) > 0 {
		trimmed := strings.TrimSpace(string(subRaw))
		if trimmed == "true" || trimmed == "false" {
			c.Subtitles = trimmed == "true"
			c.SubtitleConfig.Enabled = c.Subtitles
		} else if strings.HasPrefix(trimmed, "{") {
			var sc SubtitleConfig
			if err := json.Unmarshal(subRaw, &sc); err == nil {
				c.SubtitleConfig = sc
				c.Subtitles = sc.Enabled
				c.SubPreset = sc.Preset
				c.SubStyle = sc.Style
				c.SubSDHMode = sc.SDHMode
				c.SubEmoji = sc.Emoji
				c.SubFontSize = sc.FontSize
				c.SubFontPath = sc.FontPath
				c.TranslateLang = sc.TranslateLang
				c.UseWhisper = sc.UseWhisper
			}
		}
	}
	if aux.SubPreset != nil {
		c.SubPreset = *aux.SubPreset
		c.SubtitleConfig.Preset = *aux.SubPreset
	}
	if aux.SubStyle != nil {
		c.SubStyle = *aux.SubStyle
		c.SubtitleConfig.Style = *aux.SubStyle
	}
	if aux.SubSDHMode != nil {
		c.SubSDHMode = *aux.SubSDHMode
		c.SubtitleConfig.SDHMode = *aux.SubSDHMode
	}
	if aux.SubEmoji != nil {
		c.SubEmoji = *aux.SubEmoji
		c.SubtitleConfig.Emoji = *aux.SubEmoji
	}
	if aux.SubFontSize != nil {
		c.SubFontSize = *aux.SubFontSize
		c.SubtitleConfig.FontSize = *aux.SubFontSize
	}
	if aux.SubFontPath != nil {
		c.SubFontPath = *aux.SubFontPath
		c.SubtitleConfig.FontPath = *aux.SubFontPath
	}
	if aux.TranslateLang != nil {
		c.TranslateLang = *aux.TranslateLang
		c.SubtitleConfig.TranslateLang = *aux.TranslateLang
	}
	if aux.UseWhisper != nil {
		c.UseWhisper = *aux.UseWhisper
		c.SubtitleConfig.UseWhisper = *aux.UseWhisper
	}

	// 3. Audio parsing
	if len(aux.RawAudio) > 0 {
		var ac AudioConfig
		if err := json.Unmarshal(aux.RawAudio, &ac); err == nil {
			c.AudioConfig = ac
			c.Loudnorm = ac.Loudnorm.Enabled
			c.LoudnormI = ac.Loudnorm.I
			c.LoudnormLRA = ac.Loudnorm.LRA
			c.LoudnormTP = ac.Loudnorm.TP
			c.JumpCut = ac.JumpCut.Enabled
			c.JumpCutMinSil = ac.JumpCut.MinSilence
			c.JumpCutMargin = ac.JumpCut.Margin
			c.JumpCutNoise = ac.JumpCut.Noise
		}
	}
	if aux.Loudnorm != nil {
		c.Loudnorm = *aux.Loudnorm
		c.AudioConfig.Loudnorm.Enabled = *aux.Loudnorm
	}
	if aux.LoudnormI != nil {
		c.LoudnormI = *aux.LoudnormI
		c.AudioConfig.Loudnorm.I = *aux.LoudnormI
	}
	if aux.LoudnormLRA != nil {
		c.LoudnormLRA = *aux.LoudnormLRA
		c.AudioConfig.Loudnorm.LRA = *aux.LoudnormLRA
	}
	if aux.LoudnormTP != nil {
		c.LoudnormTP = *aux.LoudnormTP
		c.AudioConfig.Loudnorm.TP = *aux.LoudnormTP
	}
	if aux.JumpCut != nil {
		c.JumpCut = *aux.JumpCut
		c.AudioConfig.JumpCut.Enabled = *aux.JumpCut
	}
	if aux.JumpCutMinSil != nil {
		c.JumpCutMinSil = *aux.JumpCutMinSil
		c.AudioConfig.JumpCut.MinSilence = *aux.JumpCutMinSil
	}
	if aux.JumpCutMargin != nil {
		c.JumpCutMargin = *aux.JumpCutMargin
		c.AudioConfig.JumpCut.Margin = *aux.JumpCutMargin
	}
	if aux.JumpCutNoise != nil {
		c.JumpCutNoise = *aux.JumpCutNoise
		c.AudioConfig.JumpCut.Noise = *aux.JumpCutNoise
	}

	// 4. Branding parsing
	if len(aux.RawBranding) > 0 {
		var bc BrandingConfig
		if err := json.Unmarshal(aux.RawBranding, &bc); err == nil {
			c.BrandingConfig = bc
			c.WatermarkPath = bc.Watermark.Path
			c.WatermarkPos = bc.Watermark.Position
			c.OverlayText = bc.OverlayText.Text
			c.TextPos = bc.OverlayText.Position
			c.FontSize = bc.OverlayText.FontSize
			c.FontColor = bc.OverlayText.FontColor
		}
	}
	if aux.WatermarkPath != nil {
		c.WatermarkPath = *aux.WatermarkPath
		c.BrandingConfig.Watermark.Path = *aux.WatermarkPath
	}
	if aux.WatermarkPos != nil {
		c.WatermarkPos = *aux.WatermarkPos
		c.BrandingConfig.Watermark.Position = *aux.WatermarkPos
	}
	if aux.OverlayText != nil {
		c.OverlayText = *aux.OverlayText
		c.BrandingConfig.OverlayText.Text = *aux.OverlayText
	}
	if aux.TextPos != nil {
		c.TextPos = *aux.TextPos
		c.BrandingConfig.OverlayText.Position = *aux.TextPos
	}
	if aux.FontSize != nil {
		c.FontSize = *aux.FontSize
		c.BrandingConfig.OverlayText.FontSize = *aux.FontSize
	}
	if aux.FontColor != nil {
		c.FontColor = *aux.FontColor
		c.BrandingConfig.OverlayText.FontColor = *aux.FontColor
	}

	// 5. Social parsing
	if len(aux.RawSocial) > 0 {
		var sc SocialConfig
		if err := json.Unmarshal(aux.RawSocial, &sc); err == nil {
			c.SocialConfig = sc
			c.GenerateMetadata = sc.GenerateMetadata
			c.ExtractThumbnail = sc.ExtractThumbnail
			c.ThumbnailCount = sc.ThumbnailCount
		}
	}
	if aux.GenerateMetadata != nil {
		c.GenerateMetadata = *aux.GenerateMetadata
		c.SocialConfig.GenerateMetadata = *aux.GenerateMetadata
	}
	if aux.ExtractThumbnail != nil {
		c.ExtractThumbnail = *aux.ExtractThumbnail
		c.SocialConfig.ExtractThumbnail = *aux.ExtractThumbnail
	}
	if aux.ThumbnailCount != nil {
		c.ThumbnailCount = *aux.ThumbnailCount
		c.SocialConfig.ThumbnailCount = *aux.ThumbnailCount
	}

	// 6. Handle ai_config if provided as array []AIProfile
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

	c.Sync()
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
	c.Sync()
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
