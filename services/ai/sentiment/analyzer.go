package sentiment

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config holds sentiment analyzer configuration
type Config struct {
	CoinGeckoAPIKey       string  // Optional API key for higher limits
	FearGreedUpdateFreq   int     // How often to update Fear & Greed Index (seconds)
	FundingRateWindow     int     // Window for funding rate trend analysis
	SocialVolumeThreshold float64 // Threshold for social volume spike detection
	CacheDuration         int     // Cache duration in seconds
}

// SentimentAnalyzer analyzes market sentiment from multiple sources
type SentimentAnalyzer struct {
	config              Config
	httpClient          *http.Client
	mu                  sync.RWMutex
	cache               map[string]cacheEntry
	fearGreedIndex      float64
	lastFearGreedUpdate time.Time
}

// SentimentScore represents the overall sentiment analysis result
type SentimentScore struct {
	OverallScore    float64                // 0-100 overall sentiment
	OverallLabel    string                 // "Bullish", "Bearish", "Neutral"
	Components      map[string]float64     // Individual component scores
	ComponentsLabel map[string]string      // Component labels
	Sources         []string               // Data sources used
	Symbol          string                 // Analyzed symbol (empty for market-wide)
	Timestamp       time.Time              // When analyzed
	Confidence      float64                // 0-1 confidence in the score
	RawData         map[string]interface{} // Raw data from sources
}

// FearGreedData represents the Fear & Greed Index data
type FearGreedData struct {
	Value       float64 // 0-100
	ValueText   string  // "Extreme Fear", "Fear", "Neutral", "Greed", "Extreme Greed"
	Timestamp   int64
	Description string
}

// FundingRateData represents funding rate information
type FundingRateData struct {
	Symbol         string
	FundingRate    float64
	FundingRateAvg float64
	PredictedRate  float64
	MarkPrice      float64
	IndexPrice     float64
	Timestamp      time.Time
	Trend          string // "bullish", "bearish", "neutral"
}

// SocialVolumeData represents social media volume data
type SocialVolumeData struct {
	Symbol          string
	Volume          int64
	VolumeChange24h float64
	VolumeChange7d  float64
	Sentiment       float64 // -1 to 1
	Rank            int
	SpikeDetected   bool
}

// CoinGeckoTrending represents trending coins data
type CoinGeckoTrending struct {
	Coins []struct {
		Item struct {
			ID            string
			Name          string
			Symbol        string
			MarketCapRank int
			Score         float64
		}
	}
}

// cacheEntry for caching API responses
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// NewSentimentAnalyzer creates a new sentiment analyzer
func NewSentimentAnalyzer() *SentimentAnalyzer {
	return &SentimentAnalyzer{
		config: Config{
			FearGreedUpdateFreq:   3600,
			FundingRateWindow:     24,
			SocialVolumeThreshold: 2.0,
			CacheDuration:         300,
		},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]cacheEntry),
	}
}

// NewSentimentAnalyzerWithConfig creates a sentiment analyzer with custom config
func NewSentimentAnalyzerWithConfig(cfg Config) *SentimentAnalyzer {
	analyzer := NewSentimentAnalyzer()
	analyzer.config = cfg
	return analyzer
}

// GetMarketSentiment analyzes overall market sentiment
func (s *SentimentAnalyzer) GetMarketSentiment() (*SentimentScore, error) {
	return s.analyzeSentiment("", nil)
}

// GetSymbolSentiment analyzes sentiment for a specific symbol
func (s *SentimentAnalyzer) GetSymbolSentiment(symbol string) (*SentimentScore, error) {
	return s.analyzeSentiment(symbol, nil)
}

// analyzeSentiment performs the actual sentiment analysis
func (s *SentimentAnalyzer) analyzeSentiment(symbol string, overrides map[string]interface{}) (*SentimentScore, error) {
	result := &SentimentScore{
		Symbol:          symbol,
		Timestamp:       time.Now(),
		Components:      make(map[string]float64),
		ComponentsLabel: make(map[string]string),
		Sources:         make([]string, 0),
		RawData:         make(map[string]interface{}),
	}

	// 1. Fear & Greed Index (Market-wide)
	fearGreed, err := s.GetFearGreedIndex()
	if err == nil {
		result.Components["fear_greed"] = fearGreed.Value
		result.ComponentsLabel["fear_greed"] = fearGreed.ValueText
		result.Sources = append(result.Sources, "fear_greed_index")
		result.RawData["fear_greed"] = fearGreed
	}

	// 2. Symbol-specific funding rate (if symbol provided)
	if symbol != "" {
		funding, err := s.GetFundingRateTrend(symbol)
		if err == nil {
			result.Components["funding_rate"] = s.fundingRateToScore(funding)
			result.ComponentsLabel["funding_rate"] = funding.Trend
			result.Sources = append(result.Sources, "binance_funding")
			result.RawData["funding"] = funding
		}

		// 3. Social volume (if symbol provided)
		social, err := s.GetSocialVolume(symbol)
		if err == nil {
			result.Components["social_volume"] = s.socialVolumeToScore(social)
			result.ComponentsLabel["social_volume"] = s.getSocialLabel(social)
			result.Sources = append(result.Sources, "coingecko_social")
			result.RawData["social"] = social
		}
	}

	// 4. Trending coins sentiment (market-wide)
	trending, err := s.GetTrendingSentiment()
	if err == nil {
		result.Components["trending"] = trending
		result.ComponentsLabel["trending"] = s.getTrendingLabel(trending)
		result.Sources = append(result.Sources, "coingecko_trending")
	}

	// Calculate overall score
	overallScore := s.calculateOverallScore(result.Components)
	result.OverallScore = overallScore
	result.OverallLabel = s.getOverallLabel(overallScore)
	result.Confidence = s.calculateConfidence(result.Sources)

	if overrides != nil {
		for key, value := range overrides {
			result.RawData[key] = value
		}
	}

	return result, nil
}

// GetFearGreedIndex retrieves the current Fear & Greed Index
func (s *SentimentAnalyzer) GetFearGreedIndex() (*FearGreedData, error) {
	s.mu.Lock()
	if entry, ok := s.cache["fear_greed"]; ok && time.Now().Before(entry.expiresAt) {
		s.mu.Unlock()
		return entry.data.(*FearGreedData), nil
	}
	s.mu.Unlock()

	url := "https://api.alternative.me/fng/"
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Value      string
			ValueClass string
			Timestamp  string
		}
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, err
	}

	data := &FearGreedData{}
	if v, err := strconv.ParseFloat(result.Data[0].Value, 64); err == nil {
		data.Value = v
	}
	data.ValueText = result.Data[0].ValueClass
	if ts, err := strconv.ParseInt(result.Data[0].Timestamp, 10, 64); err == nil {
		data.Timestamp = ts
	}

	switch {
	case data.Value < 25:
		data.Description = "Extreme Fear - Market may be oversold"
	case data.Value < 45:
		data.Description = "Fear - Some uncertainty in market"
	case data.Value < 55:
		data.Description = "Neutral - Balanced market sentiment"
	case data.Value < 75:
		data.Description = "Greed - Growing optimism"
	default:
		data.Description = "Extreme Greed - Market may be overheated"
	}

	s.mu.Lock()
	s.cache["fear_greed"] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(time.Duration(s.config.CacheDuration) * time.Second),
	}
	s.mu.Unlock()

	return data, nil
}

// GetFundingRateTrend retrieves funding rate data and determines trend
func (s *SentimentAnalyzer) GetFundingRateTrend(symbol string) (*FundingRateData, error) {
	cacheKey := "funding_" + symbol

	s.mu.Lock()
	if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		s.mu.Unlock()
		return entry.data.(*FundingRateData), nil
	}
	s.mu.Unlock()

	url := "https://fapi.binance.com/fapi/v1/fundingRate?symbol=" + symbol + "&limit=10"
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fundingHistory []struct {
		FundingRate float64
		FundingTime int64
	}

	if err := json.Unmarshal(body, &fundingHistory); err != nil {
		return nil, err
	}

	if len(fundingHistory) == 0 {
		return nil, err
	}

	var sum float64
	for _, f := range fundingHistory {
		sum += f.FundingRate
	}
	avgRate := sum / float64(len(fundingHistory))

	latestRate := fundingHistory[len(fundingHistory)-1].FundingRate

	var trend string
	if latestRate > avgRate*1.2 {
		trend = "bullish"
	} else if latestRate < avgRate*0.8 {
		trend = "bearish"
	} else {
		trend = "neutral"
	}

	data := &FundingRateData{
		Symbol:         symbol,
		FundingRate:    latestRate,
		FundingRateAvg: avgRate,
		PredictedRate:  avgRate * 1.01,
		Timestamp:      time.Now(),
		Trend:          trend,
	}

	s.mu.Lock()
	s.cache[cacheKey] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(time.Duration(s.config.CacheDuration) * time.Second),
	}
	s.mu.Unlock()

	return data, nil
}

// GetSocialVolume retrieves social volume data from CoinGecko
func (s *SentimentAnalyzer) GetSocialVolume(symbol string) (*SocialVolumeData, error) {
	cacheKey := "social_" + symbol

	s.mu.Lock()
	if entry, ok := s.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		s.mu.Unlock()
		return entry.data.(*SocialVolumeData), nil
	}
	s.mu.Unlock()

	coinID := s.symbolToCoinGeckoID(symbol)
	url := "https://api.coingecko.com/api/v3/coins/" + coinID + "?localization=false&community_data=true"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.config.CoinGeckoAPIKey != "" {
		req.Header.Set("x-cg-api-key", s.config.CoinGeckoAPIKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var coinData struct {
		CommunityData struct {
			TwitterFollowers  int
			RedditSubscribers int
			TelegramUsers     int
		}
		PublicInterestStats struct {
			AlexaRank int
		}
	}

	if err := json.Unmarshal(body, &coinData); err != nil {
		return nil, err
	}

	totalVolume := int64(coinData.CommunityData.TwitterFollowers +
		coinData.CommunityData.RedditSubscribers +
		coinData.CommunityData.TelegramUsers)

	spikeDetected := totalVolume > 100000

	data := &SocialVolumeData{
		Symbol:        symbol,
		Volume:        totalVolume,
		Sentiment:     0.5,
		SpikeDetected: spikeDetected,
	}

	s.mu.Lock()
	s.cache[cacheKey] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(time.Duration(s.config.CacheDuration) * time.Second),
	}
	s.mu.Unlock()

	return data, nil
}

// GetTrendingSentiment returns sentiment based on trending coins
func (s *SentimentAnalyzer) GetTrendingSentiment() (float64, error) {
	s.mu.Lock()
	if entry, ok := s.cache["trending"]; ok && time.Now().Before(entry.expiresAt) {
		s.mu.Unlock()
		return entry.data.(float64), nil
	}
	s.mu.Unlock()

	url := "https://api.coingecko.com/api/v3/search/trending"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 50, err
	}

	if s.config.CoinGeckoAPIKey != "" {
		req.Header.Set("x-cg-api-key", s.config.CoinGeckoAPIKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 50, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 50, err
	}

	var trending CoinGeckoTrending
	if err := json.Unmarshal(body, &trending); err != nil {
		return 50, err
	}

	var totalScore float64
	count := 0
	for _, coin := range trending.Coins {
		if coin.Item.Score > 0 {
			totalScore += coin.Item.Score
			count++
		}
	}

	score := 50.0
	if count > 0 {
		avgScore := totalScore / float64(count)
		score = avgScore * 100
		if score > 100 {
			score = 100
		}
		if score < 0 {
			score = 0
		}
	}

	s.mu.Lock()
	s.cache["trending"] = cacheEntry{
		data:      score,
		expiresAt: time.Now().Add(time.Duration(s.config.CacheDuration) * time.Second),
	}
	s.mu.Unlock()

	return score, nil
}

// Helper functions

func (s *SentimentAnalyzer) fundingRateToScore(data *FundingRateData) float64 {
	score := 50.0

	if data.FundingRate > 0.0001 {
		score = 50 + (data.FundingRate-0.0001)*100000
		if score > 80 {
			score = 80
		}
	} else if data.FundingRate < 0 {
		score = 50 + data.FundingRate*100000
		if score < 20 {
			score = 20
		}
	}

	return score
}

func (s *SentimentAnalyzer) socialVolumeToScore(data *SocialVolumeData) float64 {
	score := 50.0

	if data.Volume > 100000 {
		score = 60
	}
	if data.Volume > 500000 {
		score = 70
	}
	if data.SpikeDetected {
		score += 10
	}

	return score
}

func (s *SentimentAnalyzer) getSocialLabel(data *SocialVolumeData) string {
	if data.SpikeDetected {
		return "High Activity"
	}
	if data.Volume > 500000 {
		return "Active"
	}
	if data.Volume > 100000 {
		return "Normal"
	}
	return "Low Activity"
}

func (s *SentimentAnalyzer) getTrendingLabel(score float64) string {
	if score > 70 {
		return "Strong Bullish"
	}
	if score > 55 {
		return "Bullish"
	}
	if score > 45 {
		return "Neutral"
	}
	if score > 30 {
		return "Bearish"
	}
	return "Strong Bearish"
}

func (s *SentimentAnalyzer) getOverallLabel(score float64) string {
	if score > 70 {
		return "Bullish"
	}
	if score > 55 {
		return "Slightly Bullish"
	}
	if score > 45 {
		return "Neutral"
	}
	if score > 30 {
		return "Slightly Bearish"
	}
	return "Bearish"
}

func (s *SentimentAnalyzer) calculateOverallScore(components map[string]float64) float64 {
	if len(components) == 0 {
		return 50.0
	}

	weights := map[string]float64{
		"fear_greed":    0.35,
		"funding_rate":  0.25,
		"social_volume": 0.20,
		"trending":      0.20,
	}

	var weightedSum float64
	var totalWeight float64

	for component, score := range components {
		weight, ok := weights[component]
		if !ok {
			weight = 0.1
		}
		weightedSum += score * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 50.0
	}

	return weightedSum / totalWeight
}

func (s *SentimentAnalyzer) calculateConfidence(sources []string) float64 {
	switch len(sources) {
	case 0:
		return 0.3
	case 1:
		return 0.5
	case 2:
		return 0.7
	case 3:
		return 0.85
	default:
		return 0.95
	}
}

func (s *SentimentAnalyzer) symbolToCoinGeckoID(symbol string) string {
	mappings := map[string]string{
		"BTCUSDT":   "bitcoin",
		"ETHUSDT":   "ethereum",
		"SOLUSDT":   "solana",
		"XRPUSDT":   "ripple",
		"ADAUSDT":   "cardano",
		"DOGEUSDT":  "dogecoin",
		"BNBUSDT":   "binancecoin",
		"AVAXUSDT":  "avalanche-2",
		"DOTUSDT":   "polkadot",
		"LINKUSDT":  "chainlink",
		"MATICUSDT": "matic-network",
		"UNIUSDT":   "uniswap",
	}

	if id, ok := mappings[symbol]; ok {
		return id
	}

	return strings.ToLower(symbol[:len(symbol)-4])
}

// ClearCache clears the sentiment cache
func (s *SentimentAnalyzer) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]cacheEntry)
}

// GetCacheStats returns cache statistics
func (s *SentimentAnalyzer) GetCacheStats() (size int, oldest time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	size = len(s.cache)
	var minTime time.Time
	for _, entry := range s.cache {
		if minTime.IsZero() || entry.expiresAt.Before(minTime) {
			minTime = entry.expiresAt
		}
	}
	return size, minTime
}
