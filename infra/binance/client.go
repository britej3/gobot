package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/britebrt/cognee/domain/trade"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type Config struct {
	APIKey    string
	APISecret string
	BaseURL   string
	Testnet   bool
	RateLimit rate.Limit
	RateBurst int
	Timeout   time.Duration
}

type Client struct {
	cfg     Config
	client  *http.Client
	limiter *rate.Limiter
}

type APIResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data json.RawMessage
}

func New(cfg Config) *Client {
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
	if cfg.RateLimit == 0 {
		cfg.RateLimit = rate.Inf
	}
	if cfg.RateBurst == 0 {
		cfg.RateBurst = 10
	}

	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		limiter: rate.NewLimiter(cfg.RateLimit, cfg.RateBurst),
	}
}

func (c *Client) CreateOrder(ctx context.Context, order *trade.Order) (*trade.Order, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

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

	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))
	params.Set("recvWindow", "5000")

	signature := c.sign(params.Encode())
	params.Set("signature", signature)

	body := strings.NewReader(params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)

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
}

func (c *Client) CancelOrder(ctx context.Context, orderID string) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/fapi/v1/order", c.cfg.BaseURL)

	params := url.Values{}
	params.Set("orderId", orderID)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	signature := c.sign(params.Encode())
	params.Set("signature", signature)

	body := strings.NewReader(params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return c.parseError(respBody)
	}

	return nil
}

func (c *Client) GetOrder(ctx context.Context, orderID string) (*trade.Order, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/fapi/v1/order", c.cfg.BaseURL)

	params := url.Values{}
	params.Set("orderId", orderID)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	signature := c.sign(params.Encode())
	params.Set("signature", signature)

	url := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)

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

	return &trade.Order{
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
	}, nil
}

func (c *Client) GetPosition(ctx context.Context, symbol string) (*trade.Position, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/fapi/v2/positionRisk", c.cfg.BaseURL)

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	signature := c.sign(params.Encode())
	params.Set("signature", signature)

	url := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)

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
}

func (c *Client) GetBalance(ctx context.Context) (float64, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return 0, err
	}

	endpoint := fmt.Sprintf("%s/fapi/v2/balance", c.cfg.BaseURL)

	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	signature := c.sign(params.Encode())
	params.Set("signature", signature)

	url := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)

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
}

func (c *Client) ClosePosition(ctx context.Context, position *trade.Position) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/fapi/v1/order", c.cfg.BaseURL)

	side := trade.SideSell
	if position.Side == trade.SideSell {
		side = trade.SideBuy
	}

	params := url.Values{}
	params.Set("symbol", position.Symbol)
	params.Set("side", string(side))
	params.Set("type", "MARKET")
	params.Set("quantity", strconv.FormatFloat(position.Quantity, 'f', -1, 64))
	params.Set("reduceOnly", "true")
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	signature := c.sign(params.Encode())
	params.Set("signature", signature)

	body := strings.NewReader(params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return c.parseError(respBody)
	}

	return nil
}

func (c *Client) Symbols(ctx context.Context) ([]string, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/fapi/v1/exchangeInfo", c.cfg.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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

	var result struct {
		Symbols []struct {
			Symbol string `json:"symbol"`
			Status string `json:"status"`
		} `json:"symbols"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var symbols []string
	for _, s := range result.Symbols {
		if s.Status == "TRADING" {
			symbols = append(symbols, s.Symbol)
		}
	}

	return symbols, nil
}

func (c *Client) Kline(ctx context.Context, symbol, interval string, limit int) ([]trade.Kline, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

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

func (c *Client) Price(ctx context.Context, symbol string) (float64, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return 0, err
	}

	endpoint := fmt.Sprintf("%s/fapi/v1/ticker/price", c.cfg.BaseURL)

	params := url.Values{}
	params.Set("symbol", symbol)

	url := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

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

	return result.Price, nil
}

func (c *Client) sign(payload string) string {
	h := c.cfg.APISecret + payload
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(h)).String()[0:32]
}

func (c *Client) parseError(respBody []byte) error {
	var errResp APIResponse
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		return fmt.Errorf("unknown error: %s", string(respBody))
	}
	return fmt.Errorf("binance API error %d: %s", errResp.Code, errResp.Msg)
}
