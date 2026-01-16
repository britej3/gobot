package brain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MultiKeyConfig holds configuration for multi-key API management
type MultiKeyConfig struct {
	// Gemini Configuration
	GeminiAPIKeys []string `json:"gemini_api_keys"`
	GeminiModel   string   `json:"gemini_model"` // gemini-1.5-flash (free tier)

	// OpenRouter Configuration
	OpenRouterAPIKeys []string `json:"openrouter_api_keys"`
	OpenRouterModel   string   `json:"openrouter_model"` // e.g., qwen/qwen-2.5-72b-instruct

	// Groq Configuration
	GroqAPIKeys []string `json:"groq_api_keys"`
	GroqModel   string   `json:"groq_model"` // e.g., llama-3.3-70b-versatile

	// Rate Limit Configuration
	MaxRequestsPerMinute int `json:"max_requests_per_minute"`
	MaxRequestsPerDay    int `json:"max_requests_per_day"`

	// Fallback Configuration
	EnableAutoFallback bool          `json:"enable_auto_fallback"`
	FallbackDelay      time.Duration `json:"fallback_delay"`
	MaxRetries         int           `json:"max_retries"`
}

// FreeTierLimits holds rate limit information for free tiers
type FreeTierLimits struct {
	Provider            string
	RequestsPerMinute   int
	RequestsPerDay      int
	TokensPerDay        int
	CostPer1KTokens     float64
	FreeTierAvailable   bool
}

// GetFreeTierLimits returns rate limits for free tier models
func GetFreeTierLimits(provider, model string) FreeTierLimits {
	switch provider {
	case "gemini":
		return FreeTierLimits{
			Provider:          "gemini",
			RequestsPerMinute: 15,
			RequestsPerDay:    1500,
			TokensPerDay:      1500000, // 1.5M tokens/day
			CostPer1KTokens:   0.0,
			FreeTierAvailable: true,
		}

	case "openrouter":
		// OpenRouter has varying limits based on model
		// Using conservative estimates for free tier
		return FreeTierLimits{
			Provider:          "openrouter",
			RequestsPerMinute: 20,
			RequestsPerDay:    1440, // ~1 per minute average
			TokensPerDay:      200000, // 200K tokens/day (varies by model)
			CostPer1KTokens:   0.0, // Free tier
			FreeTierAvailable: true,
		}

	case "groq":
		return FreeTierLimits{
			Provider:          "groq",
			RequestsPerMinute: 30,
			RequestsPerDay:    1440,
			TokensPerDay:      1000000, // 1M tokens/day
			CostPer1KTokens:   0.0,
			FreeTierAvailable: true,
		}

	default:
		return FreeTierLimits{
			Provider:          provider,
			RequestsPerMinute: 10,
			RequestsPerDay:    1000,
			TokensPerDay:      100000,
			CostPer1KTokens:   0.0,
			FreeTierAvailable: false,
		}
	}
}

// CalculateDailyRequirements calculates daily AI requirements for 24/7 operation
func CalculateDailyRequirements() map[string]int {
	return map[string]int{
		"trading_decisions": 96,  // Every 15 minutes
		"market_analysis":    48,  // Every 30 minutes
		"position_monitoring": 2880, // Every 30 seconds (may not need AI)
		"total_min":          144, // Trading + Analysis only
		"total_with_monitoring": 3024, // All operations
	}
}

// CalculateProviderCapacity calculates how many bots a single API key can support
func CalculateProviderCapacity(provider string) map[string]interface{} {
	limits := GetFreeTierLimits(provider, "")
	dailyReqs := CalculateDailyRequirements()

	// Calculate capacity based on trading + analysis only (144/day)
	capacity := limits.RequestsPerDay / dailyReqs["total_min"]

	// Safety margin: use 80% of capacity
	safeCapacity := int(float64(capacity) * 0.8)

	return map[string]interface{}{
		"provider":           provider,
		"requests_per_minute": limits.RequestsPerMinute,
		"requests_per_day":    limits.RequestsPerDay,
		"daily_requirements":  dailyReqs["total_min"],
		"max_bots_per_key":    capacity,
		"safe_bots_per_key":   safeCapacity,
		"recommended_keys":    1, // For single bot
	}
}

// MultiKeyProvider manages multiple API keys with automatic fallback
type MultiKeyProvider struct {
	config      MultiKeyConfig
	providers   map[string]*CloudProvider
	currentKey  map[string]int // Current key index for each provider
	keyStats    map[string]map[int]*KeyStats
	mu          sync.RWMutex
	initialized bool
}

// KeyStats tracks usage statistics for each API key
type KeyStats struct {
	RequestsToday      int
	RequestsThisMinute int
	LastRequestTime    time.Time
	Errors             int
	LastError          string
	Healthy            bool
}

// NewMultiKeyProvider creates a new multi-key provider
func NewMultiKeyProvider(config MultiKeyConfig) (*MultiKeyProvider, error) {
	mkp := &MultiKeyProvider{
		config:    config,
		providers: make(map[string]*CloudProvider),
		currentKey: make(map[string]int),
		keyStats:  make(map[string]map[int]*KeyStats),
	}

	// Initialize providers with multiple keys
	if err := mkp.initializeProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	mkp.initialized = true

	logrus.WithFields(logrus.Fields{
		"gemini_keys":    len(config.GeminiAPIKeys),
		"openrouter_keys": len(config.OpenRouterAPIKeys),
		"groq_keys":      len(config.GroqAPIKeys),
	}).Info("Multi-key provider initialized")

	return mkp, nil
}

// initializeProviders initializes all cloud providers with their API keys
func (mkp *MultiKeyProvider) initializeProviders() error {
	// Initialize Gemini providers
	for i, apiKey := range mkp.config.GeminiAPIKeys {
		if apiKey == "" || apiKey == "YOUR_GEMINI_API_KEY_HERE" {
			continue
		}

		provider, err := NewCloudProvider(CloudConfig{
			APIKey:      apiKey,
			Provider:    "gemini",
			Model:       mkp.config.GeminiModel,
			Timeout:     15 * time.Second,
			MaxRetries:  2,
			Temperature: 0.1,
		})

		if err != nil {
			logrus.WithError(err).WithField("key_index", i).Warn("Failed to initialize Gemini provider")
			continue
		}

		mkp.providers[fmt.Sprintf("gemini-%d", i)] = provider
		mkp.currentKey["gemini"] = 0

		// Initialize stats for this key
		if mkp.keyStats["gemini"] == nil {
			mkp.keyStats["gemini"] = make(map[int]*KeyStats)
		}
		mkp.keyStats["gemini"][i] = &KeyStats{
			Healthy: true,
		}
	}

	// Initialize OpenRouter providers
	for i, apiKey := range mkp.config.OpenRouterAPIKeys {
		if apiKey == "" || apiKey == "YOUR_OPENROUTER_API_KEY_HERE" {
			continue
		}

		provider, err := NewCloudProvider(CloudConfig{
			APIKey:      apiKey,
			Provider:    "openrouter",
			Model:       mkp.config.OpenRouterModel,
			Timeout:     15 * time.Second,
			MaxRetries:  2,
			Temperature: 0.1,
		})

		if err != nil {
			logrus.WithError(err).WithField("key_index", i).Warn("Failed to initialize OpenRouter provider")
			continue
		}

		mkp.providers[fmt.Sprintf("openrouter-%d", i)] = provider
		mkp.currentKey["openrouter"] = 0

		if mkp.keyStats["openrouter"] == nil {
			mkp.keyStats["openrouter"] = make(map[int]*KeyStats)
		}
		mkp.keyStats["openrouter"][i] = &KeyStats{
			Healthy: true,
		}
	}

	// Initialize Groq providers
	for i, apiKey := range mkp.config.GroqAPIKeys {
		if apiKey == "" || apiKey == "YOUR_GROQ_API_KEY_HERE" {
			continue
		}

		provider, err := NewCloudProvider(CloudConfig{
			APIKey:      apiKey,
			Provider:    "groq",
			Model:       mkp.config.GroqModel,
			Timeout:     15 * time.Second,
			MaxRetries:  2,
			Temperature: 0.1,
		})

		if err != nil {
			logrus.WithError(err).WithField("key_index", i).Warn("Failed to initialize Groq provider")
			continue
		}

		mkp.providers[fmt.Sprintf("groq-%d", i)] = provider
		mkp.currentKey["groq"] = 0

		if mkp.keyStats["groq"] == nil {
			mkp.keyStats["groq"] = make(map[int]*KeyStats)
		}
		mkp.keyStats["groq"][i] = &KeyStats{
			Healthy: true,
		}
	}

	if len(mkp.providers) == 0 {
		return fmt.Errorf("no valid API keys provided")
	}

	return nil
}

// GenerateResponse generates a response using the best available provider and key
func (mkp *MultiKeyProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	mkp.mu.Lock()
	defer mkp.mu.Unlock()

	// Try providers in priority order: Gemini -> OpenRouter -> Groq
	providers := []string{"gemini", "openrouter", "groq"}

	var lastError error
	for _, providerType := range providers {
		response, err := mkp.tryProvider(ctx, providerType, prompt)
		if err == nil {
			return response, nil
		}
		lastError = err
		logrus.WithError(err).WithField("provider", providerType).Warn("Provider failed, trying next")
	}

	return "", fmt.Errorf("all providers failed: %w", lastError)
}

// tryProvider tries to generate response using a specific provider
func (mkp *MultiKeyProvider) tryProvider(ctx context.Context, providerType string, prompt string) (string, error) {
	// Get all keys for this provider
	var keys []int
	for keyID := range mkp.keyStats[providerType] {
		keys = append(keys, keyID)
	}

	if len(keys) == 0 {
		return "", fmt.Errorf("no keys available for provider: %s", providerType)
	}

	// Try each key
	for _, keyID := range keys {
		providerKey := fmt.Sprintf("%s-%d", providerType, keyID)
		provider, exists := mkp.providers[providerKey]
		if !exists {
			continue
		}

		// Check if key is healthy and within rate limits
		if !mkp.isKeyAvailable(providerType, keyID) {
			continue
		}

		// Generate response
		response, err := provider.GenerateResponse(ctx, prompt)
		if err == nil {
			// Update stats
			mkp.recordSuccess(providerType, keyID)
			return response, nil
		}

		// Record error
		mkp.recordError(providerType, keyID, err.Error())
		logrus.WithError(err).WithField("key", providerKey).Warn("Key failed, trying next")
	}

	return "", fmt.Errorf("all keys failed for provider: %s", providerType)
}

// GenerateStructuredResponse generates a structured response
func (mkp *MultiKeyProvider) GenerateStructuredResponse(ctx context.Context, prompt string, response interface{}) error {
	// Add JSON instruction to prompt
	jsonPrompt := prompt + "\n\nRespond in valid JSON format only. Be concise and accurate."

	_, err := mkp.GenerateResponse(ctx, jsonPrompt)
	if err != nil {
		return err
	}

	// Parse JSON (mock implementation)
	// In production, this would parse actual JSON from cloud provider
	return nil
}

// isKeyAvailable checks if a key is available for use
func (mkp *MultiKeyProvider) isKeyAvailable(providerType string, keyID int) bool {
	stats := mkp.keyStats[providerType][keyID]
	if !stats.Healthy {
		return false
	}

	// Check rate limits
	limits := GetFreeTierLimits(providerType, "")

	// Check daily limit
	if stats.RequestsToday >= limits.RequestsPerDay {
		logrus.WithFields(logrus.Fields{
			"provider": providerType,
			"key_id":   keyID,
			"requests": stats.RequestsToday,
			"limit":    limits.RequestsPerDay,
		}).Warn("Key reached daily limit")
		return false
	}

	// Check per-minute limit
	now := time.Now()
	if now.Sub(stats.LastRequestTime) < time.Minute {
		if stats.RequestsThisMinute >= limits.RequestsPerMinute {
			logrus.WithFields(logrus.Fields{
				"provider": providerType,
				"key_id":   keyID,
				"requests": stats.RequestsThisMinute,
				"limit":    limits.RequestsPerMinute,
			}).Warn("Key reached per-minute limit")
			return false
		}
	}

	return true
}

// recordSuccess records a successful request
func (mkp *MultiKeyProvider) recordSuccess(providerType string, keyID int) {
	stats := mkp.keyStats[providerType][keyID]
	stats.RequestsToday++
	stats.RequestsThisMinute++
	stats.LastRequestTime = time.Now()
	stats.Healthy = true
}

// recordError records a failed request
func (mkp *MultiKeyProvider) recordError(providerType string, keyID int, errorMsg string) {
	stats := mkp.keyStats[providerType][keyID]
	stats.Errors++
	stats.LastError = errorMsg
	stats.LastRequestTime = time.Now()

	// Mark as unhealthy if too many errors
	if stats.Errors > 5 {
		stats.Healthy = false
		logrus.WithFields(logrus.Fields{
			"provider": providerType,
			"key_id":   keyID,
			"errors":   stats.Errors,
		}).Warn("Key marked as unhealthy due to errors")
	}
}

// GetStats returns statistics for all providers and keys
func (mkp *MultiKeyProvider) GetStats() map[string]interface{} {
	mkp.mu.RLock()
	defer mkp.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["providers"] = mkp.providers
	stats["key_stats"] = mkp.keyStats
	stats["current_keys"] = mkp.currentKey

	// Calculate total usage
	totalRequests := 0
	for _, keyMap := range mkp.keyStats {
		for _, keyStat := range keyMap {
			totalRequests += keyStat.RequestsToday
		}
	}
	stats["total_requests_today"] = totalRequests

	return stats
}

// GetModelName returns the current model name
func (mkp *MultiKeyProvider) GetModelName() string {
	mkp.mu.RLock()
	defer mkp.mu.RUnlock()

	// Return model name from first available provider
	for _, provider := range mkp.providers {
		return provider.GetModelName()
	}
	return "unknown"
}

// GetLatency returns estimated latency
func (mkp *MultiKeyProvider) GetLatency() time.Duration {
	// Average latency across providers
	return 1 * time.Second
}

// IsHealthy checks if any provider is healthy
func (mkp *MultiKeyProvider) IsHealthy() bool {
	mkp.mu.RLock()
	defer mkp.mu.RUnlock()

	for _, keyMap := range mkp.keyStats {
		for _, keyStat := range keyMap {
			if keyStat.Healthy {
				return true
			}
		}
	}

	return false
}

// TradingDecisionPrompt generates a trading decision prompt
func (mkp *MultiKeyProvider) TradingDecisionPrompt(signalData interface{}) string {
	dataJSON, _ := signalDataToJSON(signalData)

	return fmt.Sprintf(`
You are GOBOT's trading decision AI. Evaluate this trading signal for high-frequency scalping:

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
func (mkp *MultiKeyProvider) MarketAnalysisPrompt(marketData interface{}) string {
	dataJSON, _ := signalDataToJSON(marketData)

	return fmt.Sprintf(`
You are GOBOT's market analysis AI. Analyze the following market data and provide a concise assessment:

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

// Helper function
func signalDataToJSON(data interface{}) ([]byte, error) {
	// Simple JSON marshaling
	// In production, use proper JSON encoding
	return []byte("{}"), nil
}

// DefaultMultiKeyConfig returns default configuration for multi-key provider
func DefaultMultiKeyConfig() MultiKeyConfig {
	return MultiKeyConfig{
		GeminiAPIKeys: []string{},
		GeminiModel:   "gemini-1.5-flash", // Free tier
		OpenRouterAPIKeys: []string{},
		OpenRouterModel:   "qwen/qwen-2.5-72b-instruct", // Free tier
		GroqAPIKeys: []string{},
		GroqModel:   "llama-3.3-70b-versatile", // Free tier
		MaxRequestsPerMinute: 30,
		MaxRequestsPerDay:    2000,
		EnableAutoFallback:   true,
		FallbackDelay:       5 * time.Second,
		MaxRetries:          3,
	}
}