package screenshot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type Config struct {
	ServerURL   string
	Timeout     time.Duration
	AutoStart   bool
	ServicePath string
}

type Client struct {
	cfg     Config
	client  *http.Client
	log     *slog.Logger
	mu      sync.RWMutex
	running bool
}

type ScreenshotRequest struct {
	Symbol    string   `json:"symbol"`
	Intervals []string `json:"intervals,omitempty"`
}

type ScreenshotResponse struct {
	Symbol    string            `json:"symbol"`
	Intervals []string          `json:"intervals"`
	Results   map[string]string `json:"results"`
	Timestamp string            `json:"timestamp"`
}

type TradingViewResponse struct {
	Symbol     string `json:"symbol"`
	Interval   string `json:"interval"`
	Screenshot string `json:"screenshot"`
	DurationMs int64  `json:"duration_ms"`
}

func NewClient(cfg Config, log *slog.Logger) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://localhost:3000"
	}

	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		log: log,
	}
}

func (c *Client) Capture(symbol, interval string) (*TradingViewResponse, error) {
	reqBody := ScreenshotRequest{
		Symbol:    symbol,
		Intervals: []string{interval},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.client.Post(
		fmt.Sprintf("%s/capture", c.cfg.ServerURL),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf("POST request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error: %s", string(body))
	}

	var result TradingViewResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) CaptureMulti(symbol string, intervals []string) (*ScreenshotResponse, error) {
	reqBody := ScreenshotRequest{
		Symbol:    symbol,
		Intervals: intervals,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.client.Post(
		fmt.Sprintf("%s/capture-multi", c.cfg.ServerURL),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf("POST request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error: %s", string(body))
	}

	var result ScreenshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) Health() error {
	resp, err := c.client.Get(fmt.Sprintf("%s/health", c.cfg.ServerURL))
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server unhealthy")
	}

	return nil
}

func (c *Client) StartService() error {
	if c.cfg.ServicePath == "" {
		c.log.Info("Screenshot service auto-start disabled")
		return nil
	}

	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}

	c.log.Info("Starting TradingView screenshot service...", slog.String("path", c.cfg.ServicePath))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", c.cfg.ServicePath)
	if err := cmd.Start(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("start service: %w", err)
	}

	c.running = true
	c.mu.Unlock()

	// Wait for service to be ready
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		if err := c.Health(); err == nil {
			c.log.Info("Screenshot service ready")
			return nil
		}
	}

	return fmt.Errorf("service failed to start")
}
