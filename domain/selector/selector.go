package selector

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/britej3/gobot/domain/asset"
	"github.com/britej3/gobot/domain/trade"
)

type SelectorType string

const (
	SelectorVolume     SelectorType = "volume"
	SelectorVolatility SelectorType = "volatility"
	SelectorRSI        SelectorType = "rsi"
	SelectorTrend      SelectorType = "trend"
	SelectorAI         SelectorType = "ai"
	SelectorComposite  SelectorType = "composite"
	SelectorCustom     SelectorType = "custom"
)

type Selector interface {
	Type() SelectorType
	Name() string
	Configure(config SelectorConfig) error
	Validate() error
	Select(ctx context.Context, marketData map[string]*trade.MarketData) ([]asset.Asset, error)
	GetScore(ctx context.Context, market *trade.MarketData) (float64, error)
	GetRanking() []ScoredAsset
}

type SelectorConfig struct {
	Type          SelectorType       `json:"type"`
	Name          string             `json:"name"`
	Enabled       bool               `json:"enabled"`
	MinVolume     float64            `json:"min_volume"`
	MaxVolume     float64            `json:"max_volume"`
	MinVolatility float64            `json:"min_volatility"`
	MaxVolatility float64            `json:"max_volatility"`
	MinRSI        floatFilter        `json:"min_rsi"`
	MaxRSI        floatFilter        `json:"max_rsi"`
	MinConfidence float64            `json:"min_confidence"`
	MaxAssets     int                `json:"max_assets"`
	Timeframes    []string           `json:"timeframes"`
	IncludePairs  []string           `json:"include_pairs"`
	ExcludePairs  []string           `json:"exclude_pairs"`
	Weightings    map[string]float64 `json:"weightings"`
	AIEndpoint    string             `json:"ai_endpoint"`
}

type floatFilter struct {
	Enabled bool    `json:"enabled"`
	Value   float64 `json:"value"`
}

type ScoredAsset struct {
	Symbol     string
	Score      float64
	Breakdown  map[string]float64
	MarketData *trade.MarketData
	SelectedAt time.Time
}

type SelectorResult struct {
	Assets       []ScoredAsset
	TotalScanned int
	TotalPassed  int
	Duration     time.Duration
}

type SelectorRegistry interface {
	Register(name string, factory SelectorFactory) error
	Get(name string) (Selector, bool)
	List() []SelectorType
	Create(cfg SelectorConfig) (Selector, error)
}

type SelectorFactory func() Selector

type CompositeSelector struct {
	selectors []Selector
	weights   map[SelectorType]float64
}

func NewCompositeSelector() *CompositeSelector {
	return &CompositeSelector{
		selectors: make([]Selector, 0),
		weights:   make(map[SelectorType]float64),
	}
}

func (s *CompositeSelector) Add(sel Selector, weight float64) {
	s.selectors = append(s.selectors, sel)
	s.weights[sel.Type()] = weight
}

func (s *CompositeSelector) Type() SelectorType {
	return SelectorComposite
}

func (s *CompositeSelector) Name() string {
	return "composite_selector"
}

func (s *CompositeSelector) Configure(config SelectorConfig) error {
	return nil
}

func (s *CompositeSelector) Validate() error {
	if len(s.selectors) == 0 {
		return ErrNoSelectors
	}
	return nil
}

func (s *CompositeSelector) Select(ctx context.Context, marketData map[string]*trade.MarketData) ([]asset.Asset, error) {
	scores := make(map[string]ScoredAsset)

	for symbol, market := range marketData {
		var totalScore float64
		breakdown := make(map[string]float64)

		for _, sel := range s.selectors {
			selScore, err := sel.GetScore(ctx, market)
			if err != nil {
				continue
			}

			weight := s.weights[sel.Type()]
			totalScore += selScore * weight
			breakdown[string(sel.Type())] = selScore
		}

		scores[symbol] = ScoredAsset{
			Symbol:     symbol,
			Score:      totalScore,
			Breakdown:  breakdown,
			MarketData: market,
			SelectedAt: time.Now(),
		}
	}

	ranked := make([]asset.Asset, 0, len(scores))
	for _, scored := range scores {
		ranked = append(ranked, asset.Asset{
			Symbol:       scored.Symbol,
			CurrentPrice: scored.MarketData.CurrentPrice,
			Volume24h:    scored.MarketData.Volume24h,
			Volatility:   scored.MarketData.Volatility,
			RSI:          scored.MarketData.RSI,
			Confidence:   scored.Score,
			ScoredAt:     scored.SelectedAt,
		})
	}

	return ranked, nil
}

func (s *CompositeSelector) GetScore(ctx context.Context, market *trade.MarketData) (float64, error) {
	return 0, nil
}

func (s *CompositeSelector) GetRanking() []ScoredAsset {
	return nil
}

type AISelector struct {
	endpoint string
	client   *http.Client
}

type SelectorEngine struct {
	registry map[SelectorType]SelectorFactory
	mu       sync.RWMutex
}

func NewSelectorEngine() *SelectorEngine {
	return &SelectorEngine{
		registry: make(map[SelectorType]SelectorFactory),
	}
}

func (e *SelectorEngine) Register(t SelectorType, factory SelectorFactory) error {
	e.registry[t] = factory
	return nil
}

func (e *SelectorEngine) Get(t SelectorType) (Selector, bool) {
	factory, ok := e.registry[t]
	if !ok {
		return nil, false
	}
	return factory(), true
}

func (e *SelectorEngine) Create(cfg SelectorConfig) (Selector, error) {
	factory, ok := e.registry[cfg.Type]
	if !ok {
		return nil, ErrUnknownSelector
	}

	sel := factory()
	if err := sel.Configure(cfg); err != nil {
		return nil, err
	}

	return sel, nil
}

func (e *SelectorEngine) List() []SelectorType {
	types := make([]SelectorType, 0, len(e.registry))
	for t := range e.registry {
		types = append(types, t)
	}
	return types
}

var (
	ErrUnknownSelector = &SelectorError{Message: "unknown selector type"}
	ErrNoSelectors     = &SelectorError{Message: "no selectors configured"}
)

type SelectorError struct {
	Message string
}

func (e *SelectorError) Error() string {
	return e.Message
}
