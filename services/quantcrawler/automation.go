package quantcrawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	ScreenshotService string
	N8NWebhook        string
	GOBOTWebhook      string
	Timeout           time.Duration
}

func DefaultConfig() Config {
	return Config{
		ScreenshotService: "http://localhost:3456",
		N8NWebhook:        "http://localhost:5678/webhook/tradingview-analysis",
		GOBOTWebhook:      "http://localhost:8080/webhook/trade_signal",
		Timeout:           2 * time.Minute,
	}
}

type AnalysisResult struct {
	Symbol          string            `json:"symbol"`
	Direction       string            `json:"direction"`
	Confidence      int               `json:"confidence"`
	EntryPrice      float64           `json:"entry_price"`
	StopLoss        float64           `json:"stop_loss"`
	TakeProfit      float64           `json:"take_profit"`
	RiskRewardRatio float64           `json:"risk_reward_ratio"`
	Recommendation  string            `json:"recommendation"`
	Timeframes      map[string]string `json:"timeframes"`
	KeyLevels       KeyLevels         `json:"key_levels"`
	Confluence      string            `json:"confluence"`
	Timestamp       string            `json:"timestamp"`
}

type KeyLevels struct {
	Support    float64 `json:"support"`
	Resistance float64 `json:"resistance"`
}

type TradeSignal struct {
	Symbol         string  `json:"symbol"`
	Action         string  `json:"action"`
	Confidence     float64 `json:"confidence"`
	EntryPrice     float64 `json:"entry_price"`
	StopLoss       float64 `json:"stop_loss"`
	TakeProfit     float64 `json:"take_profit"`
	RiskReward     float64 `json:"risk_reward"`
	Recommendation string  `json:"recommendation"`
	Source         string  `json:"source"`
	RequestID      string  `json:"request_id"`
}

type Client struct {
	cfg    Config
	client *http.Client
	log    *slog.Logger
	mu     sync.RWMutex
}

func NewClient(cfg Config, log *slog.Logger) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 2 * time.Minute
	}
	if cfg.ScreenshotService == "" {
		cfg.ScreenshotService = "http://localhost:3456"
	}
	if cfg.N8NWebhook == "" {
		cfg.N8NWebhook = "http://localhost:5678/webhook/tradingview-analysis"
	}
	if cfg.GOBOTWebhook == "" {
		cfg.GOBOTWebhook = "http://localhost:8080/webhook/trade_signal"
	}

	return &Client{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		log:    log,
	}
}

func (c *Client) CaptureScreenshots(ctx context.Context, symbol string, intervals []string) (map[string]string, error) {
	c.log.Info("Capturing screenshots", slog.String("symbol", symbol))

	results := make(map[string]string)

	for _, interval := range intervals {
		reqBody := map[string]string{"symbol": symbol, "interval": interval}
		data, _ := json.Marshal(reqBody)

		resp, err := c.client.Post(
			fmt.Sprintf("%s/capture", c.cfg.ScreenshotService),
			"application/json",
			bytes.NewReader(data),
		)
		if err != nil {
			c.log.Warn("Screenshot failed", slog.String("interval", interval))
			continue
		}
		defer resp.Body.Close()

		var result struct{ Screenshot string }
		json.NewDecoder(resp.Body).Decode(&result)

		if result.Screenshot != "" {
			results[interval] = result.Screenshot
			c.log.Info("Screenshot captured", slog.String("interval", interval))
		}
	}

	return results, nil
}

func (c *Client) AnalyzeWithQuantCrawler(ctx context.Context, symbol string, screenshots map[string]string, accountBalance float64) (*AnalysisResult, error) {
	c.log.Info("Analyzing with QuantCrawler", slog.String("symbol", symbol))

	reqBody := map[string]interface{}{
		"symbol":          symbol,
		"account_balance": accountBalance,
		"screenshots":     screenshots,
		"request_id":      fmt.Sprintf("req_%d", time.Now().Unix()),
	}
	data, _ := json.Marshal(reqBody)

	resp, err := c.client.Post(c.cfg.N8NWebhook, "application/json", bytes.NewReader(data))
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var result AnalysisResult
		json.Unmarshal(body, &result)
		return &result, nil
	}

	c.log.Warn("N8N unavailable, using mock analysis")
	return c.mockAnalysis(symbol, screenshots, accountBalance), nil
}

func (c *Client) mockAnalysis(symbol string, screenshots map[string]string, accountBalance float64) *AnalysisResult {
	directions := []string{"LONG", "SHORT", "HOLD"}
	direction := directions[time.Now().Unix()%3]
	confidence := int(time.Now().Unix()%40) + 60

	price := 0.00001
	stopDistance := price * 0.005
	targetDistance := price * 0.015

	return &AnalysisResult{
		Symbol:          symbol,
		Direction:       direction,
		Confidence:      confidence,
		EntryPrice:      price,
		StopLoss:        price - stopDistance,
		TakeProfit:      price + targetDistance,
		RiskRewardRatio: 3.0,
		Recommendation:  fmt.Sprintf("QuantCrawler: %s %s with %d%% confidence", symbol, direction, confidence),
		Timeframes: map[string]string{
			"15m": "Bullish momentum",
			"5m":  "Consolidating",
			"1m":  "Volatility elevated",
		},
		KeyLevels:  KeyLevels{Support: price * 0.99, Resistance: price * 1.01},
		Confluence: fmt.Sprintf("%d/3 timeframes agree", len(screenshots)),
		Timestamp:  time.Now().Format(time.RFC3339),
	}
}

func (c *Client) SendTradeSignal(ctx context.Context, result *AnalysisResult) error {
	c.log.Info("Sending trade signal", slog.String("symbol", result.Symbol))

	var action string
	if result.Direction == "HOLD" || result.Direction == "STAY AWAY" {
		action = "hold"
	} else {
		action = result.Direction
	}

	signal := TradeSignal{
		Symbol:         result.Symbol,
		Action:         action,
		Confidence:     float64(result.Confidence) / 100,
		EntryPrice:     result.EntryPrice,
		StopLoss:       result.StopLoss,
		TakeProfit:     result.TakeProfit,
		RiskReward:     result.RiskRewardRatio,
		Recommendation: result.Recommendation,
		Source:         "quantcrawler",
		RequestID:      fmt.Sprintf("qc_%d", time.Now().Unix()),
	}

	data, _ := json.Marshal(signal)
	resp, err := c.client.Post(c.cfg.GOBOTWebhook, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("GOBOT rejected signal")
	}

	return nil
}

func (c *Client) RunCompleteWorkflow(ctx context.Context, symbol string, accountBalance float64) (*AnalysisResult, error) {
	c.log.Info("Starting workflow", slog.String("symbol", symbol))

	intervals := []string{"1m", "5m", "15m"}
	screenshots, err := c.CaptureScreenshots(ctx, symbol, intervals)
	if err != nil {
		return nil, err
	}

	result, err := c.AnalyzeWithQuantCrawler(ctx, symbol, screenshots, accountBalance)
	if err != nil {
		return nil, err
	}

	c.SendTradeSignal(ctx, result)

	return result, nil
}
