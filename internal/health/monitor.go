// Package health - Real-time system monitoring
package health

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ============================================================================
// Real-time Monitor Types
// ============================================================================

// SystemMetrics contains real-time system metrics
type SystemMetrics struct {
	Timestamp       time.Time     `json:"timestamp"`
	Uptime          time.Duration `json:"uptime"`
	
	// Memory
	MemoryAlloc     uint64 `json:"memory_alloc_mb"`
	MemoryTotal     uint64 `json:"memory_total_mb"`
	MemorySys       uint64 `json:"memory_sys_mb"`
	NumGoroutines   int    `json:"num_goroutines"`
	NumGC           uint32 `json:"num_gc"`
	
	// Trading
	ActivePositions int     `json:"active_positions"`
	OpenOrders      int     `json:"open_orders"`
	TotalTrades     int     `json:"total_trades"`
	WinRate         float64 `json:"win_rate"`
	DailyPnL        float64 `json:"daily_pnl"`
	TotalPnL        float64 `json:"total_pnl"`
	
	// Account
	WalletBalance   float64 `json:"wallet_balance"`
	AvailableBalance float64 `json:"available_balance"`
	MarginUsed      float64 `json:"margin_used"`
	MarginRatio     float64 `json:"margin_ratio"`
	
	// API
	APILatency      time.Duration `json:"api_latency_ms"`
	APIErrors       int           `json:"api_errors"`
	APIRateLimit    float64       `json:"api_rate_limit_pct"`
	
	// Status
	IsTrading       bool   `json:"is_trading"`
	IsFirstTrade    bool   `json:"is_first_trade"`
	SessionStatus   string `json:"session_status"`
	LastError       string `json:"last_error,omitempty"`
}

// PositionInfo contains position details for display
type PositionInfo struct {
	Symbol          string  `json:"symbol"`
	Side            string  `json:"side"` // LONG, SHORT
	Size            float64 `json:"size"`
	EntryPrice      float64 `json:"entry_price"`
	MarkPrice       float64 `json:"mark_price"`
	Leverage        int     `json:"leverage"`
	UnrealizedPnL   float64 `json:"unrealized_pnl"`
	PnLPercent      float64 `json:"pnl_percent"`
	LiquidationPrice float64 `json:"liquidation_price"`
	LiquidationDist  float64 `json:"liquidation_dist_pct"`
	Duration        time.Duration `json:"duration"`
	OpenedAt        time.Time     `json:"opened_at"`
}

// TradeInfo contains trade details for display
type TradeInfo struct {
	ID            string    `json:"id"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	EntryPrice    float64   `json:"entry_price"`
	ExitPrice     float64   `json:"exit_price"`
	Size          float64   `json:"size"`
	Leverage      int       `json:"leverage"`
	PnL           float64   `json:"pnl"`
	PnLPercent    float64   `json:"pnl_percent"`
	Fees          float64   `json:"fees"`
	Duration      time.Duration `json:"duration"`
	Reason        string    `json:"reason"`
	ExitReason    string    `json:"exit_reason"`
	OpenedAt      time.Time `json:"opened_at"`
	ClosedAt      time.Time `json:"closed_at"`
}

// WalletInfo contains wallet details for display
type WalletInfo struct {
	TotalBalance     float64 `json:"total_balance"`
	AvailableBalance float64 `json:"available_balance"`
	UnrealizedPnL    float64 `json:"unrealized_pnl"`
	MarginBalance    float64 `json:"margin_balance"`
	UsedMargin       float64 `json:"used_margin"`
	MarginRatio      float64 `json:"margin_ratio"`
	MaxWithdraw      float64 `json:"max_withdraw"`
	
	// Daily stats
	DailyStartBalance float64 `json:"daily_start_balance"`
	DailyPnL          float64 `json:"daily_pnl"`
	DailyPnLPercent   float64 `json:"daily_pnl_percent"`
	DailyTrades       int     `json:"daily_trades"`
	DailyWins         int     `json:"daily_wins"`
	DailyLosses       int     `json:"daily_losses"`
}

// TopMoverInfo contains top mover details for display
type TopMoverInfo struct {
	Symbol          string  `json:"symbol"`
	Category        string  `json:"category"`
	PriceChange     float64 `json:"price_change_pct"`
	Volume24h       float64 `json:"volume_24h"`
	LastPrice       float64 `json:"last_price"`
	MomentumScore   float64 `json:"momentum_score"`
	IsSelected      bool    `json:"is_selected"`
}

// ============================================================================
// System Monitor
// ============================================================================

// SystemMonitor provides real-time system monitoring
type SystemMonitor struct {
	mu              sync.RWMutex
	startTime       time.Time
	metrics         *SystemMetrics
	positions       []PositionInfo
	recentTrades    []TradeInfo
	wallet          *WalletInfo
	topMovers       []TopMoverInfo
	healthChecker   *HealthChecker
	
	// Callbacks for data updates
	onMetricsUpdate func(*SystemMetrics)
	onPositionUpdate func([]PositionInfo)
	onTradeUpdate    func(TradeInfo)
	onWalletUpdate   func(*WalletInfo)
	
	// Control
	stopChan        chan struct{}
	updateInterval  time.Duration
}

// NewSystemMonitor creates a new system monitor
func NewSystemMonitor(hc *HealthChecker) *SystemMonitor {
	return &SystemMonitor{
		startTime:      time.Now(),
		metrics:        &SystemMetrics{},
		positions:      make([]PositionInfo, 0),
		recentTrades:   make([]TradeInfo, 0, 100),
		wallet:         &WalletInfo{},
		topMovers:      make([]TopMoverInfo, 0),
		healthChecker:  hc,
		stopChan:       make(chan struct{}),
		updateInterval: 1 * time.Second,
	}
}

// Start begins real-time monitoring
func (m *SystemMonitor) Start(ctx context.Context) {
	go m.runMetricsLoop(ctx)
}

// Stop halts monitoring
func (m *SystemMonitor) Stop() {
	close(m.stopChan)
}

// runMetricsLoop continuously updates system metrics
func (m *SystemMonitor) runMetricsLoop(ctx context.Context) {
	ticker := time.NewTicker(m.updateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.updateRuntimeMetrics()
		}
	}
}

// updateRuntimeMetrics updates Go runtime metrics
func (m *SystemMonitor) updateRuntimeMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	m.metrics.Timestamp = time.Now()
	m.metrics.Uptime = time.Since(m.startTime)
	m.metrics.MemoryAlloc = memStats.Alloc / 1024 / 1024
	m.metrics.MemoryTotal = memStats.TotalAlloc / 1024 / 1024
	m.metrics.MemorySys = memStats.Sys / 1024 / 1024
	m.metrics.NumGoroutines = runtime.NumGoroutine()
	m.metrics.NumGC = memStats.NumGC
	
	// Call update callback if set
	if m.onMetricsUpdate != nil {
		m.onMetricsUpdate(m.metrics)
	}
}

// ============================================================================
// Update Methods (called by trading components)
// ============================================================================

// UpdatePositions updates position information
func (m *SystemMonitor) UpdatePositions(positions []PositionInfo) {
	m.mu.Lock()
	m.positions = positions
	m.metrics.ActivePositions = len(positions)
	m.mu.Unlock()
	
	if m.onPositionUpdate != nil {
		m.onPositionUpdate(positions)
	}
}

// AddTrade records a completed trade
func (m *SystemMonitor) AddTrade(trade TradeInfo) {
	m.mu.Lock()
	
	// Add to recent trades (keep last 100)
	m.recentTrades = append([]TradeInfo{trade}, m.recentTrades...)
	if len(m.recentTrades) > 100 {
		m.recentTrades = m.recentTrades[:100]
	}
	
	m.metrics.TotalTrades++
	m.metrics.TotalPnL += trade.PnL
	
	// Update win rate
	wins := 0
	for _, t := range m.recentTrades {
		if t.PnL > 0 {
			wins++
		}
	}
	if len(m.recentTrades) > 0 {
		m.metrics.WinRate = float64(wins) / float64(len(m.recentTrades)) * 100
	}
	
	m.mu.Unlock()
	
	if m.onTradeUpdate != nil {
		m.onTradeUpdate(trade)
	}
}

// UpdateWallet updates wallet information
func (m *SystemMonitor) UpdateWallet(wallet *WalletInfo) {
	m.mu.Lock()
	m.wallet = wallet
	m.metrics.WalletBalance = wallet.TotalBalance
	m.metrics.AvailableBalance = wallet.AvailableBalance
	m.metrics.MarginUsed = wallet.UsedMargin
	m.metrics.MarginRatio = wallet.MarginRatio
	m.metrics.DailyPnL = wallet.DailyPnL
	m.mu.Unlock()
	
	if m.onWalletUpdate != nil {
		m.onWalletUpdate(wallet)
	}
}

// UpdateTopMovers updates top mover information
func (m *SystemMonitor) UpdateTopMovers(movers []TopMoverInfo) {
	m.mu.Lock()
	m.topMovers = movers
	m.mu.Unlock()
}

// UpdateAPIMetrics updates API-related metrics
func (m *SystemMonitor) UpdateAPIMetrics(latency time.Duration, errors int, rateLimit float64) {
	m.mu.Lock()
	m.metrics.APILatency = latency
	m.metrics.APIErrors = errors
	m.metrics.APIRateLimit = rateLimit
	m.mu.Unlock()
}

// UpdateTradingStatus updates trading status
func (m *SystemMonitor) UpdateTradingStatus(isTrading, isFirstTrade bool, status string) {
	m.mu.Lock()
	m.metrics.IsTrading = isTrading
	m.metrics.IsFirstTrade = isFirstTrade
	m.metrics.SessionStatus = status
	m.mu.Unlock()
}

// SetLastError records the last error
func (m *SystemMonitor) SetLastError(err string) {
	m.mu.Lock()
	m.metrics.LastError = err
	m.mu.Unlock()
}

// ============================================================================
// Getter Methods
// ============================================================================

// GetMetrics returns current system metrics
func (m *SystemMonitor) GetMetrics() *SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy
	metrics := *m.metrics
	return &metrics
}

// GetPositions returns current positions
func (m *SystemMonitor) GetPositions() []PositionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	positions := make([]PositionInfo, len(m.positions))
	copy(positions, m.positions)
	return positions
}

// GetRecentTrades returns recent trades
func (m *SystemMonitor) GetRecentTrades(limit int) []TradeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if limit <= 0 || limit > len(m.recentTrades) {
		limit = len(m.recentTrades)
	}
	
	trades := make([]TradeInfo, limit)
	copy(trades, m.recentTrades[:limit])
	return trades
}

// GetWallet returns wallet information
func (m *SystemMonitor) GetWallet() *WalletInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.wallet == nil {
		return &WalletInfo{}
	}
	
	wallet := *m.wallet
	return &wallet
}

// GetTopMovers returns top movers
func (m *SystemMonitor) GetTopMovers() []TopMoverInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	movers := make([]TopMoverInfo, len(m.topMovers))
	copy(movers, m.topMovers)
	return movers
}

// ============================================================================
// Callbacks
// ============================================================================

// OnMetricsUpdate sets callback for metrics updates
func (m *SystemMonitor) OnMetricsUpdate(fn func(*SystemMetrics)) {
	m.onMetricsUpdate = fn
}

// OnPositionUpdate sets callback for position updates
func (m *SystemMonitor) OnPositionUpdate(fn func([]PositionInfo)) {
	m.onPositionUpdate = fn
}

// OnTradeUpdate sets callback for trade updates
func (m *SystemMonitor) OnTradeUpdate(fn func(TradeInfo)) {
	m.onTradeUpdate = fn
}

// OnWalletUpdate sets callback for wallet updates
func (m *SystemMonitor) OnWalletUpdate(fn func(*WalletInfo)) {
	m.onWalletUpdate = fn
}

// ============================================================================
// Summary Methods
// ============================================================================

// GetSummary returns a text summary of system state
func (m *SystemMonitor) GetSummary() string {
	metrics := m.GetMetrics()
	wallet := m.GetWallet()
	positions := m.GetPositions()
	
	status := "IDLE"
	if metrics.IsTrading {
		status = "TRADING"
		if metrics.IsFirstTrade {
			status = "FIRST TRADE"
		}
	}
	
	return fmt.Sprintf(`
=== GOBOT System Status ===
Status:     %s
Uptime:     %s
Session:    %s

=== Wallet ===
Balance:    $%.2f
Available:  $%.2f
Daily PnL:  $%.2f (%.2f%%)
Margin:     %.1f%%

=== Trading ===
Positions:  %d
Trades:     %d
Win Rate:   %.1f%%
Total PnL:  $%.2f

=== System ===
Memory:     %d MB
Goroutines: %d
API Latency: %dms
`,
		status,
		metrics.Uptime.Round(time.Second),
		metrics.SessionStatus,
		wallet.TotalBalance,
		wallet.AvailableBalance,
		wallet.DailyPnL,
		wallet.DailyPnLPercent,
		wallet.MarginRatio*100,
		len(positions),
		metrics.TotalTrades,
		metrics.WinRate,
		metrics.TotalPnL,
		metrics.MemoryAlloc,
		metrics.NumGoroutines,
		metrics.APILatency.Milliseconds(),
	)
}
