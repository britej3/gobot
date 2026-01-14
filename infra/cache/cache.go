package cache

import (
	"context"
	"sync"
	"time"
)

type Cache struct {
	mu      sync.RWMutex
	items   map[string]cacheItem
	maxSize int
	evictCh chan string
}

type cacheItem struct {
	value      interface{}
	expiresAt  time.Time
	accessedAt time.Time
}

type Config struct {
	MaxSize       int
	DefaultTTL    time.Duration
	CleanupPeriod time.Duration
}

func New(cfg Config) *Cache {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 1000
	}
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}
	if cfg.CleanupPeriod <= 0 {
		cfg.CleanupPeriod = time.Minute
	}

	cache := &Cache{
		items:   make(map[string]cacheItem),
		maxSize: cfg.MaxSize,
		evictCh: make(chan string, 100),
	}

	go cache.cleanupLoop(cfg.CleanupPeriod)

	return cache
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	return c.SetWithTTL(ctx, key, value, 0)
}

func (c *Cache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Now().Add(ttl)
	if ttl == 0 {
		expiresAt = time.Now().Add(5 * time.Minute)
	}

	c.items[key] = cacheItem{
		value:      value,
		expiresAt:  expiresAt,
		accessedAt: time.Now(),
	}

	if len(c.items) > c.maxSize {
		c.evictOldest()
	}

	return nil
}

func (c *Cache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false, nil
	}

	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false, nil
	}

	c.mu.Lock()
	item.accessedAt = time.Now()
	c.items[key] = item
	c.mu.Unlock()

	return item.value, true, nil
}

func (c *Cache) GetString(ctx context.Context, key string) (string, bool, error) {
	val, exists, err := c.Get(ctx, key)
	if err != nil || !exists {
		return "", exists, err
	}

	str, ok := val.(string)
	return str, ok, nil
}

func (c *Cache) GetFloat64(ctx context.Context, key string) (float64, bool, error) {
	val, exists, err := c.Get(ctx, key)
	if err != nil || !exists {
		return 0, exists, err
	}

	f, ok := val.(float64)
	return f, ok, nil
}

func (c *Cache) GetInt(ctx context.Context, key string) (int, bool, error) {
	val, exists, err := c.Get(ctx, key)
	if err != nil || !exists {
		return 0, exists, err
	}

	i, ok := val.(int)
	return i, ok, nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}

func (c *Cache) DeletePrefix(ctx context.Context, prefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := make([]string, 0, len(c.items))
	for k := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k)
		}
	}

	for _, k := range keys {
		delete(c.items, k)
	}

	return nil
}

func (c *Cache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]cacheItem)
	return nil
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return false, nil
	}

	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return false, nil
	}

	return true, nil
}

func (c *Cache) Keys(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	now := time.Now()

	for k, item := range c.items {
		if now.Before(item.expiresAt) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

func (c *Cache) Size(ctx context.Context) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items), nil
}

func (c *Cache) cleanupLoop(period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
				select {
				case c.evictCh <- key:
				default:
				}
			}
		}
		c.mu.Unlock()
	}
}

func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.accessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.accessedAt
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
		select {
		case c.evictCh <- oldestKey:
		default:
		}
	}
}

func (c *Cache) Evicted() <-chan string {
	return c.evictCh
}

func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Items: len(c.items),
		Max:   c.maxSize,
	}
}

type CacheStats struct {
	Items int
	Max   int
}

func (c *Cache) GetWithLoader(ctx context.Context, key string, loader func(ctx context.Context) (interface{}, error), ttl time.Duration) (interface{}, bool, error) {
	val, exists, err := c.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}

	if exists {
		return val, true, nil
	}

	val, err = loader(ctx)
	if err != nil {
		return nil, false, err
	}

	if err := c.SetWithTTL(ctx, key, val, ttl); err != nil {
		return nil, false, err
	}

	return val, false, nil
}

type CacheError struct {
	Message string
}

func (e *CacheError) Error() string {
	return e.Message
}

var (
	ErrKeyNotFound = &CacheError{Message: "key not found"}
	ErrKeyExpired  = &CacheError{Message: "key has expired"}
)

func (c *Cache) GetWithErr(ctx context.Context, key string) (interface{}, error) {
	val, exists, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrKeyNotFound
	}

	return val, nil
}
