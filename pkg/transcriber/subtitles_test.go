package transcriber

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSubtitleCacheIsolation(t *testing.T) {
	tempCacheDir, err := os.MkdirTemp("", "clipper_sub_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempCacheDir)

	video1URL := "https://www.youtube.com/watch?v=video_one_111"
	video2URL := "https://www.youtube.com/watch?v=video_two_222"

	// Create mock subtitle file for Video 1 in its isolated folder
	v1Dir := filepath.Join(tempCacheDir, "video_one_111")
	if err := os.MkdirAll(v1Dir, 0755); err != nil {
		t.Fatalf("failed to create v1 dir: %v", err)
	}
	v1VttPath := filepath.Join(v1Dir, "sub_video_one_111.id.vtt")
	v1VttContent := `WEBVTT

00:00:01.000 --> 00:00:05.000
Subtitle content for Video ONE`
	if err := os.WriteFile(v1VttPath, []byte(v1VttContent), 0644); err != nil {
		t.Fatalf("failed to write v1 vtt: %v", err)
	}

	// 1. FetchSubtitles for Video 1 should return Video 1 subtitles
	subs1, err := FetchSubtitles(video1URL, tempCacheDir, "id")
	if err != nil {
		t.Fatalf("FetchSubtitles(video1) returned error: %v", err)
	}
	if len(subs1) == 0 || subs1[0].Text != "Subtitle content for Video ONE" {
		t.Errorf("FetchSubtitles(video1) = %v; want 'Subtitle content for Video ONE'", subs1)
	}

	// 2. FetchSubtitles for Video 2 MUST NOT pick up Video 1 subtitles
	v2Dir := filepath.Join(tempCacheDir, "video_two_222")
	// Verify v2Dir does not have subtitles yet
	if _, err := os.Stat(filepath.Join(v2Dir, "sub_video_one_111.id.vtt")); !os.IsNotExist(err) {
		t.Errorf("Video 2 cache directory should not contain Video 1 subtitle file")
	}

	// Attempting to fetch subtitles for video2 without yt-dlp binary in mock env will fail to download,
	// but it must NOT return video1's subtitles!
	subs2, _ := FetchSubtitles(video2URL, tempCacheDir, "id")
	if len(subs2) > 0 && subs2[0].Text == "Subtitle content for Video ONE" {
		t.Errorf("FetchSubtitles(video2) leaked Video 1's subtitles!")
	}
}
