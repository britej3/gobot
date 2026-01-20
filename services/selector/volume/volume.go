package volume

import (
	"context"
	"time"

	"github.com/britej3/gobot/domain/asset"
	"github.com/britej3/gobot/domain/selector"
	"github.com/britej3/gobot/domain/trade"
)

type VolumeSelector struct {
	cfg selector.SelectorConfig
}

func (s *VolumeSelector) Type() selector.SelectorType {
	return selector.SelectorVolume
}

func (s *VolumeSelector) Name() string {
	return "volume_selector"
}

func (s *VolumeSelector) Configure(config selector.SelectorConfig) error {
	s.cfg = config
	return nil
}

func (s *VolumeSelector) Validate() error {
	return nil
}

func (s *VolumeSelector) Select(ctx context.Context, marketData map[string]*trade.MarketData) ([]asset.Asset, error) {
	var scoredAssets []selector.ScoredAsset

	for symbol, market := range marketData {
		if s.cfg.MinVolume > 0 && market.Volume24h < s.cfg.MinVolume {
			continue
		}

		if s.cfg.MaxVolume > 0 && market.Volume24h > s.cfg.MaxVolume {
			continue
		}

		score := s.calculateVolumeScore(market)
		breakdown := s.calculateBreakdown(market)

		scoredAssets = append(scoredAssets, selector.ScoredAsset{
			Symbol:     symbol,
			Score:      score,
			Breakdown:  breakdown,
			MarketData: market,
			SelectedAt: time.Now(),
		})
	}

	s.sortByScore(scoredAssets)

	var result []asset.Asset
	maxAssets := s.cfg.MaxAssets
	if maxAssets <= 0 {
		maxAssets = 15
	}

	for i, sa := range scoredAssets {
		if i >= maxAssets {
			break
		}

		result = append(result, asset.Asset{
			Symbol:       sa.Symbol,
			CurrentPrice: sa.MarketData.CurrentPrice,
			Volume24h:    sa.MarketData.Volume24h,
			Volatility:   sa.MarketData.Volatility,
			RSI:          sa.MarketData.RSI,
			EMAFast:      sa.MarketData.EMAFast,
			EMASlow:      sa.MarketData.EMASlow,
			Confidence:   sa.Score,
			ScoredAt:     sa.SelectedAt,
		})
	}

	return result, nil
}

func (s *VolumeSelector) GetScore(ctx context.Context, market *trade.MarketData) (float64, error) {
	return s.calculateVolumeScore(market), nil
}

func (s *VolumeSelector) GetRanking() []selector.ScoredAsset {
	return nil
}

func (s *VolumeSelector) calculateVolumeScore(market *trade.MarketData) float64 {
	score := 0.0

	volumeWeight := s.cfg.Weightings["volume"]
	if volumeWeight == 0 {
		volumeWeight = 0.4
	}

	if market.Volume24h >= s.cfg.MinVolume {
		score += volumeWeight * 100
	} else if s.cfg.MinVolume > 0 {
		score += volumeWeight * (market.Volume24h / s.cfg.MinVolume) * 100
	}

	volatilityWeight := s.cfg.Weightings["volatility"]
	if volatilityWeight == 0 {
		volatilityWeight = 0.3
	}

	if market.Volatility >= 0.5 && market.Volatility <= 3.0 {
		score += volatilityWeight * 100
	} else if market.Volatility > 0 {
		score += volatilityWeight * 50
	}

	rsiWeight := s.cfg.Weightings["rsi"]
	if rsiWeight == 0 {
		rsiWeight = 0.3
	}

	if market.RSI >= 40 && market.RSI <= 70 {
		score += rsiWeight * 100
	} else if market.RSI > 30 && market.RSI < 80 {
		score += rsiWeight * 50
	}

	return score
}

func (s *VolumeSelector) calculateBreakdown(market *trade.MarketData) map[string]float64 {
	return map[string]float64{
		"volume":     market.Volume24h,
		"volatility": market.Volatility,
		"rsi":        market.RSI,
	}
}

func (s *VolumeSelector) sortByScore(assets []selector.ScoredAsset) {
	for i := 0; i < len(assets)-1; i++ {
		for j := i + 1; j < len(assets); j++ {
			if assets[j].Score > assets[i].Score {
				assets[i], assets[j] = assets[j], assets[i]
			}
		}
	}
}
