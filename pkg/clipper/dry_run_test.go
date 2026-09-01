package clipper

import (
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
