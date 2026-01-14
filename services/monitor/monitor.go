package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/britebrt/cognee/domain/trade"
)

type Config struct {
	CheckInterval   time.Duration
	HealthThreshold float64
	AutoClose       bool
}

type Monitor struct {
	cfg       Config
	mu        sync.RWMutex
	running   bool
	positions map[string]*trackedPosition
	priceRepo PriceRepository
	executor  Executor
	stopCh    chan struct{}
}

type trackedPosition struct {
	position   *trade.Position
	lastCheck  time.Time
	health     float64
	reason     string
	checkCount int
}

type PriceRepository interface {
	Price(ctx context.Context, symbol string) (float64, error)
}

type Executor interface {
	ClosePosition(ctx context.Context, position *trade.Position, reason string) error
	GetPosition(ctx context.Context, symbol string) (*trade.Position, error)
}

func New(cfg Config, priceRepo PriceRepository, executor Executor) *Monitor {
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 30 * time.Second
	}
	if cfg.HealthThreshold <= 0 {
		cfg.HealthThreshold = 45
	}

	return &Monitor{
		cfg:       cfg,
		positions: make(map[string]*trackedPosition),
		priceRepo: priceRepo,
		executor:  executor,
		stopCh:    make(chan struct{}),
	}
}

func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	go m.run(ctx)
	return nil
}

func (m *Monitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.running = false
	close(m.stopCh)
	return nil
}

func (m *Monitor) run(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkAllPositions(ctx)
		}
	}
}

func (m *Monitor) checkAllPositions(ctx context.Context) {
	m.mu.RLock()
	positions := make([]*trackedPosition, 0, len(m.positions))
	for _, tp := range m.positions {
		positions = append(positions, tp)
	}
	m.mu.RUnlock()

	for _, tp := range positions {
		m.checkPosition(ctx, tp)
	}
}

func (m *Monitor) checkPosition(ctx context.Context, tp *trackedPosition) {
	currentPrice, err := m.priceRepo.Price(ctx, tp.position.Symbol)
	if err != nil {
		return
	}

	tp.position.CurrentPrice = currentPrice
	tp.position.UpdatePnL(currentPrice)

	health, reason := m.calculateHealth(tp.position)

	m.mu.Lock()
	tp.health = health
	tp.reason = reason
	tp.lastCheck = time.Now()
	tp.checkCount++
	m.mu.Unlock()

	if m.cfg.AutoClose && health < m.cfg.HealthThreshold {
		m.autoClosePosition(ctx, tp)
	}
}

func (m *Monitor) calculateHealth(pos *trade.Position) (float64, string) {
	if pos.PnLPercent > 5 {
		return 85, "Strong profit momentum"
	}
	if pos.PnLPercent > 2 {
		return 70, "Healthy profit"
	}
	if pos.PnLPercent > 0 {
		return 60, "Slight profit"
	}
	if pos.PnLPercent > -2 {
		return 50, "Small loss - holding"
	}
	if pos.PnLPercent > -5 {
		return 35, "Significant loss - monitor closely"
	}
	if pos.PnLPercent > -10 {
		return 20, "Large loss - consider exit"
	}
	return 10, "Critical loss - exit recommended"
}

func (m *Monitor) autoClosePosition(ctx context.Context, tp *trackedPosition) {
	if tp.checkCount < 3 {
		return
	}

	if err := m.executor.ClosePosition(ctx, tp.position, tp.reason); err != nil {
		return
	}

	m.mu.Lock()
	delete(m.positions, tp.position.Symbol)
	m.mu.Unlock()
}

func (m *Monitor) AddPosition(ctx context.Context, position *trade.Position) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.positions[position.Symbol]; exists {
		return nil
	}

	m.positions[position.Symbol] = &trackedPosition{
		position:  position,
		lastCheck: time.Now(),
		health:    50,
		reason:    "Initial check pending",
	}

	return nil
}

func (m *Monitor) RemovePosition(symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.positions, symbol)
}

func (m *Monitor) GetHealth(ctx context.Context, symbol string) (float64, string, error) {
	m.mu.RLock()
	tp, ok := m.positions[symbol]
	m.mu.RUnlock()

	if !ok {
		return 0, "", trade.ErrPositionNotFound
	}

	return tp.health, tp.reason, nil
}

func (m *Monitor) GetAllHealth(ctx context.Context) map[string]HealthInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]HealthInfo)
	for symbol, tp := range m.positions {
		result[symbol] = HealthInfo{
			Health:     tp.health,
			Reason:     tp.reason,
			LastCheck:  tp.lastCheck,
			PnL:        tp.position.PnL,
			PnLPercent: tp.position.PnLPercent,
		}
	}
	return result
}

func (m *Monitor) PositionsCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.positions)
}

type HealthInfo struct {
	Health     float64
	Reason     string
	LastCheck  time.Time
	PnL        float64
	PnLPercent float64
}
