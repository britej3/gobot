package stealth

import (
	"context"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	mrand "math/rand"
	"net/http"
	"sync"
	"time"
)

type StealthConfig struct {
	JitterRange       time.Duration
	RequestDelayMin   time.Duration
	RequestDelayMax   time.Duration
	UserAgents        []string
	RotateUserAgents  bool
	SignatureVariance float64
}

type StealthClient struct {
	cfg       StealthConfig
	rng       *mrand.Rand
	mu        sync.RWMutex
	userAgent string
}

func New(cfg StealthConfig) *StealthClient {
	source := mrand.NewSource(time.Now().UnixNano())
	return &StealthClient{
		cfg:       cfg,
		rng:       mrand.New(source),
		userAgent: cfg.UserAgents[0],
	}
}

func (s *StealthClient) WithJitter(baseDelay time.Duration) time.Duration {
	if s.cfg.JitterRange == 0 {
		return baseDelay
	}

	jitter := s.rng.Float64() * float64(s.cfg.JitterRange)
	return time.Duration(float64(baseDelay) + jitter)
}

func (s *StealthClient) RandomDelay(ctx context.Context) error {
	if s.cfg.RequestDelayMin == 0 && s.cfg.RequestDelayMax == 0 {
		return nil
	}

	delay := s.cfg.RequestDelayMin
	if s.cfg.RequestDelayMax > s.cfg.RequestDelayMin {
		delay = s.cfg.RequestDelayMin + time.Duration(s.rng.Float64()*float64(s.cfg.RequestDelayMax-s.cfg.RequestDelayMin))
	}

	delay = s.WithJitter(delay)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

func (s *StealthClient) RotateUserAgent() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.cfg.UserAgents) > 1 {
		idx := s.rng.Intn(len(s.cfg.UserAgents))
		s.userAgent = s.cfg.UserAgents[idx]
	}
}

func (s *StealthClient) GetUserAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userAgent
}

func (s *StealthClient) ObfuscateSignature(secret, payload string) string {
	if s.cfg.SignatureVariance == 0 {
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(payload))
		return hex.EncodeToString(h.Sum(nil))
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	sig := h.Sum(nil)

	idx := s.rng.Intn(len(sig))
	sig[idx] ^= byte(s.rng.Intn(256))

	return hex.EncodeToString(sig)
}

func (s *StealthClient) RandomizeHeaders(req *http.Request) {
	s.RotateUserAgent()
	req.Header.Set("User-Agent", s.GetUserAgent())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-Request-ID", generateRequestID())
}

func (s *StealthClient) WrapRequest(ctx context.Context, fn func() error) error {
	if err := s.RandomDelay(ctx); err != nil {
		return err
	}
	return fn()
}

func (s *StealthClient) ExponentialBackoffWithJitter(baseDelay time.Duration, attempt int) time.Duration {
	delay := float64(baseDelay) * math.Pow(2, float64(attempt))
	delay = math.Min(delay, float64(60*time.Second))
	delay *= (0.5 + s.rng.Float64())

	return time.Duration(delay)
}

func generateRequestID() string {
	b := make([]byte, 16)
	crand.Read(b)
	return fmt.Sprintf("%x", b)
}

type RequestPattern struct {
	mu           sync.RWMutex
	intervals    []time.Duration
	lastRequests map[string]time.Time
}

func NewRequestPattern() *RequestPattern {
	return &RequestPattern{
		intervals:    make([]time.Duration, 0),
		lastRequests: make(map[string]time.Time),
	}
}

func (p *RequestPattern) Record(endpoint string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	if last, ok := p.lastRequests[endpoint]; ok {
		p.intervals = append(p.intervals, now.Sub(last))
	}
	p.lastRequests[endpoint] = now

	if len(p.intervals) > 100 {
		p.intervals = p.intervals[1:]
	}
}

func (p *RequestPattern) GetAverageInterval() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.intervals) == 0 {
		return 0
	}

	var sum time.Duration
	for _, i := range p.intervals {
		sum += i
	}
	return sum / time.Duration(len(p.intervals))
}

func (p *RequestPattern) ShouldThrottle(endpoint string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avg := p.GetAverageInterval()
	if avg == 0 {
		return false
	}

	last, ok := p.lastRequests[endpoint]
	if !ok {
		return false
	}

	return time.Since(last) < avg/2
}

var DefaultStealthConfig = StealthConfig{
	JitterRange:       100 * time.Millisecond,
	RequestDelayMin:   50 * time.Millisecond,
	RequestDelayMax:   200 * time.Millisecond,
	UserAgents:        CommonUserAgents(),
	RotateUserAgents:  true,
	SignatureVariance: 0.01,
}

func CommonUserAgents() []string {
	return []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}
