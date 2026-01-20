package binance

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ConnectionPool manages persistent HTTP connections for optimal performance
type ConnectionPool struct {
	size        int
	connections []*http.Client
	mu          sync.RWMutex
	current     int
	logger      *logrus.Logger
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(size int) *ConnectionPool {
	if size <= 0 {
		size = 10 // Default pool size
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	pool := &ConnectionPool{
		size:        size,
		connections: make([]*http.Client, size),
		logger:      logger,
	}

	// Initialize connections
	for i := 0; i < size; i++ {
		pool.connections[i] = createOptimizedHTTPClient()
	}

	logger.WithFields(logrus.Fields{
		"pool_size": size,
	}).Info("connection_pool_initialized")

	return pool
}

// createOptimizedHTTPClient creates an HTTP client optimized for low latency
func createOptimizedHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			MaxConnsPerHost:       10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableCompression:    false,
			DisableKeepAlives:     false,
		},
	}
}

// GetConnection returns the next available connection from the pool
func (cp *ConnectionPool) GetConnection() *http.Client {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Round-robin selection
	conn := cp.connections[cp.current]
	cp.current = (cp.current + 1) % cp.size

	return conn
}

// GetConnectionByIndex returns a specific connection by index
func (cp *ConnectionPool) GetConnectionByIndex(index int) *http.Client {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if index < 0 || index >= cp.size {
		index = 0
	}

	return cp.connections[index]
}

// Size returns the size of the connection pool
func (cp *ConnectionPool) Size() int {
	return cp.size
}

// HealthCheck checks the health of all connections in the pool
func (cp *ConnectionPool) HealthCheck() error {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	healthyCount := 0
	for i, conn := range cp.connections {
		if conn != nil && conn.Transport != nil {
			healthyCount++
		} else {
			cp.logger.WithFields(logrus.Fields{
				"connection_index": i,
			}).Warn("unhealthy_connection")
		}
	}

	cp.logger.WithFields(logrus.Fields{
		"healthy_count": healthyCount,
		"total_count":   cp.size,
	}).Info("connection_pool_health_check")

	return nil
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for i, conn := range cp.connections {
		if conn != nil && conn.Transport != nil {
			if transport, ok := conn.Transport.(*http.Transport); ok {
				transport.CloseIdleConnections()
			}
		}
		cp.connections[i] = nil
	}

	cp.logger.Info("connection_pool_closed")
}

// Refresh recreates all connections in the pool
func (cp *ConnectionPool) Refresh() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for i := 0; i < cp.size; i++ {
		if cp.connections[i] != nil {
			if transport, ok := cp.connections[i].Transport.(*http.Transport); ok {
				transport.CloseIdleConnections()
			}
		}
		cp.connections[i] = createOptimizedHTTPClient()
	}

	cp.logger.Info("connection_pool_refreshed")
}

// Stats returns statistics about the connection pool
func (cp *ConnectionPool) Stats() *PoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := &PoolStats{
		Size:    cp.size,
		Current: cp.current,
		Healthy: 0,
	}

	for _, conn := range cp.connections {
		if conn != nil && conn.Transport != nil {
			stats.Healthy++
		}
	}

	return stats
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	Size    int
	Current int
	Healthy int
}
