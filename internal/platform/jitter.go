// Package platform provides anti-sniffer jitter implementation per technical specifications
package platform

import (
	"math/rand/v2"
	"time"
)

// ApplyJitter introduces a random delay following a Normal Distribution.
// mean: 15ms, stdDev: 5ms (results in ~99% of delays between 0-30ms)
func ApplyJitter() {
	mean := 15.0
	stdDev := 5.0
	
	// rand.NormFloat64 returns a value with a mean of 0 and stdDev of 1
	delayMs := mean + (rand.NormFloat64() * stdDev)
	
	// Safety: Ensure we never return a negative delay
	if delayMs < 1 {
		delayMs = 1
	}
	
	time.Sleep(time.Duration(delayMs) * time.Millisecond)
}
