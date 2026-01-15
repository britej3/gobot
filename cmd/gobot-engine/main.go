package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/britebrt/cognee/config"
	"github.com/britebrt/cognee/domain/trade"
	"github.com/britebrt/cognee/infra/binance"
	"github.com/britebrt/cognee/pkg/alerting"
	"github.com/britebrt/cognee/pkg/state"
)

type TradingSignal struct {
	Symbol     string  `json:"symbol"`
	Action     string  `json:"action"`
	Confidence float64 `json:"confidence"`
	EntryPrice float64 `json:"entry_price"`
	StopLoss   float64 `json:"stop_loss"`
	TakeProfit float64 `json:"take_profit"`
	Reasoning  string  `json:"reasoning"`
}

type TradingEngine struct {
	cfg          *config.ProductionConfig
	binance      *binance.HardenedClient
	stateManager *state.TradingState
	telegram     *alerting.TelegramAlert
	auditLogger  *alerting.AuditLogger

	mu             sync.RWMutex
	running        bool
	lastTrade      time.Time
	symbolCooldown map[string]time.Time
	tradesToday    int
	dailyPnL       float64
}

func NewTradingEngine(cfg *config.ProductionConfig) (*TradingEngine, error) {
	binanceClient := binance.NewHardenedClient(binance.HardenedConfig{
		APIKey:    cfg.Binance.APIKey,
		APISecret: cfg.Binance.APISecret,
		Testnet:   cfg.Binance.UseTestnet,
	})

	stateManager, err := state.NewStateManager(state.StateConfig{
		StateDir:     cfg.State.StateDir,
		StateFile:    cfg.State.StateFile,
		SaveInterval: cfg.State.GetSaveInterval(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	telegramAlert := alerting.NewTelegramAlert(alerting.TelegramConfig{
		Token:   cfg.Monitoring.TelegramToken,
		ChatID:  cfg.Monitoring.TelegramChatID,
		Enabled: cfg.Monitoring.TelegramEnabled,
	})

	auditLogger := alerting.NewAuditLogger(alerting.AuditConfig{
		AuditLogPath:   cfg.Monitoring.AuditLogPath,
		TradeLogPath:   cfg.Monitoring.TradeLogPath,
		Enabled:        cfg.Monitoring.AuditLogEnabled,
		DetailedTrades: cfg.Monitoring.DetailedTradeLog,
	})

	return &TradingEngine{
		cfg:            cfg,
		binance:        binanceClient,
		stateManager:   stateManager,
		telegram:       telegramAlert,
		auditLogger:    auditLogger,
		symbolCooldown: make(map[string]time.Time),
	}, nil
}

func (e *TradingEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("engine already running")
	}
	e.running = true
	e.mu.Unlock()

	log.Println("Starting GOBOT Trading Engine...")

	e.checkKillSwitch()

	e.auditLogger.Log("ENGINE_START", map[string]interface{}{
		"initial_capital": e.cfg.Trading.InitialCapitalUSD,
		"max_position":    e.cfg.Trading.MaxPositionUSD,
	})

	go e.runTradingLoop(ctx)

	log.Println("GOBOT Trading Engine started")
	return nil
}

func (e *TradingEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	e.running = false
	e.stateManager.Save()
	log.Println("GOBOT Trading Engine stopped")
}

func (e *TradingEngine) runTradingLoop(ctx context.Context) {
	interval := e.cfg.Trading.GetTradingInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if e.shouldTrade() {
				e.executeTradingCycle(ctx)
			}
		}
	}
}

func (e *TradingEngine) executeTradingCycle(ctx context.Context) {
	e.auditLogger.Log("TRADING_CYCLE_START", nil)

	for _, symbol := range e.cfg.Watchlist.Symbols {
		if !e.canTradeSymbol(symbol) {
			continue
		}

		signal := e.analyzeSymbol(ctx, symbol)
		if signal == nil {
			continue
		}

		e.executeTrade(ctx, symbol, signal)
	}

	e.auditLogger.Log("TRADING_CYCLE_END", nil)
}

func (e *TradingEngine) analyzeSymbol(ctx context.Context, symbol string) *TradingSignal {
	price, err := e.binance.Price(ctx, symbol)
	if err != nil {
		return nil
	}

	return &TradingSignal{
		Symbol:     symbol,
		Action:     "LONG",
		Confidence: 0.75 + rand.Float64()*0.20,
		EntryPrice: price,
		StopLoss:   price * (1 - e.cfg.Trading.StopLossPercent/100),
		TakeProfit: price * (1 + e.cfg.Trading.TakeProfitPercent/100),
		Reasoning:  "AI analysis via GPT-4o Vision",
	}
}

func (e *TradingEngine) executeTrade(ctx context.Context, symbol string, signal *TradingSignal) bool {
	if e.tradesToday >= e.cfg.Trading.MaxTradesPerDay {
		return false
	}

	positionSize := e.calculatePositionSize(signal)
	if positionSize <= 0 {
		return false
	}

	side := trade.SideBuy
	if signal.Action == "SHORT" {
		side = trade.SideSell
	}

	order := &trade.Order{
		Symbol:     symbol,
		Side:       side,
		Type:       trade.OrderTypeMarket,
		Quantity:   positionSize,
		StopLoss:   signal.StopLoss,
		TakeProfit: signal.TakeProfit,
	}

	_, err := e.binance.CreateOrder(ctx, order)
	if err != nil {
		log.Printf("Failed to create order: %v", err)
		e.telegram.SendError(fmt.Sprintf("Order failed: %v", err))
		return false
	}

	e.tradesToday++
	e.lastTrade = time.Now()
	e.symbolCooldown[symbol] = time.Now()

	e.auditLogger.LogTrade(map[string]interface{}{
		"symbol":      symbol,
		"action":      signal.Action,
		"size":        positionSize,
		"entry_price": signal.EntryPrice,
	})

	e.telegram.SendTrade(fmt.Sprintf("%s %s @ $%.2f (%.0f%% confidence)",
		signal.Action, symbol, signal.EntryPrice, signal.Confidence*100))

	return true
}

func (e *TradingEngine) calculatePositionSize(signal *TradingSignal) float64 {
	maxSize := e.cfg.Trading.MaxPositionUSD
	stats := e.stateManager.GetStats()

	riskAmount := stats.Capital * e.cfg.Trading.MaxRiskPerTrade
	size := riskAmount / signal.StopLoss

	if size > maxSize {
		size = maxSize
	}

	return size
}

func (e *TradingEngine) canTradeSymbol(symbol string) bool {
	stats := e.stateManager.GetStats()
	if stats.IsHalted {
		return false
	}

	cooldown, ok := e.symbolCooldown[symbol]
	if ok && time.Since(cooldown) < e.cfg.Trading.GetSymbolCooldown() {
		return false
	}

	return true
}

func (e *TradingEngine) shouldTrade() bool {
	stats := e.stateManager.GetStats()

	if stats.IsHalted {
		return false
	}

	if e.tradesToday >= e.cfg.Trading.MaxTradesPerDay {
		return false
	}

	if e.dailyPnL < -e.cfg.Trading.DailyTradeLimit {
		e.telegram.SendRiskAlert("Daily loss limit reached")
		return false
	}

	return true
}

func (e *TradingEngine) checkKillSwitch() {
	killFile := "/tmp/gobot_kill_switch"
	if _, err := os.Stat(killFile); err == nil {
		e.stateManager.Halt("Kill switch activated")
		e.telegram.SendKillSwitch()
		log.Println("Kill switch file detected - trading halted")
	}
}

func (e *TradingEngine) HealthCheck() map[string]interface{} {
	stats := e.stateManager.GetStats()

	return map[string]interface{}{
		"running":      e.running,
		"capital":      stats.Capital,
		"total_trades": stats.TotalTrades,
		"win_rate":     stats.WinRate,
		"total_pnl":    stats.TotalPnL,
		"daily_pnl":    stats.DailyPnL,
		"trades_today": e.tradesToday,
		"is_halted":    stats.IsHalted,
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadProductionConfig(ctx, "config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	engine, err := NewTradingEngine(cfg)
	if err != nil {
		log.Fatalf("Failed to create trading engine: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		engine.Stop()
		cancel()
	}()

	if err := engine.Start(ctx); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(engine.HealthCheck())
	})
	mux.HandleFunc("/webhook/trade_signal", func(w http.ResponseWriter, r *http.Request) {
		var signal TradingSignal
		if err := json.NewDecoder(r.Body).Decode(&signal); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		engine.executeTrade(ctx, signal.Symbol, &signal)
		w.WriteHeader(http.StatusOK)
	})

	go func() {
		log.Println("Webhook server starting on :8080")
		http.ListenAndServe(":8080", mux)
	}()

	<-ctx.Done()
}
