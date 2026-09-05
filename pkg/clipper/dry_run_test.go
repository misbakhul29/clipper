package clipper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetBatchInputs(t *testing.T) {
	t.Run("Comma separated inputs", func(t *testing.T) {
		cfg := Config{
			InputFile: "url1.mp4, url2.mp4 , url3.mp4",
		}
		inputs := cfg.GetBatchInputs()
		if len(inputs) != 3 {
			t.Fatalf("GetBatchInputs length = %d; want 3", len(inputs))
		}
		if inputs[0] != "url1.mp4" || inputs[1] != "url2.mp4" || inputs[2] != "url3.mp4" {
			t.Errorf("GetBatchInputs = %v; want [url1.mp4 url2.mp4 url3.mp4]", inputs)
		}
	})

	t.Run("Batch list file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "clipper_batch_test_*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		batchFile := filepath.Join(tempDir, "list.txt")
		content := "# Comment line\nhttps://youtube.com/watch?v=111\nhttps://youtube.com/watch?v=222\n"
		if err := os.WriteFile(batchFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write batch file: %v", err)
		}

		cfg := Config{
			BatchList: batchFile,
		}
		inputs := cfg.GetBatchInputs()
		if len(inputs) != 2 {
			t.Fatalf("GetBatchInputs batch length = %d; want 2", len(inputs))
		}
		if inputs[0] != "https://youtube.com/watch?v=111" || inputs[1] != "https://youtube.com/watch?v=222" {
			t.Errorf("GetBatchInputs = %v", inputs)
		}
	})
}

func TestDryRunExecution(t *testing.T) {
	app, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "clipper_dryrun_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockVideo := filepath.Join(tempDir, "test.mp4")
	if err := os.WriteFile(mockVideo, []byte("fake video header"), 0644); err != nil {
		t.Fatalf("failed to write mock video: %v", err)
	}

	cfg := Config{
		InputFile: mockVideo,
		DryRun:    true,
		Segments: []Segment{
			{Start: "00:00:10", End: "00:00:20", Title: "Segment 1"},
			{Start: "00:00:30", End: "00:00:40", Title: "Segment 2"},
		},
		OutputDir: tempDir,
	}

	err = app.Process(&cfg)
	if err != nil {
		t.Fatalf("Process(DryRun=true) failed: %v", err)
	}

	// Verify no output clips were actually created (since dry-run mode skips FFmpeg execution)
	clips, _ := filepath.Glob(filepath.Join(tempDir, "*Segment*.mp4"))
	if len(clips) > 0 {
		t.Errorf("DryRun should not generate output clip files, found: %v", clips)
	}
}

func TestConfigSubtitlesJSONParsing(t *testing.T) {
	t.Run("subtitles true", func(t *testing.T) {
		jsonData := []byte(`{"input_file":"video.mp4","subtitles":true}`)
		var cfg Config
		if err := json.Unmarshal(jsonData, &cfg); err != nil {
			t.Fatalf("failed unmarshaling: %v", err)
		}
		if !cfg.Subtitles {
			t.Errorf("expected Subtitles to be true, got false")
		}
	})

	t.Run("subtitles false", func(t *testing.T) {
		jsonData := []byte(`{"input_file":"video.mp4","subtitles":false}`)
		var cfg Config
		if err := json.Unmarshal(jsonData, &cfg); err != nil {
			t.Fatalf("failed unmarshaling: %v", err)
		}
		if cfg.Subtitles {
			t.Errorf("expected Subtitles to be false, got true")
		}
	})

	t.Run("burn_subtitles true (legacy alias)", func(t *testing.T) {
		jsonData := []byte(`{"input_file":"video.mp4","burn_subtitles":true}`)
		var cfg Config
		if err := json.Unmarshal(jsonData, &cfg); err != nil {
			t.Fatalf("failed unmarshaling: %v", err)
		}
		if !cfg.Subtitles {
			t.Errorf("expected Subtitles to be true, got false")
		}
	})
}

func TestCustomSegmentSubtitlesJSON(t *testing.T) {
	jsonData := []byte(`{
		"input_file": "video.mp4",
		"subtitles": true,
		"segments": [
			{
				"start": "00:00:10",
				"end": "00:00:30",
				"title": "Custom Cue Segment",
				"sub_position": "middle",
				"sub_preset": "neon",
				"sub_font_size": 52,
				"subtitles": [
					{"start": 0.5, "end": 2.5, "text": "Hello world"},
					{"start": 2.8, "end": 5.0, "text": "This is custom subtitle"}
				]
			}
		]
	}`)

	var cfg Config
	if err := json.Unmarshal(jsonData, &cfg); err != nil {
		t.Fatalf("failed unmarshaling config: %v", err)
	}

	if len(cfg.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(cfg.Segments))
	}
	seg := cfg.Segments[0]
	if seg.SubPosition != "middle" || seg.SubPreset != "neon" || seg.SubFontSize != 52 {
		t.Errorf("segment style overrides not matching: %+v", seg)
	}
	if len(seg.Subtitles) != 2 {
		t.Fatalf("expected 2 custom subtitle cues, got %d", len(seg.Subtitles))
	}
	if seg.Subtitles[0].Text != "Hello world" || seg.Subtitles[0].Start != 0.5 {
		t.Errorf("unexpected cue 0: %+v", seg.Subtitles[0])
	}
}

func TestDefaultAndSampleConfig(t *testing.T) {
	def := DefaultConfig()
	if def.OutputDir != "./clips" || def.Quality != "1080p" || !def.Shorts || def.ShortsStyle != "blur" {
		t.Errorf("DefaultConfig invalid defaults: %+v", def)
	}

	sample := NewSampleConfig()
	if sample.InputFile == "" {
		t.Error("NewSampleConfig should have InputFile set")
	}
	if len(sample.AIConfigs) != 3 {
		t.Errorf("expected 3 sample AI accounts, got %d", len(sample.AIConfigs))
	}
	if sample.RoutingModels.Segment != "gemini_acc_1" {
		t.Errorf("expected segment routing to gemini_acc_1, got %s", sample.RoutingModels.Segment)
	}
	if len(sample.Segments) != 2 {
		t.Errorf("expected 2 sample segments, got %d", len(sample.Segments))
	}

	// Validate sample config
	if err := sample.Validate(); err != nil {
		t.Fatalf("NewSampleConfig failed Validate(): %v", err)
	}

	// Verify MarshalJSON produces structured child keys
	marshaled, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("failed to marshal sample config: %v", err)
	}
	str := string(marshaled)
	if !strings.Contains(str, `"subtitles"`) || !strings.Contains(str, `"audio"`) || !strings.Contains(str, `"shorts"`) {
		t.Errorf("expected structured child keys in marshaled JSON, got: %s", str)
	}
}

func TestStructuredAndLegacyConfigParsing(t *testing.T) {
	t.Run("Parse Structured Nested JSON", func(t *testing.T) {
		jsonData := []byte(`{
			"input": "test.mp4",
			"cache": {
				"dir": "/custom/cache",
				"no_cache": true,
				"clean": false,
				"clean_days": 5
			},
			"shorts": {
				"enabled": true,
				"style": "smart-crop",
				"face_tracking": true,
				"pan_duration": 1.2
			},
			"subtitles": {
				"enabled": true,
				"preset": "neon",
				"sdh_mode": "top-box",
				"font_size": 56,
				"translate_lang": "en"
			},
			"audio": {
				"loudnorm": {
					"enabled": true,
					"i": -16,
					"lra": 8,
					"tp": -1.5
				},
				"jump_cut": {
					"enabled": true,
					"min_silence": 0.8,
					"margin": 0.15,
					"noise": -28.0
				}
			},
			"branding": {
				"watermark": {
					"path": "logo.png",
					"position": "bottom-right"
				},
				"overlay_text": {
					"text": "My Channel",
					"position": "top-left",
					"font_size": 40,
					"font_color": "yellow"
				}
			},
			"social": {
				"generate_metadata": true,
				"extract_thumbnail": true,
				"thumbnail_count": 2
			},
			"segments": [
				{"start": "0", "end": "10"}
			]
		}`)

		var cfg Config
		if err := json.Unmarshal(jsonData, &cfg); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if cfg.CacheDir != "/custom/cache" || !cfg.NoCache || cfg.CleanDays != 5 {
			t.Errorf("Cache syncing failed: %+v", cfg.CacheConfig)
		}
		if !cfg.Shorts || cfg.ShortsStyle != "smart-crop" || !cfg.FaceTracking || cfg.PanDuration != 1.2 {
			t.Errorf("Shorts syncing failed: %+v", cfg.ShortsConfig)
		}
		if !cfg.Subtitles || cfg.SubPreset != "neon" || cfg.SubSDHMode != "top-box" || cfg.SubFontSize != 56 || cfg.TranslateLang != "en" {
			t.Errorf("Subtitles syncing failed: %+v", cfg.SubtitleConfig)
		}
		if !cfg.Loudnorm || cfg.LoudnormI != -16 || cfg.LoudnormLRA != 8 || cfg.LoudnormTP != -1.5 {
			t.Errorf("Loudnorm syncing failed: %+v", cfg.AudioConfig.Loudnorm)
		}
		if !cfg.JumpCut || cfg.JumpCutMinSil != 0.8 || cfg.JumpCutMargin != 0.15 || cfg.JumpCutNoise != -28.0 {
			t.Errorf("JumpCut syncing failed: %+v", cfg.AudioConfig.JumpCut)
		}
		if cfg.WatermarkPath != "logo.png" || cfg.WatermarkPos != "bottom-right" {
			t.Errorf("Watermark syncing failed: %+v", cfg.BrandingConfig.Watermark)
		}
		if cfg.OverlayText != "My Channel" || cfg.TextPos != "top-left" || cfg.FontSize != 40 || cfg.FontColor != "yellow" {
			t.Errorf("OverlayText syncing failed: %+v", cfg.BrandingConfig.OverlayText)
		}
		if !cfg.GenerateMetadata || !cfg.ExtractThumbnail || cfg.ThumbnailCount != 2 {
			t.Errorf("Social syncing failed: %+v", cfg.SocialConfig)
		}
	})

	t.Run("Parse Flat Legacy JSON", func(t *testing.T) {
		jsonData := []byte(`{
			"input": "legacy.mp4",
			"cache_dir": "/legacy/cache",
			"no_cache": true,
			"clean_days": 10,
			"shorts": true,
			"shorts_style": "blur",
			"subtitles": true,
			"sub_preset": "minimal",
			"sub_font_size": 42,
			"loudnorm": true,
			"loudnorm_i": -15,
			"jump_cut": true,
			"jump_cut_min_silence": 1.2,
			"watermark": "brand.png",
			"overlay_text": "Follow for more",
			"generate_metadata": true,
			"extract_thumbnail": true,
			"thumbnail_count": 3,
			"segments": [
				{"start": "5", "end": "25"}
			]
		}`)

		var cfg Config
		if err := json.Unmarshal(jsonData, &cfg); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if cfg.CacheConfig.Dir != "/legacy/cache" || !cfg.CacheConfig.NoCache || cfg.CacheConfig.CleanDays != 10 {
			t.Errorf("CacheConfig flat sync failed: %+v", cfg.CacheConfig)
		}
		if !cfg.ShortsConfig.Enabled || cfg.ShortsConfig.Style != "blur" {
			t.Errorf("ShortsConfig flat sync failed: %+v", cfg.ShortsConfig)
		}
		if !cfg.SubtitleConfig.Enabled || cfg.SubtitleConfig.Preset != "minimal" || cfg.SubtitleConfig.FontSize != 42 {
			t.Errorf("SubtitleConfig flat sync failed: %+v", cfg.SubtitleConfig)
		}
		if !cfg.AudioConfig.Loudnorm.Enabled || cfg.AudioConfig.Loudnorm.I != -15 {
			t.Errorf("LoudnormConfig flat sync failed: %+v", cfg.AudioConfig.Loudnorm)
		}
		if !cfg.AudioConfig.JumpCut.Enabled || cfg.AudioConfig.JumpCut.MinSilence != 1.2 {
			t.Errorf("JumpCutConfig flat sync failed: %+v", cfg.AudioConfig.JumpCut)
		}
		if cfg.BrandingConfig.Watermark.Path != "brand.png" || cfg.BrandingConfig.OverlayText.Text != "Follow for more" {
			t.Errorf("BrandingConfig flat sync failed: %+v", cfg.BrandingConfig)
		}
		if !cfg.SocialConfig.GenerateMetadata || !cfg.SocialConfig.ExtractThumbnail || cfg.SocialConfig.ThumbnailCount != 3 {
			t.Errorf("SocialConfig flat sync failed: %+v", cfg.SocialConfig)
		}
	})
}
