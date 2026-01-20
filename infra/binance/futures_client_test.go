package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewFuturesClient(t *testing.T) {
	config := FuturesConfig{
		APIKey:    "test_key",
		APISecret: "test_secret",
		Testnet:   true,
		PoolSize:  5,
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
	}

	client := NewFuturesClient(config)

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.NotNil(t, client.connPool)
	assert.NotNil(t, client.wsMultiplexer)
	assert.NotNil(t, client.rateLimiter)
	assert.NotNil(t, client.circuitBreaker)
	assert.NotNil(t, client.logger)
	assert.True(t, client.testnet)
}

func TestConnectionPool(t *testing.T) {
	pool := NewConnectionPool(5)

	assert.NotNil(t, pool)
	assert.Equal(t, 5, pool.Size())

	// Test getting connections
	conn1 := pool.GetConnection()
	assert.NotNil(t, conn1)

	conn2 := pool.GetConnection()
	assert.NotNil(t, conn2)

	// Test health check
	err := pool.HealthCheck()
	assert.NoError(t, err)

	// Test stats
	stats := pool.Stats()
	assert.Equal(t, 5, stats.Size)
	assert.Equal(t, 5, stats.Healthy)

	// Test close
	pool.Close()
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewAdaptiveCircuitBreaker()

	assert.NotNil(t, cb)

	// Test initial state
	assert.Equal(t, StateClosed, cb.GetState())

	// Test allowing requests
	assert.True(t, cb.Allow())

	// Test recording success
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())

	// Test recording failures
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Test that requests are blocked when open
	assert.False(t, cb.Allow())

	// Test reset
	cb.Reset()
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreakerTransitions(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		HalfOpenRequests: 2,
	}

	cb := NewAdaptiveCircuitBreakerWithConfig(config)

	// Start in closed state
	assert.Equal(t, StateClosed, cb.GetState())

	// Trigger failures to open circuit
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open
	assert.True(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.GetState())

	// Record successes to close circuit
	cb.RecordSuccess()
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestWebSocketMultiplexer(t *testing.T) {
	wsm := NewWebSocketMultiplexer()

	assert.NotNil(t, wsm)
	assert.NotNil(t, wsm.connections)
	assert.NotNil(t, wsm.logger)

	// Test subscription
	err := wsm.SubscribeOrderBook("BTCUSDT")
	assert.NoError(t, err)
	assert.Equal(t, 1, wsm.GetConnectionCount())

	// Note: Actual WebSocket tests would require a running Binance testnet
	// This test just verifies initialization

	wsm.Close()
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"123.45", 123.45},
		{"0.001", 0.001},
		{"1000", 1000.0},
		{"0", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseFloat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"10", 10},
		{"0", 0},
		{"100", 100},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := NewAdaptiveCircuitBreaker().(*AdaptiveCircuitBreaker)

	// Record some operations
	for i := 0; i < 10; i++ {
		cb.Allow()
		if i < 5 {
			cb.RecordSuccess()
		} else {
			cb.RecordFailure()
		}
	}

	stats := cb.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, int64(10), stats.TotalRequests)
	assert.Equal(t, int64(5), stats.TotalSuccesses)
	assert.Equal(t, int64(5), stats.TotalFailures)
	assert.Equal(t, 50.0, stats.FailureRate)
}

func TestConnectionPoolRefresh(t *testing.T) {
	pool := NewConnectionPool(3)

	// Get initial connections
	conn1 := pool.GetConnection()
	assert.NotNil(t, conn1)

	// Refresh pool
	pool.Refresh()

	// Get connection after refresh
	conn2 := pool.GetConnection()
	assert.NotNil(t, conn2)

	pool.Close()
}

func TestConnectionPoolGetByIndex(t *testing.T) {
	pool := NewConnectionPool(5)

	// Test valid indices
	for i := 0; i < 5; i++ {
		conn := pool.GetConnectionByIndex(i)
		assert.NotNil(t, conn)
	}

	// Test invalid indices (should return first connection)
	conn := pool.GetConnectionByIndex(-1)
	assert.NotNil(t, conn)

	conn = pool.GetConnectionByIndex(10)
	assert.NotNil(t, conn)

	pool.Close()
}

// Benchmark tests

func BenchmarkCircuitBreakerAllow(b *testing.B) {
	cb := NewAdaptiveCircuitBreaker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Allow()
	}
}

func BenchmarkCircuitBreakerRecordSuccess(b *testing.B) {
	cb := NewAdaptiveCircuitBreaker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.RecordSuccess()
	}
}

func BenchmarkConnectionPoolGetConnection(b *testing.B) {
	pool := NewConnectionPool(10)
	defer pool.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.GetConnection()
	}
}

func BenchmarkParseFloat(b *testing.B) {
	input := "123.456789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseFloat(input)
	}
}

// Integration tests (require actual API keys and testnet)

func TestFuturesClientIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test requires actual API keys
	// Set BINANCE_API_KEY and BINANCE_API_SECRET environment variables
	// and BINANCE_TESTNET=true

	t.Skip("Integration test requires API keys")
}

func TestOrderExecutionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Skip("Integration test requires API keys")
}
