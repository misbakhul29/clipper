package clipper

import (
	"fmt"
	"runtime"
)

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
	InputFile     string      `json:"input"`
	OutputDir     string      `json:"output_dir"`
	OutputFile    string      `json:"output"`       // Used for merge mode or prefix
	Mode          Mode        `json:"mode"`         // "split" or "merge"
	Strategy      CutStrategy `json:"strategy"`     // "fast" or "accurate"
	Shorts        bool        `json:"shorts"`       // Convert to 9:16 Shorts format
	ShortsStyle   string      `json:"shorts_style"` // "crop" (center 9:16) or "blur" (blurred background 9:16)
	Quality       string      `json:"quality"`      // YouTube download quality e.g. "best", "1080p", "720p", "480p", "360p", "worst"
	CacheDir      string      `json:"cache_dir"`    // Directory to cache downloaded YouTube videos (default: "./cache")
	NoCache       bool        `json:"no_cache"`     // Disable YouTube download cache and force re-download
	Concurrency   int         `json:"concurrency"`  // Number of parallel workers for rendering clips (default: NumCPU)
	WatermarkPath string      `json:"watermark"`    // Path to watermark image (PNG)
	WatermarkPos  string      `json:"watermark_pos"`// Position: top-left, top-right, bottom-left, bottom-right, center
	OverlayText   string      `json:"overlay_text"` // Text caption to render on video
	TextPos       string      `json:"text_pos"`     // Position for overlay text
	FontSize      int         `json:"font_size"`    // Font size for overlay text
	FontColor     string      `json:"font_color"`   // Font color for overlay text (e.g. "white", "yellow")
	AutoDetect    string      `json:"auto_detect"`  // Auto detection mode: "silence" or "scene"
	Segments      []Segment   `json:"segments"`
}

// Validate checks the configuration for missing or invalid parameters.
func (c *Config) Validate() error {
	if c.InputFile == "" {
		return fmt.Errorf("input video file path is required")
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
	if c.ShortsStyle != "crop" && c.ShortsStyle != "blur" {
		return fmt.Errorf("invalid shorts_style '%s', must be 'crop' or 'blur'", c.ShortsStyle)
	}
	if c.Quality == "" {
		c.Quality = "best"
	}
	if c.CacheDir == "" {
		c.CacheDir = "./cache"
	}
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
	if c.FontColor == "" {
		c.FontColor = "white"
	}
	return nil
}
