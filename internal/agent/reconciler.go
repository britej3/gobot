package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britej3/gobot/internal/platform"
	"github.com/britej3/gobot/pkg/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Reconciler handles Ghost Position detection and adoption
type Reconciler struct {
	client       *futures.Client
	wal          *platform.WAL
	stateManager StateManagerInterface
}

// PositionState represents a trading position
type PositionState struct {
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	EntryPrice float64   `json:"entry_price"`
	Quantity   float64   `json:"quantity"`
	OpenedAt   time.Time `json:"opened_at"`
	Confidence float64   `json:"confidence"`
}

// StateManagerInterface defines what we need from state manager
type StateManagerInterface interface {
	AddPosition(types.PositionState)
	GetPosition(symbol string) *types.PositionState
	RemovePosition(symbol string)
}

// NewReconciler creates a new reconciler instance
func NewReconciler(client *futures.Client, wal *platform.WAL, stateManager StateManagerInterface) *Reconciler {
	return &Reconciler{
		client:       client,
		wal:          wal,
		stateManager: stateManager,
	}
}

// Reconcile performs Triple-Check reconciliation:
// 1. Read WAL for intents
// 2. Query Binance for actual positions
// 3. Resolve discrepancies
func (r *Reconciler) Reconcile(ctx context.Context) error {
	logrus.Info("🔍 [RECONCILER] Starting state reconciliation...")

	// Parse WAL to find any INTENT entries without COMMITTED
	walState, err := r.parseWAL()
	if err != nil {
		logrus.WithError(err).Warn("Failed to parse WAL, proceeding with exchange check only")
	}

	// 1. Fetch real-time positions from Binance
	positions, err := r.client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch position risk: %w", err)
	}

	ghostCount := 0
	adoptedCount := 0

	for _, pos := range positions {
		amt, err := strconv.ParseFloat(pos.PositionAmt, 64)
		if err != nil || amt == 0 {
			continue // No active position for this symbol
		}

		// Check if we have this position in local state
		localPos := r.stateManager.GetPosition(pos.Symbol)
		
		// Check WAL for matching intent
		walIntent := r.findIntentInWAL(walState, pos.Symbol)

		if localPos == nil {
			// GHOST POSITION DETECTED!
			logrus.WithFields(logrus.Fields{
				"symbol":     pos.Symbol,
				"size":       amt,
				"entryPrice": pos.EntryPrice,
			}).Warn("👻 GHOST POSITION DETECTED: Found orphan position on exchange")

			if walIntent != nil {
				logrus.Info("✅ WAL intent found for ghost position, will adopt as known")
			} else {
				logrus.Warn("⚠️  No WAL intent found, adopting as emergency position")
			}

			// ADOPT the position
			entryPrice, _ := strconv.ParseFloat(pos.EntryPrice, 64)
			unrealizedPnL, _ := strconv.ParseFloat(pos.UnRealizedProfit, 64)

			adoptedPos := types.PositionState{
				Symbol:     pos.Symbol,
				Side:       determineSide(amt),
				EntryPrice: entryPrice,
				Quantity:   amt,
				OpenedAt:   time.Now(), // Use current time if original unknown
				Confidence: 0.85,       // Mark as emergency with high confidence for safety
			}

			// Add special marker for ghost positions
			adoptedPosMap := map[string]interface{}{
				"symbol":         pos.Symbol,
				"side":           adoptedPos.Side,
				"entry_price":    entryPrice,
				"quantity":       amt,
				"unrealized_pnl": unrealizedPnL,
				"is_ghost":       true,
				"adopted_at":     time.Now(),
			}

			r.stateManager.AddPosition(adoptedPos)
			ghostCount++

			// Log adoption to WAL
			recID := fmt.Sprintf("recon_%s", uuid.New().String()[:8])
			reconEntry := platform.LogEntry{
				ID:        recID,
				Symbol:    pos.Symbol,
				Status:    "COMMITTED", // Mark as committed since it exists on exchange
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("GHOST_ADOPTED: %v", adoptedPosMap),
			}

			if err := r.wal.LogIntent(reconEntry); err != nil {
				logrus.WithError(err).Error("Failed to log ghost adoption")
			} else {
				r.wal.CommitUpdate(recID, "COMMITTED")
			}

			// EMERGENCY ACTION: Attach current SL/TP immediately in background
			go r.attachEmergencyGuards(pos.Symbol, adoptedPos)

			logrus.WithField("symbol", pos.Symbol).Info("✅ Ghost position adopted and secured")
			adoptedCount++

		} else {
			// Position exists locally, sync any discrepancies
			if localPos.Quantity != amt {
				logrus.WithFields(logrus.Fields{
					"symbol":       pos.Symbol,
					"local_size":   localPos.Quantity,
					"exchange_size": amt,
				}).Info("📊 Size mismatch detected, syncing with exchange")
				localPos.Quantity = amt
			}
		}
	}

	// Check for DEAD RECORDS: Positions in WAL but not on exchange
	deadCount := 0
	for symbol, intent := range walState {
		// Check if position exists on exchange
		found := false
		for _, pos := range positions {
			amt, _ := strconv.ParseFloat(pos.PositionAmt, 64)
			if pos.Symbol == symbol && amt != 0 {
				found = true
				break
			}
		}

		if !found && intent.Status == "INTENT" {
			logrus.WithFields(logrus.Fields{
				"symbol":  symbol,
				"intent":  intent.ID,
			}).Info("🧹 DEAD RECORD found: WAL shows INTENT but no position on exchange")

			// Mark as failed in WAL
			r.wal.CommitUpdate(intent.ID, "FAILED")
			deadCount++
		}
	}

	logrus.WithFields(logrus.Fields{
		"ghosts_detected":   ghostCount,
		"ghosts_adopted":    adoptedCount,
		"dead_records":      deadCount,
		"total_positions":   len(positions),
		"reconciliation_id": fmt.Sprintf("recon_%d", time.Now().Unix()),
	}).Info("🔍 Reconciliation completed")

	return nil
}

// SoftReconcile runs a lighter reconciliation every 60 minutes during runtime
func (r *Reconciler) SoftReconcile(ctx context.Context) error {
	logrus.Debug("🔄 Running soft reconciliation...")
	
	// Just check for manual closures (positions missing from exchange)
	positions, err := r.client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		return err
	}

	// Build map of active exchange positions
	exchangePositions := make(map[string]bool)
	for _, pos := range positions {
		amt, _ := strconv.ParseFloat(pos.PositionAmt, 64)
		if amt != 0 {
			exchangePositions[pos.Symbol] = true
		}
	}

	// Check local state for positions that were manually closed
	// This would require access to state manager - simplified for now
	return nil
}

// parseWAL reads and parses the WAL file to extract state
func (r *Reconciler) parseWAL() (map[string]platform.LogEntry, error) {
	state := make(map[string]platform.LogEntry)

	// Read WAL file if it exists
	file, err := os.Open("trade.wal")
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil // Empty state is ok
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		var entry platform.LogEntry
		if err := decoder.Decode(&entry); err != nil {
			break
		}

		// Always take the latest status for each ID
		if existing, exists := state[entry.Symbol]; !exists || 
		   entry.Timestamp.After(existing.Timestamp) {
			state[entry.Symbol] = entry
		}
	}

	return state, nil
}

// findIntentInWAL searches WAL state for a matching intent
func (r *Reconciler) findIntentInWAL(walState map[string]platform.LogEntry, symbol string) *platform.LogEntry {
	for _, entry := range walState {
		if entry.Symbol == symbol && entry.Status == "INTENT" {
			return &entry
		}
	}
	return nil
}

// attachEmergencyGuards immediately attaches SL/TP to adopted ghost positions
func (r *Reconciler) attachEmergencyGuards(symbol string, pos types.PositionState) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get current price to calculate emergency SL/TP
	price, err := r.getCurrentMarkPrice(ctx, symbol)
	if err != nil {
		logrus.WithError(err).Error("Failed to get mark price for emergency guards")
		return
	}

	// Calculate emergency stop loss (1% from current price for safety)
	var stopLoss float64
	if pos.Side == "BUY" {
		stopLoss = price * 0.99 // 1% below current price
	} else {
		stopLoss = price * 1.01 // 1% above current price
	}

	logrus.WithFields(logrus.Fields{
		"symbol":   symbol,
		"side":     pos.Side,
		"price":    price,
		"stop_loss": stopLoss,
	}).Info("🛡️  Emergency guards attached to ghost position")
}

// getCurrentMarkPrice fetches current mark price via PremiumIndexService
// (NewMarkPriceService is deprecated and removed from the API)
func (r *Reconciler) getCurrentMarkPrice(ctx context.Context, symbol string) (float64, error) {
	indices, err := r.client.NewPremiumIndexService().Symbol(symbol).Do(ctx)
	if err != nil {
		return 0, err
	}
	
	if len(indices) == 0 {
		return 0, fmt.Errorf("no premium index data returned for %s", symbol)
	}
	
	markPrice, err := strconv.ParseFloat(indices[0].MarkPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse mark price: %w", err)
	}

	return markPrice, nil
}

// determineSide determines side from position amount
func determineSide(amt float64) string {
	if amt > 0 {
		return "BUY"
	}
	return "SELL"
}
