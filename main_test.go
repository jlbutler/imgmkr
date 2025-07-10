package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"512KB", 512 * KB, false},
		{"1MB", 1 * MB, false},
		{"2GB", 2 * GB, false},
		{"1.5MB", int64(1.5 * MB), false},
		{"2.75GB", int64(2.75 * GB), false},
		{"1024", 1024, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"1.5XB", 0, true},
	}

	for _, test := range tests {
		result, err := parseSize(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %q, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("For input %q, expected %d, got %d", test.input, test.expected, result)
			}
		}
	}
}

func TestParseSizes(t *testing.T) {
	tests := []struct {
		input    string
		expected []int64
		hasError bool
	}{
		{"512KB,1MB,2GB", []int64{512 * KB, 1 * MB, 2 * GB}, false},
		{"1MB", []int64{1 * MB}, false},
		{"", nil, true},
		{"1MB,invalid", nil, true},
	}

	for _, test := range tests {
		result, err := parseSizes(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %q, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", test.input, err)
			}
			if len(result) != len(test.expected) {
				t.Errorf("For input %q, expected length %d, got %d", test.input, len(test.expected), len(result))
				continue
			}
			for i, expected := range test.expected {
				if result[i] != expected {
					t.Errorf("For input %q at index %d, expected %d, got %d", test.input, i, expected, result[i])
				}
			}
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{512, "512 bytes"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{int64(1.5 * MB), "1.50 MB"},
		{int64(2.75 * GB), "2.75 GB"},
	}

	for _, test := range tests {
		result := formatSize(test.input)
		if result != test.expected {
			t.Errorf("For input %d, expected %q, got %q", test.input, test.expected, result)
		}
	}
}
func TestCreateTempDir(t *testing.T) {
	// Test with empty prefix (should use system default)
	tempDir1, err := createTempDir("")
	if err != nil {
		t.Errorf("Unexpected error creating temp dir with empty prefix: %v", err)
	}
	defer os.RemoveAll(tempDir1)

	// Verify directory exists
	if _, err := os.Stat(tempDir1); os.IsNotExist(err) {
		t.Errorf("Temp directory was not created: %s", tempDir1)
	}

	// Test with custom prefix
	customPrefix := "/tmp"
	tempDir2, err := createTempDir(customPrefix)
	if err != nil {
		t.Errorf("Unexpected error creating temp dir with custom prefix: %v", err)
	}
	defer os.RemoveAll(tempDir2)

	// Verify directory exists and has correct prefix
	if _, err := os.Stat(tempDir2); os.IsNotExist(err) {
		t.Errorf("Temp directory was not created: %s", tempDir2)
	}
}
func TestCreateLayersConcurrently(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "imgmkr-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with small sizes to avoid long test times
	sizes := []int64{1024, 2048, 4096} // 1KB, 2KB, 4KB
	maxWorkers := 2

	err = createLayersConcurrently(tempDir, sizes, maxWorkers)
	if err != nil {
		t.Errorf("Unexpected error in createLayersConcurrently: %v", err)
	}

	// Verify that all layer directories were created
	for i := range sizes {
		layerDir := filepath.Join(tempDir, fmt.Sprintf("layer%d", i+1))
		if _, err := os.Stat(layerDir); os.IsNotExist(err) {
			t.Errorf("Layer directory %s was not created", layerDir)
		}
	}
}

func TestLayerJobAndResult(t *testing.T) {
	// Test LayerJob struct
	job := LayerJob{
		layerNum: 1,
		layerDir: "/tmp/test",
		size:     1024,
	}

	if job.layerNum != 1 {
		t.Errorf("Expected layerNum 1, got %d", job.layerNum)
	}
	if job.layerDir != "/tmp/test" {
		t.Errorf("Expected layerDir '/tmp/test', got %s", job.layerDir)
	}
	if job.size != 1024 {
		t.Errorf("Expected size 1024, got %d", job.size)
	}

	// Test LayerResult struct
	result := LayerResult{
		layerNum: 1,
		duration: time.Second,
		err:      nil,
	}

	if result.layerNum != 1 {
		t.Errorf("Expected layerNum 1, got %d", result.layerNum)
	}
	if result.duration != time.Second {
		t.Errorf("Expected duration 1s, got %v", result.duration)
	}
	if result.err != nil {
		t.Errorf("Expected no error, got %v", result.err)
	}
}
