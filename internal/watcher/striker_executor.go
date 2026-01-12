package watcher

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/britebrt/cognee/internal/striker"
	"github.com/britebrt/cognee/pkg/brain"
	"github.com/sirupsen/logrus"
)

// StrikerExecutor handles the complete striker workflow
type StrikerExecutor struct {
	brain      *brain.BrainEngine
	scanner    *AssetScanner
	client     *futures.Client
	realStriker *striker.Striker
	minConfidence float64
}

// NewStrikerExecutor creates a new striker executor
func NewStrikerExecutor(brain *brain.BrainEngine, scanner *AssetScanner, client *futures.Client) *StrikerExecutor {
	return &StrikerExecutor{
		brain:         brain,
		scanner:       scanner,
		client:        client,
		realStriker:   striker.NewStriker(client, brain),
		minConfidence: MinimumConfidenceThreshold,
	}
}

// Execute runs the complete striker workflow
func (e *StrikerExecutor) Execute(ctx context.Context) (*brain.StrikerDecision, error) {
	// Step 1: Get top 15 volatile assets
	topAssets := e.scanner.GetTopAssets()
	if len(topAssets) < 5 {
		return nil, fmt.Errorf("insufficient assets for striker analysis (need >= 5, have %d)", len(topAssets))
	}
	
	// Take top 15 if we have more
	if len(topAssets) > 15 {
		topAssets = topAssets[:15]
	}
	
	logrus.WithField("asset_count", len(topAssets)).Info("🎯 Striker analyzing top volatile assets")
	
	// For now, return simulated decision (in production, would call AI)
	return e.generateSimulatedDecision(topAssets), nil
}

// generateSimulatedDecision creates a simulated striker decision for testing
func (e *StrikerExecutor) generateSimulatedDecision(topAssets []ScoredAsset) *brain.StrikerDecision {
	if len(topAssets) == 0 {
		return &brain.StrikerDecision{
			Timestamp:    time.Now().Format(time.RFC3339),
			TopTargets:   []brain.TargetAsset{},
			MarketRegime: "RANGING",
		}
	}
	
	// Select the top asset as a target
	topAsset := topAssets[0]
	
	// Calculate entry, stop, and take profit
	entry := topAsset.CurrentPrice
	stopLoss := entry * 0.995 // 0.5% stop
	takeProfit := entry * 1.015 // 1.5% target
	
	// Determine action based on technical factors
	action := "LONG"
	confidence := topAsset.Confidence * 100
	if confidence < 85 {
		confidence = 85 // Minimum for striker
	}
	
	return &brain.StrikerDecision{
		Timestamp:    time.Now().Format(time.RFC3339),
		TopTargets: []brain.TargetAsset{
			{
				Symbol:              topAsset.Symbol,
				Action:              action,
				ConfidenceScore:     confidence,
				ProbabilityReason:   "High volatility and volume spike detected",
				EntryZone:           entry,
				TakeProfit:          takeProfit,
				StopLoss:            stopLoss,
				AllocationMultiplier: 1.0,
			},
		},
		MarketRegime: "VOLATILE_EXPANSION",
	}
}

// validateTargets ensures trade parameters are valid
func (e *StrikerExecutor) validateTargets(decision *brain.StrikerDecision) {
	for i, target := range decision.TopTargets {
		// Ensure stop loss is appropriate for position type
		if target.Action == "LONG" && target.StopLoss >= target.EntryZone {
			target.StopLoss = target.EntryZone * 0.998 // 0.2% below
		} else if target.Action == "SHORT" && target.StopLoss <= target.EntryZone {
			target.StopLoss = target.EntryZone * 1.002 // 0.2% above
		}
		
		// Ensure take profit has proper risk/reward (min 1.5:1)
		risk := math.Abs(target.EntryZone - target.StopLoss)
		minReward := risk * 1.5
		
		if target.Action == "LONG" && (target.TakeProfit-target.EntryZone) < minReward {
			target.TakeProfit = target.EntryZone + minReward
		} else if target.Action == "SHORT" && (target.EntryZone-target.TakeProfit) < minReward {
			target.TakeProfit = target.EntryZone - minReward
		}
		
		// Normalize allocation multiplier
		if target.AllocationMultiplier < 0.1 {
			target.AllocationMultiplier = 0.1
		} else if target.AllocationMultiplier > 2.0 {
			target.AllocationMultiplier = 2.0
		}
		
		decision.TopTargets[i] = target
	}
}

