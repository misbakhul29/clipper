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

func TestFindMatchingSubtitleFile_AvoidFalsePositiveSubstrings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "clipper_sub_lang_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create sub_xid1sE8lEec.ru.vtt (Russian subtitle file for video ID containing 'id')
	ruPath := filepath.Join(tempDir, "sub_xid1sE8lEec.ru.vtt")
	if err := os.WriteFile(ruPath, []byte("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nПривет"), 0644); err != nil {
		t.Fatalf("failed to write ru vtt: %v", err)
	}

	// 1. Searching for 'id' should NOT match sub_xid1sE8lEec.ru.vtt
	matchID := findMatchingSubtitleFile(tempDir, "id")
	if matchID != "" {
		t.Errorf("findMatchingSubtitleFile(lang='id') returned %q; want empty string because file is Russian (.ru.vtt)", matchID)
	}

	// 2. Create sub_xid1sE8lEec.id.vtt (Indonesian subtitle file)
	idPath := filepath.Join(tempDir, "sub_xid1sE8lEec.id.vtt")
	if err := os.WriteFile(idPath, []byte("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nHalo"), 0644); err != nil {
		t.Fatalf("failed to write id vtt: %v", err)
	}

	// 3. Searching for 'id' should now match sub_xid1sE8lEec.id.vtt
	matchID = findMatchingSubtitleFile(tempDir, "id")
	if matchID != idPath {
		t.Errorf("findMatchingSubtitleFile(lang='id') = %q; want %q", matchID, idPath)
	}
}

func TestFindAnySubtitleFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "clipper_any_sub_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Empty dir returns empty string
	if match := findAnySubtitleFile(tempDir); match != "" {
		t.Errorf("findAnySubtitleFile on empty dir returned %q; want empty string", match)
	}

	// 2. Create Japanese and Korean-orig subtitle files
	jaPath := filepath.Join(tempDir, "sub_sample.ja.vtt")
	_ = os.WriteFile(jaPath, []byte("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nKonnichiwa"), 0644)

	koOrigPath := filepath.Join(tempDir, "sub_sample.ko-orig.vtt")
	_ = os.WriteFile(koOrigPath, []byte("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nAnnyeong"), 0644)

	// Should prefer orig
	match := findAnySubtitleFile(tempDir)
	if match != koOrigPath {
		t.Errorf("findAnySubtitleFile = %q; want orig match %q", match, koOrigPath)
	}
}

