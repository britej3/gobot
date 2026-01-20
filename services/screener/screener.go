package screener

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/britej3/gobot/domain/asset"
)

type Config struct {
	Interval time.Duration
	MaxPairs int
	SortBy   string
	Filter   AssetFilter
}

type AssetFilter struct {
	ContractType   string
	QuoteAsset     string
	MinVolume24h   float64
	MinPriceChange float64
	MaxPriceChange float64
	IncludeSymbols []string
	ExcludeSymbols []string
	Status         string
}

type ExchangeInfo struct {
	Symbol         string
	ContractType   string
	QuoteAsset     string
	Status         string
	Volume24h      float64
	PriceChangePct float64
	LastUpdated    time.Time
}

type ExchangeClient interface {
	GetExchangeInfo(ctx context.Context) ([]ExchangeInfo, error)
}

type Screener struct {
	cfg         Config
	client      ExchangeClient
	pairs       []ExchangeInfo
	activePairs []string
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
	ticker      *time.Ticker
}

type Option func(*Config)

func NewScreener(client ExchangeClient, opts ...Option) *Screener {
	cfg := Config{
		Interval: 5 * time.Minute,
		MaxPairs: 5,
		SortBy:   "volatility",
		Filter: AssetFilter{
			ContractType:   "PERPETUAL",
			QuoteAsset:     "USDT",
			MinVolume24h:   5_000_000,
			MinPriceChange: 5.0,
			Status:         "TRADING",
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return &Screener{
		cfg:    cfg,
		client: client,
		stopCh: make(chan struct{}),
		ticker: time.NewTicker(cfg.Interval),
	}
}

func WithAssetFilter(filter AssetFilter) Option {
	return func(c *Config) {
		c.Filter = filter
	}
}

func WithInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.Interval = interval
	}
}

func WithMaxPairs(max int) Option {
	return func(c *Config) {
		c.MaxPairs = max
	}
}

func WithSortBy(sortBy string) Option {
	return func(c *Config) {
		c.SortBy = sortBy
	}
}

func (s *Screener) Initialize(ctx context.Context) error {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	if err := s.refresh(ctx); err != nil {
		return err
	}

	go s.run(ctx)
	return nil
}

func (s *Screener) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-s.ticker.C:
			s.refresh(ctx)
		}
	}
}

func (s *Screener) refresh(ctx context.Context) error {
	pairs, err := s.client.GetExchangeInfo(ctx)
	if err != nil {
		return err
	}

	filtered := s.applyFilters(pairs)

	s.mu.Lock()
	s.pairs = filtered
	s.activePairs = s.selectTopPairs(filtered)
	s.mu.Unlock()

	return nil
}

func (s *Screener) applyFilters(pairs []ExchangeInfo) []ExchangeInfo {
	filtered := make([]ExchangeInfo, 0, len(pairs))

	for _, p := range pairs {
		if !s.matchFilter(p) {
			continue
		}
		filtered = append(filtered, p)
	}

	return filtered
}

func (s *Screener) matchFilter(p ExchangeInfo) bool {
	f := s.cfg.Filter

	if f.ContractType != "" && p.ContractType != f.ContractType {
		return false
	}

	if f.QuoteAsset != "" && p.QuoteAsset != f.QuoteAsset {
		return false
	}

	if f.Status != "" && p.Status != f.Status {
		return false
	}

	if p.Volume24h < f.MinVolume24h {
		return false
	}

	if f.MinPriceChange > 0 && p.PriceChangePct < f.MinPriceChange {
		return false
	}

	if f.MaxPriceChange > 0 && p.PriceChangePct > f.MaxPriceChange {
		return false
	}

	if len(f.IncludeSymbols) > 0 {
		found := false
		for _, sym := range f.IncludeSymbols {
			if p.Symbol == sym {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	for _, sym := range f.ExcludeSymbols {
		if p.Symbol == sym {
			return false
		}
	}

	return true
}

func (s *Screener) selectTopPairs(pairs []ExchangeInfo) []string {
	sort.Slice(pairs, func(i, j int) bool {
		if s.cfg.SortBy == "volume" {
			return pairs[i].Volume24h > pairs[j].Volume24h
		}
		return pairs[i].PriceChangePct > pairs[j].PriceChangePct
	})

	maxPairs := s.cfg.MaxPairs
	if maxPairs <= 0 {
		maxPairs = 5
	}

	if len(pairs) < maxPairs {
		maxPairs = len(pairs)
	}

	result := make([]string, maxPairs)
	for i := 0; i < maxPairs; i++ {
		result[i] = pairs[i].Symbol
	}

	return result
}

func (s *Screener) GetActivePairs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, len(s.activePairs))
	copy(result, s.activePairs)
	return result
}

func (s *Screener) GetPairsInfo() []ExchangeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ExchangeInfo, len(s.pairs))
	copy(result, s.pairs)
	return result
}

func (s *Screener) IsMonitoring(symbol string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.activePairs {
		if p == symbol {
			return true
		}
	}
	return false
}

func (s *Screener) GetScore(symbol string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.pairs {
		if p.Symbol == symbol {
			if s.cfg.SortBy == "volume" {
				return p.Volume24h
			}
			return p.PriceChangePct
		}
	}
	return 0
}

func (s *Screener) ToAssets() []asset.Asset {
	s.mu.RLock()
	defer s.mu.RUnlock()

	assets := make([]asset.Asset, 0, len(s.pairs))
	for _, p := range s.pairs {
		assets = append(assets, asset.Asset{
			Symbol:     p.Symbol,
			Volume24h:  p.Volume24h,
			Confidence: s.calculateConfidence(p),
			ScoredAt:   p.LastUpdated,
		})
	}
	return assets
}

func (s *Screener) calculateConfidence(p ExchangeInfo) float64 {
	score := 0.0

	if p.Volume24h >= 10_000_000 {
		score += 0.4
	} else if p.Volume24h >= 5_000_000 {
		score += 0.3
	} else {
		score += 0.2
	}

	if p.PriceChangePct >= 10 {
		score += 0.4
	} else if p.PriceChangePct >= 5 {
		score += 0.3
	} else {
		score += 0.2
	}

	if s.cfg.Filter.IncludeSymbols != nil {
		for _, sym := range s.cfg.Filter.IncludeSymbols {
			if p.Symbol == sym {
				score += 0.2
				break
			}
		}
	}

	return score
}

func (s *Screener) Stop() {
	s.mu.Lock()
	if s.running {
		s.running = false
		close(s.stopCh)
		s.ticker.Stop()
	}
	s.mu.Unlock()
}

func (s *Screener) Stats() ScreenerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	avgVolume := 0.0
	avgChange := 0.0
	if len(s.pairs) > 0 {
		for _, p := range s.pairs {
			avgVolume += p.Volume24h
			avgChange += p.PriceChangePct
		}
		avgVolume /= float64(len(s.pairs))
		avgChange /= float64(len(s.pairs))
	}

	return ScreenerStats{
		TotalPairs:  len(s.pairs),
		ActivePairs: len(s.activePairs),
		AvgVolume:   avgVolume,
		AvgChange:   avgChange,
		LastUpdated: time.Now(),
	}
}

type ScreenerStats struct {
	TotalPairs  int
	ActivePairs int
	AvgVolume   float64
	AvgChange   float64
	LastUpdated time.Time
}

func DefaultMemeCoinFilter() AssetFilter {
	return AssetFilter{
		ContractType:   "PERPETUAL",
		QuoteAsset:     "USDT",
		MinVolume24h:   5_000_000,
		MinPriceChange: 5.0,
		Status:         "TRADING",
		IncludeSymbols: []string{
			"1000PEPEUSDT",
			"WIFUSDT",
			"POPCATUSDT",
			"TURBOUSDT",
			"MOGUSDT",
			"FWOGUSDT",
			"MEWUSDT",
			"ACTUSDT",
			"LUNAUSDT",
			"NEIROUSDT",
		},
		ExcludeSymbols: []string{
			"BTCUSDT",
			"ETHUSDT",
			"BNBUSDT",
			"SOLUSDT",
			"XRPUSDT",
		},
	}
}

func HighVolatilityFilter() Config {
	return Config{
		MaxPairs: 3,
		SortBy:   "volatility",
		Filter: AssetFilter{
			ContractType:   "PERPETUAL",
			QuoteAsset:     "USDT",
			MinVolume24h:   10_000_000,
			MinPriceChange: 15.0,
			Status:         "TRADING",
		},
	}
}

func VolumeBasedFilter() Config {
	return Config{
		MaxPairs: 10,
		SortBy:   "volume",
		Filter: AssetFilter{
			ContractType: "PERPETUAL",
			QuoteAsset:   "USDT",
			MinVolume24h: 20_000_000,
			Status:       "TRADING",
		},
	}
}

type SymbolChecker struct {
	include map[string]bool
	exclude map[string]bool
}

func NewSymbolChecker(include, exclude []string) *SymbolChecker {
	inc := make(map[string]bool)
	for _, s := range include {
		inc[strings.ToUpper(s)] = true
	}

	exc := make(map[string]bool)
	for _, s := range exclude {
		exc[strings.ToUpper(s)] = true
	}

	return &SymbolChecker{
		include: inc,
		exclude: exc,
	}
}

func (c *SymbolChecker) IsAllowed(symbol string) bool {
	symbol = strings.ToUpper(symbol)

	if c.exclude[symbol] {
		return false
	}

	if len(c.include) > 0 && !c.include[symbol] {
		return false
	}

	return true
}
