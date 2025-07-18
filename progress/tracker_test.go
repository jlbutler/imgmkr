package progress

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tracker := New(3, 6*1024*1024)

	if tracker == nil {
		t.Error("Failed to create progress tracker")
	}

	if tracker.totalLayers != 3 {
		t.Errorf("Expected totalLayers 3, got %d", tracker.totalLayers)
	}

	if tracker.totalSize != 6*1024*1024 {
		t.Errorf("Expected totalSize %d, got %d", 6*1024*1024, tracker.totalSize)
	}
}

func TestUpdate(t *testing.T) {
	tracker := New(3, 6*1024*1024)

	// Test progress updates
	tracker.Update(1, 2*1024*1024, time.Millisecond*100)

	if atomic.LoadInt64(&tracker.completedLayers) != 1 {
		t.Errorf("Expected 1 completed layer, got %d", atomic.LoadInt64(&tracker.completedLayers))
	}

	if atomic.LoadInt64(&tracker.completedSize) != 2*1024*1024 {
		t.Errorf("Expected completed size %d, got %d", 2*1024*1024, atomic.LoadInt64(&tracker.completedSize))
	}

	// Add more updates
	tracker.Update(2, 2*1024*1024, time.Millisecond*150)
	tracker.Update(3, 2*1024*1024, time.Millisecond*120)

	// Verify final counts
	if atomic.LoadInt64(&tracker.completedLayers) != 3 {
		t.Errorf("Expected 3 completed layers, got %d", atomic.LoadInt64(&tracker.completedLayers))
	}

	if atomic.LoadInt64(&tracker.completedSize) != 6*1024*1024 {
		t.Errorf("Expected completed size %d, got %d", 6*1024*1024, atomic.LoadInt64(&tracker.completedSize))
	}

	// Test Finish (just make sure it doesn't crash)
	tracker.Finish()
}
