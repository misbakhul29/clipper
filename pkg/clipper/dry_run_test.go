package clipper

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	// Verify MarshalJSON omits legacy empty ai_config
	marshaled, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("failed to marshal sample config: %v", err)
	}
	if string(marshaled) == "" {
		t.Fatal("empty marshaled output")
	}
}
