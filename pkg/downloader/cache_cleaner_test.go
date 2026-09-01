package downloader

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestCleanCache(t *testing.T) {
	tempCacheDir, err := os.MkdirTemp("", "clipper_clean_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempCacheDir)

	// Create test files
	file1 := filepath.Join(tempCacheDir, "old_file.mp4")
	file2 := filepath.Join(tempCacheDir, "new_file.mp4")

	if err := os.WriteFile(file1, []byte("old content 1234567890"), 0644); err != nil {
		t.Fatalf("failed to write old file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("new content 1234567890"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}

	// Change modtime of old_file to 10 days ago
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	_ = os.Chtimes(file1, oldTime, oldTime)

	t.Run("Clean older than 5 days removes old_file only", func(t *testing.T) {
		freed, count, err := CleanCache(tempCacheDir, 5)
		if err != nil {
			t.Fatalf("CleanCache error: %v", err)
		}
		if count != 1 {
			t.Errorf("CleanCache count = %d; want 1", count)
		}
		if freed <= 0 {
			t.Errorf("CleanCache freed = %d; want > 0", freed)
		}
		if fileExists(file1) {
			t.Errorf("old_file.mp4 should have been deleted")
		}
		if !fileExists(file2) {
			t.Errorf("new_file.mp4 should still exist")
		}
	})

	t.Run("Clean all (days=0) removes remaining files", func(t *testing.T) {
		freed, count, err := CleanCache(tempCacheDir, 0)
		if err != nil {
			t.Fatalf("CleanCache error: %v", err)
		}
		if count != 1 {
			t.Errorf("CleanCache count = %d; want 1", count)
		}
		if freed <= 0 {
			t.Errorf("CleanCache freed = %d; want > 0", freed)
		}
		if fileExists(file2) {
			t.Errorf("new_file.mp4 should have been deleted")
		}
	})
}
