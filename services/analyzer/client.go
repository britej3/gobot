package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/britej3/gobot/domain/trade"
)

type Config struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

type Analyzer struct {
	cfg    Config
	client *http.Client
}

type AnalysisRequest struct {
	Symbol       string  `json:"symbol"`
	CurrentPrice float64 `json:"current_price"`
	High24h      float64 `json:"high_24h"`
	Low24h       float64 `json:"low_24h"`
	Volume24h    float64 `json:"volume_24h"`
	Volatility   float64 `json:"volatility"`
	RSI          float64 `json:"rsi"`
	EMAFast      float64 `json:"ema_fast"`
	EMASlow      float64 `json:"ema_slow"`
}

type AnalysisResponse struct {
	Action       string  `json:"action"`
	Symbol       string  `json:"symbol"`
	PositionSize float64 `json:"position_size"`
	EntryPrice   float64 `json:"entry_price"`
	StopLoss     float64 `json:"stop_loss"`
	TakeProfit   float64 `json:"take_profit"`
	Confidence   float64 `json:"confidence"`
	Reasoning    string  `json:"reasoning"`
}

type MarketDataProvider interface {
	GetMarketData(ctx context.Context, symbol string) (*trade.MarketData, error)
}

func New(cfg Config) *Analyzer {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = time.Second
	}

	return &Analyzer{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (a *Analyzer) Analyze(ctx context.Context, data *trade.MarketData) (*AnalysisResponse, error) {
	if data == nil {
		return nil, fmt.Errorf("market data is required")
	}

	req := AnalysisRequest{
		Symbol:       data.Symbol,
		CurrentPrice: data.CurrentPrice,
		High24h:      data.High24h,
		Low24h:       data.Low24h,
		Volume24h:    data.Volume24h,
		Volatility:   data.Volatility,
		RSI:          data.RSI,
		EMAFast:      data.EMAFast,
		EMASlow:      data.EMASlow,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *AnalysisResponse
	var lastErr error

	for attempt := 0; attempt <= a.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(a.cfg.RetryDelay * time.Duration(attempt)):
			}
		}

		resp, lastErr = a.sendRequest(ctx, body)
		if lastErr == nil {
			return resp, nil
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", a.cfg.MaxRetries+1, lastErr)
}

func (a *Analyzer) sendRequest(ctx context.Context, body []byte) (*AnalysisResponse, error) {
	url := fmt.Sprintf("%s/analyze", a.cfg.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result AnalysisResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (a *Analyzer) AnalyzeBatch(ctx context.Context, markets []*trade.MarketData) ([]*AnalysisResponse, error) {
	results := make([]*AnalysisResponse, 0, len(markets))

	for _, data := range markets {
		resp, err := a.Analyze(ctx, data)
		if err != nil {
			continue
		}
		results = append(results, resp)
	}

	return results, nil
}

func (a *Analyzer) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", a.cfg.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}
