package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// OllamaConfig holds configuration for Ollama provider
type OllamaConfig struct {
	Model       string        `json:"model"`
	BaseURL     string        `json:"base_url"`
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int           `json:"max_retries"`
	Temperature float64       `json:"temperature"`
}

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	config     OllamaConfig
	httpClient *http.Client
	modelName  string
}

// OllamaRequest represents a request to MSTY API (OpenAI-compatible)
type OllamaRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Options struct {
		Temperature float64 `json:"temperature"`
	} `json:"options"`
}

// MSTYRequest represents a request to MSTY API (OpenAI-compatible)
type MSTYRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaResponse represents a response from Ollama API
type OllamaResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// MSTYResponse represents a response from MSTY API (OpenAI-compatible)
type MSTYResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(config OllamaConfig) (*OllamaProvider, error) {
	if config.Model == "" {
		config.Model = "qwen3:0.6b" // Updated to available msty model
	}
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11964" // Updated to msty port with LFM2.5
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second // Faster timeout for scalping with LFM2.5
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 2
	}
	if config.Temperature == 0 {
		config.Temperature = 0.05 // Even lower temperature for more consistent decisions
	}

	provider := &OllamaProvider{
		config:    config,
		modelName: config.Model,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	// Test connection
	if err := provider.testConnection(); err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama at %s: %w", config.BaseURL, err)
	}

	logrus.WithFields(logrus.Fields{
		"model":    config.Model,
		"base_url": config.BaseURL,
		"timeout":  config.Timeout,
	}).Info("GOBOT LiquidAI LFM2.5 provider initialized")

	return provider, nil
}

// GenerateResponse generates a response using MSTY (OpenAI-compatible API)
func (p *OllamaProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// Create MSTY-compatible request
	request := MSTYRequest{
		Model: p.config.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: p.config.Temperature,
		Stream:      false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request with retries
	var response string
	for attempt := 0; attempt < p.config.MaxRetries; attempt++ {
		resp, err := p.makeRequest(ctx, jsonData)
		if err == nil {
			response = resp
			break
		}

		if attempt < p.config.MaxRetries-1 {
			logrus.WithError(err).WithField("attempt", attempt+1).Debug("Request failed, retrying")
			time.Sleep(time.Duration(attempt+1) * time.Millisecond * 100)
		}
	}

	if response == "" && err != nil {
		return "", fmt.Errorf("failed to generate response after %d attempts: %w", p.config.MaxRetries, err)
	}

	return response, nil
}

// makeRequest makes a single HTTP request to MSTY (OpenAI-compatible API)
func (p *OllamaProvider) makeRequest(ctx context.Context, jsonData []byte) (string, error) {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse MSTY response
	var mstyResp MSTYResponse
	if err := json.Unmarshal(body, &mstyResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(mstyResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	response := mstyResp.Choices[0].Message.Content

	latency := time.Since(startTime)
	logrus.WithFields(logrus.Fields{
		"model":   p.config.Model,
		"latency": latency,
	}).Debug("LiquidAI LFM2.5 response generated")

	return response, nil
}

// GenerateStructuredResponse generates a structured response
func (p *OllamaProvider) GenerateStructuredResponse(ctx context.Context, prompt string, response interface{}) error {
	// Add JSON instruction to prompt
	jsonPrompt := prompt + "\n\nRespond in valid JSON format only."

	textResponse, err := p.GenerateResponse(ctx, jsonPrompt)
	if err != nil {
		return err
	}

	// Clean response (remove any markdown formatting)
	textResponse = cleanJSONResponse(textResponse)

	// Parse JSON
	if err := json.Unmarshal([]byte(textResponse), response); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w, response: %s", err, textResponse)
	}

	return nil
}

// cleanJSONResponse removes markdown formatting and extracts JSON
func cleanJSONResponse(response string) string {
	// Remove markdown code blocks
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	return strings.TrimSpace(response)
}

// GetModelName returns the model name
func (p *OllamaProvider) GetModelName() string {
	return p.config.Model
}

// GetLatency returns the last known latency (estimated)
func (p *OllamaProvider) GetLatency() time.Duration {
	// LFM2.5 is faster than LFM2.6B - estimated 300ms
	return 300 * time.Millisecond
}

// IsHealthy checks if Ollama is accessible
func (p *OllamaProvider) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simple health check
	_, err := p.GenerateResponse(ctx, "Hello")
	return err == nil
}

// testConnection tests the connection to MSTY (modified Ollama API)
func (p *OllamaProvider) testConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// MSTY uses /v1/models endpoint instead of /api/tags
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/v1/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to MSTY at %s: %w", p.config.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MSTY health check failed: status %d", resp.StatusCode)
	}

	// Check if our model is available
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read health check response: %w", err)
	}

	var models struct {
		Data []struct {
			ID     string `json:"id"`
			Object string `json:"object"`
		} `json:"data"`
		Object string `json:"object"`
	}

	if err := json.Unmarshal(body, &models); err != nil {
		return fmt.Errorf("failed to parse models response: %w", err)
	}

	// Check if our model exists
	modelFound := false
	for _, model := range models.Data {
		if model.ID == p.config.Model {
			modelFound = true
			break
		}
	}

	if !modelFound {
		return fmt.Errorf("model '%s' not found in MSTY at %s. Available models: %v",
			p.config.Model, p.config.BaseURL, models.Data)
	}

	logrus.WithFields(logrus.Fields{
		"model":    p.config.Model,
		"base_url": p.config.BaseURL,
	}).Info("GOBOT LiquidAI LFM2.5 connection test successful")
	return nil
}

// GetStats returns provider statistics
func (p *OllamaProvider) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"model":      p.config.Model,
		"base_url":   p.config.BaseURL,
		"healthy":    p.IsHealthy(),
		"latency_ms": p.GetLatency().Milliseconds(),
	}
}

// OptimizeForScalping configures the provider for high-frequency trading
func (p *OllamaProvider) OptimizeForScalping() {
	// LFM2.5 optimizations for scalping
	p.config.Temperature = 0.03        // Ultra-low temperature for maximum consistency
	p.config.Timeout = 8 * time.Second // Faster timeout for LFM2.5

	logrus.Info("GOBOT LiquidAI LFM2.5 optimized for ultra-fast scalping")
}

// TradingDecisionSchema provides the expected schema for trading decisions
func (p *OllamaProvider) TradingDecisionSchema() string {
	return `{
  "decision": "BUY|SELL|HOLD",
  "confidence": 0.0-1.0,
  "reasoning": "Brief explanation",
  "risk_level": "LOW|MEDIUM|HIGH",
  "recommended_leverage": 1-25,
  "fvg_confidence": 0.0-1.0,
  "cvd_divergence": true|false
}`
}

// TradingDecisionPrompt generates a trading decision prompt
func (p *OllamaProvider) TradingDecisionPrompt(signalData interface{}) string {
	dataJSON, _ := json.Marshal(signalData)

	return fmt.Sprintf(`
You are GOBOT's trading decision AI powered by LiquidAI LFM2.5. Evaluate this trading signal for ultra-high-frequency scalping:

Signal Data: %s

Decision criteria for LFM2.5:
- FVG confidence > 0.75
- CVD divergence present and strong
- Volatility within optimal range (0.5-2.0%%)
- No high-impact news events
- Market microstructure favorable

Provide your decision in JSON format:
{
  "decision": "BUY/SELL/HOLD",
  "confidence": 0.85,
  "reasoning": "Strong bullish FVG at $49,400 with CVD divergence",
  "risk_level": "LOW/MEDIUM/HIGH",
  "recommended_leverage": 20
}

Respond ultra-fast - this is for millisecond execution with LFM2.5.
`, string(dataJSON))
}

// MarketAnalysisPrompt generates a market analysis prompt
func (p *OllamaProvider) MarketAnalysisPrompt(marketData interface{}) string {
	dataJSON, _ := json.Marshal(marketData)

	return fmt.Sprintf(`
You are GOBOT's market analysis AI powered by LiquidAI LFM2.5. Analyze the following market data with millisecond precision:

Market Data: %s

Please provide ultra-fast assessment:
1. Current market regime (RANGING/TRENDING/VOLATILE)
2. Confidence level (0.0-1.0) for LFM2.5
3. Key microstructure factors
4. Optimal strategy adjustments for LFM2.5

Respond in JSON format:
{
  "market_regime": "RANGING",
  "confidence": 0.82,
  "key_factors": ["low volatility", "tight spreads", "high liquidity"],
  "strategy_adjustments": {
    "max_leverage": 25,
    "fvg_confidence_min": 0.75,
    "execution_speed": "millisecond"
  }
}
`, string(dataJSON))
}
