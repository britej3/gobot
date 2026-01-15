package binance

import (
	"context"
	"math"
	"sync"

	"github.com/britebrt/cognee/domain/platform"
	"github.com/britebrt/cognee/domain/trade"
	"github.com/britebrt/cognee/pkg/stealth"
	"golang.org/x/time/rate"
)

type MarketDataProvider struct {
	client     *Client
	rateClient *RateLimitedClient
	limiter    *rate.Limiter
	subs       map[string][]func(*trade.MarketData)
	mu         sync.RWMutex
}

func NewMarketDataProvider(client *Client) *MarketDataProvider {
	return &MarketDataProvider{
		client:  client,
		limiter: rate.NewLimiter(rate.Limit(10), 20),
		subs:    make(map[string][]func(*trade.MarketData)),
	}
}

func NewMarketDataProviderWithStealth(client *RateLimitedClient, stealth *stealth.StealthClient) *MarketDataProvider {
	return &MarketDataProvider{
		rateClient: client,
		limiter:    rate.NewLimiter(rate.Limit(10), 20),
		subs:       make(map[string][]func(*trade.MarketData)),
	}
}

func (m *MarketDataProvider) GetMarketData(ctx context.Context, symbol string) (*trade.MarketData, error) {
	if err := m.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	var price float64
	var klines []trade.Kline
	var err error

	if m.rateClient != nil {
		price, err = m.rateClient.Price(ctx, symbol)
	} else if m.client != nil {
		price, err = m.client.Price(ctx, symbol)
	}
	if err != nil {
		return nil, err
	}

	if m.rateClient != nil {
		klines, err = m.rateClient.Kline(ctx, symbol, "15m", 100)
	} else if m.client != nil {
		klines, err = m.client.Kline(ctx, symbol, "15m", 100)
	}
	if err != nil {
		return nil, err
	}

	market := buildMarketData(symbol, price, klines)

	go m.notifySubscribers(market)

	return market, nil
}

func (m *MarketDataProvider) GetAllMarketData(ctx context.Context) (map[string]*trade.MarketData, error) {
	var symbols []string
	var err error

	if m.rateClient != nil {
		symbols, err = m.rateClient.Symbols(ctx)
	} else if m.client != nil {
		symbols, err = m.client.Symbols(ctx)
	}
	if err != nil {
		return nil, err
	}

	result := make(map[string]*trade.MarketData)

	for _, symbol := range symbols {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		var price float64
		if m.rateClient != nil {
			price, err = m.rateClient.Price(ctx, symbol)
		} else if m.client != nil {
			price, err = m.client.Price(ctx, symbol)
		}
		if err != nil {
			continue
		}

		klines, err := m.client.Kline(ctx, symbol, "15m", 100)
		if err != nil {
			continue
		}

		market := buildMarketData(symbol, price, klines)
		result[symbol] = market

		m.notifySubscribers(market)
	}

	return result, nil
}

func (m *MarketDataProvider) Subscribe(symbol string, handler func(*trade.MarketData)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subs[symbol] = append(m.subs[symbol], handler)
	return nil
}

func (m *MarketDataProvider) Unsubscribe(symbol string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subs, symbol)
	return nil
}

func (m *MarketDataProvider) notifySubscribers(market *trade.MarketData) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	handlers, ok := m.subs[market.Symbol]
	if !ok {
		return
	}

	for _, handler := range handlers {
		go handler(market)
	}
}

func buildMarketData(symbol string, currentPrice float64, klines []trade.Kline) *trade.MarketData {
	if len(klines) == 0 {
		return &trade.MarketData{
			Symbol:       symbol,
			CurrentPrice: currentPrice,
		}
	}

	var high24h, low24h, volume24h float64
	for _, k := range klines {
		if k.High > high24h {
			high24h = k.High
		}
		if low24h == 0 || k.Low < low24h {
			low24h = k.Low
		}
		volume24h += k.Volume
	}

	var emaFast, emaSlow float64
	if len(klines) >= 9 {
		emaFast = calculateEMA(klines, 9)
	}
	if len(klines) >= 21 {
		emaSlow = calculateEMA(klines, 21)
	}

	rsi := calculateRSI(klines)

	volatility := 0.0
	if len(klines) > 1 {
		for i := len(klines) - 20; i < len(klines); i++ {
			if i >= 0 {
				volatility += (klines[i].High - klines[i].Low) / klines[i].Close * 100
			}
		}
		volatility /= float64(min(20, len(klines)))
	}

	return &trade.MarketData{
		Symbol:       symbol,
		CurrentPrice: currentPrice,
		High24h:      high24h,
		Low24h:       low24h,
		Volume24h:    volume24h,
		Volatility:   math.Round(volatility*100) / 100,
		RSI:          math.Round(rsi*100) / 100,
		EMAFast:      math.Round(emaFast*10000000) / 10000000,
		EMASlow:      math.Round(emaSlow*10000000) / 10000000,
	}
}

func calculateEMA(klines []trade.Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}

	var ema float64
	multiplier := 2.0 / float64(period+1)

	sum := 0.0
	for i := len(klines) - period; i < len(klines); i++ {
		sum += klines[i].Close
	}
	ema = sum / float64(period)

	for i := len(klines) - period - 1; i >= 0; i-- {
		ema = (klines[i].Close-ema)*multiplier + ema
	}

	return ema
}

func calculateRSI(klines []trade.Kline) float64 {
	if len(klines) < 15 {
		return 50
	}

	var gains, losses float64
	for i := len(klines) - 15; i < len(klines)-1; i++ {
		change := klines[i+1].Close - klines[i].Close
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / 14
	avgLoss := losses / 14

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ platform.MarketDataProvider = (*MarketDataProvider)(nil)
