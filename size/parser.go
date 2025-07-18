package size

import (
	"fmt"
	"strconv"
	"strings"
)

// Constants for size units
const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
)

// Parse parses a string like "512KB", "1.5MB", "2.75GB", "8150", "8B" into bytes
func Parse(sizeStr string) (int64, error) {
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

// ParseList parses a comma-separated list of sizes
func ParseList(sizesStr string) ([]int64, error) {
	if sizesStr == "" {
		return nil, fmt.Errorf("layer sizes cannot be empty")
	}

	sizeStrs := strings.Split(sizesStr, ",")
	sizes := make([]int64, len(sizeStrs))

	for i, sizeStr := range sizeStrs {
		size, err := Parse(sizeStr)
		if err != nil {
			return nil, err
		}
		sizes[i] = size
	}

	return sizes, nil
}

// Format formats a size in bytes to a human-readable string
func Format(size int64) string {
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
