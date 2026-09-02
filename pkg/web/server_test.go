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

	t.Run("POST API Clip Validation", func(t *testing.T) {
		// Empty payload should fail with 400 Bad Request
		payload := []byte(`{"input_file":""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/clip", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for empty payload, got %d", rec.Code)
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
}
