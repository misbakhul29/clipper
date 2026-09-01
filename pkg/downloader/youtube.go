package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kkdai/youtube/v2"
)

// IsYouTubeURL checks whether a string is a YouTube video URL.
func IsYouTubeURL(url string) bool {
	u := strings.ToLower(strings.TrimSpace(url))
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") ||
		strings.Contains(u, "youtube.com/") || strings.Contains(u, "youtu.be/")
}

// GetVideoCacheDir returns a per-video cache directory under baseCacheDir.
// For YouTube URLs, it uses the video ID (e.g. ./cache/v12345).
// For local files, it uses the sanitized video file title (e.g. ./cache/video_name).
func GetVideoCacheDir(baseCacheDir, inputStr string) string {
	if baseCacheDir == "" {
		baseCacheDir = "./cache"
	}
	inputStr = strings.TrimSpace(inputStr)
	if IsYouTubeURL(inputStr) {
		if videoID := ExtractVideoID(inputStr); videoID != "" {
			return filepath.Join(baseCacheDir, videoID)
		}
	} else if inputStr != "" && !strings.HasSuffix(inputStr, ".vtt") && !strings.HasSuffix(inputStr, ".srt") {
		baseName := strings.TrimSuffix(filepath.Base(inputStr), filepath.Ext(inputStr))
		cleanName := sanitizeFilename(baseName)
		if cleanName != "" {
			return filepath.Join(baseCacheDir, cleanName)
		}
	}
	return baseCacheDir
}

// DownloadYouTubeVideo downloads a YouTube video given its URL to outputDir with requested quality.
// If noCache is false, it looks for a cached file in per-video subfolder matching the YouTube video ID.
func DownloadYouTubeVideo(urlStr, outputDir, quality string, noCache bool) (string, error) {
	urlStr = strings.TrimSpace(urlStr)
	if !IsYouTubeURL(urlStr) {
		return "", fmt.Errorf("invalid YouTube URL: %s", urlStr)
	}

	if outputDir == "" {
		outputDir = "./cache"
	}
	if quality == "" {
		quality = "best"
	}

	videoCacheDir := GetVideoCacheDir(outputDir, urlStr)
	if err := os.MkdirAll(videoCacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory '%s': %w", videoCacheDir, err)
	}

	videoID := ExtractVideoID(urlStr)

	// Step 0: Check cache if noCache is false
	if !noCache {
		matches, _ := filepath.Glob(filepath.Join(videoCacheDir, "*.mp4"))
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && info.Size() > 0 {
				fmt.Printf("[CACHE HIT] Found cached video (%s): %s\n", videoID, m)
				return filepath.Abs(m)
			}
		}
		if videoID != "" {
			matchesRoot, _ := filepath.Glob(filepath.Join(outputDir, "*"+videoID+"*.mp4"))
			for _, m := range matchesRoot {
				if info, err := os.Stat(m); err == nil && info.Size() > 0 {
					fmt.Printf("[CACHE HIT] Found legacy cached video (%s): %s\n", videoID, m)
					return filepath.Abs(m)
				}
			}
		}
	}

	// Step 1: Find yt-dlp binary (system PATH, ./bin/yt-dlp, or auto-install)
	binPath, err := ensureYtDlpBinary()
	if err == nil && binPath != "" {
		fmt.Printf("Downloading YouTube video via yt-dlp (quality: %s)...\n", quality)
		file, err := downloadWithYtDlp(binPath, urlStr, videoCacheDir, quality, videoID)
		if err == nil {
			return file, nil
		}
		fmt.Printf("yt-dlp download failed: %v. Falling back to Go internal downloader...\n", err)
	}

	// Step 2: Fallback to pure Go youtube library
	return downloadWithGoLibrary(urlStr, videoCacheDir)
}

// ExtractVideoID extracts the YouTube video ID string from various URL formats.
func ExtractVideoID(urlStr string) string {
	s := strings.TrimSpace(urlStr)
	if idx := strings.Index(s, "v="); idx != -1 {
		id := s[idx+2:]
		if amp := strings.Index(id, "&"); amp != -1 {
			id = id[:amp]
		}
		return id
	}
	if idx := strings.Index(s, "youtu.be/"); idx != -1 {
		id := s[idx+9:]
		if q := strings.Index(id, "?"); q != -1 {
			id = id[:q]
		}
		return id
	}
	if idx := strings.Index(s, "shorts/"); idx != -1 {
		id := s[idx+7:]
		if q := strings.Index(id, "?"); q != -1 {
			id = id[:q]
		}
		return id
	}
	return ""
}

func ensureYtDlpBinary() (string, error) {
	// 1. Check system PATH
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("youtube-dl"); err == nil {
		return path, nil
	}

	// 2. Check User Cache Directory (~/.cache/clipper/bin/yt-dlp)
	userBin := GetYtDlpCachePath()
	if _, err := os.Stat(userBin); err == nil {
		return userBin, nil
	}

	// 3. Check Legacy ./bin/yt-dlp in current working directory
	legacyBin := filepath.Join("bin", "yt-dlp")
	if runtime.GOOS == "windows" {
		legacyBin = filepath.Join("bin", "yt-dlp.exe")
	}
	if _, err := os.Stat(legacyBin); err == nil {
		return filepath.Abs(legacyBin)
	}

	// 4. Auto-download yt-dlp standalone binary into user cache directory
	fmt.Printf("yt-dlp binary not found in PATH. Auto-downloading yt-dlp to %s...\n", userBin)
	binDir := filepath.Dir(userBin)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", err
	}

	downloadURL := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	if runtime.GOOS == "windows" {
		downloadURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch yt-dlp binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download yt-dlp, status code: %d", resp.StatusCode)
	}

	out, err := os.OpenFile(userBin, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create yt-dlp binary file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("failed to save yt-dlp binary: %w", err)
	}

	return userBin, nil
}

// GetYtDlpCachePath returns the OS-specific user cache path for stored yt-dlp binary
func GetYtDlpCachePath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = "."
	}
	appBinDir := filepath.Join(cacheDir, "clipper", "bin")
	binName := "yt-dlp"
	if runtime.GOOS == "windows" {
		binName = "yt-dlp.exe"
	}
	return filepath.Join(appBinDir, binName)
}

func downloadWithYtDlp(binPath, urlStr, outputDir, quality, videoID string) (string, error) {
	outTemplate := filepath.Join(outputDir, "%(title)s_%(id)s.%(ext)s")
	formatSpec := getYtDlpFormatSpec(quality)

	args := []string{
		"-f", formatSpec,
		"--merge-output-format", "mp4",
		"-o", outTemplate,
		"--print", "after_move:filepath",
		urlStr,
	}

	cmd := exec.Command(binPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("yt-dlp download failed: %w\nOutput: %s", err, string(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("yt-dlp did not return output filepath")
	}

	downloadedFile := strings.TrimSpace(lines[len(lines)-1])
	if _, err := os.Stat(downloadedFile); err != nil {
		return "", fmt.Errorf("downloaded file not found at: %s", downloadedFile)
	}

	return downloadedFile, nil
}

func getYtDlpFormatSpec(quality string) string {
	q := strings.ToLower(strings.TrimSpace(quality))
	switch q {
	case "1080p", "1080":
		return "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=1080]+bestaudio/best[height<=1080]"
	case "720p", "720":
		return "bestvideo[height<=720][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=720]+bestaudio/best[height<=720]"
	case "480p", "480":
		return "bestvideo[height<=480][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=480]+bestaudio/best[height<=480]"
	case "360p", "360":
		return "bestvideo[height<=360][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=360]+bestaudio/best[height<=360]"
	case "worst", "low":
		return "worstvideo+worstaudio/worst"
	default: // "best"
		return "b[ext=mp4]/bestvideo[ext=mp4]+bestaudio[ext=m4a]/best"
	}
}

func downloadWithGoLibrary(urlStr, outputDir string) (string, error) {
	client := youtube.Client{}

	video, err := client.GetVideo(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to fetch YouTube video info: %w", err)
	}

	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		formats = video.Formats
	}
	if len(formats) == 0 {
		return "", fmt.Errorf("no suitable video format found for video: %s", video.Title)
	}

	targetFormat := &formats[0]

	stream, size, err := client.GetStream(video, targetFormat)
	if err != nil {
		return "", fmt.Errorf("failed to get YouTube video stream: %w", err)
	}
	defer stream.Close()

	cleanTitle := sanitizeFilename(video.Title)
	if cleanTitle == "" {
		cleanTitle = video.ID
	}
	outPath := filepath.Join(outputDir, fmt.Sprintf("%s.mp4", cleanTitle))

	file, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file '%s': %w", outPath, err)
	}
	defer file.Close()

	fmt.Printf("Downloading YouTube video (%s, size: %.2f MB) -> %s...\n",
		video.Title, float64(size)/(1024*1024), outPath)

	_, err = io.Copy(file, stream)
	if err != nil {
		return "", fmt.Errorf("failed to write YouTube stream to file: %w", err)
	}

	fmt.Printf("Download completed: %s\n", outPath)
	return outPath, nil
}

func sanitizeFilename(s string) string {
	var builder strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			builder.WriteRune(r)
		} else if r == ' ' {
			builder.WriteRune('_')
		}
	}
	return builder.String()
}
