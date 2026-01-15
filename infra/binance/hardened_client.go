package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/britebrt/cognee/domain/trade"
	"github.com/britebrt/cognee/pkg/circuitbreaker"
	"golang.org/x/time/rate"
)

type HardenedConfig struct {
	APIKey            string
	APISecret         string
	BaseURL           string
	Testnet           bool
	RateLimitRPS      float64
	RateBurst         int
	Timeout           time.Duration
	RecvWindow        time.Duration
	SignatureVariance float64
}

type HardenedClient struct {
	cfg            HardenedConfig
	client         *http.Client
	limiter        *rate.Limiter
	circuitBreaker *circuitbreaker.CircuitBreaker
	requestCache   *RequestCache
	mu             sync.RWMutex
	lastRequest    time.Time
	minInterval    time.Duration
}

type RequestCache struct {
	mu       sync.RWMutex
	cache    map[string]cacheEntry
	duration time.Duration
}

type cacheEntry struct {
	response interface{}
	expiry   time.Time
}

func NewHardenedClient(cfg HardenedConfig) *HardenedClient {
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
	if cfg.RateLimitRPS == 0 {
		cfg.RateLimitRPS = 8
	}
	if cfg.RateBurst == 0 {
		cfg.RateBurst = 16
	}
	if cfg.RecvWindow == 0 {
		cfg.RecvWindow = 5000 * time.Millisecond
	}
	if cfg.SignatureVariance == 0 {
		cfg.SignatureVariance = 0.01
	}

	return &HardenedClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		limiter: rate.NewLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateBurst),
		circuitBreaker: circuitbreaker.New(circuitbreaker.CircuitBreakerConfig{
			Name:             "binance-api",
			FailureThreshold: 5,
			RecoveryTimeout:  30 * time.Second,
			FailureWindow:    60 * time.Second,
			HalfOpenRequests: 3,
		}),
		requestCache: &RequestCache{
			cache:    make(map[string]cacheEntry),
			duration: 5 * time.Second,
		},
		minInterval: 50 * time.Millisecond,
	}
}

func (c *HardenedClient) CreateOrder(ctx context.Context, order *trade.Order) (*trade.Order, error) {
	return circuitbreaker.Execute(c.circuitBreaker, func() (*trade.Order, error) {
		c.waitForRateLimit(ctx)

		endpoint := fmt.Sprintf("%s/fapi/v1/order", c.cfg.BaseURL)

		params := url.Values{}
		params.Set("symbol", order.Symbol)
		params.Set("side", string(order.Side))
		params.Set("type", string(order.Type))
		params.Set("quantity", strconv.FormatFloat(order.Quantity, 'f', -1, 64))

		if order.Type == trade.OrderTypeLimit {
			params.Set("price", strconv.FormatFloat(order.Price, 'f', -1, 64))
			params.Set("timeInForce", "GTC")
		}

		if order.StopLoss > 0 {
			params.Set("stopPrice", strconv.FormatFloat(order.StopLoss, 'f', -1, 64))
			params.Set("workingType", "MARK_PRICE")
		}

		timestamp := time.Now().UnixMilli() + int64(rand.Float64()*100)
		params.Set("timestamp", strconv.FormatInt(timestamp, 10))
		params.Set("recvWindow", strconv.FormatInt(int64(c.cfg.RecvWindow.Milliseconds()), 10))

		signature := c.sign(params.Encode())
		params.Set("signature", signature)

		body := strings.NewReader(params.Encode())

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)
		req.Header.Set("X-MBX-USER-IP", c.getRandomIP())

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, c.parseError(respBody)
		}

		var result struct {
			OrderID     int64   `json:"orderId"`
			Symbol      string  `json:"symbol"`
			Status      string  `json:"status"`
			Side        string  `json:"side"`
			Type        string  `json:"type"`
			Price       float64 `json:"price"`
			AvgPrice    float64 `json:"avgPrice"`
			OrigQty     float64 `json:"origQty"`
			ExecutedQty float64 `json:"executedQty"`
			StopPrice   float64 `json:"stopPrice"`
			UpdateTime  int64   `json:"updateTime"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		order.ID = strconv.FormatInt(result.OrderID, 10)
		order.Status = trade.OrderStatus(result.Status)
		order.AvgFillPrice = result.AvgPrice
		order.FilledQty = result.ExecutedQty
		order.UpdatedAt = time.UnixMilli(result.UpdateTime)

		return order, nil
	})
}

func (c *HardenedClient) GetOrder(ctx context.Context, orderID, symbol string) (*trade.Order, error) {
	return circuitbreaker.Execute(c.circuitBreaker, func() (*trade.Order, error) {
		c.waitForRateLimit(ctx)

		cacheKey := fmt.Sprintf("order:%s:%s", symbol, orderID)
		if cached := c.requestCache.Get(cacheKey); cached != nil {
			if order, ok := cached.(*trade.Order); ok {
				return order, nil
			}
		}

		endpoint := fmt.Sprintf("%s/fapi/v1/order", c.cfg.BaseURL)

		params := url.Values{}
		params.Set("orderId", orderID)
		params.Set("symbol", symbol)
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli()+int64(rand.Float64()*100), 10))
		params.Set("recvWindow", strconv.FormatInt(int64(c.cfg.RecvWindow.Milliseconds()), 10))

		signature := c.sign(params.Encode())
		params.Set("signature", signature)

		url := endpoint + "?" + params.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)
		req.Header.Set("X-MBX-USER-IP", c.getRandomIP())

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, c.parseError(respBody)
		}

		var result struct {
			OrderID     int64   `json:"orderId"`
			Symbol      string  `json:"symbol"`
			Status      string  `json:"status"`
			Side        string  `json:"side"`
			Type        string  `json:"type"`
			Price       float64 `json:"price"`
			AvgPrice    float64 `json:"avgPrice"`
			OrigQty     float64 `json:"origQty"`
			ExecutedQty float64 `json:"executedQty"`
			UpdateTime  int64   `json:"updateTime"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		order := &trade.Order{
			ID:           strconv.FormatInt(result.OrderID, 10),
			Symbol:       result.Symbol,
			Side:         trade.Side(result.Side),
			Type:         trade.OrderType(result.Type),
			Price:        result.Price,
			AvgFillPrice: result.AvgPrice,
			Quantity:     result.OrigQty,
			FilledQty:    result.ExecutedQty,
			Status:       trade.OrderStatus(result.Status),
			UpdatedAt:    time.UnixMilli(result.UpdateTime),
		}

		c.requestCache.Set(cacheKey, order)

		return order, nil
	})
}

func (c *HardenedClient) GetPosition(ctx context.Context, symbol string) (*trade.Position, error) {
	return circuitbreaker.Execute(c.circuitBreaker, func() (*trade.Position, error) {
		c.waitForRateLimit(ctx)

		endpoint := fmt.Sprintf("%s/fapi/v2/positionRisk", c.cfg.BaseURL)

		params := url.Values{}
		params.Set("symbol", symbol)
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli()+int64(rand.Float64()*100), 10))
		params.Set("recvWindow", strconv.FormatInt(int64(c.cfg.RecvWindow.Milliseconds()), 10))

		signature := c.sign(params.Encode())
		params.Set("signature", signature)

		url := endpoint + "?" + params.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)
		req.Header.Set("X-MBX-USER-IP", c.getRandomIP())

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, c.parseError(respBody)
		}

		var result []struct {
			Symbol           string  `json:"symbol"`
			PositionSide     string  `json:"positionSide"`
			PositionAmt      float64 `json:"positionAmt"`
			EntryPrice       float64 `json:"entryPrice"`
			MarkPrice        float64 `json:"markPrice"`
			UnRealizedProfit float64 `json:"unRealizedProfit"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		for _, pos := range result {
			if pos.Symbol == symbol && pos.PositionAmt != 0 {
				side := trade.SideBuy
				if pos.PositionSide == "SHORT" {
					side = trade.SideSell
				}

				pnlPercent := 0.0
				if pos.EntryPrice > 0 {
					pnlPercent = (pos.MarkPrice - pos.EntryPrice) / pos.EntryPrice * 100
					if side == trade.SideSell {
						pnlPercent = -pnlPercent
					}
				}

				return &trade.Position{
					Symbol:       symbol,
					Side:         side,
					Quantity:     pos.PositionAmt,
					EntryPrice:   pos.EntryPrice,
					CurrentPrice: pos.MarkPrice,
					PnL:          pos.UnRealizedProfit,
					PnLPercent:   pnlPercent,
					UpdatedAt:    time.Now(),
				}, nil
			}
		}

		return nil, trade.ErrPositionNotFound
	})
}

func (c *HardenedClient) GetBalance(ctx context.Context) (float64, error) {
	return circuitbreaker.Execute(c.circuitBreaker, func() (float64, error) {
		c.waitForRateLimit(ctx)

		endpoint := fmt.Sprintf("%s/fapi/v2/balance", c.cfg.BaseURL)

		params := url.Values{}
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli()+int64(rand.Float64()*100), 10))
		params.Set("recvWindow", strconv.FormatInt(int64(c.cfg.RecvWindow.Milliseconds()), 10))

		signature := c.sign(params.Encode())
		params.Set("signature", signature)

		url := endpoint + "?" + params.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)
		req.Header.Set("X-MBX-USER-IP", c.getRandomIP())

		resp, err := c.client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}

		if resp.StatusCode != http.StatusOK {
			return 0, c.parseError(respBody)
		}

		var result []struct {
			Asset   string  `json:"asset"`
			Balance float64 `json:"balance"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			return 0, fmt.Errorf("failed to parse response: %w", err)
		}

		for _, bal := range result {
			if bal.Asset == "USDT" {
				return bal.Balance, nil
			}
		}

		return 0, nil
	})
}

func (c *HardenedClient) Price(ctx context.Context, symbol string) (float64, error) {
	cacheKey := fmt.Sprintf("price:%s", symbol)
	if cached := c.requestCache.Get(cacheKey); cached != nil {
		if price, ok := cached.(float64); ok {
			return price, nil
		}
	}

	c.waitForRateLimit(ctx)

	endpoint := fmt.Sprintf("%s/fapi/v1/ticker/price", c.cfg.BaseURL)

	params := url.Values{}
	params.Set("symbol", symbol)

	url := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-MBX-USER-IP", c.getRandomIP())

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, c.parseError(respBody)
	}

	var result struct {
		Price float64 `json:"price"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	c.requestCache.Set(cacheKey, result.Price)

	return result.Price, nil
}

func (c *HardenedClient) Kline(ctx context.Context, symbol, interval string, limit int) ([]trade.Kline, error) {
	c.waitForRateLimit(ctx)

	endpoint := fmt.Sprintf("%s/fapi/v1/klines", c.cfg.BaseURL)

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	params.Set("limit", strconv.Itoa(limit))

	url := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(respBody)
	}

	var raw [][]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var klines []trade.Kline
	for _, k := range raw {
		klines = append(klines, trade.Kline{
			OpenTime:  time.UnixMilli(int64(k[0].(float64))),
			Open:      k[1].(float64),
			High:      k[2].(float64),
			Low:       k[3].(float64),
			Close:     k[4].(float64),
			Volume:    k[5].(float64),
			CloseTime: time.UnixMilli(int64(k[6].(float64))),
		})
	}

	return klines, nil
}

func (c *HardenedClient) waitForRateLimit(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.limiter.Wait(ctx); err != nil {
		return
	}

	since := time.Since(c.lastRequest)
	if since < c.minInterval {
		jitter := time.Duration(rand.Float64() * float64(c.cfg.SignatureVariance*float64(c.minInterval)))
		time.Sleep(c.minInterval - since + jitter)
	}
	c.lastRequest = time.Now()
}

func (c *HardenedClient) sign(payload string) string {
	h := hmac.New(sha256.New, []byte(c.cfg.APISecret))
	h.Write([]byte(payload))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (c *HardenedClient) getRandomIP() string {
	return fmt.Sprintf("192.168.%d.%d", rand.Intn(256), rand.Intn(256))
}

func (c *HardenedClient) parseError(respBody []byte) error {
	var errResp struct {
		Code int64  `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		return fmt.Errorf("unknown error: %s", string(respBody))
	}
	return fmt.Errorf("binance API error %d: %s", errResp.Code, errResp.Msg)
}

func (c *RequestCache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.cache[key]
	if !ok || time.Now().After(entry.expiry) {
		return nil
	}
	return entry.response
}

func (c *RequestCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = cacheEntry{
		response: value,
		expiry:   time.Now().Add(c.duration),
	}
}

func (c *HardenedClient) GetCircuitBreakerStats() circuitbreaker.Stats {
	return c.circuitBreaker.GetStats()
}
