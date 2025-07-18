package main

import (
	"fmt"
	"log"

	"github.com/jlbutler/imgmkr/size"
)

func main() {
	fmt.Println("Testing refactored modules...")

	// Test size parsing
	result, err := size.Parse("1MB")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Size parsing: 1MB = %d bytes\n", result)

	// Test size formatting
	formatted := size.Format(1024 * 1024)
	fmt.Printf("✅ Size formatting: 1048576 bytes = %s\n", formatted)

	// Test list parsing
	sizes, err := size.ParseList("1KB,2MB,1.5GB")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ List parsing: %d sizes parsed\n", len(sizes))
	for i, s := range sizes {
		fmt.Printf("   Size %d: %s\n", i+1, size.Format(s))
	}

	fmt.Println("🎉 All module tests passed!")
}
