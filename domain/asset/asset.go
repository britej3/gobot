package asset

import (
	"context"
	"time"
)

type Asset struct {
	Symbol       string
	CurrentPrice float64
	Volume24h    float64
	Volatility   float64
	RSI          float64
	EMAFast      float64
	EMASlow      float64
	Confidence   float64
	ScoredAt     time.Time
}

type Criteria struct {
	MinVolume     float64
	MaxVolume     float64
	MinVolatility float64
	MaxVolatility float64
	MinConfidence float64
}

func (c Criteria) Validate() error {
	if c.MinVolume < 0 {
		return ErrInvalidMinVolume
	}
	if c.MinVolatility < 0 {
		return ErrInvalidMinVolatility
	}
	if c.MinConfidence < 0 || c.MinConfidence > 1 {
		return ErrInvalidConfidence
	}
	return nil
}

func (a *Asset) Score(c Criteria) float64 {
	if a == nil {
		return 0
	}

	score := 0.0

	if a.Volatility >= c.MinVolatility && a.Volatility <= c.MaxVolatility {
		score += 30.0
	}

	if a.Volume24h >= c.MinVolume {
		score += 30.0
	}

	if a.Confidence >= c.MinConfidence {
		score += a.Confidence * 40
	}

	return score
}

func (a *Asset) IsQualified(c Criteria) bool {
	return a.Volume24h >= c.MinVolume &&
		a.Volatility >= c.MinVolatility &&
		a.Volatility <= c.MaxVolatility &&
		a.Confidence >= c.MinConfidence
}

type ScoredAsset struct {
	Asset
	Score float64
}

type ScoredAssets []ScoredAsset

func (s ScoredAssets) Len() int           { return len(s) }
func (s ScoredAssets) Less(i, j int) bool { return s[i].Score > s[j].Score }
func (s ScoredAssets) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s ScoredAssets) Top(n int) []ScoredAsset {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

type Scorer interface {
	Score(ctx context.Context, asset Asset) (float64, error)
}

type DefaultScorer struct {
	Criteria Criteria
}

func (s *DefaultScorer) Score(ctx context.Context, a Asset) (float64, error) {
	return a.Score(s.Criteria), nil
}

type AssetError struct {
	Symbol string
	Err    error
}

func (e *AssetError) Unwrap() error {
	return e.Err
}

func (e *AssetError) Error() string {
	return "asset error for " + e.Symbol + ": " + e.Err.Error()
}

var (
	ErrInvalidMinVolume     = &AssetError{Symbol: "Criteria", Err: ErrMinVolumeNegative}
	ErrInvalidMinVolatility = &AssetError{Symbol: "Criteria", Err: ErrVolatilityNegative}
	ErrInvalidConfidence    = &AssetError{Symbol: "Criteria", Err: ErrConfidenceOutOfRange}
	ErrMinVolumeNegative    = &ConfigError{Field: "MinVolume", Message: "must be non-negative"}
	ErrVolatilityNegative   = &ConfigError{Field: "MinVolatility", Message: "must be non-negative"}
	ErrConfidenceOutOfRange = &ConfigError{Field: "MinConfidence", Message: "must be between 0 and 1"}
)

type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}
