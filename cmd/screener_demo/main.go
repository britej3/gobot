package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/britebrt/cognee/infra/binance"
	"github.com/britebrt/cognee/services/screener"
)

func main() {
	log.SetFlags(0)
	log.Println("=== GOBOT Meme Coin Screener Demo ===")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := binance.Config{
		BaseURL: "https://fapi.binance.com",
		Timeout: 30 * time.Second,
	}

	client := binance.NewScreenerClient(cfg)
	adapter := binance.NewScreenerAdapter(client)

	screenerInstance := screener.NewScreener(adapter,
		screener.WithAssetFilter(screener.DefaultMemeCoinFilter()),
		screener.WithInterval(5*time.Minute),
		screener.WithMaxPairs(10),
		screener.WithSortBy("volatility"),
	)

	log.Println("Starting screener...")
	if err := screenerInstance.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize screener: %v", err)
	}

	time.Sleep(2 * time.Second)

	pairs := screenerInstance.GetActivePairs()
	pairsInfo := screenerInstance.GetPairsInfo()
	stats := screenerInstance.Stats()
	assets := screenerInstance.ToAssets()

	log.Println("\n=== Screener Results ===")
	log.Printf("Status: Running | Pairs: %d/%d", stats.ActivePairs, stats.TotalPairs)

	log.Println("\n--- Active Trading Pairs ---")
	for i, symbol := range pairs {
		score := screenerInstance.GetScore(symbol)
		log.Printf("%d. %s (score: %.2f)", i+1, symbol, score)
	}

	log.Println("\n--- Pair Details ---")
	for _, p := range pairsInfo {
		log.Printf("- %s: $%.0f vol | %.1f%% change | %s",
			p.Symbol, p.Volume24h, p.PriceChangePct, p.Status)
	}

	log.Println("\n--- Generated Assets ---")
	for i, a := range assets {
		log.Printf("%d. %s: $%.0f vol | confidence: %.2f",
			i+1, a.Symbol, a.Volume24h, a.Confidence)
	}

	log.Printf("\n--- Summary ---")
	log.Printf("Total pairs found: %d", stats.TotalPairs)
	log.Printf("Active pairs: %d", stats.ActivePairs)
	log.Printf("Avg volume: $%.0f", stats.AvgVolume)
	log.Printf("Avg price change: %.1f%%", stats.AvgChange)
	log.Printf("Last updated: %s", stats.LastUpdated.Format(time.RFC3339))

	screenerInstance.Stop()
	log.Println("\nScreener stopped. Demo complete.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
