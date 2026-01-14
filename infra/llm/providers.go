package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/britebrt/cognee/domain/llm"
)

type OpenAIProvider struct {
	cfg   llm.ProviderConfig
	state llm.ProviderState
}

func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{}
}

func (p *OpenAIProvider) Type() llm.ProviderType {
	return llm.ProviderOpenAI
}

func (p *OpenAIProvider) Name() string {
	return "openai_provider"
}

func (p *OpenAIProvider) Configure(config llm.ProviderConfig) error {
	p.cfg = config
	return nil
}

func (p *OpenAIProvider) Validate() error {
	if len(p.cfg.APIKeys) == 0 {
		return fmt.Errorf("OpenAI API key required")
	}
	return nil
}

func (p *OpenAIProvider) Chat(ctx context.Context, req llm.LLMRequest) (*llm.LLMResponse, error) {
	start := time.Now()

	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": buildMessages(req),
	}

	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	if req.JSONMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKeys[0])

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OpenAI API error: %s", string(respBody))
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	choices := result["choices"].([]interface{})
	message := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	content := message["content"].(string)

	usage := result["usage"].(map[string]interface{})
	tokens := 0
	if usage["total_tokens"] != nil {
		tokens = int(usage["total_tokens"].(float64))
	}

	cost := float64(tokens) * 0.01 / 1000

	return &llm.LLMResponse{
		Content:    content,
		TokensUsed: tokens,
		Cost:       cost,
		Provider:   llm.ProviderOpenAI,
		Model:      req.Model,
		Latency:    time.Since(start),
	}, nil
}

func (p *OpenAIProvider) GetRateLimit() llm.RateLimit {
	return llm.RateLimit{
		RequestsPerMinute: 60,
		RequestsPerHour:   3600,
	}
}

func (p *OpenAIProvider) GetState() llm.ProviderState {
	return p.state
}

func (p *OpenAIProvider) IsHealthy(ctx context.Context) bool {
	return p.state.IsHealthy
}

func buildMessages(req llm.LLMRequest) []map[string]interface{} {
	messages := make([]map[string]interface{}, 0)

	if req.SystemPrompt != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	return messages
}

type GeminiProvider struct {
	cfg   llm.ProviderConfig
	state llm.ProviderState
}

func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{}
}

func (p *GeminiProvider) Type() llm.ProviderType {
	return llm.ProviderGemini
}

func (p *GeminiProvider) Name() string {
	return "gemini_provider"
}

func (p *GeminiProvider) Configure(config llm.ProviderConfig) error {
	p.cfg = config
	return nil
}

func (p *GeminiProvider) Validate() error {
	if len(p.cfg.APIKeys) == 0 {
		return fmt.Errorf("Gemini API key required")
	}
	return nil
}

func (p *GeminiProvider) Chat(ctx context.Context, req llm.LLMRequest) (*llm.LLMResponse, error) {
	start := time.Now()

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.cfg.BaseURL, req.Model, p.cfg.APIKeys[0])

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": buildGeminiParts(req),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Gemini API error: %s", string(respBody))
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	candidates := result["candidates"].([]interface{})
	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	text := parts[0].(map[string]interface{})["text"].(string)

	tokens := estimateTokens(text)

	cost := float64(tokens) * 0.0005 / 1000

	return &llm.LLMResponse{
		Content:    text,
		TokensUsed: tokens,
		Cost:       cost,
		Provider:   llm.ProviderGemini,
		Model:      req.Model,
		Latency:    time.Since(start),
	}, nil
}

func (p *GeminiProvider) GetRateLimit() llm.RateLimit {
	return llm.RateLimit{
		RequestsPerMinute: 15,
		RequestsPerHour:   90,
	}
}

func (p *GeminiProvider) GetState() llm.ProviderState {
	return p.state
}

func (p *GeminiProvider) IsHealthy(ctx context.Context) bool {
	return p.state.IsHealthy
}

func buildGeminiParts(req llm.LLMRequest) []map[string]interface{} {
	parts := make([]map[string]interface{}, 0)

	if req.SystemPrompt != "" {
		parts = append(parts, map[string]interface{}{
			"text": req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		parts = append(parts, map[string]interface{}{
			"text": msg.Content,
		})
	}

	return parts
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func main() {
	router := llm.NewRouter(llm.RouterConfig{
		Providers: []llm.ProviderConfig{
			{
				Type:       llm.ProviderOpenAI,
				Name:       "OpenAI",
				APIKeys:    []string{os.Getenv("OPENAI_API_KEY")},
				BaseURL:    "https://api.openai.com/v1",
				RateLimits: llm.RateLimit{RequestsPerMinute: 60},
				Priority:   1,
				Enabled:    true,
			},
			{
				Type:       llm.ProviderGemini,
				Name:       "Google Gemini",
				APIKeys:    []string{os.Getenv("GEMINI_API_KEY")},
				BaseURL:    "https://generativelanguage.googleapis.com",
				RateLimits: llm.RateLimit{RequestsPerMinute: 15},
				Priority:   2,
				Enabled:    true,
			},
		},
		EnableFailover:      true,
		EnableLoadBalancing: true,
		MaxRetries:          3,
		RetryDelay:          time.Second,
	})

	openai := NewOpenAIProvider()
	openai.Configure(llm.ProviderConfig{
		Type:       llm.ProviderOpenAI,
		APIKeys:    []string{os.Getenv("OPENAI_API_KEY")},
		BaseURL:    "https://api.openai.com/v1",
		RateLimits: llm.RateLimit{RequestsPerMinute: 60},
	})

	gemini := NewGeminiProvider()
	gemini.Configure(llm.ProviderConfig{
		Type:       llm.ProviderGemini,
		APIKeys:    []string{os.Getenv("GEMINI_API_KEY")},
		BaseURL:    "https://generativelanguage.googleapis.com",
		RateLimits: llm.RateLimit{RequestsPerMinute: 15},
	})

	router.RegisterProvider(openai)
	router.RegisterProvider(gemini)

	ctx := context.Background()
	resp, err := router.Chat(ctx, llm.LLMRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    []llm.Message{{Role: "user", Content: "Hello"}},
		Temperature: 0.7,
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.Content)
	fmt.Printf("Provider: %s\n", resp.Provider)
	fmt.Printf("Latency: %v\n", resp.Latency)
	fmt.Printf("Cost: $%.6f\n", resp.Cost)
}
