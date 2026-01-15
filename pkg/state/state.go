package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TradingState struct {
	mu           sync.RWMutex
	filePath     string
	dirty        bool
	lastSave     time.Time
	saveInterval time.Duration

	Capital           float64
	TotalTrades       int
	Wins              int
	Losses            int
	TotalPnL          float64
	DailyPnL          float64
	WeeklyPnL         float64
	CurrentPositions  []Position
	TradeHistory      []Trade
	LastTradeTime     time.Time
	LastSignalTime    time.Time
	ConsecutiveLosses int
	APIErrorCount     int
	LastAPIErrorTime  time.Time
	IsHalted          bool
	HaltReason        string
}

type Position struct {
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	Size       float64   `json:"size"`
	EntryPrice float64   `json:"entry_price"`
	StopLoss   float64   `json:"stop_loss"`
	TakeProfit float64   `json:"take_profit"`
	OpenTime   time.Time `json:"open_time"`
}

type Trade struct {
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	Size       float64   `json:"size"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	PnL        float64   `json:"pnl"`
	PnLPercent float64   `json:"pnl_percent"`
	StopLoss   float64   `json:"stop_loss"`
	TakeProfit float64   `json:"take_profit"`
	Confidence float64   `json:"confidence"`
	Reasoning  string    `json:"reasoning"`
	EntryTime  time.Time `json:"entry_time"`
	ExitTime   time.Time `json:"exit_time"`
	Status     string    `json:"status"`
}

type StateConfig struct {
	StateDir     string
	StateFile    string
	SaveInterval time.Duration
	MaxHistory   int
}

func NewStateManager(cfg StateConfig) (*TradingState, error) {
	if cfg.StateDir == "" {
		cfg.StateDir = "/Users/britebrt/GOBOT/state"
	}
	if cfg.StateFile == "" {
		cfg.StateFile = "trading_state.json"
	}
	if cfg.SaveInterval == 0 {
		cfg.SaveInterval = 30 * time.Second
	}
	if cfg.MaxHistory == 0 {
		cfg.MaxHistory = 1000
	}

	state := &TradingState{
		filePath:     filepath.Join(cfg.StateDir, cfg.StateFile),
		saveInterval: cfg.SaveInterval,
		Capital:      100,
	}

	if err := os.MkdirAll(cfg.StateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	if err := state.Load(); err != nil {
		state.Save()
	}

	go state.autoSaveLoop()

	return state, nil
}

func (s *TradingState) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	return nil
}

func (s *TradingState) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	s.dirty = false
	s.lastSave = time.Now()

	return nil
}

func (s *TradingState) autoSaveLoop() {
	for range time.Tick(s.saveInterval) {
		s.mu.RLock()
		needsSave := s.dirty
		s.mu.RUnlock()

		if needsSave {
			if err := s.Save(); err != nil {
				fmt.Printf("Error saving state: %v\n", err)
			}
		}
	}
}

func (s *TradingState) MarkDirty() {
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()
}

func (s *TradingState) AddTrade(trade Trade) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TradeHistory = append(s.TradeHistory, trade)
	if len(s.TradeHistory) > 1000 {
		s.TradeHistory = s.TradeHistory[len(s.TradeHistory)-1000:]
	}

	s.TotalTrades++
	s.TotalPnL += trade.PnL
	s.DailyPnL += trade.PnL
	s.WeeklyPnL += trade.PnL

	if trade.PnL > 0 {
		s.Wins++
		s.ConsecutiveLosses = 0
	} else {
		s.Losses++
		s.ConsecutiveLosses++
	}

	s.LastTradeTime = trade.ExitTime
	s.dirty = true
}

func (s *TradingState) AddPosition(pos Position) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CurrentPositions = append(s.CurrentPositions, pos)
	s.dirty = true
}

func (s *TradingState) ClosePosition(symbol string, exitPrice float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, pos := range s.CurrentPositions {
		if pos.Symbol == symbol {
			s.CurrentPositions = append(s.CurrentPositions[:i], s.CurrentPositions[i+1:]...)
			s.dirty = true
			return
		}
	}
}

func (s *TradingState) UpdateCapital(pnl float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Capital += pnl
	s.TotalPnL += pnl
	s.DailyPnL += pnl
	s.WeeklyPnL += pnl
	s.dirty = true
}

func (s *TradingState) ResetDailyStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.DailyPnL = 0
	s.dirty = true
}

func (s *TradingState) ResetWeeklyStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.WeeklyPnL = 0
	s.DailyPnL = 0
	s.dirty = true
}

func (s *TradingState) RecordAPIError() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.APIErrorCount++
	s.LastAPIErrorTime = time.Now()
	s.dirty = true
}

func (s *TradingState) ResetAPIErrorCount() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.APIErrorCount = 0
	s.dirty = true
}

func (s *TradingState) Halt(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsHalted = true
	s.HaltReason = reason
	s.dirty = true
}

func (s *TradingState) Resume() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsHalted = false
	s.HaltReason = ""
	s.dirty = true
}

func (s *TradingState) GetStats() StateStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	winRate := 0.0
	if s.TotalTrades > 0 {
		winRate = float64(s.Wins) / float64(s.TotalTrades) * 100
	}

	return StateStats{
		Capital:           s.Capital,
		TotalTrades:       s.TotalTrades,
		Wins:              s.Wins,
		Losses:            s.Losses,
		WinRate:           winRate,
		TotalPnL:          s.TotalPnL,
		DailyPnL:          s.DailyPnL,
		WeeklyPnL:         s.WeeklyPnL,
		OpenPositions:     len(s.CurrentPositions),
		TradeHistory:      len(s.TradeHistory),
		LastTradeTime:     s.LastTradeTime,
		ConsecutiveLosses: s.ConsecutiveLosses,
		APIErrorCount:     s.APIErrorCount,
		IsHalted:          s.IsHalted,
		HaltReason:        s.HaltReason,
	}
}

type StateStats struct {
	Capital           float64
	TotalTrades       int
	Wins              int
	Losses            int
	WinRate           float64
	TotalPnL          float64
	DailyPnL          float64
	WeeklyPnL         float64
	OpenPositions     int
	TradeHistory      int
	LastTradeTime     time.Time
	ConsecutiveLosses int
	APIErrorCount     int
	IsHalted          bool
	HaltReason        string
}
