package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestScreenerAdapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fapi/v1/exchangeInfo" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"symbols": [
					{"symbol": "1000PEPEUSDT", "contractType": "PERPETUAL", "quoteAsset": "USDT", "status": "TRADING"},
					{"symbol": "WIFUSDT", "contractType": "PERPETUAL", "quoteAsset": "USDT", "status": "TRADING"},
					{"symbol": "BTCUSDT", "contractType": "PERPETUAL", "quoteAsset": "USDT", "status": "TRADING"}
				]
			}`))
		} else if r.URL.Path == "/fapi/v1/ticker/24hr" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{"symbol": "1000PEPEUSDT", "quoteVolume": "10000000", "priceChangePercent": "15.0"},
				{"symbol": "WIFUSDT", "quoteVolume": "8000000", "priceChangePercent": "8.5"},
				{"symbol": "BTCUSDT", "quoteVolume": "50000000", "priceChangePercent": "2.0"}
			]`))
		}
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL, Timeout: 5 * time.Second}
	client := NewScreenerClient(cfg)
	adapter := NewScreenerAdapter(client)

	ctx := context.Background()
	info, err := adapter.GetExchangeInfo(ctx)
	if err != nil {
		t.Fatalf("GetExchangeInfo failed: %v", err)
	}

	if len(info) != 3 {
		t.Errorf("expected 3 pairs, got %d", len(info))
	}

	for _, p := range info {
		if p.Symbol == "1000PEPEUSDT" {
			if p.Volume24h != 10000000 {
				t.Errorf("PEPE volume expected 10000000, got %f", p.Volume24h)
			}
			if p.PriceChangePct != 15.0 {
				t.Errorf("PEPE change expected 15.0, got %f", p.PriceChangePct)
			}
		}
	}
}

func TestScreenerAdapter_TopMemeCoins(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fapi/v1/exchangeInfo" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"symbols": [
					{"symbol": "1000PEPEUSDT", "contractType": "PERPETUAL", "quoteAsset": "USDT", "status": "TRADING"},
					{"symbol": "WIFUSDT", "contractType": "PERPETUAL", "quoteAsset": "USDT", "status": "TRADING"},
					{"symbol": "BTCUSDT", "contractType": "PERPETUAL", "quoteAsset": "USDT", "status": "TRADING"}
				]
			}`))
		} else if r.URL.Path == "/fapi/v1/ticker/24hr" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{"symbol": "1000PEPEUSDT", "quoteVolume": "15000000", "priceChangePercent": "12.0"},
				{"symbol": "WIFUSDT", "quoteVolume": "10000000", "priceChangePercent": "10.0"},
				{"symbol": "BTCUSDT", "quoteVolume": "50000000", "priceChangePercent": "1.5"}
			]`))
		}
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL, Timeout: 5 * time.Second}
	client := NewScreenerClient(cfg)
	adapter := NewScreenerAdapter(client)

	ctx := context.Background()
	pairs, err := adapter.GetTopMemeCoins(ctx, 2)
	if err != nil {
		t.Fatalf("GetTopMemeCoins failed: %v", err)
	}

	if len(pairs) != 2 {
		t.Errorf("expected 2 meme coins, got %d", len(pairs))
	}

	if pairs[0].Symbol != "1000PEPEUSDT" {
		t.Errorf("expected PEPE first by volume, got %s", pairs[0].Symbol)
	}
}

func TestScreenerClient_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fapi/v1/exchangeInfo" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"symbols": [{"symbol": "TEST", "contractType": "PERPETUAL", "quoteAsset": "USDT", "status": "TRADING"}]}`))
		} else if r.URL.Path == "/fapi/v1/ticker/24hr" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"symbol": "TEST", "quoteVolume": "12345.67", "priceChangePercent": "5.0"}]`))
		}
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL}
	client := NewScreenerClient(cfg)

	ctx := context.Background()
	info, err := client.GetExchangeInfo(ctx)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	if len(info) != 1 || info[0].Symbol != "TEST" {
		t.Error("unexpected result")
	}
}

func TestScreenerClient_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL}
	client := NewScreenerClient(cfg)

	ctx := context.Background()
	_, err := client.GetExchangeInfo(ctx)
	if err == nil {
		t.Error("expected error")
	}
}

func TestScreenerClient_GetVolume24h(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("symbol") == "PEPEUSDT" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"symbol": "PEPEUSDT", "quoteVolume": "12345678.90"}`))
		}
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL}
	client := NewScreenerClient(cfg)

	vol, err := client.GetVolume24h("PEPEUSDT")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	if vol != 12345678.90 {
		t.Errorf("expected 12345678.90, got %f", vol)
	}
}

func TestTicker24hr_JSON(t *testing.T) {
	data := `{
		"symbol": "PEPEUSDT",
		"price": "0.00001234",
		"priceChange": "0.00000123",
		"priceChangePercent": "11.11",
		"volume": "999999999",
		"quoteVolume": "12345.67"
	}`

	var ticker Ticker24hr
	if err := json.Unmarshal([]byte(data), &ticker); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if ticker.Symbol != "PEPEUSDT" {
		t.Errorf("symbol mismatch: %s", ticker.Symbol)
	}
	if ticker.PriceChangePercent != "11.11" {
		t.Errorf("priceChangePercent mismatch: %s", ticker.PriceChangePercent)
	}
}

func TestSymbolInfo_JSON(t *testing.T) {
	data := `{
		"symbol": "1000PEPEUSDT",
		"contractType": "PERPETUAL",
		"quoteAsset": "USDT",
		"status": "TRADING"
	}`

	var info SymbolInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if info.Symbol != "1000PEPEUSDT" {
		t.Errorf("symbol mismatch: %s", info.Symbol)
	}
	if info.ContractType != "PERPETUAL" {
		t.Errorf("contractType mismatch: %s", info.ContractType)
	}
}
