package downloader

import (
	"path/filepath"
	"testing"
)

func TestExtractVideoID(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://www.youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=30s", "dQw4w9WgXcQ"},
	}

	for _, tt := range tests {
		got := ExtractVideoID(tt.url)
		if got != tt.expected {
			t.Errorf("ExtractVideoID(%q) = %q; want %q", tt.url, got, tt.expected)
		}
	}
}

func TestGetVideoCacheDir(t *testing.T) {
	baseDir := "/tmp/test_cache"

	t.Run("YouTube URL gets video ID subfolder", func(t *testing.T) {
		got := GetVideoCacheDir(baseDir, "https://www.youtube.com/watch?v=video123")
		expected := filepath.Join(baseDir, "video123")
		if got != expected {
			t.Errorf("GetVideoCacheDir YouTube = %q; want %q", got, expected)
		}
	})

	t.Run("Local video file gets sanitized basename subfolder", func(t *testing.T) {
		got := GetVideoCacheDir(baseDir, "/path/to/my_awesome_video.mp4")
		expected := filepath.Join(baseDir, "my_awesome_video")
		if got != expected {
			t.Errorf("GetVideoCacheDir Local = %q; want %q", got, expected)
		}
	})
}
