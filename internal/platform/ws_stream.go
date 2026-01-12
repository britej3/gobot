package platform

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

type StreamManager struct {
	client  *futures.Client
	symbols []string
	stopCh  chan struct{}
	doneCh  chan struct{}
}

func NewStreamManager(client *futures.Client, symbols []string) *StreamManager {
	return &StreamManager{
		client:  client,
		symbols: symbols,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

// Start initiates the resilient stream loop
func (sm *StreamManager) Start(ctx context.Context, handler func(*futures.WsKlineEvent)) {
	go sm.reconnectionLoop(ctx, handler)
}

func (sm *StreamManager) reconnectionLoop(ctx context.Context, handler func(*futures.WsKlineEvent)) {
	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Printf("🔌 [WS] Connecting to Binance Futures Streams (Attempt %d)...", attempts+1)
			
			// Map symbols to intervals for the combined stream
			symbolIntervals := make(map[string]string)
			for _, s := range sm.symbols {
				symbolIntervals[s] = "1m"
			}

			// Serve Combined Stream
			doneC, stopC, err := futures.WsCombinedKlineServe(symbolIntervals, handler, sm.errHandler)
			if err != nil {
				attempts++
				delay := sm.calculateBackoff(baseDelay, maxDelay, attempts)
				log.Printf("❌ [WS] Connection failed: %v. Retrying in %v...", err, delay)
				time.Sleep(delay)
				continue
			}

			// Reset attempts on successful connection
			attempts = 0
			log.Println("✅ [WS] Stream connected and active.")

			// Setup 23h 50m Rotation Timer
			rotationTimer := time.NewTimer(23*time.Hour + 50*time.Minute)

			select {
			case <-rotationTimer.C:
				log.Println("🔄 [WS] Scheduled 24h Rotation. Gracefully reconnecting...")
				stopC <- struct{}{}
			case <-doneC:
				log.Println("⚠️ [WS] Connection closed by server. Initiating reconnect...")
			case <-sm.stopCh:
				stopC <- struct{}{}
				return
			case <-ctx.Done():
				stopC <- struct{}{}
				return
			}
		}
	}
}

// calculateBackoff implements Exponential Backoff with Jitter
func (sm *StreamManager) calculateBackoff(base, max time.Duration, attempts int) time.Duration {
	// Calculate exponential backoff using integer arithmetic
	delay := time.Duration(int64(base) * (1 << uint(attempts)))
	if delay > max {
		delay = max
	}
	// Add 15% random jitter
	jitter := time.Duration(float64(delay) * (rand.Float64()*0.3 - 0.15))
	return delay + jitter
}

// handleCloseError implements reply_unknown.md error code specifications
func (sm *StreamManager) handleCloseError(code int) time.Duration {
	switch code {
	case 1008:
		// Too Many Requests (Queued) - Slow down. Increase jitter and reduce scan frequency
		log.Printf("🚨 [WS] Error 1008: Too Many Requests. Waiting 2 minutes...")
		return 2 * time.Minute
	case 429:
		// Rate Limit Hit - Back off. Disconnect all WebSockets and wait for Retry-After
		log.Printf("🚨 [WS] Error 429: Rate Limit Hit. Waiting 5 minutes...")
		return 5 * time.Minute
	case -1003:
		// Internal Server Error - Hold. Pause all new orders for 30 seconds
		log.Printf("🚨 [WS] Error -1003: Internal Error. Pausing for 30 seconds...")
		return 30 * time.Second
	default:
		log.Printf("🚨 [WS] Stream Error: %v", code)
		return 0 // Use default backoff
	}
}

func (sm *StreamManager) errHandler(err error) {
	log.Printf("🚨 [WS] Stream Error: %v", err)
}