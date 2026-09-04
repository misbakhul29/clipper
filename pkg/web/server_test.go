package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/misbakhul29/clipper/pkg/clipper"
)

func TestWebServerEndpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipper_web_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srv := NewServer(":9999", &clipper.Config{
		OutputDir: tmpDir,
	})
	handler := srv.Router()

	t.Run("GET Index HTML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			t.Errorf("expected Content-Type text/html, got %s", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("GET styles.css", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/styles.css", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("GET app.js", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("GET API Status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var st statusResponse
		if err := json.NewDecoder(rec.Body).Decode(&st); err != nil {
			t.Fatalf("failed decoding status response: %v", err)
		}
		if st.IsRunning {
			t.Errorf("expected is_running = false")
		}
	})

	t.Run("GET API Clips", func(t *testing.T) {
		// Create a dummy video file and thumbnail in tmpDir
		dummyVideo := filepath.Join(tmpDir, "sample_clip_001.mp4")
		dummyThumb := filepath.Join(tmpDir, "sample_clip_001.jpg")
		_ = os.WriteFile(dummyVideo, []byte("fake video content"), 0644)
		_ = os.WriteFile(dummyThumb, []byte("fake thumbnail"), 0644)

		req := httptest.NewRequest(http.MethodGet, "/api/clips", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var clips []ClipItem
		if err := json.NewDecoder(rec.Body).Decode(&clips); err != nil {
			t.Fatalf("failed decoding clips: %v", err)
		}
		if len(clips) != 1 {
			t.Fatalf("expected 1 clip, got %d", len(clips))
		}
		if clips[0].Name != "sample_clip_001.mp4" {
			t.Errorf("expected sample_clip_001.mp4, got %s", clips[0].Name)
		}
		if clips[0].ThumbnailURL == "" {
			t.Errorf("expected non-empty thumbnail URL")
		}
	})

	t.Run("POST API Clip and Render Validation", func(t *testing.T) {
		// Empty payload on /api/clip should fail with 400 Bad Request
		payload := []byte(`{"input_file":""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/clip", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for empty payload on /api/clip, got %d", rec.Code)
		}

		// Empty payload on /api/render should also fail with 400 Bad Request
		reqRender := httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(payload))
		recRender := httptest.NewRecorder()
		handler.ServeHTTP(recRender, reqRender)

		if recRender.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for empty payload on /api/render, got %d", recRender.Code)
		}
	})

	t.Run("POST API Prepare", func(t *testing.T) {
		// Empty source fails with 400
		req := httptest.NewRequest(http.MethodPost, "/api/prepare", bytes.NewReader([]byte(`{"source":""}`)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for empty source, got %d", rec.Code)
		}

		// Valid existing file returns ready
		sampleFile := filepath.Join(tmpDir, "sample_clip_001.mp4")
		req2 := httptest.NewRequest(http.MethodPost, "/api/prepare", bytes.NewReader([]byte(`{"source":"`+sampleFile+`"}`)))
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusOK {
			t.Errorf("expected status 200 for existing file, got %d: %s", rec2.Code, rec2.Body.String())
		}
		var prep prepareResponse
		if err := json.NewDecoder(rec2.Body).Decode(&prep); err != nil {
			t.Fatalf("failed decoding prepareResponse: %v", err)
		}
		if prep.Status != "ready" || !strings.Contains(prep.PreviewURL, "/preview?path=") {
			t.Errorf("unexpected prepare response: %+v", prep)
		}
	})

	t.Run("POST API Auto-Detect Validation", func(t *testing.T) {
		// Empty input fails with 400
		req := httptest.NewRequest(http.MethodPost, "/api/auto-detect", bytes.NewReader([]byte(`{"input_file":""}`)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for empty input_file, got %d", rec.Code)
		}
	})

	t.Run("POST API Clip with AutoDetect and 0 segments", func(t *testing.T) {
		sampleFile := filepath.Join(tmpDir, "sample_clip_001.mp4")
		payload := []byte(`{"input_file":"` + sampleFile + `","auto_detect":"silence","segments":[]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/clip", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Should not fail with 400 "at least 1 segment required"
		if rec.Code == http.StatusBadRequest && strings.Contains(rec.Body.String(), "at least 1 segment") {
			t.Errorf("expected auto_detect to be valid without manual segments, got: %s", rec.Body.String())
		}
	})

	t.Run("POST API Auto-Detect AI Mode Without Subtitles", func(t *testing.T) {
		sampleFile := filepath.Join(tmpDir, "sample_clip_001.mp4")
		payload := []byte(`{"input_file":"` + sampleFile + `","mode":"ai"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/auto-detect", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp autoDetectResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed decoding autoDetectResponse: %v", err)
		}
		if len(resp.Segments) == 0 {
			t.Errorf("expected at least 1 detected segment, got 0")
		}
	})

	t.Run("POST API Transcribe Validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/transcribe", bytes.NewReader([]byte(`{}`)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for empty input_file, got %d", rec.Code)
		}
	})

	t.Run("POST API AI Subtitles Emojis", func(t *testing.T) {
		payload := []byte(`{"action":"emojis","cues":[{"start":0.0,"end":2.0,"text":"This video is fire and money"}]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/ai/subtitles", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Cues []clipper.SubtitleCue `json:"cues"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed decoding: %v", err)
		}
		if len(resp.Cues) == 0 {
			t.Fatalf("expected at least 1 cue returned")
		}
		if !strings.Contains(resp.Cues[0].Text, "🔥") && !strings.Contains(resp.Cues[0].Text, "💰") {
			t.Errorf("expected emoji in text, got %s", resp.Cues[0].Text)
		}
	})

	t.Run("GET API Storage Stats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/storage/stats", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var stats StorageStatsResponse
		if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
			t.Fatalf("failed decoding storage stats: %v", err)
		}
		if stats.ClipsCount < 1 {
			t.Errorf("expected clips_count >= 1, got %d", stats.ClipsCount)
		}
		if stats.ClipsSizeBytes <= 0 {
			t.Errorf("expected clips_size_bytes > 0, got %d", stats.ClipsSizeBytes)
		}
	})

	t.Run("DELETE API Clip (Single File)", func(t *testing.T) {
		// Missing parameter fails
		reqEmpty := httptest.NewRequest(http.MethodDelete, "/api/clips", nil)
		recEmpty := httptest.NewRecorder()
		handler.ServeHTTP(recEmpty, reqEmpty)
		if recEmpty.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for missing name param, got %d", recEmpty.Code)
		}

		// Delete existing clip
		reqDel := httptest.NewRequest(http.MethodDelete, "/api/clips?name=sample_clip_001.mp4", nil)
		recDel := httptest.NewRecorder()
		handler.ServeHTTP(recDel, reqDel)
		if recDel.Code != http.StatusOK {
			t.Errorf("expected status 200 for clip deletion, got %d: %s", recDel.Code, recDel.Body.String())
		}

		// Verify file and thumb are gone
		if _, err := os.Stat(filepath.Join(tmpDir, "sample_clip_001.mp4")); !os.IsNotExist(err) {
			t.Errorf("expected sample_clip_001.mp4 to be deleted")
		}
		if _, err := os.Stat(filepath.Join(tmpDir, "sample_clip_001.jpg")); !os.IsNotExist(err) {
			t.Errorf("expected sample_clip_001.jpg thumbnail to be deleted")
		}

		// Deleting non-existent clip returns 404
		req404 := httptest.NewRequest(http.MethodDelete, "/api/clips?name=sample_clip_001.mp4", nil)
		rec404 := httptest.NewRecorder()
		handler.ServeHTTP(rec404, req404)
		if rec404.Code != http.StatusNotFound {
			t.Errorf("expected status 404 for missing clip, got %d", rec404.Code)
		}
	})

	t.Run("POST API Clean Cache and Clips", func(t *testing.T) {
		// Clean Cache
		reqCache := httptest.NewRequest(http.MethodPost, "/api/storage/clean-cache", nil)
		recCache := httptest.NewRecorder()
		handler.ServeHTTP(recCache, reqCache)
		if recCache.Code != http.StatusOK {
			t.Errorf("expected status 200 for clean cache, got %d: %s", recCache.Code, recCache.Body.String())
		}

		// Clean Clips
		reqClips := httptest.NewRequest(http.MethodPost, "/api/storage/clean-clips", nil)
		recClips := httptest.NewRecorder()
		handler.ServeHTTP(recClips, reqClips)
		if recClips.Code != http.StatusOK {
			t.Errorf("expected status 200 for clean clips, got %d: %s", recClips.Code, recClips.Body.String())
		}
	})
}
