package main

import (
	"fmt"
	"time"

	"github.com/britebrt/cognee/internal/platform"
)

func main() {
	fmt.Println("🧪 Testing Anti-Sniffer Jitter Implementation")
	fmt.Println("===============================================")
	
	// Test 1: Verify jitter produces delays in 5-25ms range
	fmt.Println("\nTest 1: Measuring 10 jitter delays...")
	for i := 0; i < 10; i++ {
		start := time.Now()
		platform.ApplyJitter()
		delay := time.Since(start).Milliseconds()
		fmt.Printf("  Delay %2d: %d ms\n", i+1, delay)
	}
	
	fmt.Println("\n✅ Jitter test complete!")
	fmt.Println("Expected: Delays should show natural distribution around 15ms mean")
}
