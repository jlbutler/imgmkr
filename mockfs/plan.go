package mockfs

import (
	"math/rand"

	"github.com/jlbutler/imgmkr/size"
)

// Plan represents a plan for creating files of different sizes
type Plan struct {
	VeryLargeFiles []int64 // 512MB - 50% of layer size
	LargeFiles     []int64 // 10MB - 512MB
	MediumFiles    []int64 // 100KB - 10MB
	SmallFiles     []int64 // 1KB - 100KB
}

// CreatePlan creates a realistic distribution of file sizes
func CreatePlan(totalSize int64, targetFiles int) Plan {
	plan := Plan{}
	remainingSize := totalSize
	remainingFiles := targetFiles

	// For large layers (>= 1GB), include some very large files
	if totalSize >= size.GB && remainingFiles > 10 {
		numVeryLarge := 1 + rand.Intn(3) // 1-3 very large files
		if numVeryLarge > remainingFiles/4 {
			numVeryLarge = remainingFiles / 4 // Don't use more than 25% of files for very large
		}

		maxVeryLargeSize := totalSize / 2 // Up to 50% of total size
		minVeryLargeSize := int64(512 * size.MB)

		for i := 0; i < numVeryLarge && remainingSize > minVeryLargeSize && remainingFiles > 0; i++ {
			// Random size between 512MB and maxVeryLargeSize
			fileSize := rand.Int63n(maxVeryLargeSize-minVeryLargeSize) + minVeryLargeSize
			if fileSize > remainingSize/2 { // Don't use more than half remaining size
				fileSize = remainingSize / 2
			}
			if fileSize < minVeryLargeSize {
				fileSize = minVeryLargeSize
			}

			plan.VeryLargeFiles = append(plan.VeryLargeFiles, fileSize)
			remainingSize -= fileSize
			remainingFiles--
		}
	}

	// Large files: 10MB - 512MB (10% of remaining files)
	if remainingFiles > 10 {
		numLarge := remainingFiles / 10
		if numLarge > 20 {
			numLarge = 20 // Cap at 20 large files
		}

		for i := 0; i < numLarge && remainingSize > 10*size.MB && remainingFiles > 0; i++ {
			maxSize := int64(512 * size.MB)
			if remainingSize/int64(remainingFiles) < maxSize {
				maxSize = remainingSize / int64(remainingFiles) * 2 // Allow up to 2x average
			}
			if maxSize < 10*size.MB {
				break
			}

			fileSize := rand.Int63n(maxSize-10*size.MB) + 10*size.MB
			plan.LargeFiles = append(plan.LargeFiles, fileSize)
			remainingSize -= fileSize
			remainingFiles--
		}
	}

	// Medium files: 100KB - 10MB (20% of remaining files)
	if remainingFiles > 5 {
		numMedium := remainingFiles / 5
		if numMedium > 50 {
			numMedium = 50 // Cap at 50 medium files
		}

		for i := 0; i < numMedium && remainingSize > 100*size.KB && remainingFiles > 0; i++ {
			maxSize := int64(10 * size.MB)
			if remainingSize/int64(remainingFiles) < maxSize {
				maxSize = remainingSize / int64(remainingFiles) * 2
			}
			if maxSize < 100*size.KB {
				break
			}

			fileSize := rand.Int63n(maxSize-100*size.KB) + 100*size.KB
			plan.MediumFiles = append(plan.MediumFiles, fileSize)
			remainingSize -= fileSize
			remainingFiles--
		}
	}

	// Small files: 1KB - 100KB (remaining files)
	for remainingFiles > 0 && remainingSize > 1024 {
		maxSize := int64(100 * size.KB)
		if remainingSize/int64(remainingFiles) < maxSize {
			maxSize = remainingSize / int64(remainingFiles)
		}
		if maxSize < 1024 {
			maxSize = 1024
		}

		var fileSize int64
		if maxSize <= 1024 {
			fileSize = remainingSize // Use all remaining size
			remainingFiles = 1       // This will be the last file
		} else {
			fileSize = rand.Int63n(maxSize-1024) + 1024
		}

		plan.SmallFiles = append(plan.SmallFiles, fileSize)
		remainingSize -= fileSize
		remainingFiles--
	}

	// If there's remaining size, distribute it among existing files or create a new medium file
	if remainingSize > 0 {
		if remainingSize >= 100*size.KB {
			// Create a new medium file with the remaining size
			plan.MediumFiles = append(plan.MediumFiles, remainingSize)
		} else if len(plan.SmallFiles) > 0 {
			// Add to the last small file only if it keeps it in the small range
			lastSmallIdx := len(plan.SmallFiles) - 1
			if plan.SmallFiles[lastSmallIdx]+remainingSize < 100*size.KB {
				plan.SmallFiles[lastSmallIdx] += remainingSize
			} else {
				// Create a new small file with remaining size
				plan.SmallFiles = append(plan.SmallFiles, remainingSize)
			}
		}
	}

	return plan
}
