package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

// BrainConfig holds configuration for the brain engine
type BrainConfig struct {
	InferenceMode          string        `json:"inference_mode"`
	LocalModel             string        `json:"local_model"`
	LocalBaseURL           string        `json:"local_base_url"`
	CloudAPIKey            string        `json:"cloud_api_key"`
	CloudProvider          string        `json:"cloud_provider"`
	EnableRecovery         bool          `json:"enable_recovery"`
	RecoveryInterval       time.Duration `json:"recovery_interval"`
	DecisionTimeout        time.Duration `json:"decision_timeout"`
	MaxConcurrentDecisions int           `json:"max_concurrent_decisions"`
}

// BrainEngine is the main AI engine that coordinates all brain functions
type BrainEngine struct {
	provider       Provider
	feedback       interface{} // Simplified - would be *feedback.CogneeFeedbackSystem
	client         *futures.Client
	
	config         BrainConfig
	
	// State management
	mu             sync.RWMutex
	isRunning      bool
	shutdownChan   chan struct{}
	wg             sync.WaitGroup
	
	// Performance tracking
	startTime      time.Time
	decisionsMade  int
	recoveryCount  int
}

// NewBrainEngine creates a new brain engine
func NewBrainEngine(client *futures.Client, feedback interface{}, config BrainConfig) (*BrainEngine, error) {
	// Set default configuration for LiquidAI LFM2.5
	if config.LocalModel == "" {
		config.LocalModel = "lfm2.5-1.2b-instruct-q8_0:latest" // Available model from msty
	}
	if config.LocalBaseURL == "" {
		config.LocalBaseURL = "http://localhost:11964" // msty port
	}
	
	// Create the main provider using the configuration
	providerConfig := ProviderConfig{
		Mode:        InferenceMode(config.InferenceMode),
		LocalModel:  config.LocalModel,
		LocalBaseURL: config.LocalBaseURL,
		CloudAPIKey: config.CloudAPIKey,
		CloudProvider: config.CloudProvider,
		MaxRetries:  3,
		Timeout:     config.DecisionTimeout,
		ComplexityThreshold: 500,
	}
	
	provider, err := NewLLMProviderWithConfig(providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize provider: %w", err)
	}
	
	engine := &BrainEngine{
		provider:     provider,
		feedback:     feedback,
		client:       client,
		config:       config,
		shutdownChan: make(chan struct{}),
		startTime:    time.Now(),
	}

	logrus.WithFields(logrus.Fields{
		"inference_mode": config.InferenceMode,
		"local_model":    config.LocalModel,
		"local_base_url": config.LocalBaseURL,
		"cloud_provider": config.CloudProvider,
		"enable_recovery": config.EnableRecovery,
	}).Info("GOBOT LiquidAI LFM2.5 brain engine initialized")

	return engine, nil
}

// Start begins the brain engine operation
func (e *BrainEngine) Start() error {
	e.mu.Lock()
	if e.isRunning {
		return fmt.Errorf("brain engine already running")
	}
	e.isRunning = true
	e.mu.Unlock()

	logrus.Info("🧠 GOBOT LIQUIDAI: Starting LFM2.5 AI engine...")

	// Start background monitoring
	e.startBackgroundMonitoring()

	logrus.Info("✅ GOBOT LIQUIDAI: LFM2.5 AI engine started successfully")
	return nil
}

// Stop gracefully shuts down the brain engine
func (e *BrainEngine) Stop() error {
	e.mu.Lock()
	if !e.isRunning {
		return fmt.Errorf("brain engine not running")
	}
	e.isRunning = false
	e.mu.Unlock()

	logrus.Info("🛑 GOBOT LIQUIDAI: Shutting down LFM2.5 AI engine...")

	// Signal shutdown
	close(e.shutdownChan)
	
	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		logrus.Info("✅ GOBOT LIQUIDAI: All goroutines stopped")
	case <-time.After(30 * time.Second):
		logrus.Warn("⚠️ GOBOT LIQUIDAI: Some goroutines did not stop gracefully")
	}

	// Generate final report
	e.generateFinalReport()

	logrus.Info("✅ GOBOT LIQUIDAI: Shutdown complete")
	return nil
}

func (e *BrainEngine) startBackgroundMonitoring() {
	// Health monitoring
	e.wg.Add(1)
	go e.healthMonitoring()
	
	// Performance monitoring
	e.wg.Add(1)
	go e.performanceMonitoring()
}

// MakeTradingDecision makes a real-time trading decision
func (e *BrainEngine) MakeTradingDecision(ctx context.Context, signalData interface{}) (*TradingDecision, error) {
	e.mu.Lock()
	e.decisionsMade++
	e.mu.Unlock()

	// Create decision prompt
	prompt := e.provider.TradingDecisionPrompt(signalData)

	// Generate decision with timeout - faster for LFM2.5
	ctx, cancel := context.WithTimeout(ctx, e.config.DecisionTimeout)
	defer cancel()

	var decision TradingDecision
	if err := e.provider.GenerateStructuredResponse(ctx, prompt, &decision); err != nil {
		return nil, fmt.Errorf("failed to generate trading decision: %w", err)
	}

	// Validate decision
	if err := e.validateDecision(&decision); err != nil {
		return nil, fmt.Errorf("invalid trading decision: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"decision":   decision.Decision,
		"confidence": decision.Confidence,
		"symbol":     decision.Symbol,
		"reasoning":  decision.Reasoning,
		"latency_ms": time.Since(ctx.Value("start_time").(time.Time)).Milliseconds(),
	}).Info("GOBOT LFM2.5 trading decision generated")

	return &decision, nil
}

// AnalyzeMarket performs comprehensive market analysis
func (e *BrainEngine) AnalyzeMarket(ctx context.Context, marketData interface{}) (*MarketAnalysis, error) {
	// Create analysis prompt
	prompt := e.provider.MarketAnalysisPrompt(marketData)

	// Generate analysis with timeout - faster for LFM2.5
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second) // Reduced from 30s
	defer cancel()

	var analysis MarketAnalysis
	if err := e.provider.GenerateStructuredResponse(ctx, prompt, &analysis); err != nil {
		return nil, fmt.Errorf("failed to generate market analysis: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"market_regime": analysis.MarketRegime,
		"confidence":    analysis.Confidence,
		"key_factors":   analysis.KeyFactors,
	}).Info("GOBOT LFM2.5 market analysis generated")

	return &analysis, nil
}

// TradingDecision represents an AI-generated trading decision
type TradingDecision struct {
	Decision           string  `json:"decision"`
	Confidence         float64 `json:"confidence"`
	Reasoning          string `json:"reasoning"`
	RiskLevel          string `json:"risk_level"`
	RecommendedLeverage int    `json:"recommended_leverage"`
	Symbol             string `json:"symbol"`
	FVGConfidence      float64 `json:"fvg_confidence"`
	CVDDivergence      bool    `json:"cvd_divergence"`
}

// MarketAnalysis represents AI-generated market analysis
type MarketAnalysis struct {
	MarketRegime       string                 `json:"market_regime"`
	Confidence         float64                `json:"confidence"`
	KeyFactors         []string               `json:"key_factors"`
	StrategyAdjustments map[string]interface{} `json:"strategy_adjustments"`
}

// GetProviderStats returns provider usage statistics
func (e *BrainEngine) GetProviderStats() ProviderStats {
	if llmProvider, ok := e.provider.(*LLMProvider); ok {
		return llmProvider.GetStats()
	}
	return ProviderStats{}
}

// SwitchProviderMode allows runtime switching of inference modes
func (e *BrainEngine) SwitchProviderMode(mode InferenceMode) error {
	if llmProvider, ok := e.provider.(*LLMProvider); ok {
		llmProvider.SwitchMode(mode)
		logrus.WithField("mode", mode).Info("Switched GOBOT inference mode")
		return nil
	}
	return fmt.Errorf("provider does not support mode switching")
}

// GetEngineStats returns comprehensive engine statistics
func (e *BrainEngine) GetEngineStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"uptime":         time.Since(e.startTime).Round(time.Second),
		"decisions_made": e.decisionsMade,
		"recoveries":     e.recoveryCount,
		"is_running":     e.isRunning,
		"provider": map[string]interface{}{
			"model":        e.provider.GetModelName(),
			"healthy":      e.provider.IsHealthy(),
			"latency_ms":   e.provider.GetLatency().Milliseconds(),
			"base_url":     e.config.LocalBaseURL,
		},
	}
}

// IncrementDecisions increments the decisions made counter
func (e *BrainEngine) IncrementDecisions() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.decisionsMade++
	
	logrus.WithField("total_decisions", e.decisionsMade).Debug("Trading decision executed")
}

// validateDecision validates the AI-generated decision
func (e *BrainEngine) validateDecision(decision *TradingDecision) error {
	// Validate decision type
	if decision.Decision != "BUY" && decision.Decision != "SELL" && decision.Decision != "HOLD" {
		return fmt.Errorf("invalid decision: %s", decision.Decision)
	}

	// Validate confidence
	if decision.Confidence < 0 || decision.Confidence > 1 {
		return fmt.Errorf("invalid confidence: %f", decision.Confidence)
	}

	// Validate leverage
	if decision.RecommendedLeverage < 1 || decision.RecommendedLeverage > 125 {
		return fmt.Errorf("invalid leverage: %d", decision.RecommendedLeverage)
	}

	// Validate FVG confidence - higher threshold for LFM2.5
	if decision.FVGConfidence < 0 || decision.FVGConfidence > 1 {
		return fmt.Errorf("invalid FVG confidence: %f", decision.FVGConfidence)
	}

	return nil
}

func (e *BrainEngine) healthMonitoring() {
	defer e.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for e.isRunning {
		select {
		case <-ticker.C:
			// Check provider health
			if !e.provider.IsHealthy() {
				logrus.Error("GOBOT LiquidAI provider is unhealthy")
				// Attempt to reinitialize or switch providers
			}
		case <-e.shutdownChan:
			return
		}
	}
}

func (e *BrainEngine) performanceMonitoring() {
	defer e.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for e.isRunning {
		select {
		case <-ticker.C:
			e.mu.RLock()
			uptime := time.Since(e.startTime)
			decisions := e.decisionsMade
			recoveries := e.recoveryCount
			e.mu.RUnlock()
			
			logrus.WithFields(logrus.Fields{
				"uptime":          uptime.Round(time.Second),
				"decisions_made":  decisions,
				"recoveries":      recoveries,
				"provider":        e.provider.GetModelName(),
				"provider_healthy": e.provider.IsHealthy(),
				"base_url":        e.config.LocalBaseURL,
			}).Info("GOBOT LFM2.5 performance metrics")
		case <-e.shutdownChan:
			return
		}
	}
}

func (e *BrainEngine) generateFinalReport() {
	stats := e.GetEngineStats()
	
	logrus.WithFields(logrus.Fields{
		"uptime":         stats["uptime"],
		"total_decisions": stats["decisions_made"],
		"recoveries":     stats["recoveries"],
		"provider_model": e.provider.GetModelName(),
		"base_url":       e.config.LocalBaseURL,
	}).Info("GOBOT LFM2.5 - Final shutdown report")

	// Save detailed report to file
	reportFile := fmt.Sprintf("gobot_lfm25_report_%s.json", time.Now().Format("20060102_150405"))
	if err := e.saveReportToFile(reportFile, stats); err != nil {
		logrus.WithError(err).Warn("Failed to save final report")
	}
}

func (e *BrainEngine) saveReportToFile(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// DefaultBrainConfig returns default configuration for the brain engine
func DefaultBrainConfig() BrainConfig {
	return BrainConfig{
		InferenceMode:          "LOCAL",
		LocalModel:             "LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf", // Available model from msty
		LocalBaseURL:           "http://localhost:11454", // msty port
		CloudAPIKey:            os.Getenv("OPENAI_API_KEY"),
		CloudProvider:          "openai",
		EnableRecovery:         true,
		RecoveryInterval:       30 * time.Second,
		DecisionTimeout:        8 * time.Second, // Faster for LFM2.5
		MaxConcurrentDecisions: 5,
	}
}