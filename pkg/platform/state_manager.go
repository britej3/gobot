// Package platform provides state persistence for crash recovery
package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/britej3/gobot/pkg/types"
	"github.com/sirupsen/logrus"
)

// PositionState is a type alias for types.PositionState to maintain compatibility
type PositionState = types.PositionState

// PlatformState represents complete platform state for persistence
type PlatformState struct {
	SessionID        string          `json:"session_id"`
	Timestamp        time.Time       `json:"timestamp"`
	OpenPositions    []PositionState `json:"open_positions"`
	TotalBalance     float64         `json:"total_balance"`
	AvailableBalance float64         `json:"available_balance"`
	LastTradeID      string          `json:"last_trade_id"`
	MarketRegime     string          `json:"market_regime"`
}

// StateManager handles platform state persistence
type StateManager struct {
	mu       sync.RWMutex
	state    PlatformState
	filePath string
}

// NewStateManager creates a new state manager
func NewStateManager(sessionID string) *StateManager {
	return &StateManager{
		state: PlatformState{
			SessionID:     sessionID,
			Timestamp:     time.Now(),
			OpenPositions: make([]PositionState, 0),
		},
		filePath: fmt.Sprintf("state/session_%s.json", sessionID),
	}
}

// Save persists the current state to disk
func (sm *StateManager) Save() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Create state directory if not exists
	if err := os.MkdirAll("state", 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal state to JSON
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file then rename for atomicity
	tmpPath := sm.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, sm.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	logrus.WithField("file", sm.filePath).Debug("State saved successfully")
	return nil
}

// Load restores state from disk
func (sm *StateManager) Load() (*PlatformState, error) {
	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Debug("No state file found, starting fresh")
			return nil, nil // Fresh start
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state PlatformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"file":           sm.filePath,
		"positions":      len(state.OpenPositions),
		"balance":        state.TotalBalance,
	}).Info("State loaded successfully")

	return &state, nil
}

// AddPosition adds an open position to state
func (sm *StateManager) AddPosition(pos PositionState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.OpenPositions = append(sm.state.OpenPositions, pos)
	sm.state.Timestamp = time.Now()

	// Auto-save on position open
	if err := sm.Save(); err != nil {
		logrus.WithError(err).Error("Failed to auto-save state")
	}
}

// RemovePosition removes a closed position
func (sm *StateManager) RemovePosition(symbol string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	filtered := make([]PositionState, 0)
	for _, pos := range sm.state.OpenPositions {
		if pos.Symbol != symbol {
			filtered = append(filtered, pos)
		}
	}

	sm.state.OpenPositions = filtered
	sm.state.Timestamp = time.Now()

	// Auto-save on position close
	if err := sm.Save(); err != nil {
		logrus.WithError(err).Error("Failed to auto-save state")
	}
}

// GetPosition retrieves a position by symbol
func (sm *StateManager) GetPosition(symbol string) *PositionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, pos := range sm.state.OpenPositions {
		if pos.Symbol == symbol {
			return &pos
		}
	}
	return nil
}

// UpdateBalance updates balance in state
func (sm *StateManager) UpdateBalance(total, available float64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.TotalBalance = total
	sm.state.AvailableBalance = available
	sm.state.Timestamp = time.Now()
}

// StartAutoSave starts background auto-save every 30 seconds
func (sm *StateManager) StartAutoSave() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			if err := sm.Save(); err != nil {
				logrus.WithError(err).Error("Auto-save failed")
			}
		}
	}()
}