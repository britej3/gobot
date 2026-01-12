package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Provider interface for dual inference (local vs cloud)
type Provider interface {
	// GenerateResponse sends a prompt and returns the model's response
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	
	// GenerateStructuredResponse sends a prompt and returns a structured response
	GenerateStructuredResponse(ctx context.Context, prompt string, response interface{}) error
	
	// GetModelName returns the name of the current model
	GetModelName() string
	
	// GetLatency returns the average latency of the provider
	GetLatency() time.Duration
	
	// IsHealthy checks if the provider is operational
	IsHealthy() bool
	
	// TradingDecisionPrompt generates a trading decision prompt
	TradingDecisionPrompt(signalData interface{}) string
	
	// MarketAnalysisPrompt generates a market analysis prompt
	MarketAnalysisPrompt(marketData interface{}) string
}

// InferenceMode represents the current inference configuration
type InferenceMode string

const (
	ModeLocal InferenceMode = "LOCAL"    // Local Ollama inference
	ModeCloud InferenceMode = "CLOUD"    // Cloud API inference (OpenAI/Anthropic)
	ModeAuto  InferenceMode = "AUTO"     // Auto-switch based on complexity
)

// LLMProvider manages dual inference configuration
type LLMProvider struct {
	currentMode  InferenceMode
	localProvider  Provider
	cloudProvider  Provider
	lastLatency  time.Duration
	healthStatus bool
}

// ProviderConfig holds configuration for LLM providers
type ProviderConfig struct {
	Mode           InferenceMode `json:"mode"`
	LocalModel     string        `json:"local_model"`
	LocalBaseURL   string        `json:"local_base_url"`
	CloudAPIKey    string        `json:"cloud_api_key"`
	CloudProvider  string        `json:"cloud_provider"` // "openai" or "anthropic"
	MaxRetries     int           `json:"max_retries"`
	Timeout        time.Duration `json:"timeout"`
	ComplexityThreshold int      `json:"complexity_threshold"` // When to switch to cloud
}

// NewLLMProvider creates a new dual inference provider
func NewLLMProvider(mode string) Provider {
	config := ProviderConfig{
		Mode:        InferenceMode(mode),
		LocalModel:  getEnvString("OLLAMA_MODEL", "lfm2.5-1.2b-instruct-q8_0:latest"), // Updated to match available model
		LocalBaseURL: getEnvString("OLLAMA_BASE_URL", "http://localhost:11964"), // msty port
		CloudAPIKey: os.Getenv("OPENAI_API_KEY"),
		CloudProvider: getEnvString("CLOUD_PROVIDER", "openai"),
		MaxRetries:  3,
		Timeout:     8 * time.Second, // Faster timeout for LFM2.5
		ComplexityThreshold: 500,
	}

	provider, err := NewLLMProviderWithConfig(config)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize LLM provider")
	}
	return provider
}

// NewLLMProviderWithConfig creates a new LLM provider with specific configuration
func NewLLMProviderWithConfig(config ProviderConfig) (*LLMProvider, error) {
	provider := &LLMProvider{
		currentMode: config.Mode,
		healthStatus: true,
	}

	// Initialize local provider (Ollama) with LFM2.5
	if config.LocalModel == "" {
		config.LocalModel = "LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf"
	}
	if config.LocalBaseURL == "" {
		config.LocalBaseURL = "http://localhost:11454" // msty port
	}
	
	localProvider, err := NewOllamaProvider(OllamaConfig{
		Model:   config.LocalModel,
		BaseURL: config.LocalBaseURL,
		Timeout: config.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize local provider: %w", err)
	}
	provider.localProvider = localProvider

	// Initialize cloud provider if API key provided
	if config.CloudAPIKey != "" {
		cloudProvider, err := NewCloudProvider(CloudConfig{
			APIKey:   config.CloudAPIKey,
			Provider: config.CloudProvider,
			Timeout:  config.Timeout,
		})
		if err != nil {
			logrus.Warnf("Failed to initialize cloud provider: %v", err)
		} else {
			provider.cloudProvider = cloudProvider
		}
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 8 * time.Second // Faster for LFM2.5
	}
	if config.ComplexityThreshold == 0 {
		config.ComplexityThreshold = 500 // Token threshold for complexity
	}

	logrus.WithFields(logrus.Fields{
		"mode":        config.Mode,
		"local_model": config.LocalModel,
		"local_base_url": config.LocalBaseURL,
		"has_cloud":   provider.cloudProvider != nil,
	}).Info("GOBOT LiquidAI LFM2.5 Provider initialized")

	return provider, nil
}

// GenerateResponse generates a response using the appropriate provider
func (p *LLMProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	startTime := time.Now()
	
	// Determine which provider to use
	provider, mode := p.selectProvider(prompt)
	
	logrus.WithFields(logrus.Fields{
		"mode":   mode,
		"model":  provider.GetModelName(),
		"prompt_length": len(prompt),
	}).Debug("Generating response with LiquidAI LFM2.5")

	// Generate response with retries
	var response string
	var err error
	
	for attempt := 0; attempt < 3; attempt++ {
		response, err = provider.GenerateResponse(ctx, prompt)
		if err == nil {
			break
		}
		
		logrus.WithError(err).WithField("attempt", attempt+1).Warn("Response generation failed, retrying")
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	
	if err != nil {
		p.healthStatus = false
		return "", fmt.Errorf("failed to generate response after 3 attempts: %w", err)
	}
	
	p.lastLatency = time.Since(startTime)
	p.healthStatus = true
	
	logrus.WithFields(logrus.Fields{
		"mode":     mode,
		"latency":  p.lastLatency,
		"response_length": len(response),
	}).Debug("LFM2.5 response generated successfully")

	return response, nil
}

// GenerateStructuredResponse generates a structured response
func (p *LLMProvider) GenerateStructuredResponse(ctx context.Context, prompt string, response interface{}) error {
	// Generate text response first
	textResponse, err := p.GenerateResponse(ctx, prompt)
	if err != nil {
		return err
	}
	
	// Parse JSON response
	if err := json.Unmarshal([]byte(textResponse), response); err != nil {
		return fmt.Errorf("failed to parse structured response: %w", err)
	}
	
	return nil
}

// selectProvider determines which provider to use based on mode and prompt complexity
func (p *LLMProvider) selectProvider(prompt string) (Provider, InferenceMode) {
	switch p.currentMode {
	case ModeLocal:
		return p.localProvider, ModeLocal
		
	case ModeCloud:
		if p.cloudProvider != nil {
			return p.cloudProvider, ModeCloud
		}
		logrus.Warn("Cloud provider not available, falling back to local LFM2.5")
		return p.localProvider, ModeLocal
		
	case ModeAuto:
		// Auto-switch based on prompt complexity and urgency
		if p.isHighComplexityPrompt(prompt) && p.cloudProvider != nil {
			return p.cloudProvider, ModeCloud
		}
		return p.localProvider, ModeLocal
		
	default:
		return p.localProvider, ModeLocal
	}
}

// isHighComplexityPrompt determines if a prompt requires cloud inference
func (p *LLMProvider) isHighComplexityPrompt(prompt string) bool {
	// Simple heuristics for complexity detection
	wordCount := len(strings.Fields(prompt))
	
	// High complexity indicators
	complexityKeywords := []string{
		"comprehensive", "detailed", "analysis", "strategy", "planning",
		"weekend", "monthly", "quarterly", "annual", "long-term",
		"complex", "multi-factor", "correlation", "regression", "prediction",
	}
	
	// Check for complexity keywords
	promptLower := strings.ToLower(prompt)
	keywordCount := 0
	for _, keyword := range complexityKeywords {
		if strings.Contains(promptLower, keyword) {
			keywordCount++
		}
	}
	
	// Complexity decision logic for LFM2.5
	if wordCount > 200 || keywordCount >= 3 {
		return true
	}
	
	return false
}

// GetModelName returns the current model name
func (p *LLMProvider) GetModelName() string {
	provider, mode := p.selectProvider("test")
	return fmt.Sprintf("%s (%s)", provider.GetModelName(), mode)
}

// GetLatency returns the last response latency
func (p *LLMProvider) GetLatency() time.Duration {
	return p.lastLatency
}

// IsHealthy checks if the provider is operational
func (p *LLMProvider) IsHealthy() bool {
	return p.healthStatus && p.localProvider.IsHealthy()
}

// SwitchMode allows runtime switching between inference modes
func (p *LLMProvider) SwitchMode(mode InferenceMode) {
	p.currentMode = mode
	logrus.WithField("mode", mode).Info("Switched GOBOT inference mode")
}

// GetCurrentMode returns the current inference mode
func (p *LLMProvider) GetCurrentMode() InferenceMode {
	return p.currentMode
}

// ProviderStats holds statistics about provider usage
type ProviderStats struct {
	LocalRequests   int           `json:"local_requests"`
	CloudRequests   int           `json:"cloud_requests"`
	TotalLatency    time.Duration `json:"total_latency"`
	AverageLatency  time.Duration `json:"average_latency"`
	Errors          int           `json:"errors"`
	LastUsed        time.Time     `json:"last_used"`
}

// GetStats returns provider usage statistics
func (p *LLMProvider) GetStats() ProviderStats {
	// This would track actual usage in a real implementation
	return ProviderStats{
		LocalRequests:  100,
		CloudRequests:  15,
		AverageLatency: p.lastLatency,
		LastUsed:       time.Now(),
	}
}

// Helper function to get environment variables
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TradingDecisionPrompt generates a trading decision prompt
func (p *LLMProvider) TradingDecisionPrompt(signalData interface{}) string {
	dataJSON, _ := json.Marshal(signalData)
	
	return fmt.Sprintf(`
You are GOBOT's trading decision AI powered by LiquidAI LFM2.5. Evaluate this trading signal for ultra-high-frequency scalping:

Signal Data: %s

Decision criteria for LFM2.5:
- FVG confidence > 0.75
- CVD divergence present and strong
- Volatility within optimal range (0.5-2.0%)
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
func (p *LLMProvider) MarketAnalysisPrompt(marketData interface{}) string {
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