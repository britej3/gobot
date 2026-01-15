package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type Policy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Jitter     float64
}

func (p Policy) Backoff(attempt int) time.Duration {
	if attempt >= p.MaxRetries {
		return -1
	}

	delay := float64(p.BaseDelay) * math.Pow(2, float64(attempt))
	delay = math.Min(delay, float64(p.MaxDelay))

	if p.Jitter > 0 {
		jitterRange := delay * p.Jitter
		delay += (rand.Float64()*2 - 1) * jitterRange
	}

	return time.Duration(delay)
}

func Do[T any](ctx context.Context, fn func() (T, error), opts ...Option) (T, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	var lastErr error
	var result T

	for attempt := 0; attempt <= cfg.Policy.MaxRetries; attempt++ {
		result, lastErr = fn()
		if lastErr == nil {
			return result, nil
		}

		if !cfg.IsRetryable(lastErr) {
			return result, lastErr
		}

		delay := cfg.Policy.Backoff(attempt)
		if delay < 0 {
			return result, lastErr
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}

	return result, lastErr
}

type Option func(*config)

func WithPolicy(p Policy) Option {
	return func(c *config) {
		c.Policy = p
	}
}

func WithRetryableFn(fn func(error) bool) Option {
	return func(c *config) {
		c.IsRetryable = fn
	}
}

type config struct {
	Policy      Policy
	IsRetryable func(error) bool
}

func defaultConfig() config {
	return config{
		Policy: Policy{
			MaxRetries: 3,
			BaseDelay:  100 * time.Millisecond,
			MaxDelay:   5 * time.Second,
			Jitter:     0.2,
		},
		IsRetryable: func(err error) bool {
			return err != nil
		},
	}
}

var DefaultPolicy = Policy{
	MaxRetries: 3,
	BaseDelay:  100 * time.Millisecond,
	MaxDelay:   5 * time.Second,
	Jitter:     0.25,
}
