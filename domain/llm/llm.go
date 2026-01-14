package llm

import (
	"context"
	"sync"
	"time"
)

type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderGemini    ProviderType = "gemini"
	ProviderOllama    ProviderType = "ollama"
	ProviderGroq      ProviderType = "groq"
	ProviderDeepSeek  ProviderType = "deepseek"
	ProviderMistral   ProviderType = "mistral"
)

type ModelInfo struct {
	Name           string
	ContextLength  int
	CostPer1KToken float64
	MaxTokens      int
	SupportsJSON   bool
}

type RateLimit struct {
	RequestsPerMinute int
	RequestsPerHour   int
	TokensPerMinute   int
	TokensPerDay      int
}

type ProviderConfig struct {
	Type       ProviderType
	Name       string
	APIKeys    []string
	BaseURL    string
	Models     []ModelInfo
	RateLimits RateLimit
	Priority   int
	Enabled    bool
	Timeout    time.Duration
}

type ProviderState struct {
	LastRequest   time.Time
	RequestCount  int
	TokenCount    int
	CurrentTokens int
	IsHealthy     bool
	FailureCount  int
}

type LLMRequest struct {
	Model        string
	Messages     []Message
	SystemPrompt string
	Temperature  float64
	MaxTokens    int
	JSONMode     bool
}

type Message struct {
	Role    string
	Content string
}

type LLMResponse struct {
	Content    string
	TokensUsed int
	Cost       float64
	Provider   ProviderType
	Model      string
	Latency    time.Duration
}

type UsageStats struct {
	TotalRequests int
	TotalTokens   int
	TotalCost     float64
	ProviderUsage map[ProviderType]ProviderStats
	RateLimitHits int
	Failures      int
}

type ProviderStats struct {
	Requests int
	Tokens   int
	Cost     float64
	Failures int
	LastUsed time.Time
}

type LLMProvider interface {
	Type() ProviderType
	Name() string
	Configure(config ProviderConfig) error
	Validate() error
	Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error)
	GetRateLimit() RateLimit
	GetState() ProviderState
	IsHealthy(ctx context.Context) bool
}

type RouterConfig struct {
	Providers           []ProviderConfig
	DefaultProvider     ProviderType
	EnableFailover      bool
	EnableLoadBalancing bool
	HealthCheckInterval time.Duration
	RequestTimeout      time.Duration
	MaxRetries          int
	RetryDelay          time.Duration
}

type Router struct {
	cfg       RouterConfig
	providers map[ProviderType]LLMProvider
	state     map[ProviderType]*ProviderState
	usage     UsageStats
	mu        sync.RWMutex
}

func NewRouter(cfg RouterConfig) *Router {
	return &Router{
		cfg:       cfg,
		providers: make(map[ProviderType]LLMProvider),
		state:     make(map[ProviderType]*ProviderState),
		usage: UsageStats{
			ProviderUsage: make(map[ProviderType]ProviderStats),
		},
	}
}

func (r *Router) RegisterProvider(p LLMProvider) error {
	r.providers[p.Type()] = p
	r.state[p.Type()] = &ProviderState{
		IsHealthy: true,
	}
	return nil
}

func (r *Router) Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	var lastErr error
	providers := r.getOrderedProviders()

	for attempt := 0; attempt <= r.cfg.MaxRetries; attempt++ {
		for _, providerType := range providers {
			if attempt > 0 && attempt == 1 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(r.cfg.RetryDelay):
				}
			}

			provider, ok := r.providers[providerType]
			if !ok {
				continue
			}

			r.mu.RLock()
			state := r.state[providerType]
			r.mu.RUnlock()

			if state != nil && !state.IsHealthy {
				continue
			}

			if r.isRateLimited(providerType) {
				r.mu.Lock()
				r.usage.RateLimitHits++
				r.mu.Unlock()
				continue
			}

			resp, err := provider.Chat(ctx, req)
			if err != nil {
				lastErr = err
				r.mu.Lock()
				if r.state[providerType] != nil {
					r.state[providerType].FailureCount++
				}
				r.usage.Failures++
				if existing, ok := r.usage.ProviderUsage[providerType]; ok {
					existing.Failures++
					r.usage.ProviderUsage[providerType] = existing
				}
				r.mu.Unlock()
				continue
			}

			r.updateStats(providerType, resp)
			return resp, nil
		}
	}

	return nil, lastErr
}

func (r *Router) getOrderedProviders() []ProviderType {
	types := make([]ProviderType, 0, len(r.providers))
	for t := range r.providers {
		types = append(types, t)
	}

	if r.cfg.EnableLoadBalancing {
		r.sortByLoad(types)
	} else {
		r.sortByPriority(types)
	}

	return types
}

func (r *Router) sortByPriority(types []ProviderType) {
	for i := 0; i < len(types)-1; i++ {
		for j := i + 1; j < len(types); j++ {
			cfg := r.getProviderConfig(types[i])
			cfg2 := r.getProviderConfig(types[j])
			if cfg.Priority > cfg2.Priority {
				types[i], types[j] = types[j], types[i]
			}
		}
	}
}

func (r *Router) sortByLoad(types []ProviderType) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := 0; i < len(types)-1; i++ {
		for j := i + 1; j < len(types); j++ {
			stateI := r.state[types[i]]
			stateJ := r.state[types[j]]
			countI, countJ := 0, 0
			if stateI != nil {
				countI = stateI.RequestCount
			}
			if stateJ != nil {
				countJ = stateJ.RequestCount
			}
			if countJ < countI {
				types[i], types[j] = types[j], types[i]
			}
		}
	}
}

func (r *Router) isRateLimited(providerType ProviderType) bool {
	r.mu.RLock()
	state := r.state[providerType]
	r.mu.RUnlock()

	if state == nil {
		return false
	}

	now := time.Now()
	if state.LastRequest.IsZero() {
		return false
	}

	cfg := r.getProviderConfig(providerType)

	if now.Sub(state.LastRequest) < time.Minute {
		if state.RequestCount >= cfg.RateLimits.RequestsPerMinute {
			return true
		}
	}

	if now.Sub(state.LastRequest) < time.Hour {
		if state.RequestCount >= cfg.RateLimits.RequestsPerHour {
			return true
		}
	}

	return false
}

func (r *Router) updateStats(providerType ProviderType, resp *LLMResponse) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state[providerType] != nil {
		r.state[providerType].RequestCount++
		r.state[providerType].CurrentTokens = resp.TokensUsed
	}

	if existing, ok := r.usage.ProviderUsage[providerType]; ok {
		existing.Requests++
		existing.Tokens += resp.TokensUsed
		existing.Cost += resp.Cost
		existing.LastUsed = time.Now()
		r.usage.ProviderUsage[providerType] = existing
	} else {
		r.usage.ProviderUsage[providerType] = ProviderStats{
			Requests: 1,
			Tokens:   resp.TokensUsed,
			Cost:     resp.Cost,
			LastUsed: time.Now(),
		}
	}

	r.usage.TotalRequests++
	r.usage.TotalTokens += resp.TokensUsed
	r.usage.TotalCost += resp.Cost
}

func (r *Router) getProviderConfig(providerType ProviderType) ProviderConfig {
	for _, cfg := range r.cfg.Providers {
		if cfg.Type == providerType {
			return cfg
		}
	}
	return ProviderConfig{}
}

func (r *Router) GetUsageStats() UsageStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.usage
}

type FreeProviderManager struct {
	apiKeys map[ProviderType][]APIKeyEntry
}

type APIKeyEntry struct {
	Key       string
	Label     string
	Requests  int
	Tokens    int
	ExpiresAt time.Time
	LastUsed  time.Time
	Healthy   bool
}

func NewFreeProviderManager() *FreeProviderManager {
	return &FreeProviderManager{
		apiKeys: make(map[ProviderType][]APIKeyEntry),
	}
}

func (m *FreeProviderManager) AddAPIKey(provider ProviderType, key, label string) {
	m.apiKeys[provider] = append(m.apiKeys[provider], APIKeyEntry{
		Key:     key,
		Label:   label,
		Healthy: true,
	})
}

func (m *FreeProviderManager) GetBestKey(provider ProviderType) (string, error) {
	keys := m.apiKeys[provider]
	if len(keys) == 0 {
		return "", ErrNoAPIKeys
	}

	var bestKey *APIKeyEntry
	for i := range keys {
		key := &keys[i]
		if !key.Healthy {
			continue
		}
		if key.ExpiresAt.Before(time.Now()) {
			continue
		}
		if bestKey == nil || key.Requests < bestKey.Requests {
			bestKey = key
		}
	}

	if bestKey == nil {
		return "", ErrAllKeysExhausted
	}

	return bestKey.Key, nil
}

func (m *FreeProviderManager) UseKey(provider ProviderType, key string, tokens int) {
	keys := m.apiKeys[provider]
	for i := range keys {
		if keys[i].Key == key {
			keys[i].Requests++
			keys[i].Tokens += tokens
			keys[i].LastUsed = time.Now()
			break
		}
	}
}

func (m *FreeProviderManager) MarkKeyUnhealthy(provider ProviderType, key string) {
	keys := m.apiKeys[provider]
	for i := range keys {
		if keys[i].Key == key {
			keys[i].Healthy = false
			break
		}
	}
}

var (
	ErrNoAPIKeys        = &LLMError{Message: "no API keys configured for provider"}
	ErrAllKeysExhausted = &LLMError{Message: "all API keys exhausted or unhealthy"}
)

type LLMError struct {
	Message string
}

func (e *LLMError) Error() string {
	return e.Message
}
