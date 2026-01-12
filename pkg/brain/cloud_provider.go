package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// CloudConfig holds configuration for cloud providers
type CloudConfig struct {
	APIKey      string        `json:"api_key"`
	Provider    string        `json:"provider"` // "openai" or "anthropic"
	Model       string        `json:"model"`
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int           `json:"max_retries"`
	Temperature float64       `json:"temperature"`
}

// CloudProvider implements the Provider interface for cloud APIs
type CloudProvider struct {
	config     CloudConfig
	modelName  string
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
		default:
			config.Model = "gpt-4-turbo-preview"
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
	// Simulate cloud response for now
	// In production, this would make actual API calls
	
	startTime := time.Now()
	
	// Simulate processing time
	time.Sleep(2 * time.Second)
	
	// Generate mock response based on prompt
	response := p.generateMockResponse(prompt)
	
	latency := time.Since(startTime)
	logrus.WithFields(logrus.Fields{
		"provider": p.config.Provider,
		"model":    p.config.Model,
		"latency":  latency,
		"prompt_length": len(prompt),
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

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
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
	default:
		return 0.01
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