package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/britej3/gobot/services/screener"
)

type mockExchangeClient struct{}

func (m *mockExchangeClient) GetExchangeInfo(ctx context.Context) ([]screener.ExchangeInfo, error) {
	now := time.Now()
	return []screener.ExchangeInfo{
		{Symbol: "1000PEPEUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 15000000, PriceChangePct: 15.0, LastUpdated: now},
		{Symbol: "WIFUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 10000000, PriceChangePct: 12.0, LastUpdated: now},
		{Symbol: "POPCATUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 8000000, PriceChangePct: 18.0, LastUpdated: now},
		{Symbol: "TURBOUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 6000000, PriceChangePct: 8.5, LastUpdated: now},
		{Symbol: "MOGUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 5000000, PriceChangePct: 22.0, LastUpdated: now},
		{Symbol: "FWOGUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 4000000, PriceChangePct: 10.0, LastUpdated: now},
		{Symbol: "MEWUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 3500000, PriceChangePct: 9.0, LastUpdated: now},
		{Symbol: "ACTUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 3000000, PriceChangePct: 25.0, LastUpdated: now},
		{Symbol: "BTCUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 500000000, PriceChangePct: 1.5, LastUpdated: now},
		{Symbol: "ETHUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 250000000, PriceChangePct: 2.0, LastUpdated: now},
		{Symbol: "SOLUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 100000000, PriceChangePct: -1.0, LastUpdated: now},
		{Symbol: "LOWVOLUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 100000, PriceChangePct: 50.0, LastUpdated: now},
	}, nil
}

func main() {
	log.SetFlags(0)
	log.Println("=== GOBOT Meme Coin Screener - Generated Assets ===\n")

	client := &mockExchangeClient{}

	screenerInstance := screener.NewScreener(client,
		screener.WithAssetFilter(screener.DefaultMemeCoinFilter()),
		screener.WithMaxPairs(10),
		screener.WithSortBy("volatility"),
	)

	ctx := context.Background()
	if err := screenerInstance.Initialize(ctx); err != nil {
		log.Fatalf("Failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	stats := screenerInstance.Stats()
	pairs := screenerInstance.GetActivePairs()
	assets := screenerInstance.ToAssets()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("SCREENER STATUS                                         \n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Total pairs scanned:    %d\n", stats.TotalPairs)
	fmt.Printf("  Active pairs:           %d\n", stats.ActivePairs)
	fmt.Printf("  Avg volume (24h):       $%.0f\n", stats.AvgVolume)
	fmt.Printf("  Avg price change:       %.1f%%\n", stats.AvgChange)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n📊 GENERATED ASSETS (domain/asset.Asset)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for i, a := range assets {
		fmt.Printf("  %2d. %-15s $%10.0f  conf: %.2f  scored: %s\n",
			i+1, a.Symbol, a.Volume24h, a.Confidence, a.ScoredAt.Format("15:04:05"))
	}

	fmt.Println("\n🎯 ACTIVE TRADING PAIRS (sorted by volatility)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for i, symbol := range pairs {
		score := screenerInstance.GetScore(symbol)
		fmt.Printf("  %2d. %-15s score: %.1f\n", i+1, symbol, score)
	}

	fmt.Println("\n✅ Assets generated successfully!")
	fmt.Printf("\nAsset struct implements: asset.Asset {\n")
	fmt.Printf("  Symbol:       string\n")
	fmt.Printf("  Volume24h:    float64\n")
	fmt.Printf("  Confidence:   float64 (0-1)\n")
	fmt.Printf("  ScoredAt:     time.Time\n")
	fmt.Printf("}\n")

	screenerInstance.Stop()
}
