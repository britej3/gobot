package screener

import (
	"context"
	"testing"
	"time"
)

type mockExchangeClient struct {
	info []ExchangeInfo
	err  error
}

func (m *mockExchangeClient) GetExchangeInfo(ctx context.Context) ([]ExchangeInfo, error) {
	return m.info, m.err
}

func TestScreener_ApplyFilters(t *testing.T) {
	client := &mockExchangeClient{
		info: []ExchangeInfo{
			{Symbol: "BTCUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 50000000, PriceChangePct: 2.5},
			{Symbol: "1000PEPEUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 8000000, PriceChangePct: 15.0},
			{Symbol: "WIFUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 6000000, PriceChangePct: 8.5},
			{Symbol: "ETHUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 45000000, PriceChangePct: 3.0},
			{Symbol: "SOLUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 25000000, PriceChangePct: -2.0},
			{Symbol: "SMALLCOINUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 6000000, PriceChangePct: 50.0},
		},
	}

	screener := NewScreener(client,
		WithAssetFilter(AssetFilter{
			ContractType:   "PERPETUAL",
			QuoteAsset:     "USDT",
			MinVolume24h:   5_000_000,
			MinPriceChange: 5.0,
			Status:         "TRADING",
		}),
		WithMaxPairs(3),
		WithSortBy("volatility"),
	)

	ctx := context.Background()
	err := screener.refresh(ctx)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	pairs := screener.GetActivePairs()
	if len(pairs) != 3 {
		t.Errorf("expected 3 active pairs, got %d: %v", len(pairs), pairs)
	}

	expected := []string{"SMALLCOINUSDT", "1000PEPEUSDT", "WIFUSDT"}
	for i, p := range pairs {
		if p != expected[i] {
			t.Errorf("pair %d: expected %s, got %s", i, expected[i], p)
		}
	}
}

func TestScreener_IncludeSymbols(t *testing.T) {
	client := &mockExchangeClient{
		info: []ExchangeInfo{
			{Symbol: "BTCUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 50000000, PriceChangePct: 1.0},
			{Symbol: "1000PEPEUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 1000000, PriceChangePct: 10.0},
			{Symbol: "WIFUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 2000000, PriceChangePct: 8.0},
		},
	}

	screener := NewScreener(client,
		WithAssetFilter(AssetFilter{
			ContractType:   "PERPETUAL",
			QuoteAsset:     "USDT",
			MinVolume24h:   1_000_000,
			MinPriceChange: 5.0,
			Status:         "TRADING",
			IncludeSymbols: []string{"1000PEPEUSDT", "WIFUSDT"},
		}),
	)

	ctx := context.Background()
	_ = screener.refresh(ctx)

	pairs := screener.GetActivePairs()
	if len(pairs) != 2 {
		t.Errorf("expected 2 pairs (only included symbols), got %d: %v", len(pairs), pairs)
	}
}

func TestScreener_ExcludeSymbols(t *testing.T) {
	client := &mockExchangeClient{
		info: []ExchangeInfo{
			{Symbol: "BTCUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 50000000, PriceChangePct: 5.0},
			{Symbol: "ETHUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 40000000, PriceChangePct: 6.0},
			{Symbol: "1000PEPEUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 10000000, PriceChangePct: 10.0},
		},
	}

	screener := NewScreener(client,
		WithAssetFilter(AssetFilter{
			ContractType:   "PERPETUAL",
			QuoteAsset:     "USDT",
			MinVolume24h:   1_000_000,
			Status:         "TRADING",
			ExcludeSymbols: []string{"BTCUSDT", "ETHUSDT"},
		}),
		WithMaxPairs(10),
	)

	ctx := context.Background()
	_ = screener.refresh(ctx)

	pairs := screener.GetActivePairs()
	if len(pairs) != 1 || pairs[0] != "1000PEPEUSDT" {
		t.Errorf("expected only PEPE, got %v", pairs)
	}
}

func TestScreener_Stats(t *testing.T) {
	client := &mockExchangeClient{
		info: []ExchangeInfo{
			{Symbol: "PEPEUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 10000000, PriceChangePct: 10.0},
			{Symbol: "WIFUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 8000000, PriceChangePct: 8.0},
		},
	}

	screener := NewScreener(client, WithMaxPairs(10))
	ctx := context.Background()
	_ = screener.refresh(ctx)

	stats := screener.Stats()
	if stats.TotalPairs != 2 {
		t.Errorf("expected 2 total pairs, got %d", stats.TotalPairs)
	}
	if stats.ActivePairs != 2 {
		t.Errorf("expected 2 active pairs, got %d", stats.ActivePairs)
	}
	if stats.AvgVolume != 9000000 {
		t.Errorf("expected avg volume 9000000, got %f", stats.AvgVolume)
	}
	if stats.AvgChange != 9.0 {
		t.Errorf("expected avg change 9.0, got %f", stats.AvgChange)
	}
}

func TestScreener_IsMonitoring(t *testing.T) {
	client := &mockExchangeClient{
		info: []ExchangeInfo{
			{Symbol: "PEPEUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 10000000, PriceChangePct: 10.0},
			{Symbol: "WIFUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 8000000, PriceChangePct: 8.0},
		},
	}

	screener := NewScreener(client, WithMaxPairs(5))
	ctx := context.Background()
	_ = screener.refresh(ctx)

	if !screener.IsMonitoring("PEPEUSDT") {
		t.Error("PEPEUSDT should be monitored")
	}
	if !screener.IsMonitoring("WIFUSDT") {
		t.Error("WIFUSDT should be monitored")
	}
	if screener.IsMonitoring("NOTEXIST") {
		t.Error("NOTEXIST should not be monitored")
	}
}

func TestScreener_GetScore(t *testing.T) {
	client := &mockExchangeClient{
		info: []ExchangeInfo{
			{Symbol: "PEPEUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 15000000, PriceChangePct: 15.0},
		},
	}

	screener := NewScreener(client, WithSortBy("volatility"))
	ctx := context.Background()
	_ = screener.refresh(ctx)

	score := screener.GetScore("PEPEUSDT")
	if score != 15.0 {
		t.Errorf("expected score 15.0, got %f", score)
	}
}

func TestScreener_SortByVolume(t *testing.T) {
	client := &mockExchangeClient{
		info: []ExchangeInfo{
			{Symbol: "LOWVOLUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 5000000, PriceChangePct: 20.0},
			{Symbol: "HIGHVOLUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 20000000, PriceChangePct: 5.0},
			{Symbol: "MIDVOLUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 10000000, PriceChangePct: 10.0},
		},
	}

	screener := NewScreener(client,
		WithAssetFilter(AssetFilter{
			ContractType: "PERPETUAL",
			QuoteAsset:   "USDT",
			MinVolume24h: 1_000_000,
			Status:       "TRADING",
		}),
		WithSortBy("volume"),
		WithMaxPairs(3),
	)

	ctx := context.Background()
	_ = screener.refresh(ctx)

	pairs := screener.GetActivePairs()
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	expected := []string{"HIGHVOLUSDT", "MIDVOLUSDT", "LOWVOLUSDT"}
	for i, p := range pairs {
		if p != expected[i] {
			t.Errorf("sort by volume - pair %d: expected %s, got %s", i, expected[i], p)
		}
	}
}

func TestScreener_ErrorHandling(t *testing.T) {
	client := &mockExchangeClient{
		err: context.DeadlineExceeded,
	}

	screener := NewScreener(client)
	ctx := context.Background()

	err := screener.refresh(ctx)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestScreener_ToAssets(t *testing.T) {
	client := &mockExchangeClient{
		info: []ExchangeInfo{
			{Symbol: "PEPEUSDT", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 10000000, PriceChangePct: 10.0, LastUpdated: time.Now()},
		},
	}

	screener := NewScreener(client)
	ctx := context.Background()
	_ = screener.refresh(ctx)

	assets := screener.ToAssets()
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	if assets[0].Symbol != "PEPEUSDT" {
		t.Errorf("expected symbol PEPEUSDT, got %s", assets[0].Symbol)
	}
	if assets[0].Volume24h != 10000000 {
		t.Errorf("expected volume 10000000, got %f", assets[0].Volume24h)
	}
}

func TestSymbolChecker(t *testing.T) {
	checker := NewSymbolChecker(
		[]string{"PEPE", "WIF", "MOG"},
		[]string{"BTC", "ETH", "SOL"},
	)

	tests := []struct {
		symbol   string
		expected bool
	}{
		{"PEPE", true},
		{"WIF", true},
		{"MOG", true},
		{"BTC", false},
		{"ETH", false},
		{"SOL", false},
		{"UNKNOWN", false},
		{"pepe", true},
		{"PePe", true},
	}

	for _, tt := range tests {
		result := checker.IsAllowed(tt.symbol)
		if result != tt.expected {
			t.Errorf("IsAllowed(%s): expected %v, got %v", tt.symbol, tt.expected, result)
		}
	}
}

func TestDefaultMemeCoinFilter(t *testing.T) {
	filter := DefaultMemeCoinFilter()

	if filter.ContractType != "PERPETUAL" {
		t.Errorf("expected contract type PERPETUAL, got %s", filter.ContractType)
	}
	if filter.QuoteAsset != "USDT" {
		t.Errorf("expected quote asset USDT, got %s", filter.QuoteAsset)
	}
	if filter.MinVolume24h != 5_000_000 {
		t.Errorf("expected min volume 5000000, got %f", filter.MinVolume24h)
	}
	if len(filter.IncludeSymbols) == 0 {
		t.Error("expected include symbols to be set")
	}
	if len(filter.ExcludeSymbols) == 0 {
		t.Error("expected exclude symbols to be set")
	}
}

func TestHighVolatilityFilter(t *testing.T) {
	cfg := HighVolatilityFilter()

	if cfg.MaxPairs != 3 {
		t.Errorf("expected max pairs 3, got %d", cfg.MaxPairs)
	}
	if cfg.SortBy != "volatility" {
		t.Errorf("expected sort by volatility, got %s", cfg.SortBy)
	}
	if cfg.Filter.MinPriceChange != 15.0 {
		t.Errorf("expected min price change 15.0, got %f", cfg.Filter.MinPriceChange)
	}
	if cfg.Filter.MinVolume24h != 10_000_000 {
		t.Errorf("expected min volume 10000000, got %f", cfg.Filter.MinVolume24h)
	}
}

func TestVolumeBasedFilter(t *testing.T) {
	cfg := VolumeBasedFilter()

	if cfg.MaxPairs != 10 {
		t.Errorf("expected max pairs 10, got %d", cfg.MaxPairs)
	}
	if cfg.SortBy != "volume" {
		t.Errorf("expected sort by volume, got %s", cfg.SortBy)
	}
	if cfg.Filter.MinVolume24h != 20_000_000 {
		t.Errorf("expected min volume 20000000, got %f", cfg.Filter.MinVolume24h)
	}
}

func TestScreener_ConfidenceCalculation(t *testing.T) {
	client := &mockExchangeClient{
		info: []ExchangeInfo{
			{Symbol: "HIGHVOLHIGHCHANGE", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 15000000, PriceChangePct: 15.0},
			{Symbol: "MIDCONFIDENCE", ContractType: "PERPETUAL", QuoteAsset: "USDT", Status: "TRADING", Volume24h: 7000000, PriceChangePct: 7.0},
		},
	}

	screener := NewScreener(client,
		WithAssetFilter(AssetFilter{
			ContractType:   "PERPETUAL",
			QuoteAsset:     "USDT",
			MinVolume24h:   1_000_000,
			Status:         "TRADING",
			IncludeSymbols: []string{"HIGHVOLHIGHCHANGE", "MIDCONFIDENCE"},
		}),
	)

	ctx := context.Background()
	_ = screener.refresh(ctx)

	assets := screener.ToAssets()
	for _, a := range assets {
		if a.Confidence > 1.0 {
			t.Errorf("confidence should be <= 1.0, got %f for %s", a.Confidence, a.Symbol)
		}
	}
}
