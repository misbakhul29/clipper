package downloader

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CleanCache scans cacheDir and deletes files/directories older than maxAgeDays.
// If maxAgeDays is <= 0, all files/subdirectories inside cacheDir are deleted.
// Returns total freed bytes and total removed count.
func CleanCache(cacheDir string, maxAgeDays int) (int64, int, error) {
	if cacheDir == "" {
		cacheDir = "./cache"
	}
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	var freedBytes int64
	var removedCount int
	cutoffTime := time.Now().Add(-time.Duration(maxAgeDays) * 24 * time.Hour)

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read cache dir '%s': %w", cacheDir, err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(cacheDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if maxAgeDays <= 0 || info.ModTime().Before(cutoffTime) {
			size := getPathSize(fullPath)
			if err := os.RemoveAll(fullPath); err == nil {
				freedBytes += size
				removedCount++
			}
		}
	}

	return freedBytes, removedCount, nil
}

func getPathSize(path string) int64 {
	var total int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}
