package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ScreenerClient struct {
	cfg    Config
	client *http.Client
}

type ExchangeInfo struct {
	Symbol         string
	ContractType   string
	QuoteAsset     string
	Status         string
	Volume24h      float64
	PriceChangePct float64
	LastUpdated    time.Time
}

type Ticker24hr struct {
	Symbol             string `json:"symbol"`
	Price              string `json:"price"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
}

type SymbolInfo struct {
	Symbol       string `json:"symbol"`
	ContractType string `json:"contractType"`
	QuoteAsset   string `json:"quoteAsset"`
	Status       string `json:"status"`
}

type ExchangeInfoResponse struct {
	Symbols []SymbolInfo `json:"symbols"`
}

func NewScreenerClient(cfg Config) *ScreenerClient {
	if cfg.BaseURL == "" {
		if cfg.Testnet {
			cfg.BaseURL = "https://testnet.binancefuture.com"
		} else {
			cfg.BaseURL = "https://fapi.binance.com"
		}
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	return &ScreenerClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *ScreenerClient) GetExchangeInfo(ctx context.Context) ([]ExchangeInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	symbolURL := fmt.Sprintf("%s/fapi/v1/exchangeInfo", c.cfg.BaseURL)
	tickerURL := fmt.Sprintf("%s/fapi/v1/ticker/24hr", c.cfg.BaseURL)

	symbolReq, err := http.NewRequestWithContext(ctx, http.MethodGet, symbolURL, nil)
	if err != nil {
		return nil, err
	}

	symbolResp, err := c.client.Do(symbolReq)
	if err != nil {
		return nil, err
	}
	defer symbolResp.Body.Close()

	symbolBody, _ := io.ReadAll(symbolResp.Body)

	var exchangeResp ExchangeInfoResponse
	if err := json.Unmarshal(symbolBody, &exchangeResp); err != nil {
		return nil, fmt.Errorf("failed to parse exchange info: %w", err)
	}

	tickerReq, err := http.NewRequestWithContext(ctx, http.MethodGet, tickerURL, nil)
	if err != nil {
		return nil, err
	}

	tickerResp, err := c.client.Do(tickerReq)
	if err != nil {
		return nil, err
	}
	defer tickerResp.Body.Close()

	tickerBody, _ := io.ReadAll(tickerResp.Body)

	var tickers []Ticker24hr
	if err := json.Unmarshal(tickerBody, &tickers); err != nil {
		return nil, fmt.Errorf("failed to parse tickers: %w", err)
	}

	tickerMap := make(map[string]*Ticker24hr)
	for i := range tickers {
		tickerMap[tickers[i].Symbol] = &tickers[i]
	}

	pairs := make([]ExchangeInfo, 0, len(exchangeResp.Symbols))

	for _, symbol := range exchangeResp.Symbols {
		ticker, ok := tickerMap[symbol.Symbol]
		if !ok {
			continue
		}

		vol, _ := strconv.ParseFloat(ticker.QuoteVolume, 64)
		change, _ := strconv.ParseFloat(ticker.PriceChangePercent, 64)

		pairs = append(pairs, ExchangeInfo{
			Symbol:         symbol.Symbol,
			ContractType:   symbol.ContractType,
			QuoteAsset:     symbol.QuoteAsset,
			Status:         symbol.Status,
			Volume24h:      vol,
			PriceChangePct: change,
			LastUpdated:    time.Now(),
		})
	}

	return pairs, nil
}

func (c *ScreenerClient) GetUSDMFuturesPairs(ctx context.Context) ([]ExchangeInfo, error) {
	allPairs, err := c.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]ExchangeInfo, 0)
	for _, p := range allPairs {
		if p.ContractType == "PERPETUAL" && p.QuoteAsset == "USDT" {
			filtered = append(filtered, p)
		}
	}

	return filtered, nil
}

func (c *ScreenerClient) GetTopMemeCoins(ctx context.Context, limit int) ([]ExchangeInfo, error) {
	pairs, err := c.GetUSDMFuturesPairs(ctx)
	if err != nil {
		return nil, err
	}

	memeKeywords := []string{
		"PEPE", "WIF", "POPCAT", "TURBO", "MOG", "FWOG", "MEW", "ACT",
		"LUNA", "NEIRO", "BOME", "SLERF", "BONK", "JUP", "WEN",
		"MEME", "DOGE", "SHIB", "FLOKI", "ARBIT", "STRK", "SUNDOG",
		"PIPPIN", "GRIFFAIN", "NEG",
	}

	memePairs := make([]ExchangeInfo, 0)
	otherPairs := make([]ExchangeInfo, 0)

	for _, p := range pairs {
		isMeme := false
		symbolUpper := strings.ToUpper(p.Symbol)
		for _, keyword := range memeKeywords {
			if strings.Contains(symbolUpper, keyword) {
				isMeme = true
				break
			}
		}

		if isMeme {
			memePairs = append(memePairs, p)
		} else {
			otherPairs = append(otherPairs, p)
		}
	}

	if limit <= 0 {
		limit = 10
	}

	result := make([]ExchangeInfo, 0, limit)

	sort.Slice(memePairs, func(i, j int) bool {
		return memePairs[i].Volume24h > memePairs[j].Volume24h
	})
	sort.Slice(otherPairs, func(i, j int) bool {
		return otherPairs[i].Volume24h > otherPairs[j].Volume24h
	})

	for _, p := range memePairs {
		if len(result) >= limit {
			break
		}
		result = append(result, p)
	}

	for _, p := range otherPairs {
		if len(result) >= limit {
			break
		}
		result = append(result, p)
	}

	return result, nil
}

func (c *ScreenerClient) GetVolume24h(symbol string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/fapi/v1/ticker/24hr?symbol=%s", c.cfg.BaseURL, symbol)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var ticker Ticker24hr
	if err := json.Unmarshal(body, &ticker); err != nil {
		return 0, err
	}

	vol, _ := strconv.ParseFloat(ticker.QuoteVolume, 64)
	return vol, nil
}
