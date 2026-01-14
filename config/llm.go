package config

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/britebrt/cognee/domain/llm"
)

type LLMConfig struct {
	Router    RouterConfig     `json:"router"`
	Providers []ProviderConfig `json:"providers"`
	CostTrack CostTrackConfig  `json:"cost_tracking"`
}

type RouterConfig struct {
	EnableFailover      bool          `json:"enable_failover"`
	EnableLoadBalancing bool          `json:"enable_load_balancing"`
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay_ms"`
	RequestTimeout      time.Duration `json:"request_timeout_ms"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

type ProviderConfig struct {
	Type       string        `json:"type"`
	Name       string        `json:"name"`
	Enabled    bool          `json:"enabled"`
	Priority   int           `json:"priority"`
	APIKeys    []string      `json:"api_keys"`
	APIKeysEnv string        `json:"api_keys_env"`
	BaseURL    string        `json:"base_url"`
	Models     []string      `json:"models"`
	RateLimits llm.RateLimit `json:"rate_limits"`
	Timeout    time.Duration `json:"timeout"`
}

type CostTrackConfig struct {
	DailyBudget    float64            `json:"daily_budget"`
	ProviderLimits map[string]float64 `json:"provider_limits"`
}

type N8NConfig struct {
	BaseURL     string        `json:"base_url"`
	APIKey      string        `json:"api_key"`
	WebhookAuth WebhookAuth   `json:"webhook_auth"`
	Workflows   []N8NWorkflow `json:"workflows"`
	Timeout     time.Duration `json:"timeout"`
}

type WebhookAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type N8NWorkflow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TriggerType string `json:"trigger_type"`
	Enabled     bool   `json:"enabled"`
}

func LoadLLMConfig(ctx context.Context) (*LLMConfig, error) {
	cfg := &LLMConfig{
		Router: RouterConfig{
			EnableFailover:      true,
			EnableLoadBalancing: true,
			MaxRetries:          3,
			RetryDelay:          1 * time.Second,
			RequestTimeout:      30 * time.Second,
			HealthCheckInterval: 1 * time.Minute,
		},
		Providers: []ProviderConfig{
			{
				Type:       "groq",
				Name:       "Groq (Fastest Free)",
				Enabled:    true,
				Priority:   1,
				APIKeysEnv: "GROQ_API_KEYS",
				BaseURL:    "https://api.groq.com/openai/v1",
				Models:     []string{"llama3-8b-8192", "llama3-70b-8192", "mixtral-8x7b-32768"},
				RateLimits: llm.RateLimit{RequestsPerMinute: 60, RequestsPerHour: 1440},
				Timeout:    10 * time.Second,
			},
			{
				Type:       "ollama",
				Name:       "Ollama (Local)",
				Enabled:    true,
				Priority:   2,
				APIKeys:    []string{},
				BaseURL:    getEnv("OLLAMA_BASE_URL", "http://localhost:11434/v1"),
				Models:     []string{"llama3", "llama3.2", "codellama", "mistral"},
				RateLimits: llm.RateLimit{RequestsPerMinute: 1000, RequestsPerHour: 10000},
				Timeout:    5 * time.Second,
			},
			{
				Type:       "deepseek",
				Name:       "DeepSeek",
				Enabled:    true,
				Priority:   3,
				APIKeysEnv: "DEEPSEEK_API_KEYS",
				BaseURL:    "https://api.deepseek.com/v1",
				Models:     []string{"deepseek-chat", "deepseek-coder"},
				RateLimits: llm.RateLimit{RequestsPerMinute: 60, RequestsPerHour: 3600},
				Timeout:    15 * time.Second,
			},
			{
				Type:       "gemini",
				Name:       "Google Gemini",
				Enabled:    true,
				Priority:   4,
				APIKeysEnv: "GEMINI_API_KEYS",
				BaseURL:    "https://generativelanguage.googleapis.com/v1beta",
				Models:     []string{"gemini-1.5-flash", "gemini-1.5-pro"},
				RateLimits: llm.RateLimit{RequestsPerMinute: 15, RequestsPerHour: 90},
				Timeout:    20 * time.Second,
			},
			{
				Type:       "huggingface",
				Name:       "HuggingFace",
				Enabled:    false,
				Priority:   5,
				APIKeysEnv: "HUGGINGFACE_API_KEYS",
				BaseURL:    "https://api-inference.huggingface.co/models",
				Models:     []string{"meta-llama/Llama-3.2-3B-Instruct", "microsoft/Phi-3-mini-4k-instruct"},
				RateLimits: llm.RateLimit{RequestsPerMinute: 50, RequestsPerHour: 1000},
				Timeout:    30 * time.Second,
			},
			{
				Type:       "openai",
				Name:       "OpenAI",
				Enabled:    false,
				Priority:   6,
				APIKeysEnv: "OPENAI_API_KEYS",
				BaseURL:    "https://api.openai.com/v1",
				Models:     []string{"gpt-3.5-turbo", "gpt-4o-mini"},
				RateLimits: llm.RateLimit{RequestsPerMinute: 60, RequestsPerHour: 3600},
				Timeout:    30 * time.Second,
			},
		},
		CostTrack: CostTrackConfig{
			DailyBudget:    10.0,
			ProviderLimits: map[string]float64{"openai": 5.0, "anthropic": 5.0},
		},
	}

	for i := range cfg.Providers {
		cfg.Providers[i].APIKeys = loadAPIKeys(cfg.Providers[i].APIKeysEnv)
	}

	return cfg, nil
}

func LoadN8NConfig(ctx context.Context) (*N8NConfig, error) {
	return &N8NConfig{
		BaseURL: getEnv("N8N_BASE_URL", "http://localhost:5678"),
		APIKey:  getEnv("N8N_API_KEY", ""),
		WebhookAuth: WebhookAuth{
			Username: getEnv("N8N_WEBHOOK_USER", "gobot"),
			Password: getEnv("N8N_WEBHOOK_PASS", "secure_password"),
		},
		Workflows: []N8NWorkflow{
			{ID: "trade_signal", Name: "Trade Signal Handler", TriggerType: "trade_signal", Enabled: true},
			{ID: "risk_alert", Name: "Risk Alert Handler", TriggerType: "risk_alert", Enabled: true},
			{ID: "market_analysis", Name: "AI Market Analysis", TriggerType: "market_data", Enabled: true},
			{ID: "position_update", Name: "Position Update", TriggerType: "position_update", Enabled: true},
			{ID: "daily_report", Name: "Daily Report", TriggerType: "schedule", Enabled: true},
		},
		Timeout: 30 * time.Second,
	}, nil
}

func loadAPIKeys(envVar string) []string {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return []string{}
	}

	keys := strings.Split(envValue, ",")
	for i := range keys {
		keys[i] = strings.TrimSpace(keys[i])
	}

	return keys
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (c *LLMConfig) ToLLMRouterConfig() llm.RouterConfig {
	providerCfgs := make([]llm.ProviderConfig, len(c.Providers))
	for i, p := range c.Providers {
		providerCfgs[i] = llm.ProviderConfig{
			Type:       llm.ProviderType(p.Type),
			Name:       p.Name,
			APIKeys:    p.APIKeys,
			BaseURL:    p.BaseURL,
			RateLimits: p.RateLimits,
			Priority:   p.Priority,
			Enabled:    p.Enabled,
			Timeout:    p.Timeout,
		}
	}

	return llm.RouterConfig{
		Providers:           providerCfgs,
		EnableFailover:      c.Router.EnableFailover,
		EnableLoadBalancing: c.Router.EnableLoadBalancing,
		MaxRetries:          c.Router.MaxRetries,
		RetryDelay:          c.Router.RetryDelay,
	}
}

type LLMConfigError struct {
	Message string
}

func (e *LLMConfigError) Error() string {
	return e.Message
}

var (
	ErrLLMMissingAPIKey = &LLMConfigError{Message: "API key is required but not set"}
	ErrLLMNoHealthyKeys = &LLMConfigError{Message: "no healthy API keys available"}
)
