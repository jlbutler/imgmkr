package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jlbutler/imgmkr/cleanup"
	"github.com/jlbutler/imgmkr/mockfs"
	"github.com/jlbutler/imgmkr/progress"
	"github.com/jlbutler/imgmkr/size"
)

func TestSizeModule(t *testing.T) {
	// Test parsing
	result, err := size.Parse("1MB")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 1024*1024 {
		t.Errorf("Expected 1048576, got %d", result)
	}

	// Test formatting
	formatted := size.Format(1024 * 1024)
	if formatted != "1.00 MB" {
		t.Errorf("Expected '1.00 MB', got %s", formatted)
	}

	// Test list parsing
	sizes, err := size.ParseList("1KB,2MB")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(sizes) != 2 {
		t.Errorf("Expected 2 sizes, got %d", len(sizes))
	}
}

func TestProgressModule(t *testing.T) {
	tracker := progress.New(3, 6*1024*1024)
	if tracker == nil {
		t.Error("Failed to create progress tracker")
	}
	// Note: We can't easily test the output, but we can test it doesn't crash
}

func TestCleanupModule(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "imgmkr-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create a cleanup manager
	cm := cleanup.New(tempDir)
	if cm == nil {
		t.Error("Failed to create cleanup manager")
	}

	// Test graceful cleanup
	cm.GracefulCleanup()

	// Verify the directory was removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("Temp directory should be removed after cleanup: %s", tempDir)
	}
}

func TestMockfsModule(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "imgmkr-mockfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test creating a mock filesystem
	layerDir := filepath.Join(tempDir, "test-layer")
	err = mockfs.Create(layerDir, 10*1024, 2, 5) // 10KB, depth 2, 5 files
	if err != nil {
		t.Errorf("Unexpected error creating mock filesystem: %v", err)
	}

	// Verify that the layer directory was created
	if _, err := os.Stat(layerDir); os.IsNotExist(err) {
		t.Errorf("Layer directory %s was not created", layerDir)
	}
}
