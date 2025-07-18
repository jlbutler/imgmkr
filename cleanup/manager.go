package cleanup

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Manager handles graceful shutdown and cleanup
type Manager struct {
	buildDir    string
	cleanupDone chan bool
	interrupted bool
	mu          sync.Mutex
}

// New creates a new cleanup manager
func New(buildDir string) *Manager {
	return &Manager{
		buildDir:    buildDir,
		cleanupDone: make(chan bool, 1),
	}
}

// SetupSignalHandling sets up signal handlers for graceful shutdown
func (cm *Manager) SetupSignalHandling() {
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

// cleanup performs the cleanup operation
func (cm *Manager) cleanup() {
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
func (cm *Manager) GracefulCleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.interrupted {
		cm.cleanup()
	}
}
