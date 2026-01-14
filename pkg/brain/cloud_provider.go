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

// CloudConfig holds configuration for cloud providers
type CloudConfig struct {
	APIKey      string        `json:"api_key"`
	Provider    string        `json:"provider"` // "openai", "anthropic", or "gemini"
	Model       string        `json:"model"`
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int           `json:"max_retries"`
	Temperature float64       `json:"temperature"`
}

// CloudProvider implements the Provider interface for cloud APIs
type CloudProvider struct {
	config    CloudConfig
	modelName string
}

// NewCloudProvider creates a new cloud provider
func NewCloudProvider(config CloudConfig) (*CloudProvider, error) {
	if config.Provider == "" {
		config.Provider = "openai"
	}

	// Set default models based on provider
	if config.Model == "" {
		switch config.Provider {
		case "openai":
			config.Model = "gpt-4-turbo-preview" // Fast and capable
		case "anthropic":
			config.Model = "claude-3-haiku-20240307" // Fastest Claude model
		case "gemini":
			config.Model = "gemini-1.5-flash" // Free tier, fast model
		default:
			config.Model = "gemini-1.5-flash"
		}
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 2
	}
	if config.Temperature == 0 {
		config.Temperature = 0.1 // Low temperature for consistent decisions
	}

	provider := &CloudProvider{
		config:    config,
		modelName: config.Model,
	}

	// Test connection
	if err := provider.testConnection(); err != nil {
		return nil, fmt.Errorf("failed to connect to cloud provider: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"provider": config.Provider,
		"model":    config.Model,
	}).Info("Cloud provider initialized")

	return provider, nil
}

// GenerateResponse generates a response using cloud provider
func (p *CloudProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	startTime := time.Now()

	var response string
	var err error

	switch p.config.Provider {
	case "gemini":
		response, err = p.callGeminiAPI(ctx, prompt)
	case "openai":
		response, err = p.callOpenAIAPI(ctx, prompt)
	case "anthropic":
		response, err = p.callAnthropicAPI(ctx, prompt)
	default:
		response, err = p.callGeminiAPI(ctx, prompt)
	}

	if err != nil {
		return "", fmt.Errorf("cloud provider error: %w", err)
	}

	latency := time.Since(startTime)
	logrus.WithFields(logrus.Fields{
		"provider":        p.config.Provider,
		"model":           p.config.Model,
		"latency":         latency,
		"prompt_length":   len(prompt),
		"response_length": len(response),
	}).Debug("Cloud response generated")

	return response, nil
}

// GenerateStructuredResponse generates a structured response
func (p *CloudProvider) GenerateStructuredResponse(ctx context.Context, prompt string, response interface{}) error {
	// Add JSON instruction to prompt
	jsonPrompt := prompt + "\n\nRespond in valid JSON format only. Be concise and accurate."

	_, err := p.GenerateResponse(ctx, jsonPrompt)
	if err != nil {
		return err
	}

	// Parse JSON (mock implementation)
	// In production, this would parse actual JSON from cloud provider
	return nil
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// callGeminiAPI makes a call to Google's Gemini API
func (p *CloudProvider) callGeminiAPI(ctx context.Context, prompt string) (string, error) {
	if p.config.APIKey == "" {
		return "", fmt.Errorf("Gemini API key not configured")
	}

	// Gemini API endpoint for gemini-1.5-flash (free tier)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		p.config.Model, p.config.APIKey)

	// Prepare request body
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     p.config.Temperature,
			"maxOutputTokens": 1024,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: p.config.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var geminiResponse struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &geminiResponse); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if len(geminiResponse.Candidates) == 0 || len(geminiResponse.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response content from Gemini")
	}

	return geminiResponse.Candidates[0].Content.Parts[0].Text, nil
}

// callOpenAIAPI makes a call to OpenAI API
func (p *CloudProvider) callOpenAIAPI(ctx context.Context, prompt string) (string, error) {
	if p.config.APIKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	url := "https://api.openai.com/v1/chat/completions"

	requestBody := map[string]interface{}{
		"model": p.config.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": p.config.Temperature,
		"max_tokens":  1024,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	client := &http.Client{Timeout: p.config.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var openaiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &openaiResponse); err != nil {
		return "", fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	if len(openaiResponse.Choices) == 0 {
		return "", fmt.Errorf("no response content from OpenAI")
	}

	return openaiResponse.Choices[0].Message.Content, nil
}

// callAnthropicAPI makes a call to Anthropic Claude API
func (p *CloudProvider) callAnthropicAPI(ctx context.Context, prompt string) (string, error) {
	if p.config.APIKey == "" {
		return "", fmt.Errorf("Anthropic API key not configured")
	}

	url := "https://api.anthropic.com/v1/messages"

	requestBody := map[string]interface{}{
		"model":      p.config.Model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: p.config.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResponse struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &anthropicResponse); err != nil {
		return "", fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	if len(anthropicResponse.Content) == 0 {
		return "", fmt.Errorf("no response content from Anthropic")
	}

	return anthropicResponse.Content[0].Text, nil
}

// generateMockResponse creates a mock response for demonstration
func (p *CloudProvider) generateMockResponse(prompt string) string {
	if contains(prompt, "trading decision") {
		return `{"decision": "BUY", "confidence": 0.85, "reasoning": "Strong bullish FVG with CVD divergence", "risk_level": "MEDIUM", "recommended_leverage": 20, "fvg_confidence": 0.82, "cvd_divergence": true}`
	}

	if contains(prompt, "market analysis") {
		return `{"market_regime": "RANGING", "confidence": 0.78, "key_factors": ["low volatility", "clear support/resistance"], "strategy_adjustments": {"max_leverage": 25, "fvg_confidence_min": 0.75}}`
	}

	return `{"response": "Analysis complete", "confidence": 0.9}`
}

// GetModelName returns the model name
func (p *CloudProvider) GetModelName() string {
	return p.config.Model
}

// GetLatency returns estimated latency
func (p *CloudProvider) GetLatency() time.Duration {
	// Cloud providers typically have higher latency
	switch p.config.Provider {
	case "openai":
		return 2 * time.Second // GPT-4 turbo
	case "anthropic":
		return 1 * time.Second // Claude Haiku
	case "gemini":
		return 1 * time.Second // Gemini Flash is very fast
	default:
		return 2 * time.Second
	}
}

// IsHealthy checks if cloud provider is accessible
func (p *CloudProvider) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Simple health check
	_, err := p.GenerateResponse(ctx, "Hello, are you working?")
	return err == nil
}

// testConnection tests the connection to cloud provider
func (p *CloudProvider) testConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Simple connection test
	_, err := p.GenerateResponse(ctx, "Test connection. Respond with 'OK' only.")
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"provider": p.config.Provider,
		"model":    p.config.Model,
	}).Info("Cloud provider connection test successful")

	return nil
}

// GetStats returns provider statistics
func (p *CloudProvider) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"provider":   p.config.Provider,
		"model":      p.config.Model,
		"healthy":    p.IsHealthy(),
		"latency_ms": p.GetLatency().Milliseconds(),
	}
}

// EstimateCost returns estimated cost per 1K tokens
func (p *CloudProvider) EstimateCost() float64 {
	switch p.config.Provider {
	case "openai":
		if strings.Contains(p.config.Model, "gpt-4") {
			return 0.03 // ~$0.03 per 1K tokens for GPT-4
		}
		return 0.002 // ~$0.002 per 1K tokens for GPT-3.5
	case "anthropic":
		if strings.Contains(p.config.Model, "claude-3-haiku") {
			return 0.00025 // ~$0.00025 per 1K tokens for Claude Haiku
		}
		return 0.008 // ~$0.008 per 1K tokens for Claude Sonnet
	case "gemini":
		return 0.0000 // FREE for gemini-1.5-flash free tier
	default:
		return 0.0000
	}
}

// TradingDecisionPrompt generates a trading decision prompt
func (p *CloudProvider) TradingDecisionPrompt(signalData interface{}) string {
	dataJSON, _ := json.Marshal(signalData)

	return fmt.Sprintf(`
You are Cognee's trading decision AI. Evaluate this trading signal for high-frequency scalping:

Signal Data: %s

Decision criteria:
- FVG confidence > 0.7
- CVD divergence present
- Volatility within acceptable range
- No high-impact news events

Provide your decision in JSON format:
{
  "decision": "BUY/SELL/HOLD",
  "confidence": 0.8,
  "reasoning": "Bullish FVG at $49,400 with CVD divergence",
  "risk_level": "LOW/MEDIUM/HIGH",
  "recommended_leverage": 20
}

Respond quickly - this is for real-time execution.
`, string(dataJSON))
}

// MarketAnalysisPrompt generates a market analysis prompt
func (p *CloudProvider) MarketAnalysisPrompt(marketData interface{}) string {
	dataJSON, _ := json.Marshal(marketData)

	return fmt.Sprintf(`
You are Cognee's market analysis AI. Analyze the following market data and provide a concise assessment:

Market Data: %s

Please provide:
1. Current market regime (RANGING/TRENDING/VOLATILE)
2. Confidence level (0.0-1.0)
3. Key factors influencing your assessment
4. Recommended strategy adjustments

Respond in JSON format:
{
  "market_regime": "RANGING",
  "confidence": 0.75,
  "key_factors": ["low volatility", "support/resistance levels"],
  "strategy_adjustments": {
    "max_leverage": 20,
    "fvg_confidence_min": 0.7
  }
}
`, string(dataJSON))
}
