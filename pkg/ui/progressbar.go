package ui

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

// ProgressBar displays a dynamic, real-time terminal progress bar.
type ProgressBar struct {
	mu         sync.Mutex
	Title      string
	Total      float64
	Current    float64
	Width      int
	Speed      string
	ETA        string
	IsTTY      bool
	lastUpdate time.Time
	lastPct    int
	finished   bool
}

// NewProgressBar initializes a new progress bar.
func NewProgressBar(title string, total float64) *ProgressBar {
	if total <= 0 {
		total = 100.0
	}
	isTTY := false
	if fi, err := os.Stdout.Stat(); err == nil {
		isTTY = (fi.Mode() & os.ModeCharDevice) != 0
	}
	return &ProgressBar{
		Title:      title,
		Total:      total,
		Width:      25,
		IsTTY:      isTTY,
		lastUpdate: time.Now(),
		lastPct:    -1,
	}
}

// Update updates the progress bar state and re-renders if in a TTY terminal.
func (p *ProgressBar) Update(current float64, speed, eta string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return
	}

	p.Current = current
	p.Speed = speed
	p.ETA = eta

	pct := int(math.Min(100.0, math.Max(0.0, (p.Current/p.Total)*100.0)))

	now := time.Now()
	// Throttle redraws to at most 12-15 times per sec to prevent terminal flicker
	if p.IsTTY && now.Sub(p.lastUpdate) < 70*time.Millisecond && pct == p.lastPct {
		return
	}
	p.lastUpdate = now
	p.lastPct = pct

	if p.IsTTY {
		filled := (pct * p.Width) / 100
		empty := p.Width - filled
		if filled < 0 {
			filled = 0
		}
		if empty < 0 {
			empty = 0
		}

		bar := strings.Repeat("=", filled)
		if empty > 0 && filled > 0 {
			bar = strings.Repeat("=", filled-1) + ">"
		}
		space := strings.Repeat(" ", empty)

		extra := ""
		if speed != "" {
			extra += fmt.Sprintf(" | %s", speed)
		}
		if eta != "" {
			extra += fmt.Sprintf(" | ETA: %s", eta)
		}

		fmt.Printf("\r\033[K%s: [%s%s] %3d%%%s", p.Title, bar, space, pct, extra)
	}
}

// Finish completes the progress bar and prints the final state.
func (p *ProgressBar) Finish(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.finished {
		return
	}
	p.finished = true

	if message == "" {
		message = "Done!"
	}

	isFailure := strings.Contains(strings.ToLower(message), "fail") || strings.Contains(strings.ToLower(message), "error")

	if p.IsTTY {
		if isFailure {
			fmt.Printf("\r\033[K%s: %s\n", p.Title, message)
		} else {
			bar := strings.Repeat("=", p.Width)
			fmt.Printf("\r\033[K%s: [%s] 100%% %s\n", p.Title, bar, message)
		}
	} else {
		if isFailure {
			fmt.Printf("%s: %s\n", p.Title, message)
		} else {
			fmt.Printf("%s: 100%% %s\n", p.Title, message)
		}
	}
}

// FormatDuration formats seconds into MM:SS format.
func FormatDuration(sec float64) string {
	if sec < 0 || math.IsNaN(sec) || math.IsInf(sec, 0) {
		return "--:--"
	}
	totalSec := int(math.Round(sec))
	m := totalSec / 60
	s := totalSec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
