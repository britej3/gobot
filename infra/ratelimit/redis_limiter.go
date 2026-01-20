package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(endpoint string) bool
	Reset(endpoint string) error
	GetUsage(endpoint string) (int, int, error) // current, limit
}

// RedisRateLimiter implements distributed rate limiting using Redis
type RedisRateLimiter struct {
	client *redis.Client
	limits map[string]RateLimit
	logger *logrus.Logger
}

// RateLimit defines rate limit configuration for an endpoint
type RateLimit struct {
	Endpoint      string
	RequestsPerMinute int
	BurstCapacity int
	SafetyMargin  float64 // 5.0 for 5x safety margin
}

// Config holds Redis configuration
type Config struct {
	Addr     string
	Password string
	DB       int
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(config Config) *RedisRateLimiter {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	limiter := &RedisRateLimiter{
		client: client,
		limits: make(map[string]RateLimit),
		logger: logger,
	}

	// Initialize Binance Futures rate limits with 5x safety margin
	limiter.initializeBinanceLimits()

	return limiter
}

// initializeBinanceLimits sets up Binance Futures API rate limits
func (rrl *RedisRateLimiter) initializeBinanceLimits() {
	// Binance Futures official limits: 2400 requests/minute
	// Our limit with 5x safety margin: 480 requests/minute (2400 / 5)
	
	binanceLimit := 2400
	safetyMargin := 5.0
	ourLimit := int(float64(binanceLimit) / safetyMargin)

	// General endpoints
	rrl.limits["exchange_info"] = RateLimit{
		Endpoint:          "exchange_info",
		RequestsPerMinute: ourLimit,
		BurstCapacity:     ourLimit / 10, // 10% burst
		SafetyMargin:      safetyMargin,
	}

	rrl.limits["get_account"] = RateLimit{
		Endpoint:          "get_account",
		RequestsPerMinute: ourLimit,
		BurstCapacity:     ourLimit / 10,
		SafetyMargin:      safetyMargin,
	}

	rrl.limits["get_balance"] = RateLimit{
		Endpoint:          "get_balance",
		RequestsPerMinute: ourLimit,
		BurstCapacity:     ourLimit / 10,
		SafetyMargin:      safetyMargin,
	}

	rrl.limits["get_positions"] = RateLimit{
		Endpoint:          "get_positions",
		RequestsPerMinute: ourLimit,
		BurstCapacity:     ourLimit / 10,
		SafetyMargin:      safetyMargin,
	}

	// Market data endpoints (higher frequency)
	rrl.limits["get_mark_price"] = RateLimit{
		Endpoint:          "get_mark_price",
		RequestsPerMinute: ourLimit,
		BurstCapacity:     ourLimit / 5, // 20% burst for market data
		SafetyMargin:      safetyMargin,
	}

	rrl.limits["get_funding_rate"] = RateLimit{
		Endpoint:          "get_funding_rate",
		RequestsPerMinute: ourLimit,
		BurstCapacity:     ourLimit / 5,
		SafetyMargin:      safetyMargin,
	}

	// Order endpoints (critical, lower limit)
	orderLimit := int(float64(binanceLimit) / (safetyMargin * 2)) // Extra safety for orders
	
	rrl.limits["create_order"] = RateLimit{
		Endpoint:          "create_order",
		RequestsPerMinute: orderLimit,
		BurstCapacity:     orderLimit / 20, // 5% burst for orders
		SafetyMargin:      safetyMargin * 2,
	}

	rrl.limits["cancel_order"] = RateLimit{
		Endpoint:          "cancel_order",
		RequestsPerMinute: orderLimit,
		BurstCapacity:     orderLimit / 20,
		SafetyMargin:      safetyMargin * 2,
	}

	rrl.limits["get_order"] = RateLimit{
		Endpoint:          "get_order",
		RequestsPerMinute: ourLimit,
		BurstCapacity:     ourLimit / 10,
		SafetyMargin:      safetyMargin,
	}

	// Leverage and margin endpoints
	rrl.limits["set_leverage"] = RateLimit{
		Endpoint:          "set_leverage",
		RequestsPerMinute: orderLimit,
		BurstCapacity:     orderLimit / 20,
		SafetyMargin:      safetyMargin * 2,
	}

	rrl.limits["set_margin_type"] = RateLimit{
		Endpoint:          "set_margin_type",
		RequestsPerMinute: orderLimit,
		BurstCapacity:     orderLimit / 20,
		SafetyMargin:      safetyMargin * 2,
	}

	rrl.limits["set_position_mode"] = RateLimit{
		Endpoint:          "set_position_mode",
		RequestsPerMinute: orderLimit,
		BurstCapacity:     orderLimit / 20,
		SafetyMargin:      safetyMargin * 2,
	}

	rrl.limits["get_position_mode"] = RateLimit{
		Endpoint:          "get_position_mode",
		RequestsPerMinute: ourLimit,
		BurstCapacity:     ourLimit / 10,
		SafetyMargin:      safetyMargin,
	}

	rrl.logger.WithFields(logrus.Fields{
		"binance_limit": binanceLimit,
		"our_limit":     ourLimit,
		"safety_margin": safetyMargin,
	}).Info("rate_limits_initialized")
}

// Allow checks if a request is allowed under the rate limit
func (rrl *RedisRateLimiter) Allow(endpoint string) bool {
	limit, exists := rrl.limits[endpoint]
	if !exists {
		// Unknown endpoint, use default limit
		limit = RateLimit{
			Endpoint:          endpoint,
			RequestsPerMinute: 100,
			BurstCapacity:     10,
			SafetyMargin:      5.0,
		}
		rrl.limits[endpoint] = limit
	}

	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:%s", endpoint)

	// Use sliding window algorithm
	now := time.Now()
	windowStart := now.Add(-1 * time.Minute)

	// Remove old entries
	rrl.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count requests in current window
	count, err := rrl.client.ZCount(ctx, key, fmt.Sprintf("%d", windowStart.UnixNano()), "+inf").Result()
	if err != nil {
		rrl.logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
			"error":    err.Error(),
		}).Error("rate_limit_check_failed")
		return false
	}

	// Check if under limit
	if int(count) >= limit.RequestsPerMinute {
		rrl.logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
			"count":    count,
			"limit":    limit.RequestsPerMinute,
		}).Warn("rate_limit_exceeded")
		return false
	}

	// Add current request
	member := fmt.Sprintf("%d", now.UnixNano())
	rrl.client.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: member,
	})

	// Set expiration on key
	rrl.client.Expire(ctx, key, 2*time.Minute)

	// Check burst capacity
	recentWindow := now.Add(-10 * time.Second)
	recentCount, _ := rrl.client.ZCount(ctx, key, fmt.Sprintf("%d", recentWindow.UnixNano()), "+inf").Result()
	
	if int(recentCount) > limit.BurstCapacity {
		rrl.logger.WithFields(logrus.Fields{
			"endpoint":       endpoint,
			"recent_count":   recentCount,
			"burst_capacity": limit.BurstCapacity,
		}).Warn("burst_capacity_exceeded")
		return false
	}

	return true
}

// Reset resets the rate limit for an endpoint
func (rrl *RedisRateLimiter) Reset(endpoint string) error {
	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:%s", endpoint)

	err := rrl.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	rrl.logger.WithFields(logrus.Fields{
		"endpoint": endpoint,
	}).Info("rate_limit_reset")

	return nil
}

// GetUsage returns current usage and limit for an endpoint
func (rrl *RedisRateLimiter) GetUsage(endpoint string) (int, int, error) {
	limit, exists := rrl.limits[endpoint]
	if !exists {
		return 0, 0, fmt.Errorf("unknown endpoint: %s", endpoint)
	}

	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:%s", endpoint)

	now := time.Now()
	windowStart := now.Add(-1 * time.Minute)

	count, err := rrl.client.ZCount(ctx, key, fmt.Sprintf("%d", windowStart.UnixNano()), "+inf").Result()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get usage: %w", err)
	}

	return int(count), limit.RequestsPerMinute, nil
}

// GetAllUsage returns usage statistics for all endpoints
func (rrl *RedisRateLimiter) GetAllUsage() map[string]UsageStats {
	stats := make(map[string]UsageStats)

	for endpoint := range rrl.limits {
		current, limit, err := rrl.GetUsage(endpoint)
		if err != nil {
			continue
		}

		percentage := float64(current) / float64(limit) * 100

		stats[endpoint] = UsageStats{
			Endpoint:   endpoint,
			Current:    current,
			Limit:      limit,
			Percentage: percentage,
		}
	}

	return stats
}

// UsageStats represents usage statistics for an endpoint
type UsageStats struct {
	Endpoint   string
	Current    int
	Limit      int
	Percentage float64
}

// Close closes the Redis connection
func (rrl *RedisRateLimiter) Close() error {
	return rrl.client.Close()
}

// HealthCheck checks if Redis connection is healthy
func (rrl *RedisRateLimiter) HealthCheck() error {
	ctx := context.Background()
	return rrl.client.Ping(ctx).Err()
}

// GetStats returns overall rate limiting statistics
func (rrl *RedisRateLimiter) GetStats() *Stats {
	allUsage := rrl.GetAllUsage()

	totalCurrent := 0
	totalLimit := 0
	maxUsagePercentage := 0.0

	for _, usage := range allUsage {
		totalCurrent += usage.Current
		totalLimit += usage.Limit
		if usage.Percentage > maxUsagePercentage {
			maxUsagePercentage = usage.Percentage
		}
	}

	avgUsagePercentage := 0.0
	if totalLimit > 0 {
		avgUsagePercentage = float64(totalCurrent) / float64(totalLimit) * 100
	}

	return &Stats{
		TotalCurrent:       totalCurrent,
		TotalLimit:         totalLimit,
		AvgUsagePercentage: avgUsagePercentage,
		MaxUsagePercentage: maxUsagePercentage,
		EndpointCount:      len(allUsage),
	}
}

// Stats represents overall rate limiting statistics
type Stats struct {
	TotalCurrent       int
	TotalLimit         int
	AvgUsagePercentage float64
	MaxUsagePercentage float64
	EndpointCount      int
}

// Monitor starts a monitoring goroutine that logs rate limit usage
func (rrl *RedisRateLimiter) Monitor(interval time.Duration) chan struct{} {
	stopChan := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				stats := rrl.GetStats()
				
				rrl.logger.WithFields(logrus.Fields{
					"total_current":        stats.TotalCurrent,
					"total_limit":          stats.TotalLimit,
					"avg_usage_percentage": fmt.Sprintf("%.2f%%", stats.AvgUsagePercentage),
					"max_usage_percentage": fmt.Sprintf("%.2f%%", stats.MaxUsagePercentage),
					"endpoint_count":       stats.EndpointCount,
				}).Info("rate_limit_stats")

				// Alert if usage is too high (>70%)
				if stats.MaxUsagePercentage > 70 {
					rrl.logger.WithFields(logrus.Fields{
						"max_usage_percentage": fmt.Sprintf("%.2f%%", stats.MaxUsagePercentage),
					}).Warn("high_rate_limit_usage")
				}
			}
		}
	}()

	return stopChan
}
