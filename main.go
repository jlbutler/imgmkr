package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Constants for size units
const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
)

// Command line arguments
var (
	layerSizes    = flag.String("layer-sizes", "", "Comma-separated list of layer sizes (e.g., 512KB,1MB,2GB,8150)")
	tmpdirPrefix  = flag.String("tmpdir-prefix", "", "Directory prefix for temporary build files (default: system temp dir)")
	maxConcurrent = flag.Int("max-concurrent", 5, "Maximum number of layers to create concurrently")
	mockFS        = flag.Bool("mock-fs", false, "Create mock filesystem structure instead of single files")
	maxDepth      = flag.Int("max-depth", 3, "Maximum directory depth for mock filesystem (only used with --mock-fs)")
	targetFiles   = flag.Int("target-files", 0, "Target number of files per layer for mock filesystem (default: calculated based on layer size)")
)

// parseSize parses a string like "512KB", "1.5MB", "2.75GB", "8150", "8B" into bytes
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Convert to uppercase for easier matching
	upperStr := strings.ToUpper(sizeStr)

	// Check for size suffixes (case-insensitive, with variations)
	var multiplier float64 = 1
	var numStr string

	// Bytes: B, b, bytes (or no suffix)
	if strings.HasSuffix(upperStr, "B") && !strings.HasSuffix(upperStr, "KB") && !strings.HasSuffix(upperStr, "MB") && !strings.HasSuffix(upperStr, "GB") {
		multiplier = 1
		numStr = sizeStr[:len(sizeStr)-1]
	} else if strings.HasSuffix(upperStr, "BYTES") {
		multiplier = 1
		numStr = sizeStr[:len(sizeStr)-5]
	} else if strings.HasSuffix(upperStr, "BYTE") {
		multiplier = 1
		numStr = sizeStr[:len(sizeStr)-4]
		// Kilobytes: KB, K, kb, k
	} else if strings.HasSuffix(upperStr, "KB") {
		multiplier = float64(KB)
		numStr = sizeStr[:len(sizeStr)-2]
	} else if strings.HasSuffix(upperStr, "K") {
		multiplier = float64(KB)
		numStr = sizeStr[:len(sizeStr)-1]
		// Megabytes: MB, M, mb, m
	} else if strings.HasSuffix(upperStr, "MB") {
		multiplier = float64(MB)
		numStr = sizeStr[:len(sizeStr)-2]
	} else if strings.HasSuffix(upperStr, "M") {
		multiplier = float64(MB)
		numStr = sizeStr[:len(sizeStr)-1]
		// Gigabytes: GB, G, gb, g
	} else if strings.HasSuffix(upperStr, "GB") {
		multiplier = float64(GB)
		numStr = sizeStr[:len(sizeStr)-2]
	} else if strings.HasSuffix(upperStr, "G") {
		multiplier = float64(GB)
		numStr = sizeStr[:len(sizeStr)-1]
	} else {
		// No suffix - assume bytes
		numStr = sizeStr
	}

	// Parse the numeric part as float64 to handle decimal values
	size, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	// Convert to int64 after multiplication
	return int64(size * multiplier), nil
}

// parseSizes parses a comma-separated list of sizes
func parseSizes(sizesStr string) ([]int64, error) {
	if sizesStr == "" {
		return nil, fmt.Errorf("layer sizes cannot be empty")
	}

	sizeStrs := strings.Split(sizesStr, ",")
	sizes := make([]int64, len(sizeStrs))

	for i, sizeStr := range sizeStrs {
		size, err := parseSize(sizeStr)
		if err != nil {
			return nil, err
		}
		sizes[i] = size
	}

	return sizes, nil
}

// createTempDir creates a temporary directory for building the image
func createTempDir(prefix string) (string, error) {
	tempDir, err := os.MkdirTemp(prefix, "imgmkr-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	return tempDir, nil
}

// LayerJob represents a layer creation job
type LayerJob struct {
	layerNum int
	layerDir string
	size     int64
}

// LayerResult represents the result of a layer creation job
type LayerResult struct {
	layerNum int
	duration time.Duration
	err      error
}

// ProgressTracker tracks progress across concurrent operations
type ProgressTracker struct {
	totalLayers     int
	completedLayers int64
	totalSize       int64
	completedSize   int64
	startTime       time.Time
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(totalLayers int, totalSize int64) *ProgressTracker {
	return &ProgressTracker{
		totalLayers: totalLayers,
		totalSize:   totalSize,
		startTime:   time.Now(),
	}
}

// UpdateProgress updates the progress and displays current status
func (pt *ProgressTracker) UpdateProgress(layerNum int, layerSize int64, duration time.Duration) {
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
		formatSize(completedSize), formatSize(pt.totalSize), sizeProgressPercent,
		layerNum, duration.Round(time.Millisecond),
		eta.Round(time.Second))
}

// Finish completes the progress display
func (pt *ProgressTracker) Finish() {
	elapsed := time.Since(pt.startTime)
	fmt.Printf("\n✅ All layers completed in %s\n", elapsed.Round(time.Millisecond))
}

// CleanupManager handles graceful shutdown and cleanup
type CleanupManager struct {
	buildDir    string
	cleanupDone chan bool
	interrupted bool
	mu          sync.Mutex
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(buildDir string) *CleanupManager {
	return &CleanupManager{
		buildDir:    buildDir,
		cleanupDone: make(chan bool, 1),
	}
}

// SetupSignalHandling sets up signal handlers for graceful shutdown
func (cm *CleanupManager) SetupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		cm.mu.Lock()
		if !cm.interrupted {
			cm.interrupted = true
			fmt.Printf("\n\n🛑 Received %s signal, cleaning up...\n", sig)
			cm.cleanup()
			fmt.Println("✅ Cleanup completed")
			os.Exit(130) // Standard exit code for SIGINT
		}
		cm.mu.Unlock()
	}()
}

// Cleanup performs the cleanup operation
func (cm *CleanupManager) cleanup() {
	if cm.buildDir != "" {
		err := os.RemoveAll(cm.buildDir)
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to clean up temporary directory %s: %v\n", cm.buildDir, err)
		} else {
			fmt.Printf("🗑️  Removed temporary directory: %s\n", cm.buildDir)
		}
		// Clear the buildDir to prevent double cleanup
		cm.buildDir = ""
	}
}

// GracefulCleanup performs cleanup if not already interrupted
func (cm *CleanupManager) GracefulCleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.interrupted {
		cm.cleanup()
	}
}

// createLayersConcurrently creates multiple layers concurrently using a worker pool
func createLayersConcurrently(buildDir string, sizes []int64, maxWorkers int) error {
	// Calculate total size for progress tracking
	var totalSize int64
	for _, size := range sizes {
		totalSize += size
	}

	// Create progress tracker
	progress := NewProgressTracker(len(sizes), totalSize)
	jobs := make(chan LayerJob, len(sizes))
	results := make(chan LayerResult, len(sizes))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				startTime := time.Now()
				var err error
				if *mockFS {
					err = createMockFilesystem(job.layerDir, job.size, *maxDepth, *targetFiles)
				} else {
					err = createLayerFile(job.layerDir, job.size)
				}
				results <- LayerResult{
					layerNum: job.layerNum,
					duration: time.Since(startTime),
					err:      err,
				}
			}
		}()
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for i, size := range sizes {
			layerDir := filepath.Join(buildDir, fmt.Sprintf("layer%d", i+1))
			jobs <- LayerJob{
				layerNum: i + 1,
				layerDir: layerDir,
				size:     size,
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results and report progress
	completed := make(map[int]LayerResult)
	for result := range results {
		if result.err != nil {
			return fmt.Errorf("error creating layer %d: %w", result.layerNum, result.err)
		}
		completed[result.layerNum] = result
		progress.UpdateProgress(result.layerNum, sizes[result.layerNum-1], result.duration)
	}

	// Finish progress display
	progress.Finish()

	return nil
}

// createMockFilesystem creates a mock filesystem structure with multiple files and directories
func createMockFilesystem(layerDir string, size int64, maxDepth int, targetFiles int) error {
	// Create the layer directory if it doesn't exist
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		return fmt.Errorf("failed to create layer directory: %w", err)
	}

	// Calculate target files if not specified (roughly 1 file per 10MB, min 5, max 1000)
	if targetFiles == 0 {
		targetFiles = int(size / (10 * MB))
		if targetFiles < 5 {
			targetFiles = 5
		}
		if targetFiles > 1000 {
			targetFiles = 1000
		}
	}

	// Create realistic file size distribution
	filePlan := createFileSizePlan(size, targetFiles)

	// Create directory structure and files based on the plan
	return createFilesFromPlan(layerDir, filePlan, maxDepth, 0)
}

// FileSizePlan represents a plan for creating files of different sizes
type FileSizePlan struct {
	VeryLargeFiles []int64 // 512MB - 50% of layer size
	LargeFiles     []int64 // 10MB - 512MB
	MediumFiles    []int64 // 100KB - 10MB
	SmallFiles     []int64 // 1KB - 100KB
}

// createFileSizePlan creates a realistic distribution of file sizes
func createFileSizePlan(totalSize int64, targetFiles int) FileSizePlan {
	plan := FileSizePlan{}
	remainingSize := totalSize
	remainingFiles := targetFiles

	// For large layers (>= 1GB), include some very large files
	if totalSize >= GB && remainingFiles > 10 {
		numVeryLarge := 1 + rand.Intn(3) // 1-3 very large files
		if numVeryLarge > remainingFiles/4 {
			numVeryLarge = remainingFiles / 4 // Don't use more than 25% of files for very large
		}

		maxVeryLargeSize := totalSize / 2 // Up to 50% of total size
		minVeryLargeSize := int64(512 * MB)

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

		for i := 0; i < numLarge && remainingSize > 10*MB && remainingFiles > 0; i++ {
			maxSize := int64(512 * MB)
			if remainingSize/int64(remainingFiles) < maxSize {
				maxSize = remainingSize / int64(remainingFiles) * 2 // Allow up to 2x average
			}
			if maxSize < 10*MB {
				break
			}

			fileSize := rand.Int63n(maxSize-10*MB) + 10*MB
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

		for i := 0; i < numMedium && remainingSize > 100*KB && remainingFiles > 0; i++ {
			maxSize := int64(10 * MB)
			if remainingSize/int64(remainingFiles) < maxSize {
				maxSize = remainingSize / int64(remainingFiles) * 2
			}
			if maxSize < 100*KB {
				break
			}

			fileSize := rand.Int63n(maxSize-100*KB) + 100*KB
			plan.MediumFiles = append(plan.MediumFiles, fileSize)
			remainingSize -= fileSize
			remainingFiles--
		}
	}

	// Small files: 1KB - 100KB (remaining files)
	for remainingFiles > 0 && remainingSize > 1024 {
		maxSize := int64(100 * KB)
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
		if remainingSize >= 100*KB {
			// Create a new medium file with the remaining size
			plan.MediumFiles = append(plan.MediumFiles, remainingSize)
		} else if len(plan.SmallFiles) > 0 {
			// Add to the last small file only if it keeps it in the small range
			lastSmallIdx := len(plan.SmallFiles) - 1
			if plan.SmallFiles[lastSmallIdx]+remainingSize < 100*KB {
				plan.SmallFiles[lastSmallIdx] += remainingSize
			} else {
				// Create a new small file with remaining size
				plan.SmallFiles = append(plan.SmallFiles, remainingSize)
			}
		}
	}

	return plan
}

// createFilesFromPlan creates files based on the file size plan
func createFilesFromPlan(dir string, plan FileSizePlan, maxDepth int, currentDepth int) error {
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
		fileName := fmt.Sprintf("%s-file", formatSize(fileSize))
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
				subdirPlan := FileSizePlan{}
				for _, size := range subdirFiles {
					// Categorize files back into size buckets for recursive call
					switch {
					case size >= 512*MB:
						subdirPlan.VeryLargeFiles = append(subdirPlan.VeryLargeFiles, size)
					case size >= 10*MB:
						subdirPlan.LargeFiles = append(subdirPlan.LargeFiles, size)
					case size >= 100*KB:
						subdirPlan.MediumFiles = append(subdirPlan.MediumFiles, size)
					default:
						subdirPlan.SmallFiles = append(subdirPlan.SmallFiles, size)
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
func createSingleFile(filePath string, size int64) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Fill the file with data in chunks
	const chunkSize = 10 * MB
	remaining := size

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

// createLayerFile creates a file of the specified size filled with random data
func createLayerFile(layerDir string, size int64) error {
	// Create the layer directory if it doesn't exist
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		return fmt.Errorf("failed to create layer directory: %w", err)
	}

	// Create a file with the size as part of the name
	var sizeStr string
	switch {
	case size >= GB:
		sizeStr = fmt.Sprintf("%.2fGB-file", float64(size)/float64(GB))
	case size >= MB:
		sizeStr = fmt.Sprintf("%.2fMB-file", float64(size)/float64(MB))
	default:
		sizeStr = fmt.Sprintf("%.2fKB-file", float64(size)/float64(KB))
	}

	filePath := filepath.Join(layerDir, sizeStr)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Fill the file with random data
	// We'll write in chunks to avoid memory issues with large files
	const chunkSize = 10 * MB
	remaining := size

	for remaining > 0 {
		writeSize := remaining
		if writeSize > chunkSize {
			writeSize = chunkSize
		}

		// Create a buffer with random data
		data := make([]byte, writeSize)
		_, err := io.ReadFull(strings.NewReader(strings.Repeat("x", int(writeSize))), data)
		if err != nil {
			return fmt.Errorf("failed to generate random data: %w", err)
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

// createDockerfile creates a Dockerfile that adds each layer
func createDockerfile(buildDir string, numLayers int) error {
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	file, err := os.Create(dockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to create Dockerfile: %w", err)
	}
	defer file.Close()

	// Start with a scratch image
	_, err = file.WriteString("FROM scratch\n")
	if err != nil {
		return fmt.Errorf("failed to write to Dockerfile: %w", err)
	}

	// Add each layer
	for i := 1; i <= numLayers; i++ {
		layerDir := fmt.Sprintf("layer%d", i)
		_, err = file.WriteString(fmt.Sprintf("ADD %s /\n", layerDir))
		if err != nil {
			return fmt.Errorf("failed to write to Dockerfile: %w", err)
		}
	}

	return nil
}

// buildImage builds the Docker image using finch or docker
func buildImage(buildDir string, repoTag string) error {
	// Try finch first, fallback to docker if not available
	var cmdName string
	_, err := exec.LookPath("finch")
	if err == nil {
		cmdName = "finch"
	} else {
		_, err = exec.LookPath("docker")
		if err == nil {
			cmdName = "docker"
		} else {
			return fmt.Errorf("neither finch nor docker command found")
		}
	}

	// Build the image
	cmd := exec.Command(cmdName, "build", "-t", repoTag, ".")
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Building image with %s...\n", cmdName)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	return nil
}

func main() {
	// Parse command line flags
	flag.Parse()

	// Validate required flags
	if *layerSizes == "" {
		log.Fatal("--layer-sizes is required")
	}

	// Get the repository:tag argument
	args := flag.Args()
	if len(args) != 1 {
		log.Fatal("Repository:tag argument is required")
	}
	repoTag := args[0]

	// Parse layer sizes
	sizes, err := parseSizes(*layerSizes)
	if err != nil {
		log.Fatalf("Error parsing layer sizes: %v", err)
	}

	// Number of layers is inferred from the layer sizes
	numLayers := len(sizes)

	// Create a temporary build directory
	fmt.Println("Creating temporary build directory...")
	buildDir, err := createTempDir(*tmpdirPrefix)
	if err != nil {
		log.Fatalf("Error creating temporary directory: %v", err)
	}

	// Setup cleanup manager and signal handling
	cleanupManager := NewCleanupManager(buildDir)
	cleanupManager.SetupSignalHandling()
	defer cleanupManager.GracefulCleanup()

	// Create layer files
	fmt.Printf("Creating layer files (max %d concurrent)...\n", *maxConcurrent)
	err = createLayersConcurrently(buildDir, sizes, *maxConcurrent)
	if err != nil {
		log.Fatalf("Error creating layer files: %v", err)
	}

	// Create Dockerfile
	fmt.Println("Creating Dockerfile...")
	err = createDockerfile(buildDir, numLayers)
	if err != nil {
		log.Fatalf("Error creating Dockerfile: %v", err)
	}

	// Build the image
	err = buildImage(buildDir, repoTag)
	if err != nil {
		log.Fatalf("Error building image: %v", err)
	}

	fmt.Printf("Successfully built image %s\n", repoTag)
}

// formatSize formats a size in bytes to a human-readable string
func formatSize(size int64) string {
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
