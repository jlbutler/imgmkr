package mockfs

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/jlbutler/imgmkr/size"
)

// Create creates a mock filesystem structure with multiple files and directories
func Create(layerDir string, layerSize int64, maxDepth int, targetFiles int) error {
	// Create the layer directory if it doesn't exist
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		return fmt.Errorf("failed to create layer directory: %w", err)
	}

	// Calculate target files if not specified (roughly 1 file per 10MB, min 5, max 1000)
	if targetFiles == 0 {
		targetFiles = int(layerSize / (10 * size.MB))
		if targetFiles < 5 {
			targetFiles = 5
		}
		if targetFiles > 1000 {
			targetFiles = 1000
		}
	}

	// Create realistic file size distribution
	filePlan := CreatePlan(layerSize, targetFiles)

	// Create directory structure and files based on the plan
	return createFilesFromPlan(layerDir, filePlan, maxDepth, 0)
}

// createFilesFromPlan creates files based on the file size plan
func createFilesFromPlan(dir string, plan Plan, maxDepth int, currentDepth int) error {
	// Calculate total files to distribute
	totalFiles := len(plan.VeryLargeFiles) + len(plan.LargeFiles) + len(plan.MediumFiles) + len(plan.SmallFiles)
	if totalFiles == 0 {
		return nil
	}

	// Create all file sizes in one slice for easier distribution
	allFiles := make([]int64, 0, totalFiles)
	allFiles = append(allFiles, plan.VeryLargeFiles...)
	allFiles = append(allFiles, plan.LargeFiles...)
	allFiles = append(allFiles, plan.MediumFiles...)
	allFiles = append(allFiles, plan.SmallFiles...)

	// Shuffle to distribute different sizes across directories
	for i := range allFiles {
		j := rand.Intn(i + 1)
		allFiles[i], allFiles[j] = allFiles[j], allFiles[i]
	}

	// Decide how many files to create at this level vs subdirectories
	filesAtThisLevel := totalFiles / 3 // Roughly 1/3 of files at current level
	if filesAtThisLevel < 1 {
		filesAtThisLevel = totalFiles
	}
	if currentDepth >= maxDepth {
		filesAtThisLevel = totalFiles // All files at this level if max depth reached
	}

	// Create files at this level
	for i := 0; i < filesAtThisLevel && i < len(allFiles); i++ {
		fileSize := allFiles[i]
		fileName := fmt.Sprintf("%s-file", size.Format(fileSize))
		filePath := filepath.Join(dir, fileName)

		err := createSingleFile(filePath, fileSize)
		if err != nil {
			return err
		}
	}

	// Create subdirectories with remaining files
	remainingFiles := allFiles[filesAtThisLevel:]
	if len(remainingFiles) > 0 && currentDepth < maxDepth {
		// Create 2-4 subdirectories
		numSubdirs := 2 + rand.Intn(3) // 2-4 subdirectories
		if numSubdirs > len(remainingFiles) {
			numSubdirs = len(remainingFiles)
		}

		filesPerSubdir := len(remainingFiles) / numSubdirs
		for i := 0; i < numSubdirs; i++ {
			subdirName := fmt.Sprintf("dir%d", i+1)
			subdirPath := filepath.Join(dir, subdirName)

			if err := os.MkdirAll(subdirPath, 0755); err != nil {
				return fmt.Errorf("failed to create subdirectory: %w", err)
			}

			// Calculate files for this subdirectory
			startIdx := i * filesPerSubdir
			endIdx := startIdx + filesPerSubdir
			if i == numSubdirs-1 {
				endIdx = len(remainingFiles) // Last subdir gets remaining files
			}

			if startIdx < len(remainingFiles) {
				subdirFiles := remainingFiles[startIdx:endIdx]

				// Create a plan for this subdirectory
				subdirPlan := Plan{}
				for _, fileSize := range subdirFiles {
					// Categorize files back into size buckets for recursive call
					switch {
					case fileSize >= 512*size.MB:
						subdirPlan.VeryLargeFiles = append(subdirPlan.VeryLargeFiles, fileSize)
					case fileSize >= 10*size.MB:
						subdirPlan.LargeFiles = append(subdirPlan.LargeFiles, fileSize)
					case fileSize >= 100*size.KB:
						subdirPlan.MediumFiles = append(subdirPlan.MediumFiles, fileSize)
					default:
						subdirPlan.SmallFiles = append(subdirPlan.SmallFiles, fileSize)
					}
				}

				err := createFilesFromPlan(subdirPath, subdirPlan, maxDepth, currentDepth+1)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// createSingleFile creates a single file of the specified size
func createSingleFile(filePath string, fileSize int64) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Fill the file with data in chunks
	const chunkSize = 10 * size.MB
	remaining := fileSize

	for remaining > 0 {
		writeSize := remaining
		if writeSize > chunkSize {
			writeSize = chunkSize
		}

		// Create a buffer with data
		data := make([]byte, writeSize)
		for i := range data {
			data[i] = byte(rand.Intn(256))
		}

		// Write the data to the file
		_, err = file.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write data to file: %w", err)
		}

		remaining -= writeSize
	}

	return nil
}
