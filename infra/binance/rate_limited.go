package binance

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/britebrt/cognee/domain/trade"
	"github.com/britebrt/cognee/pkg/retry"
	"golang.org/x/time/rate"
)

type RateLimitedClient struct {
	client  *Client
	limiter *rate.Limiter
	mu      sync.RWMutex
}

func NewRateLimitedClient(client *Client, rps float64, burst int) *RateLimitedClient {
	return &RateLimitedClient{
		client:  client,
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

func (c *RateLimitedClient) Price(ctx context.Context, symbol string) (float64, error) {
	var price float64
	_, err := retry.Do(ctx, func() (struct{}, error) {
		if waitErr := c.limiter.Wait(ctx); waitErr != nil {
			return struct{}{}, waitErr
		}
		var retryErr error
		price, retryErr = c.client.Price(ctx, symbol)
		return struct{}{}, retryErr
	}, retry.WithPolicy(retry.DefaultPolicy))
	return price, err
}

func (c *RateLimitedClient) Kline(ctx context.Context, symbol, interval string, limit int) ([]trade.Kline, error) {
	var result []trade.Kline
	_, err := retry.Do(ctx, func() (struct{}, error) {
		if waitErr := c.limiter.Wait(ctx); waitErr != nil {
			return struct{}{}, waitErr
		}
		var retryErr error
		result, retryErr = c.client.Kline(ctx, symbol, interval, limit)
		return struct{}{}, retryErr
	}, retry.WithPolicy(retry.DefaultPolicy))
	return result, err
}

func (c *RateLimitedClient) GetBalance(ctx context.Context) (float64, error) {
	var balance float64
	_, err := retry.Do(ctx, func() (struct{}, error) {
		if waitErr := c.limiter.Wait(ctx); waitErr != nil {
			return struct{}{}, waitErr
		}
		var retryErr error
		balance, retryErr = c.client.GetBalance(ctx)
		return struct{}{}, retryErr
	}, retry.WithPolicy(retry.DefaultPolicy))
	return balance, err
}

func (c *RateLimitedClient) CreateOrder(ctx context.Context, order *trade.Order) (*trade.Order, error) {
	var result *trade.Order
	_, err := retry.Do(ctx, func() (struct{}, error) {
		if waitErr := c.limiter.Wait(ctx); waitErr != nil {
			return struct{}{}, waitErr
		}
		var retryErr error
		result, retryErr = c.client.CreateOrder(ctx, order)
		return struct{}{}, retryErr
	}, retry.WithPolicy(retry.Policy{
		MaxRetries: 3,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   10 * time.Second,
		Jitter:     0.3,
	}))
	return result, err
}

func (c *RateLimitedClient) GetPosition(ctx context.Context, symbol string) (*trade.Position, error) {
	var result *trade.Position
	_, err := retry.Do(ctx, func() (struct{}, error) {
		if waitErr := c.limiter.Wait(ctx); waitErr != nil {
			return struct{}{}, waitErr
		}
		var retryErr error
		result, retryErr = c.client.GetPosition(ctx, symbol)
		return struct{}{}, retryErr
	}, retry.WithPolicy(retry.DefaultPolicy))
	return result, err
}

func (c *RateLimitedClient) Symbols(ctx context.Context) ([]string, error) {
	var result []string
	_, err := retry.Do(ctx, func() (struct{}, error) {
		if waitErr := c.limiter.Wait(ctx); waitErr != nil {
			return struct{}{}, waitErr
		}
		var retryErr error
		result, retryErr = c.client.Symbols(ctx)
		return struct{}{}, retryErr
	}, retry.WithPolicy(retry.DefaultPolicy))
	return result, err
}

type FanOutClient struct {
	clients []*RateLimitedClient
}

func NewFanOutClient(clients []*RateLimitedClient) *FanOutClient {
	return &FanOutClient{clients: clients}
}

func (f *FanOutClient) FetchAllPrices(ctx context.Context, symbols []string) map[string]float64 {
	results := make(map[string]float64)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			clientIdx := rand.Intn(len(f.clients))
			price, err := f.clients[clientIdx].Price(ctx, sym)
			if err == nil {
				mu.Lock()
				results[sym] = price
				mu.Unlock()
			}
		}(symbol)
	}

	wg.Wait()
	return results
}
