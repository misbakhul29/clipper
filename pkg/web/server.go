package web

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/misbakhul29/clipper/pkg/clipper"
	"github.com/misbakhul29/clipper/pkg/downloader"
)

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

	sub, err := fs.Sub(StaticFS, "static")
	if err == nil {
		mux.Handle("/", http.FileServer(http.FS(sub)))
	}

	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/clips", s.handleClips)
	mux.HandleFunc("/api/prepare", s.handlePrepare)
	mux.HandleFunc("/api/auto-detect", s.handleAutoDetect)
	mux.HandleFunc("/api/clip", s.handleClip)
	mux.HandleFunc("/preview", s.handlePreview)

	// Ensure output directory exists for static video serving
	_ = os.MkdirAll(s.OutDir, 0755)
	mux.Handle("/clips/", http.StripPrefix("/clips/", http.FileServer(http.Dir(s.OutDir))))

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

func (s *Server) handleClips(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

		previewURL := "/preview?path=" + filepath.ToSlash(videoPath)
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

	previewURL := "/preview?path=" + filepath.ToSlash(absPath)
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

	// Serve video with automatic HTTP Range support for seeking
	http.ServeFile(w, r, path)
}

type clipRequestPayload struct {
	InputFile        string            `json:"input_file"`
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
	SubEmoji         bool              `json:"sub_emoji"`
	Loudnorm         bool              `json:"loudnorm"`
	JumpCut          bool              `json:"jump_cut"`
	GenerateMetadata bool              `json:"generate_metadata"`
	ExtractThumbnail bool              `json:"extract_thumbnail"`
	ThumbnailCount   int               `json:"thumbnail_count"`
	HWAccel          string            `json:"hwaccel"`
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
	if req.AIRouter != "" {
		cfg.AIConfig.APIRouter = req.AIRouter
	}
	if req.APIKey != "" {
		cfg.AIConfig.APIKey = req.APIKey
	}
	if req.AIModel != "" {
		cfg.AIConfig.Model = req.AIModel
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
	cfg.SubEmoji = req.SubEmoji
	cfg.Loudnorm = req.Loudnorm
	cfg.JumpCut = req.JumpCut
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
	if cfg.OutputDir == "" {
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
