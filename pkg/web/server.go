package web

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/misbakhul29/clipper/pkg/ai"
	"github.com/misbakhul29/clipper/pkg/clipper"
	"github.com/misbakhul29/clipper/pkg/downloader"
	"github.com/misbakhul29/clipper/pkg/transcriber"
)

func init() {
	_ = mime.AddExtensionType(".mp4", "video/mp4")
	_ = mime.AddExtensionType(".m4v", "video/mp4")
	_ = mime.AddExtensionType(".webm", "video/webm")
	_ = mime.AddExtensionType(".mov", "video/quicktime")
	_ = mime.AddExtensionType(".mkv", "video/x-matroska")
	_ = mime.AddExtensionType(".ts", "video/mp2t")
	_ = mime.AddExtensionType(".avi", "video/x-msvideo")
	_ = mime.AddExtensionType(".ogg", "video/ogg")
	_ = mime.AddExtensionType(".ogv", "video/ogg")
	_ = mime.AddExtensionType(".jpg", "image/jpeg")
	_ = mime.AddExtensionType(".jpeg", "image/jpeg")
	_ = mime.AddExtensionType(".png", "image/png")
	_ = mime.AddExtensionType(".webp", "image/webp")
}

// Server handles the local web UI dashboard and REST API.
type Server struct {
	Addr          string
	OutDir        string
	DefaultConfig *clipper.Config
	mu            sync.RWMutex
	isRunning     bool
	currentTask   string
	progressPct   int
	completed     bool
	lastError     string
}

// NewServer initializes a new Server instance.
func NewServer(addr string, defaultCfg *clipper.Config) *Server {
	if addr == "" {
		addr = ":8080"
	}
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}
	outDir := "./clips"
	if defaultCfg != nil && defaultCfg.OutputDir != "" && defaultCfg.OutputDir != "." {
		outDir = defaultCfg.OutputDir
	}
	return &Server{
		Addr:          addr,
		OutDir:        outDir,
		DefaultConfig: defaultCfg,
	}
}

// Router configures the HTTP multiplexer.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Serve static files: prioritize live disk ./pkg/web/static for instant dev reload, fallback to embed.FS
	var staticHandler http.Handler
	if _, err := os.Stat("./pkg/web/static"); err == nil {
		staticHandler = http.FileServer(http.Dir("./pkg/web/static"))
	} else if sub, err := fs.Sub(StaticFS, "static"); err == nil {
		staticHandler = http.FileServer(http.FS(sub))
	}

	if staticHandler != nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Prevent browser caching during local development
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			staticHandler.ServeHTTP(w, r)
		})
	}

	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/clips", s.handleClips)
	mux.HandleFunc("/api/storage/stats", s.handleStorageStats)
	mux.HandleFunc("/api/storage/clean-cache", s.handleCleanCache)
	mux.HandleFunc("/api/storage/clean-clips", s.handleCleanClips)
	mux.HandleFunc("/api/prepare", s.handlePrepare)
	mux.HandleFunc("/api/auto-detect", s.handleAutoDetect)
	mux.HandleFunc("/api/transcribe", s.handleTranscribe)
	mux.HandleFunc("/api/ai/subtitles", s.handleAISubtitles)
	mux.HandleFunc("/api/clip", s.handleClip)
	mux.HandleFunc("/api/render", s.handleClip)
	mux.HandleFunc("/preview", s.handlePreview)

	// Ensure output directory exists for static video serving
	_ = os.MkdirAll(s.OutDir, 0755)
	clipFS := http.FileServer(http.Dir(s.OutDir))
	mux.HandleFunc("/clips/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		if ext == ".mp4" || ext == ".m4v" {
			w.Header().Set("Content-Type", "video/mp4")
		} else if ext == ".webm" {
			w.Header().Set("Content-Type", "video/webm")
		}
		http.StripPrefix("/clips/", clipFS).ServeHTTP(w, r)
	})

	return mux
}

// Start runs the HTTP web server.
func (s *Server) Start() error {
	handler := s.Router()
	server := &http.Server{
		Addr:         s.Addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Minute, // Allow long video streams
	}

	fmt.Printf("\n🚀 Clipper Web Studio running at: http://localhost%s\n", s.Addr)
	fmt.Printf("📂 Serving clips from directory: %s\n", s.OutDir)
	fmt.Println("Press Ctrl+C to stop the web server.")

	return server.ListenAndServe()
}

type statusResponse struct {
	IsRunning   bool   `json:"is_running"`
	CurrentTask string `json:"current_task"`
	ProgressPct int    `json:"progress_pct"`
	Completed   bool   `json:"completed"`
	LastError   string `json:"last_error"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(statusResponse{
		IsRunning:   s.isRunning,
		CurrentTask: s.currentTask,
		ProgressPct: s.progressPct,
		Completed:   s.completed,
		LastError:   s.lastError,
	})
}

// ClipItem represents a rendered clip in the output directory.
type ClipItem struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	SizeStr      string `json:"size_str"`
	DurationStr  string `json:"duration_str"`
	ModTime      string `json:"mod_time"`
}

func (s *Server) getCacheDir() string {
	if s.DefaultConfig != nil && s.DefaultConfig.CacheDir != "" {
		return s.DefaultConfig.CacheDir
	}
	return "./cache"
}

func (s *Server) handleClips(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodDelete {
		fileName := strings.TrimSpace(r.URL.Query().Get("name"))
		if fileName == "" {
			fileName = strings.TrimSpace(r.URL.Query().Get("file"))
		}
		if fileName == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "name or file query parameter is required"})
			return
		}

		// Prevent path traversal
		cleanName := filepath.Base(fileName)
		dirsToSearch := []string{s.OutDir, filepath.Join(s.OutDir, "shorts")}
		found := false

		for _, dir := range dirsToSearch {
			targetVideo := filepath.Join(dir, cleanName)
			if fi, err := os.Stat(targetVideo); err == nil && !fi.IsDir() {
				_ = os.Remove(targetVideo)
				found = true

				// Also clean up matching thumbnails, metadata, and subtitle files
				ext := filepath.Ext(cleanName)
				base := strings.TrimSuffix(cleanName, ext)
				associatedCandidates := []string{
					base + ".jpg",
					base + "_hook1.jpg",
					base + "_hook2.jpg",
					base + ".png",
					base + ".webp",
					base + "_metadata.json",
					base + "_metadata.txt",
					base + ".srt",
					base + ".vtt",
					base + ".ass",
				}
				for _, tc := range associatedCandidates {
					_ = os.Remove(filepath.Join(dir, tc))
				}
			}
		}

		if !found {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("clip '%s' not found", cleanName)})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "deleted",
			"message": fmt.Sprintf("Clip '%s' deleted successfully", cleanName),
			"name":    cleanName,
		})
		return
	}

	var items []ClipItem

	// Search in OutDir and OutDir/shorts
	dirsToScan := []string{s.OutDir, filepath.Join(s.OutDir, "shorts")}
	seen := make(map[string]bool)

	for _, dir := range dirsToScan {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			ext := strings.ToLower(filepath.Ext(name))
			if ext != ".mp4" && ext != ".mkv" && ext != ".webm" {
				continue
			}

			fullPath := filepath.Join(dir, name)
			if seen[fullPath] {
				continue
			}
			seen[fullPath] = true

			fi, fErr := e.Info()
			if fErr != nil {
				continue
			}

			rel, _ := filepath.Rel(s.OutDir, fullPath)
			url := "/clips/" + filepath.ToSlash(rel)

			// Look for matching thumbnail
			baseWithoutExt := strings.TrimSuffix(name, ext)
			var thumbURL string
			thumbCandidates := []string{
				baseWithoutExt + ".jpg",
				baseWithoutExt + "_hook1.jpg",
				baseWithoutExt + "_hook2.jpg",
			}
			for _, tc := range thumbCandidates {
				candidatePath := filepath.Join(dir, tc)
				if _, sErr := os.Stat(candidatePath); sErr == nil {
					relThumb, _ := filepath.Rel(s.OutDir, candidatePath)
					thumbURL = "/clips/" + filepath.ToSlash(relThumb)
					break
				}
			}

			sizeStr := formatFileSize(fi.Size())

			items = append(items, ClipItem{
				Name:         name,
				URL:          url,
				ThumbnailURL: thumbURL,
				SizeStr:      sizeStr,
				DurationStr:  "--",
				ModTime:      fi.ModTime().Format("15:04:05 02 Jan"),
			})
		}
	}

	if items == nil {
		items = []ClipItem{}
	}

	_ = json.NewEncoder(w).Encode(items)
}

// StorageStatsResponse represents disk usage data for cache and clips directories.
type StorageStatsResponse struct {
	CacheDir       string `json:"cache_dir"`
	CacheSizeBytes int64  `json:"cache_size_bytes"`
	CacheSizeStr   string `json:"cache_size_str"`
	CacheFileCount int    `json:"cache_file_count"`

	ClipsDir       string `json:"clips_dir"`
	ClipsSizeBytes int64  `json:"clips_size_bytes"`
	ClipsSizeStr   string `json:"clips_size_str"`
	ClipsCount     int    `json:"clips_count"`
}

func (s *Server) handleStorageStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cacheDir := s.getCacheDir()
	var cacheBytes int64
	var cacheCount int
	if _, err := os.Stat(cacheDir); err == nil {
		_ = filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				cacheBytes += info.Size()
				cacheCount++
			}
			return nil
		})
	}

	var clipsBytes int64
	var clipsCount int
	dirsToScan := []string{s.OutDir, filepath.Join(s.OutDir, "shorts")}
	seen := make(map[string]bool)

	for _, dir := range dirsToScan {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fullPath := filepath.Join(dir, e.Name())
			if seen[fullPath] {
				continue
			}
			seen[fullPath] = true

			fi, err := e.Info()
			if err == nil {
				clipsBytes += fi.Size()
				ext := strings.ToLower(filepath.Ext(e.Name()))
				if ext == ".mp4" || ext == ".mkv" || ext == ".webm" {
					clipsCount++
				}
			}
		}
	}

	_ = json.NewEncoder(w).Encode(StorageStatsResponse{
		CacheDir:       cacheDir,
		CacheSizeBytes: cacheBytes,
		CacheSizeStr:   formatFileSize(cacheBytes),
		CacheFileCount: cacheCount,

		ClipsDir:       s.OutDir,
		ClipsSizeBytes: clipsBytes,
		ClipsSizeStr:   formatFileSize(clipsBytes),
		ClipsCount:     clipsCount,
	})
}

func (s *Server) handleCleanCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	cacheDir := s.getCacheDir()
	freed, count, err := downloader.CleanCache(cacheDir, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to clean cache: %v", err)})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"freed_bytes":   freed,
		"freed_str":     formatFileSize(freed),
		"removed_count": count,
		"message":       fmt.Sprintf("Cleaned %d cache files, freeing %s", count, formatFileSize(freed)),
	})
}

func (s *Server) handleCleanClips(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var freedBytes int64
	var removedClips int

	dirsToClean := []string{filepath.Join(s.OutDir, "shorts"), s.OutDir}
	for _, dir := range dirsToClean {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fullPath := filepath.Join(dir, e.Name())
			if fi, sErr := e.Info(); sErr == nil {
				freedBytes += fi.Size()
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext == ".mp4" || ext == ".mkv" || ext == ".webm" {
				removedClips++
			}
			_ = os.Remove(fullPath)
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"freed_bytes":   freedBytes,
		"freed_str":     formatFileSize(freedBytes),
		"removed_count": removedClips,
		"message":       fmt.Sprintf("Deleted %d clips and media assets, freeing %s", removedClips, formatFileSize(freedBytes)),
	})
}

type prepareRequestPayload struct {
	Source string `json:"source"`
}

type prepareResponse struct {
	Status     string `json:"status"`
	Path       string `json:"path"`
	PreviewURL string `json:"preview_url"`
	IsYouTube  bool   `json:"is_youtube"`
}

func (s *Server) handlePrepare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req prepareRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid request: %v"}`, err), http.StatusBadRequest)
		return
	}

	src := strings.TrimSpace(req.Source)
	if src == "" {
		http.Error(w, `{"error":"source path or URL is required"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if downloader.IsYouTubeURL(src) {
		cacheDir := "./cache"
		if s.DefaultConfig != nil && s.DefaultConfig.CacheDir != "" {
			cacheDir = s.DefaultConfig.CacheDir
		}

		videoPath, err := downloader.DownloadYouTubeVideo(src, cacheDir, "best", false)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Failed to download YouTube video: %v", err)})
			return
		}

		previewURL := "/preview?path=" + url.QueryEscape(videoPath)
		_ = json.NewEncoder(w).Encode(prepareResponse{
			Status:     "ready",
			Path:       videoPath,
			PreviewURL: previewURL,
			IsYouTube:  true,
		})
		return
	}

	// Local file
	absPath, err := filepath.Abs(src)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid file path"})
		return
	}

	fi, err := os.Stat(absPath)
	if err != nil || fi.IsDir() {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Local video file '%s' not found", src)})
		return
	}

	previewURL := "/preview?path=" + url.QueryEscape(absPath)
	_ = json.NewEncoder(w).Encode(prepareResponse{
		Status:     "ready",
		Path:       absPath,
		PreviewURL: previewURL,
		IsYouTube:  false,
	})
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "missing path query param", http.StatusBadRequest)
		return
	}

	if downloader.IsYouTubeURL(path) {
		cacheDir := "./cache"
		if s.DefaultConfig != nil && s.DefaultConfig.CacheDir != "" {
			cacheDir = s.DefaultConfig.CacheDir
		}
		cached, err := downloader.DownloadYouTubeVideo(path, cacheDir, "best", false)
		if err == nil && cached != "" {
			path = cached
		}
	}

	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Accept-Ranges", "bytes")
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp4", ".m4v":
		w.Header().Set("Content-Type", "video/mp4")
	case ".webm":
		w.Header().Set("Content-Type", "video/webm")
	case ".mov":
		w.Header().Set("Content-Type", "video/quicktime")
	case ".ogg", ".ogv":
		w.Header().Set("Content-Type", "video/ogg")
	case ".mkv":
		w.Header().Set("Content-Type", "video/x-matroska")
	}

	// Serve video with automatic HTTP Range support for seeking
	http.ServeFile(w, r, path)
}

type clipRequestPayload struct {
	InputFile        string            `json:"input_file"`
	OutputDir        string            `json:"output_dir"`
	OutputFile       string            `json:"output"`
	Mode             clipper.Mode        `json:"mode"`
	Strategy         clipper.CutStrategy `json:"strategy"`
	Quality          string              `json:"quality"`
	Segments         []clipper.Segment `json:"segments"`
	AutoDetect       string            `json:"auto_detect"`
	AIRouter         string            `json:"ai_router"`
	APIKey           string            `json:"api_key"`
	AIModel          string            `json:"ai_model"`
	Shorts           bool              `json:"shorts"`
	ShortsStyle      string            `json:"shorts_style"`
	Subtitles        *bool             `json:"subtitles,omitempty"`
	BurnSubtitles    *bool             `json:"burn_subtitles,omitempty"`
	SubPreset        string            `json:"sub_preset"`
	SubSDHMode       string            `json:"sub_sdh_mode"`
	SubEmoji         bool              `json:"sub_emoji"`
	Loudnorm         bool              `json:"loudnorm"`
	JumpCut          bool              `json:"jump_cut"`
	JumpCutMinSil    float64           `json:"jump_cut_min_silence"`
	JumpCutMargin    float64           `json:"jump_cut_margin"`
	JumpCutNoise     float64           `json:"jump_cut_noise"`
	Watermark        string            `json:"watermark"`
	WatermarkPos     string            `json:"watermark_pos"`
	OverlayText      string            `json:"overlay_text"`
	TextPos          string            `json:"text_pos"`
	FontSize         int               `json:"font_size"`
	FontColor        string            `json:"font_color"`
	GenerateMetadata bool              `json:"generate_metadata"`
	ExtractThumbnail bool              `json:"extract_thumbnail"`
	ThumbnailCount   int               `json:"thumbnail_count"`
	TargetDuration   float64           `json:"target_duration"`
	HWAccel          string            `json:"hwaccel"`
	Concurrency      int               `json:"concurrency"`
	AIConfig         *struct {
		APIKey       string `json:"api_key"`
		SegmentModel string `json:"segment_model"`
		STTModel     string `json:"stt_model"`
	} `json:"ai_config,omitempty"`
}

type autoDetectRequestPayload struct {
	InputFile      string  `json:"input_file"`
	Mode           string  `json:"mode"` // "ai", "silence", "scene"
	AIRouter       string  `json:"ai_router"`
	APIKey         string  `json:"api_key"`
	Model          string  `json:"model"`
	UseWhisper     bool    `json:"use_whisper"`
	Shorts         bool    `json:"shorts"`
	TargetDuration float64 `json:"target_duration"`
}

type autoDetectResponse struct {
	Segments []clipper.Segment `json:"segments"`
	Count    int               `json:"count"`
	Mode     string            `json:"mode"`
}

func (s *Server) handleAutoDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req autoDetectRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid request: %v"}`, err), http.StatusBadRequest)
		return
	}

	src := strings.TrimSpace(req.InputFile)
	if src == "" {
		http.Error(w, `{"error":"input_file is required for auto detection"}`, http.StatusBadRequest)
		return
	}

	cfg := clipper.Config{}
	if s.DefaultConfig != nil {
		cfg = *s.DefaultConfig
	}

	cfg.InputFile = src
	cfg.AutoDetect = req.Mode
	if cfg.AutoDetect == "" {
		cfg.AutoDetect = "ai"
	}
	cfg.Shorts = req.Shorts
	cfg.UseWhisper = req.UseWhisper
	cfg.AIConfig.TargetDuration = req.TargetDuration

	if req.AIRouter != "" {
		cfg.AIConfig.APIRouter = req.AIRouter
	}
	if req.APIKey != "" {
		cfg.AIConfig.APIKey = req.APIKey
	}
	if req.Model != "" {
		cfg.AIConfig.Model = req.Model
	}

	c, err := clipper.New()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Failed to initialize clipper engine: " + err.Error()})
		return
	}

	segments, err := c.DetectSegments(&cfg)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(autoDetectResponse{
		Segments: segments,
		Count:    len(segments),
		Mode:     cfg.AutoDetect,
	})
}

func (s *Server) handleClip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		http.Error(w, `{"error":"a clipping job is already running"}`, http.StatusConflict)
		return
	}

	var req clipRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.mu.Unlock()
		http.Error(w, fmt.Sprintf(`{"error":"invalid request: %v"}`, err), http.StatusBadRequest)
		return
	}

	if req.InputFile == "" || (len(req.Segments) == 0 && req.AutoDetect == "") {
		s.mu.Unlock()
		http.Error(w, `{"error":"input_file and at least 1 segment (or auto_detect mode) required"}`, http.StatusBadRequest)
		return
	}

	s.isRunning = true
	s.currentTask = "Initializing job..."
	s.progressPct = 5
	s.completed = false
	s.lastError = ""
	s.mu.Unlock()

	// Clone default config and override with request parameters
	cfg := clipper.Config{}
	if s.DefaultConfig != nil {
		cfg = *s.DefaultConfig
	}

	cfg.InputFile = req.InputFile
	cfg.Segments = req.Segments
	cfg.AutoDetect = req.AutoDetect
	if req.Mode != "" {
		cfg.Mode = req.Mode
	}
	if req.OutputFile != "" {
		cfg.OutputFile = req.OutputFile
	}
	if req.Strategy != "" {
		cfg.Strategy = req.Strategy
	}
	if req.Quality != "" {
		cfg.Quality = req.Quality
	}
	if req.TargetDuration > 0 {
		cfg.TargetDuration = req.TargetDuration
		cfg.AIConfig.TargetDuration = req.TargetDuration
	}
	if req.AIRouter != "" {
		cfg.AIConfig.APIRouter = req.AIRouter
	}
	if req.APIKey != "" {
		cfg.AIConfig.APIKey = req.APIKey
	}
	if req.AIModel != "" {
		cfg.AIConfig.Model = req.AIModel
	}
	if req.AIConfig != nil {
		if req.AIConfig.APIKey != "" {
			cfg.AIConfig.APIKey = req.AIConfig.APIKey
		}
		if req.AIConfig.SegmentModel != "" {
			cfg.AIConfig.Model = req.AIConfig.SegmentModel
		}
	}
	cfg.Shorts = req.Shorts
	if req.ShortsStyle != "" {
		cfg.ShortsStyle = req.ShortsStyle
	}
	if req.Subtitles != nil {
		cfg.Subtitles = *req.Subtitles
	} else if req.BurnSubtitles != nil {
		cfg.Subtitles = *req.BurnSubtitles
	}
	if req.SubPreset != "" {
		cfg.SubPreset = req.SubPreset
	}
	if req.SubSDHMode != "" {
		cfg.SubSDHMode = req.SubSDHMode
	}
	cfg.SubEmoji = req.SubEmoji
	cfg.Loudnorm = req.Loudnorm
	cfg.JumpCut = req.JumpCut
	if req.JumpCutMinSil > 0 {
		cfg.JumpCutMinSil = req.JumpCutMinSil
	}
	if req.JumpCutMargin > 0 {
		cfg.JumpCutMargin = req.JumpCutMargin
	}
	if req.JumpCutNoise != 0 {
		cfg.JumpCutNoise = req.JumpCutNoise
	}
	if req.Watermark != "" {
		cfg.WatermarkPath = req.Watermark
		cfg.WatermarkPos = req.WatermarkPos
	}
	if req.OverlayText != "" {
		cfg.OverlayText = req.OverlayText
		cfg.TextPos = req.TextPos
		cfg.FontSize = req.FontSize
		cfg.FontColor = req.FontColor
	}
	cfg.GenerateMetadata = req.GenerateMetadata
	cfg.ExtractThumbnail = req.ExtractThumbnail
	if req.ThumbnailCount > 0 {
		cfg.ThumbnailCount = req.ThumbnailCount
	} else {
		cfg.ThumbnailCount = 1
	}
	if req.HWAccel != "" {
		cfg.HWAccel = req.HWAccel
	}
	if req.Concurrency > 0 {
		cfg.Concurrency = req.Concurrency
	}
	if req.OutputDir != "" {
		cfg.OutputDir = req.OutputDir
	} else if cfg.OutputDir == "" {
		cfg.OutputDir = s.OutDir
	}

	// Run clipping job in background goroutine
	go func() {
		defer func() {
			s.mu.Lock()
			s.isRunning = false
			s.mu.Unlock()
		}()

		c, err := clipper.New()
		if err != nil {
			s.mu.Lock()
			s.lastError = err.Error()
			s.mu.Unlock()
			return
		}

		s.mu.Lock()
		s.currentTask = fmt.Sprintf("Processing %d segments...", len(cfg.Segments))
		s.progressPct = 25
		s.mu.Unlock()

		if err := c.Process(&cfg); err != nil {
			s.mu.Lock()
			s.lastError = err.Error()
			s.mu.Unlock()
			return
		}

		s.mu.Lock()
		s.progressPct = 100
		s.completed = true
		s.currentTask = "All clips completed!"
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "job_started",
		"message": "Clipping job started successfully in background",
	})
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

type transcribeRequestPayload struct {
	InputFile  string `json:"input_file"`
	Start      string `json:"start"`
	End        string `json:"end"`
	Lang       string `json:"lang"`
	UseWhisper bool   `json:"use_whisper"`
	APIKey     string `json:"api_key"`
	Model      string `json:"model"`
}

type transcribeResponsePayload struct {
	Cues  []clipper.SubtitleCue `json:"cues"`
	Count int                   `json:"count"`
}

func (s *Server) handleTranscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req transcribeRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid request: %v"}`, err), http.StatusBadRequest)
		return
	}

	src := strings.TrimSpace(req.InputFile)
	if src == "" {
		http.Error(w, `{"error":"input_file is required"}`, http.StatusBadRequest)
		return
	}

	cacheDir := "./cache"
	if s.DefaultConfig != nil && s.DefaultConfig.CacheDir != "" {
		cacheDir = s.DefaultConfig.CacheDir
	}

	videoPath := src
	if downloader.IsYouTubeURL(src) {
		cached, err := downloader.DownloadYouTubeVideo(src, cacheDir, "best", false)
		if err == nil && cached != "" {
			videoPath = cached
		}
	}

	lang := req.Lang
	if lang == "" {
		lang = "id"
	}

	startSec, _ := clipper.ParseTimeSeconds(req.Start)
	endSec, _ := clipper.ParseTimeSeconds(req.End)
	if endSec <= startSec {
		endSec = startSec + 60
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	// 1. Try Gemini Audio STT if API key is provided or user selected Gemini STT
	if apiKey != "" && (req.UseWhisper || !downloader.IsYouTubeURL(src)) {
		tmpAudio := filepath.Join(cacheDir, fmt.Sprintf(".stt_%d.mp3", time.Now().UnixNano()))
		cmd := exec.Command("ffmpeg", "-y", "-ss", fmt.Sprintf("%.2f", startSec), "-to", fmt.Sprintf("%.2f", endSec),
			"-i", videoPath, "-vn", "-acodec", "libmp3lame", "-b:a", "64k", "-ar", "24000", tmpAudio)
		if err := cmd.Run(); err == nil {
			if audioBytes, rErr := os.ReadFile(tmpAudio); rErr == nil && len(audioBytes) > 0 {
				_ = os.Remove(tmpAudio)
				model := req.Model
				if model == "" {
					model = "gemini-3.6-flash"
				}
				geminiCues, gErr := ai.TranscribeAudioGemini(apiKey, model, lang, audioBytes, "audio/mp3")
				if gErr == nil && len(geminiCues) > 0 {
					var cues []clipper.SubtitleCue
					for _, gc := range geminiCues {
						cues = append(cues, clipper.SubtitleCue{
							Start: gc.Start,
							End:   gc.End,
							Text:  gc.Text,
						})
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(transcribeResponsePayload{
						Cues:  cues,
						Count: len(cues),
					})
					return
				}
			} else {
				_ = os.Remove(tmpAudio)
			}
		}
	}

	// 2. Fetch from YouTube subtitles or fallback to local Whisper
	var subs []transcriber.SubtitleEntry
	if req.UseWhisper {
		subs, _ = transcriber.TranscribeWithWhisper(videoPath, cacheDir, lang)
	} else {
		subs, _ = transcriber.FetchSubtitles(src, cacheDir, lang)
		if len(subs) == 0 && lang != "en" {
			subs, _ = transcriber.FetchSubtitles(src, cacheDir, "en")
		}
		if len(subs) == 0 {
			// If YouTube has no CC and Gemini API key is provided, transcribe with Gemini!
			if apiKey != "" {
				tmpAudio := filepath.Join(cacheDir, fmt.Sprintf(".stt_%d.mp3", time.Now().UnixNano()))
				cmd := exec.Command("ffmpeg", "-y", "-ss", fmt.Sprintf("%.2f", startSec), "-to", fmt.Sprintf("%.2f", endSec),
					"-i", videoPath, "-vn", "-acodec", "libmp3lame", "-b:a", "64k", "-ar", "24000", tmpAudio)
				if err := cmd.Run(); err == nil {
					if audioBytes, rErr := os.ReadFile(tmpAudio); rErr == nil && len(audioBytes) > 0 {
						_ = os.Remove(tmpAudio)
						model := req.Model
						if model == "" {
							model = "gemini-3.6-flash"
						}
						geminiCues, gErr := ai.TranscribeAudioGemini(apiKey, model, lang, audioBytes, "audio/mp3")
						if gErr == nil && len(geminiCues) > 0 {
							var cues []clipper.SubtitleCue
							for _, gc := range geminiCues {
								cues = append(cues, clipper.SubtitleCue{
									Start: gc.Start,
									End:   gc.End,
									Text:  gc.Text,
								})
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(transcribeResponsePayload{
								Cues:  cues,
								Count: len(cues),
							})
							return
						}
					} else {
						_ = os.Remove(tmpAudio)
					}
				}
			}
			subs, _ = transcriber.TranscribeWithWhisper(videoPath, cacheDir, lang)
		}
	}

	var cues []clipper.SubtitleCue
	if len(subs) > 0 {
		sliced := transcriber.SliceSubtitles(subs, startSec, endSec)
		for _, e := range sliced {
			cueStart, _ := clipper.ParseTimeSeconds(e.Start)
			cueEnd, _ := clipper.ParseTimeSeconds(e.End)
			relStart := cueStart - startSec
			relEnd := cueEnd - startSec
			if relStart < 0 {
				relStart = 0
			}
			if relEnd <= relStart {
				relEnd = relStart + 1.5
			}
			cleanText := strings.TrimSpace(e.Text)
			if cleanText != "" {
				cues = append(cues, clipper.SubtitleCue{
					Start: math.Round(relStart*100) / 100,
					End:   math.Round(relEnd*100) / 100,
					Text:  cleanText,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(transcribeResponsePayload{
		Cues:  cues,
		Count: len(cues),
	})
}

type aiSubtitlesRequestPayload struct {
	Action     string                `json:"action"` // "translate", "emojis", "polish"
	Cues       []clipper.SubtitleCue `json:"cues"`
	TargetLang string                `json:"target_lang"`
	AIRouter   string                `json:"ai_router"`
	APIKey     string                `json:"api_key"`
	Model      string                `json:"model"`
}

func (s *Server) handleAISubtitles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req aiSubtitlesRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid request: %v"}`, err), http.StatusBadRequest)
		return
	}

	if len(req.Cues) == 0 {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"cues": req.Cues, "count": 0})
		return
	}

	var entries []transcriber.SubtitleEntry
	for _, c := range req.Cues {
		entries = append(entries, transcriber.SubtitleEntry{
			Start: ai.FormatSecondsToTime(c.Start),
			End:   ai.FormatSecondsToTime(c.End),
			Text:  c.Text,
		})
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "emojis":
		enhanced := transcriber.InjectContextualEmojis(entries)
		var outCues []clipper.SubtitleCue
		for i, e := range enhanced {
			outCues = append(outCues, clipper.SubtitleCue{
				Start: req.Cues[i].Start,
				End:   req.Cues[i].End,
				Text:  e.Text,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"cues": outCues, "count": len(outCues)})
		return

	case "translate":
		targetLang := req.TargetLang
		if targetLang == "" {
			targetLang = "id"
		}
		aiCfg := ai.AIProviderConfig{
			APIRouter: req.AIRouter,
			APIKey:    req.APIKey,
			Model:     req.Model,
		}
		translated, err := ai.TranslateSubtitlesMultiProvider(entries, aiCfg, targetLang)
		if err != nil || len(translated) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"cues": req.Cues, "count": len(req.Cues), "warn": fmt.Sprintf("%v", err)})
			return
		}
		var outCues []clipper.SubtitleCue
		for i, e := range translated {
			origStart := req.Cues[0].Start
			origEnd := req.Cues[0].End
			if i < len(req.Cues) {
				origStart = req.Cues[i].Start
				origEnd = req.Cues[i].End
			}
			outCues = append(outCues, clipper.SubtitleCue{
				Start: origStart,
				End:   origEnd,
				Text:  e.Text,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"cues": outCues, "count": len(outCues)})
		return

	case "polish":
		var outCues []clipper.SubtitleCue
		for _, c := range req.Cues {
			t := strings.TrimSpace(c.Text)
			if len(t) > 0 {
				t = strings.ToUpper(t[:1]) + t[1:]
			}
			outCues = append(outCues, clipper.SubtitleCue{
				Start: c.Start,
				End:   c.End,
				Text:  t,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"cues": outCues, "count": len(outCues)})
		return

	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"cues": req.Cues, "count": len(req.Cues)})
		return
	}
}
