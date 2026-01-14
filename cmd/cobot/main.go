package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/britebrt/cognee/config"
	"github.com/britebrt/cognee/domain/automation"
	"github.com/britebrt/cognee/domain/executor"
	"github.com/britebrt/cognee/domain/llm"
	"github.com/britebrt/cognee/domain/platform"
	"github.com/britebrt/cognee/domain/selector"
	"github.com/britebrt/cognee/domain/strategy"
	"github.com/britebrt/cognee/infra/binance"
	"github.com/britebrt/cognee/services/executor/market"
	"github.com/britebrt/cognee/services/screenshot"
	"github.com/britebrt/cognee/services/selector/volume"
	"github.com/britebrt/cognee/services/strategy/scalper"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Starting GOBOT v2.0 with N8N + LLM Router...")

	binanceClient := binance.New(binance.Config{
		APIKey:    os.Getenv("BINANCE_API_KEY"),
		APISecret: os.Getenv("BINANCE_API_SECRET"),
		Testnet:   os.Getenv("BINANCE_USE_TESTNET") == "true",
	})
	_ = binanceClient

	llmCfg, err := config.LoadLLMConfig(ctx)
	if err != nil {
		log.Printf("Warning: Failed to load LLM config: %v", err)
	}

	router := llm.NewRouter(llmCfg.ToLLMRouterConfig())

	n8nCfg, err := config.LoadN8NConfig(ctx)
	if err != nil {
		log.Printf("Warning: Failed to load N8N config: %v", err)
	}

	engine := platform.NewPlatformEngine()

	engine.RegisterStrategy(strategy.StrategyScalper, func() strategy.Strategy {
		return &scalper.ScalperStrategy{}
	})

	engine.RegisterSelector(selector.SelectorVolume, func() selector.Selector {
		return &volume.VolumeSelector{}
	})

	engine.RegisterExecutor(executor.ExecutionMarket, func() executor.Executor {
		return &market.MarketExecutor{}
	})

	engine.RegisterAutomation(automation.AutomationN8N, func() automation.Automation {
		return &automation.N8NAutomation{}
	})

	p := &platform.Platform{
		Cfg: platform.PlatformConfig{
			Name:    "GOBOT",
			Version: "2.0.0",
			StrategyConfig: strategy.StrategyConfig{
				Type:    strategy.StrategyScalper,
				Name:    "scalper_strategy",
				Version: "1.0.0",
				Enabled: true,
				RiskParameters: strategy.RiskConfig{
					StopLossPercent:   0.5,
					TakeProfitPercent: 1.5,
					RiskPerTrade:      0.02,
				},
			},
			SelectorConfig: selector.SelectorConfig{
				Type:          selector.SelectorVolume,
				Name:          "volume_selector",
				Enabled:       true,
				MinVolume:     1000000,
				MaxAssets:     15,
				MinConfidence: 0.65,
			},
			ExecutorConfig: executor.ExecutionConfig{
				Type:              executor.ExecutionMarket,
				Name:              "market_executor",
				Enabled:           true,
				SlippageTolerance: 0.001,
				MaxRetries:        3,
			},
			AutomationConfig: automation.AutomationConfig{
				Type:    automation.AutomationN8N,
				Name:    "n8n_automation",
				Enabled: true,
				N8NConfig: automation.N8NConfig{
					BaseURL:   n8nCfg.BaseURL,
					APIKey:    n8nCfg.APIKey,
					Workflows: convertN8NWorkflows(n8nCfg.Workflows),
				},
			},
		},
		Engine:     engine,
		Components: &platform.Components{},
	}

	if err := p.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize platform: %v", err)
	}

	if err := p.Start(ctx); err != nil {
		log.Fatalf("Failed to start platform: %v", err)
	}

	go startWebhookServer(ctx, n8nCfg)

	go runTradingCycle(ctx, p)

	log.Println("GOBOT started successfully!")
	log.Printf("N8N Webhooks available at: %s/webhook/", n8nCfg.BaseURL)
	log.Printf("LLM Router active with %d providers", len(llmCfg.Providers))

	stats := router.GetUsageStats()
	log.Printf("LLM Stats - Requests: %d, Tokens: %d, Cost: $%.4f",
		stats.TotalRequests, stats.TotalTokens, stats.TotalCost)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()
	p.Stop()
	log.Println("Shutdown complete")
}

func startWebhookServer(ctx context.Context, cfg *config.N8NConfig) {
	mux := http.NewServeMux()

	mux.HandleFunc("/webhook/trade_signal", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		log.Printf("Received trade signal from N8N: %v", data)
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/webhook/risk-alert", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		log.Printf("Received risk alert from N8N: %v", data)
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/webhook/market-analysis", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		log.Printf("Received market analysis from N8N: %v", data)
		w.WriteHeader(http.StatusOK)
	})

	// TradingView Screenshot endpoint - triggered by GOBOT
	mux.HandleFunc("/webhook/capture-chart", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Symbol    string   `json:"symbol"`
			Intervals []string `json:"intervals,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Symbol == "" {
			http.Error(w, "Missing symbol", http.StatusBadRequest)
			return
		}

		if len(req.Intervals) == 0 {
			req.Intervals = []string{"1m", "5m", "15m"}
		}

		log.Printf("📸 Capturing charts for %s at %v", req.Symbol, req.Intervals)

		// Call TradingView screenshot service
		screenshotClient := screenshot.NewClient(screenshot.Config{
			ServerURL: "http://localhost:3000",
		}, slog.Default())

		result, err := screenshotClient.CaptureMulti(req.Symbol, req.Intervals)
		if err != nil {
			log.Printf("Screenshot failed: %v", err)
			http.Error(w, fmt.Sprintf("Screenshot failed: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Captured %d charts for %s", len(result.Results), req.Symbol)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// Trigger QuantCrawler analysis with screenshots
	mux.HandleFunc("/webhook/analyze-symbol", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Symbol         string  `json:"symbol"`
			AccountBalance float64 `json:"account_balance"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Symbol == "" {
			http.Error(w, "Missing symbol", http.StatusBadRequest)
			return
		}

		log.Printf("🎯 Starting analysis workflow for %s", req.Symbol)

		// Step 1: Capture screenshots
		screenshotClient := screenshot.NewClient(screenshot.Config{
			ServerURL: "http://localhost:3000",
		}, slog.Default())

		result, err := screenshotClient.CaptureMulti(req.Symbol, []string{"1m", "5m", "15m"})
		if err != nil {
			log.Printf("Screenshot failed: %v", err)
		}

		log.Printf("📊 Charts captured, ready for analysis")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"symbol":      req.Symbol,
			"screenshots": result.Results,
			"status":      "ready_for_analysis",
			"next_step":   "Send to QuantCrawler for AI analysis",
		})
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Close()
	}()

	log.Printf("Webhook server started on :8080")
	server.ListenAndServe()
}

func runTradingCycle(ctx context.Context, p *platform.Platform) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.RunCycle(ctx); err != nil {
				log.Printf("Trading cycle error: %v", err)
			}
		}
	}
}

func convertN8NWorkflows(workflows []config.N8NWorkflow) []automation.N8NWorkflow {
	result := make([]automation.N8NWorkflow, len(workflows))
	for i, w := range workflows {
		result[i] = automation.N8NWorkflow{
			ID:          w.ID,
			Name:        w.Name,
			TriggerType: w.TriggerType,
			Enabled:     w.Enabled,
		}
	}
	return result
}

func init() {
	fmt.Println("GOBOT v2.0 - Modular Trading Platform with N8N + LLM Router")
}
