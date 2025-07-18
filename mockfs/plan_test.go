package mockfs

import (
	"testing"

	"github.com/jlbutler/imgmkr/size"
)

func TestCreatePlan(t *testing.T) {
	// Test small layer (should not have very large files)
	smallPlan := CreatePlan(10*size.MB, 20)
	if len(smallPlan.VeryLargeFiles) > 0 {
		t.Errorf("Small layer should not have very large files, got %d", len(smallPlan.VeryLargeFiles))
	}

	totalFiles := len(smallPlan.LargeFiles) + len(smallPlan.MediumFiles) + len(smallPlan.SmallFiles)
	if totalFiles == 0 {
		t.Errorf("Expected some files to be created, got %d", totalFiles)
	}

	// Test large layer (should have very large files)
	largePlan := CreatePlan(2*size.GB, 100)
	if len(largePlan.VeryLargeFiles) == 0 {
		t.Errorf("Large layer should have very large files")
	}

	totalLargeFiles := len(largePlan.VeryLargeFiles) + len(largePlan.LargeFiles) + len(largePlan.MediumFiles) + len(largePlan.SmallFiles)
	if totalLargeFiles == 0 {
		t.Errorf("Expected some files to be created, got %d", totalLargeFiles)
	}

	// Verify very large files are in correct range
	for _, fileSize := range largePlan.VeryLargeFiles {
		if fileSize < 512*size.MB {
			t.Errorf("Very large file too small: %d", fileSize)
		}
	}

	// Verify large files are in correct range
	for _, fileSize := range largePlan.LargeFiles {
		if fileSize < 10*size.MB {
			t.Errorf("Large file too small: %d", fileSize)
		}
	}

	// Verify small files are at least 1KB
	for _, fileSize := range largePlan.SmallFiles {
		if fileSize < 1024 {
			t.Errorf("Small file too small: %d", fileSize)
		}
	}
}
