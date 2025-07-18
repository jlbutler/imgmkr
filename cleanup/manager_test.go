package cleanup

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	cm := New("/tmp/test")

	if cm == nil {
		t.Error("Failed to create cleanup manager")
	}

	if cm.buildDir != "/tmp/test" {
		t.Errorf("Expected buildDir '/tmp/test', got %s", cm.buildDir)
	}
}

func TestGracefulCleanup(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "imgmkr-cleanup-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create a cleanup manager
	cm := New(tempDir)

	// Verify the directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Temp directory should exist before cleanup: %s", tempDir)
	}

	// Test graceful cleanup
	cm.GracefulCleanup()

	// Verify the directory was removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("Temp directory should be removed after cleanup: %s", tempDir)
	}
}

func TestDoubleCleanup(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "imgmkr-double-cleanup-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create a cleanup manager
	cm := New(tempDir)

	// Test that double cleanup doesn't cause issues
	cm.GracefulCleanup()
	cm.GracefulCleanup() // Should be safe to call multiple times

	// Verify the directory was removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("Temp directory should be removed after cleanup: %s", tempDir)
	}
}

func TestInterrupted(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "imgmkr-interrupted-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a cleanup manager
	cm := New(tempDir)

	// Simulate interruption by setting interrupted flag
	cm.mu.Lock()
	cm.interrupted = true
	cm.mu.Unlock()

	// GracefulCleanup should not clean up if already interrupted
	cm.GracefulCleanup()

	// Directory should still exist since cleanup was skipped
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Temp directory should still exist when interrupted: %s", tempDir)
	}
}
