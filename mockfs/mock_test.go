package mockfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreate(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "imgmkr-mockfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test creating a mock filesystem
	layerDir := filepath.Join(tempDir, "test-layer")
	err = Create(layerDir, 10*1024, 2, 5) // 10KB, depth 2, 5 files
	if err != nil {
		t.Errorf("Unexpected error creating mock filesystem: %v", err)
	}

	// Verify that the layer directory was created
	if _, err := os.Stat(layerDir); os.IsNotExist(err) {
		t.Errorf("Layer directory %s was not created", layerDir)
	}

	// Verify that some files were created
	files, err := filepath.Glob(filepath.Join(layerDir, "**/*"))
	if err != nil {
		t.Errorf("Error checking created files: %v", err)
	}
	if len(files) == 0 {
		t.Errorf("No files were created in mock filesystem")
	}
}
