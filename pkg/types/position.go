// Package types provides common data types used across the application
package types

import (
	"time"
)

// PositionState represents an open position for persistence
// This type is used to avoid circular dependencies between internal/agent and pkg/platform
type PositionState struct {
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	EntryPrice float64   `json:"entry_price"`
	Quantity   float64   `json:"quantity"`
	StopLoss   float64   `json:"stop_loss"`
	TakeProfit float64   `json:"take_profit"`
	OpenedAt   time.Time `json:"opened_at"`
	Confidence float64   `json:"confidence"`
}
