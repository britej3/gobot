// Package platform provides market data integration per technical specifications
package platform

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CGResponse is the CoinGecko API response structure
type CGResponse struct {
	MarketData struct {
		CirculatingSupply float64 `json:"circulating_supply"`
	} `json:"market_data"`
}

// MarketCapCache provides cached market cap data per reply_unknown.md
type MarketCapCache struct {
	data map[string]CachedData
	mu   sync.RWMutex
}

type CachedData struct {
	CirculatingSupply float64
	Timestamp         time.Time
}

// NewMarketCapCache creates a new market cap cache (24h per reply_unknown.md)
func NewMarketCapCache() *MarketCapCache {
	return &MarketCapCache{
		data: make(map[string]CachedData),
	}
}

// GetMarketCap returns market cap for a symbol, using cache if available
func (c *MarketCapCache) GetMarketCap(symbol string, price float64) (float64, string, error) {
	c.mu.RLock()
	cached, exists := c.data[symbol]
	c.mu.RUnlock()
	
	// Return cached if less than 24 hours old (per reply_unknown.md)
	if exists && time.Since(cached.Timestamp) < 24*time.Hour {
		marketCap := price * cached.CirculatingSupply
		return marketCap, "cached", nil
	}
	
	// Fetch from CoinGecko
	supply, err := c.fetchCirculatingSupply(symbol)
	if err != nil {
		// Fallback: use cached if exists (reply_unknown.md fallback)
		if exists {
			marketCap := price * cached.CirculatingSupply
			return marketCap, "stale_cache", nil
		}
		// No cache available: flag as high risk (reply_unknown.md)
		return 0, "high_risk", fmt.Errorf("no cached data and API failed: %w", err)
	}
	
	// Cache the result
	c.mu.Lock()
	c.data[symbol] = CachedData{
		CirculatingSupply: supply,
		Timestamp:         time.Now(),
	}
	c.mu.Unlock()
	
	marketCap := price * supply
	return marketCap, "fresh", nil
}

// fetchCirculatingSupply pulls from CoinGecko public API per specifications
// Note: Use https://pro-api.coingecko.com for Pro key as mentioned in specs
func (c *MarketCapCache) fetchCirculatingSupply(coinID string) (float64, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s?localization=false&tickers=false&community_data=false&developer_data=false&sparkline=false", coinID)
	
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("coingecko API error: %d", resp.StatusCode)
	}
	
	var data CGResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	
	if data.MarketData.CirculatingSupply == 0 {
		return 0, fmt.Errorf("no circulating supply data available")
	}
	
	return data.MarketData.CirculatingSupply, nil
}

// ClearCache clears the market cap cache
func (c *MarketCapCache) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]CachedData)
}
