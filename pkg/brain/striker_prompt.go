package brain

// StrikerDecision represents the AI's trading decision
type StrikerDecision struct {
	Timestamp    string        `json:"timestamp"`
	TopTargets   []TargetAsset `json:"top_targets"`
	MarketRegime string        `json:"market_regime"`
}

// TargetAsset represents a single trading target
type TargetAsset struct {
	Symbol              string  `json:"symbol"`
	Action              string  `json:"action"`
	ConfidenceScore     float64 `json:"confidence_score"`
	ProbabilityReason   string  `json:"probability_reason"`
	EntryZone           float64 `json:"entry_zone"`
	TakeProfit          float64 `json:"take_profit"`
	StopLoss            float64 `json:"stop_loss"`
	AllocationMultiplier float64 `json:"allocation_multiplier"`
}

// MinimumConfidenceThreshold is the minimum confidence required to execute
const MinimumConfidenceThreshold = 85.0

// ExecuteWithThreshold determines if a target meets execution criteria
func (t *TargetAsset) ExecuteWithThreshold() bool {
	return t.ConfidenceScore >= MinimumConfidenceThreshold
}