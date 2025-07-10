package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
	numLayers     = flag.Int("num-layers", 0, "Number of layers to create")
	layerSizes    = flag.String("layer-sizes", "", "Comma-separated list of layer sizes (e.g., 512KB,1MB,2GB)")
	tmpdirPrefix  = flag.String("tmpdir-prefix", "", "Directory prefix for temporary build files (default: system temp dir)")
	maxConcurrent = flag.Int("max-concurrent", 5, "Maximum number of layers to create concurrently")
)

// parseSize parses a string like "512KB", "1.5MB", "2.75GB" into bytes
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)

	// Check for size suffixes
	var multiplier float64 = 1
	var numStr string

	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = float64(KB)
		numStr = sizeStr[:len(sizeStr)-2]
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = float64(MB)
		numStr = sizeStr[:len(sizeStr)-2]
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = float64(GB)
		numStr = sizeStr[:len(sizeStr)-2]
	} else {
		// Assume bytes if no suffix
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

// createLayersConcurrently creates multiple layers concurrently using a worker pool
func createLayersConcurrently(buildDir string, sizes []int64, maxWorkers int) error {
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
				err := createLayerFile(job.layerDir, job.size)
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
		fmt.Printf("Layer %d (%s) created in %s\n",
			result.layerNum,
			formatSize(sizes[result.layerNum-1]),
			result.duration)
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
	// Determine whether to use finch or docker
	var cmdName string
	if runtime.GOOS == "darwin" {
		cmdName = "finch"
	} else {
		cmdName = "docker"
	}

	// Check if the command exists
	_, err := exec.LookPath(cmdName)
	if err != nil {
		return fmt.Errorf("%s command not found: %w", cmdName, err)
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

// cleanup removes the temporary build directory
func cleanup(buildDir string) {
	fmt.Println("Cleaning up temporary files...")
	err := os.RemoveAll(buildDir)
	if err != nil {
		fmt.Printf("Warning: Failed to clean up temporary directory %s: %v\n", buildDir, err)
	}
}

func main() {
	// Parse command line flags
	flag.Parse()

	// Validate required flags
	if *numLayers <= 0 {
		log.Fatal("--num-layers must be a positive integer")
	}

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

	// Validate that the number of layers matches the number of sizes
	if len(sizes) != *numLayers {
		log.Fatalf("Number of layer sizes (%d) does not match number of layers (%d)", len(sizes), *numLayers)
	}

	// Create a temporary build directory
	fmt.Println("Creating temporary build directory...")
	buildDir, err := createTempDir(*tmpdirPrefix)
	if err != nil {
		log.Fatalf("Error creating temporary directory: %v", err)
	}
	defer cleanup(buildDir)

	// Create layer files
	fmt.Printf("Creating layer files (max %d concurrent)...\n", *maxConcurrent)
	err = createLayersConcurrently(buildDir, sizes, *maxConcurrent)
	if err != nil {
		log.Fatalf("Error creating layer files: %v", err)
	}

	// Create Dockerfile
	fmt.Println("Creating Dockerfile...")
	err = createDockerfile(buildDir, *numLayers)
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
