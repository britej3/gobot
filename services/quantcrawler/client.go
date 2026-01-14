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

type N8NRequest struct {
	Symbol         string  `json:"symbol"`
	AccountBalance float64 `json:"account_balance"`
	AccountEquity  float64 `json:"account_equity"`
	CurrentPrice   float64 `json:"current_price"`
	Timestamp      string  `json:"timestamp"`
	RequestID      string  `json:"request_id"`
}

type N8NResponse struct {
	Symbol         string            `json:"symbol"`
	Ticker         string            `json:"ticker"`
	CurrentPrice   float64           `json:"current_price"`
	Entry          float64           `json:"entry"`
	Confidence     int               `json:"confidence"`
	Direction      string            `json:"direction"`
	Recommendation string            `json:"recommendation"`
	Options        []PositionOption  `json:"options"`
	Timeframes     TimeframeAnalysis `json:"timeframes"`
	KeyLevels      KeyLevels         `json:"key_levels"`
	Risks          string            `json:"risks"`
	Confluence     string            `json:"confluence"`
	RequestID      string            `json:"request_id"`
	ProcessedAt    string            `json:"processed_at"`
}

type PositionOption struct {
	Name            string  `json:"name"`
	Contracts       int     `json:"contracts"`
	RiskPerContract float64 `json:"risk_per_contract"`
	TotalRisk       float64 `json:"total_risk"`
	StopDistance    float64 `json:"stop_distance"`
	TargetDistance  float64 `json:"target_distance"`
	StopPrice       float64 `json:"stop_price"`
	TargetPrice     float64 `json:"target_price"`
	RiskRewardRatio float64 `json:"risk_reward_ratio"`
	BestFor         string  `json:"best_for"`
	Recommended     bool    `json:"recommended"`
}

type TimeframeAnalysis struct {
	TF15m string `json:"15m"`
	TF5m  string `json:"5m"`
	TF1m  string `json:"1m"`
}

type KeyLevels struct {
	Support    float64 `json:"support"`
	Resistance float64 `json:"resistance"`
}

type TradeDecision struct {
	Symbol           string
	ShouldTrade      bool
	Direction        string
	Confidence       float64
	EntryPrice       float64
	StopLossPrice    float64
	TakeProfitPrice  float64
	PositionSizeUSDT float64
	Contracts        int
	Leverage         int
	RiskAmount       float64
	RiskPercent      float64
	RiskRewardRatio  float64
	Reasoning        string
	Confluence       string
	Timestamp        time.Time
}

type Config struct {
	N8NWebhookURL   string
	Timeout         time.Duration
	MaxRetries      int
	MinConfidence   int
	MaxRiskPercent  float64
	PreferredOption string
}

func DefaultConfig() Config {
	return Config{
		N8NWebhookURL:   "http://localhost:5678/webhook/quantcrawler-analysis",
		Timeout:         60 * time.Second,
		MaxRetries:      2,
		MinConfidence:   50,
		MaxRiskPercent:  2.0,
		PreferredOption: "multiple",
	}
}

type Client struct {
	cfg           Config
	httpClient    *http.Client
	log           *slog.Logger
	mu            sync.RWMutex
	totalRequests int
	successCount  int
	tradeCount    int
	skipCount     int
}

type Option func(*Client)

func WithN8NWebhook(url string) Option {
	return func(c *Client) {
		c.cfg.N8NWebhookURL = url
	}
}

func WithMinConfidence(conf int) Option {
	return func(c *Client) {
		c.cfg.MinConfidence = conf
	}
}

func WithMaxRisk(pct float64) Option {
	return func(c *Client) {
		c.cfg.MaxRiskPercent = pct
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Client) {
		c.log = log
	}
}

func NewClient(opts ...Option) *Client {
	c := &Client{
		cfg: DefaultConfig(),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		log: slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.httpClient.Timeout = c.cfg.Timeout
	return c
}

func (c *Client) GetTradeRecommendation(
	ctx context.Context,
	symbol string,
	accountBalance float64,
	accountEquity float64,
	currentPrice float64,
) (*TradeDecision, error) {
	c.mu.Lock()
	c.totalRequests++
	requestID := fmt.Sprintf("req_%d_%d", time.Now().Unix(), c.totalRequests)
	c.mu.Unlock()

	c.log.Info("requesting trade analysis",
		slog.String("symbol", symbol),
		slog.String("request_id", requestID))

	req := N8NRequest{
		Symbol:         symbol,
		AccountBalance: accountBalance,
		AccountEquity:  accountEquity,
		CurrentPrice:   currentPrice,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		RequestID:      requestID,
	}

	var resp *N8NResponse
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			c.log.Info("retrying request", slog.Int("attempt", attempt))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}

		resp, lastErr = c.callN8N(ctx, req)
		if lastErr == nil {
			break
		}

		c.log.Warn("request failed",
			slog.Int("attempt", attempt),
			slog.Any("error", lastErr))
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all retries failed: %w", lastErr)
	}

	c.mu.Lock()
	c.successCount++
	c.mu.Unlock()

	decision := c.processResponse(resp, accountBalance)

	if decision.ShouldTrade {
		c.mu.Lock()
		c.tradeCount++
		c.mu.Unlock()

		c.log.Info("TRADE SIGNAL",
			slog.String("symbol", decision.Symbol),
			slog.String("direction", decision.Direction),
			slog.Float64("confidence", decision.Confidence),
			slog.Float64("entry", decision.EntryPrice),
			slog.Float64("stop", decision.StopLossPrice),
			slog.Float64("target", decision.TakeProfitPrice))
	} else {
		c.mu.Lock()
		c.skipCount++
		c.mu.Unlock()

		c.log.Info("NO TRADE",
			slog.String("symbol", decision.Symbol),
			slog.String("reason", decision.Reasoning))
	}

	return decision, nil
}

func (c *Client) callN8N(ctx context.Context, req N8NRequest) (*N8NResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.cfg.N8NWebhookURL,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var n8nResp N8NResponse
	if err := json.NewDecoder(resp.Body).Decode(&n8nResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &n8nResp, nil
}

func (c *Client) processResponse(resp *N8NResponse, accountBalance float64) *TradeDecision {
	decision := &TradeDecision{
		Symbol:     resp.Symbol,
		Timestamp:  time.Now(),
		Reasoning:  resp.Recommendation,
		Confluence: resp.Confluence,
	}

	if resp.Direction == "STAY AWAY" || resp.Confidence < c.cfg.MinConfidence {
		decision.ShouldTrade = false
		if resp.Direction == "STAY AWAY" {
			decision.Reasoning = fmt.Sprintf("AI recommends STAY AWAY: %s", resp.Recommendation)
		} else {
			decision.Reasoning = fmt.Sprintf("Confidence too low: %d%% < %d%%", resp.Confidence, c.cfg.MinConfidence)
		}
		return decision
	}

	option := c.selectPositionOption(resp.Options)
	if option == nil {
		decision.ShouldTrade = false
		decision.Reasoning = "No valid position sizing options available"
		return decision
	}

	riskPercent := (option.TotalRisk / accountBalance) * 100.0
	if riskPercent > c.cfg.MaxRiskPercent {
		decision.ShouldTrade = false
		decision.Reasoning = fmt.Sprintf("Risk too high: %.2f%% > %.2f%%", riskPercent, c.cfg.MaxRiskPercent)
		return decision
	}

	decision.ShouldTrade = true
	decision.Direction = resp.Direction
	decision.Confidence = float64(resp.Confidence) / 100.0
	decision.EntryPrice = resp.Entry
	decision.StopLossPrice = option.StopPrice
	decision.TakeProfitPrice = option.TargetPrice
	decision.PositionSizeUSDT = option.TotalRisk / (riskPercent / 100.0)
	decision.Contracts = option.Contracts
	decision.RiskAmount = option.TotalRisk
	decision.RiskPercent = riskPercent
	decision.RiskRewardRatio = option.RiskRewardRatio

	return decision
}

func (c *Client) selectPositionOption(options []PositionOption) *PositionOption {
	if len(options) == 0 {
		return nil
	}

	for i := range options {
		if options[i].Recommended {
			return &options[i]
		}
	}

	for i := range options {
		switch c.cfg.PreferredOption {
		case "single":
			if options[i].Contracts == 1 {
				return &options[i]
			}
		case "multiple":
			if options[i].Contracts > 1 {
				return &options[i]
			}
		case "structure":
			if contains(options[i].Name, "structure") || contains(options[i].Name, "chart") {
				return &options[i]
			}
		}
	}

	return &options[0]
}

func (c *Client) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	successRate := 0.0
	if c.totalRequests > 0 {
		successRate = float64(c.successCount) / float64(c.totalRequests) * 100.0
	}

	tradeRate := 0.0
	if c.successCount > 0 {
		tradeRate = float64(c.tradeCount) / float64(c.successCount) * 100.0
	}

	return map[string]interface{}{
		"total_requests": c.totalRequests,
		"success_count":  c.successCount,
		"trade_count":    c.tradeCount,
		"skip_count":     c.skipCount,
		"success_rate":   successRate,
		"trade_rate":     tradeRate,
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)))
}
