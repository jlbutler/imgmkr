package progress

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jlbutler/imgmkr/size"
)

// Tracker tracks progress across concurrent operations
type Tracker struct {
	totalLayers     int
	completedLayers int64
	totalSize       int64
	completedSize   int64
	startTime       time.Time
}

// New creates a new progress tracker
func New(totalLayers int, totalSize int64) *Tracker {
	return &Tracker{
		totalLayers: totalLayers,
		totalSize:   totalSize,
		startTime:   time.Now(),
	}
}

// Update updates the progress and displays current status
func (pt *Tracker) Update(layerNum int, layerSize int64, duration time.Duration) {
	atomic.AddInt64(&pt.completedLayers, 1)
	atomic.AddInt64(&pt.completedSize, layerSize)

	completed := atomic.LoadInt64(&pt.completedLayers)
	completedSize := atomic.LoadInt64(&pt.completedSize)

	// Calculate progress percentage
	progressPercent := float64(completed) / float64(pt.totalLayers) * 100
	sizeProgressPercent := float64(completedSize) / float64(pt.totalSize) * 100

	// Calculate ETA
	elapsed := time.Since(pt.startTime)
	var eta time.Duration
	if completed > 0 {
		avgTimePerLayer := elapsed / time.Duration(completed)
		remainingLayers := int64(pt.totalLayers) - completed
		eta = avgTimePerLayer * time.Duration(remainingLayers)
	}

	// Create progress bar
	barWidth := 30
	filledWidth := int(float64(barWidth) * progressPercent / 100)
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

	// Display progress
	fmt.Printf("\r[%s] %d/%d layers (%.1f%%) | %s/%s (%.1f%%) | Layer %d: %s | ETA: %s",
		bar,
		completed, pt.totalLayers, progressPercent,
		size.Format(completedSize), size.Format(pt.totalSize), sizeProgressPercent,
		layerNum, duration.Round(time.Millisecond),
		eta.Round(time.Second))
}

// Finish completes the progress display
func (pt *Tracker) Finish() {
	elapsed := time.Since(pt.startTime)
	fmt.Printf("\n✅ All layers completed in %s\n", elapsed.Round(time.Millisecond))
}
