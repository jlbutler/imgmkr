package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jlbutler/imgmkr/cleanup"
	"github.com/jlbutler/imgmkr/mockfs"
	"github.com/jlbutler/imgmkr/progress"
	"github.com/jlbutler/imgmkr/size"
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
	// Calculate total size for progress tracking
	var totalSize int64
	for _, size := range sizes {
		totalSize += size
	}

	// Create progress tracker
	tracker := progress.New(len(sizes), totalSize)
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
					err = mockfs.Create(job.layerDir, job.size, *maxDepth, *targetFiles)
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
		tracker.Update(result.layerNum, sizes[result.layerNum-1], result.duration)
	}

	// Finish progress display
	tracker.Finish()

	return nil
}

// createLayerFile creates a file of the specified size filled with random data
func createLayerFile(layerDir string, fileSize int64) error {
	// Create the layer directory if it doesn't exist
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		return fmt.Errorf("failed to create layer directory: %w", err)
	}

	// Create a file with the size as part of the name
	fileName := fmt.Sprintf("%s-file", size.Format(fileSize))
	filePath := filepath.Join(layerDir, fileName)
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
		_, err := io.ReadFull(strings.NewReader(strings.Repeat("x", int(writeSize))), data)
		if err != nil {
			return fmt.Errorf("failed to generate data: %w", err)
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
	sizes, err := size.ParseList(*layerSizes)
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
	cleanupManager := cleanup.New(buildDir)
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
