// Package market defines types for Binance Futures market data
package market

import (
	"fmt"
	"strconv"
	"time"
)

// ============================================================================
// Binance API Response Types
// ============================================================================

// Ticker24hr represents the response from GET /fapi/v1/ticker/24hr
// https://developers.binance.com/docs/derivatives/usds-margined-futures/market-data/rest-api/24hr-Ticker-Price-Change-Statistics
type Ticker24hr struct {
	Symbol             string  `json:"symbol"`
	PriceChange        string  `json:"priceChange"`
	PriceChangePercent string  `json:"priceChangePercent"`
	WeightedAvgPrice   string  `json:"weightedAvgPrice"`
	LastPrice          string  `json:"lastPrice"`
	LastQty            string  `json:"lastQty"`
	OpenPrice          string  `json:"openPrice"`
	HighPrice          string  `json:"highPrice"`
	LowPrice           string  `json:"lowPrice"`
	Volume             string  `json:"volume"`      // Base asset volume
	QuoteVolume        string  `json:"quoteVolume"` // Quote asset volume (USDT)
	OpenTime           int64   `json:"openTime"`
	CloseTime          int64   `json:"closeTime"`
	FirstID            int64   `json:"firstId"`
	LastID             int64   `json:"lastId"`
	Count              int64   `json:"count"` // Trade count
}

// OrderBook represents the response from GET /fapi/v1/depth
type OrderBook struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"` // [price, quantity]
	Asks         [][]string `json:"asks"` // [price, quantity]
}

// Kline represents a single kline/candlestick from GET /fapi/v1/klines
type Kline struct {
	OpenTime                 int64
	Open                     string
	High                     string
	Low                      string
	Close                    string
	Volume                   string
	CloseTime                int64
	QuoteAssetVolume         string
	NumberOfTrades           int64
	TakerBuyBaseAssetVolume  string
	TakerBuyQuoteAssetVolume string
}

// FundingRate represents the response from GET /fapi/v1/fundingRate
type FundingRate struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"fundingRate"`
	FundingTime int64  `json:"fundingTime"`
	MarkPrice   string `json:"markPrice"`
}

// AccountInfo represents account information from GET /fapi/v2/account
type AccountInfo struct {
	TotalWalletBalance    string `json:"totalWalletBalance"`
	TotalUnrealizedProfit string `json:"totalUnrealizedProfit"`
	TotalMarginBalance    string `json:"totalMarginBalance"`
	AvailableBalance      string `json:"availableBalance"`
	MaxWithdrawAmount     string `json:"maxWithdrawAmount"`
}

// PositionRisk represents position from GET /fapi/v2/positionRisk
type PositionRisk struct {
	Symbol           string `json:"symbol"`
	PositionAmt      string `json:"positionAmt"`
	EntryPrice       string `json:"entryPrice"`
	MarkPrice        string `json:"markPrice"`
	UnRealizedProfit string `json:"unRealizedProfit"`
	LiquidationPrice string `json:"liquidationPrice"`
	Leverage         string `json:"leverage"`
	MarginType       string `json:"marginType"`
	PositionSide     string `json:"positionSide"` // BOTH, LONG, SHORT
	Notional         string `json:"notional"`
}

// ============================================================================
// Parsed Types (with proper Go types, validated)
// ============================================================================

// ParsedTicker24hr is the validated, parsed version of Ticker24hr
type ParsedTicker24hr struct {
	Symbol             string
	PriceChange        float64
	PriceChangePercent float64
	WeightedAvgPrice   float64
	LastPrice          float64
	LastQty            float64
	OpenPrice          float64
	HighPrice          float64
	LowPrice           float64
	Volume             float64 // Base asset volume
	QuoteVolume        float64 // Quote asset volume (USDT)
	OpenTime           time.Time
	CloseTime          time.Time
	TradeCount         int64
	
	// Computed fields
	Spread             float64 // (Ask - Bid) / Ask * 100
	VolatilityPercent  float64 // (High - Low) / Low * 100
	IsValid            bool
	ParseErrors        []string
}

// ParsedOrderBook is the validated order book
type ParsedOrderBook struct {
	Symbol       string
	LastUpdateID int64
	BestBid      float64
	BestAsk      float64
	BidDepth     float64 // Total bid volume in top 20 levels
	AskDepth     float64 // Total ask volume in top 20 levels
	Spread       float64 // (BestAsk - BestBid) / BestAsk * 100
	SpreadAbs    float64 // BestAsk - BestBid
	MidPrice     float64 // (BestBid + BestAsk) / 2
	Bids         []PriceLevel
	Asks         []PriceLevel
	IsValid      bool
	ParseErrors  []string
}

// PriceLevel represents a price/quantity pair in the order book
type PriceLevel struct {
	Price    float64
	Quantity float64
}

// ParsedAccountInfo is the validated account info
type ParsedAccountInfo struct {
	TotalWalletBalance    float64
	TotalUnrealizedProfit float64
	TotalMarginBalance    float64
	AvailableBalance      float64
	MaxWithdrawAmount     float64
	MarginRatio           float64 // Used margin / Total margin
	IsValid               bool
	ParseErrors           []string
}

// ParsedPosition is the validated position
type ParsedPosition struct {
	Symbol           string
	PositionAmt      float64
	EntryPrice       float64
	MarkPrice        float64
	UnrealizedProfit float64
	LiquidationPrice float64
	Leverage         int
	MarginType       string
	PositionSide     string
	Notional         float64
	PnLPercent       float64 // Unrealized PnL as percentage of entry
	IsLong           bool
	IsShort          bool
	IsOpen           bool
	IsValid          bool
	ParseErrors      []string
}

// ============================================================================
// Parsing Functions
// ============================================================================

// ParseFloat safely parses a string to float64
func ParseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// MustParseFloat parses or returns 0 on error
func MustParseFloat(s string) float64 {
	v, _ := ParseFloat(s)
	return v
}

// ParseTicker24hr parses and validates a Ticker24hr response
func ParseTicker24hr(t *Ticker24hr) *ParsedTicker24hr {
	p := &ParsedTicker24hr{
		Symbol:  t.Symbol,
		IsValid: true,
	}
	
	var err error
	
	// Parse all numeric fields
	p.PriceChange, err = ParseFloat(t.PriceChange)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("priceChange: %v", err))
	}
	
	p.PriceChangePercent, err = ParseFloat(t.PriceChangePercent)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("priceChangePercent: %v", err))
	}
	
	p.WeightedAvgPrice, err = ParseFloat(t.WeightedAvgPrice)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("weightedAvgPrice: %v", err))
	}
	
	p.LastPrice, err = ParseFloat(t.LastPrice)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("lastPrice: %v", err))
	}
	
	p.LastQty, err = ParseFloat(t.LastQty)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("lastQty: %v", err))
	}
	
	p.OpenPrice, err = ParseFloat(t.OpenPrice)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("openPrice: %v", err))
	}
	
	p.HighPrice, err = ParseFloat(t.HighPrice)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("highPrice: %v", err))
	}
	
	p.LowPrice, err = ParseFloat(t.LowPrice)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("lowPrice: %v", err))
	}
	
	p.Volume, err = ParseFloat(t.Volume)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("volume: %v", err))
	}
	
	p.QuoteVolume, err = ParseFloat(t.QuoteVolume)
	if err != nil {
		p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("quoteVolume: %v", err))
	}
	
	// Parse timestamps
	p.OpenTime = time.UnixMilli(t.OpenTime)
	p.CloseTime = time.UnixMilli(t.CloseTime)
	p.TradeCount = t.Count
	
	// Compute derived fields
	if p.LowPrice > 0 {
		p.VolatilityPercent = (p.HighPrice - p.LowPrice) / p.LowPrice * 100
	}
	
	// Validate
	if len(p.ParseErrors) > 0 {
		p.IsValid = false
	}
	
	// Basic sanity checks
	if p.LastPrice <= 0 {
		p.IsValid = false
		p.ParseErrors = append(p.ParseErrors, "lastPrice must be > 0")
	}
	
	if p.QuoteVolume < 0 {
		p.IsValid = false
		p.ParseErrors = append(p.ParseErrors, "quoteVolume must be >= 0")
	}
	
	return p
}

// ParseOrderBook parses and validates an OrderBook response
func ParseOrderBook(symbol string, ob *OrderBook) *ParsedOrderBook {
	p := &ParsedOrderBook{
		Symbol:       symbol,
		LastUpdateID: ob.LastUpdateID,
		IsValid:      true,
	}
	
	// Parse bids
	for i, bid := range ob.Bids {
		if len(bid) < 2 {
			p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("bid[%d]: insufficient data", i))
			continue
		}
		price, _ := ParseFloat(bid[0])
		qty, _ := ParseFloat(bid[1])
		p.Bids = append(p.Bids, PriceLevel{Price: price, Quantity: qty})
		
		if i == 0 {
			p.BestBid = price
		}
		if i < 20 {
			p.BidDepth += qty * price // Volume in quote currency
		}
	}
	
	// Parse asks
	for i, ask := range ob.Asks {
		if len(ask) < 2 {
			p.ParseErrors = append(p.ParseErrors, fmt.Sprintf("ask[%d]: insufficient data", i))
			continue
		}
		price, _ := ParseFloat(ask[0])
		qty, _ := ParseFloat(ask[1])
		p.Asks = append(p.Asks, PriceLevel{Price: price, Quantity: qty})
		
		if i == 0 {
			p.BestAsk = price
		}
		if i < 20 {
			p.AskDepth += qty * price // Volume in quote currency
		}
	}
	
	// Compute spread
	if p.BestAsk > 0 && p.BestBid > 0 {
		p.SpreadAbs = p.BestAsk - p.BestBid
		p.Spread = p.SpreadAbs / p.BestAsk * 100
		p.MidPrice = (p.BestAsk + p.BestBid) / 2
	}
	
	// Validate
	if len(p.ParseErrors) > 0 {
		p.IsValid = false
	}
	
	if p.BestBid <= 0 || p.BestAsk <= 0 {
		p.IsValid = false
		p.ParseErrors = append(p.ParseErrors, "invalid bid/ask prices")
	}
	
	if p.BestBid >= p.BestAsk {
		p.IsValid = false
		p.ParseErrors = append(p.ParseErrors, "bid >= ask (crossed book)")
	}
	
	return p
}

// ParseAccountInfo parses and validates AccountInfo
func ParseAccountInfo(a *AccountInfo) *ParsedAccountInfo {
	p := &ParsedAccountInfo{IsValid: true}
	
	p.TotalWalletBalance = MustParseFloat(a.TotalWalletBalance)
	p.TotalUnrealizedProfit = MustParseFloat(a.TotalUnrealizedProfit)
	p.TotalMarginBalance = MustParseFloat(a.TotalMarginBalance)
	p.AvailableBalance = MustParseFloat(a.AvailableBalance)
	p.MaxWithdrawAmount = MustParseFloat(a.MaxWithdrawAmount)
	
	// Calculate margin ratio
	if p.TotalMarginBalance > 0 {
		usedMargin := p.TotalMarginBalance - p.AvailableBalance
		p.MarginRatio = usedMargin / p.TotalMarginBalance
	}
	
	// Validate
	if p.TotalWalletBalance < 0 {
		p.IsValid = false
		p.ParseErrors = append(p.ParseErrors, "totalWalletBalance < 0")
	}
	
	return p
}

// ParsePosition parses and validates PositionRisk
func ParsePosition(pr *PositionRisk) *ParsedPosition {
	p := &ParsedPosition{
		Symbol:     pr.Symbol,
		MarginType: pr.MarginType,
		PositionSide: pr.PositionSide,
		IsValid:    true,
	}
	
	p.PositionAmt = MustParseFloat(pr.PositionAmt)
	p.EntryPrice = MustParseFloat(pr.EntryPrice)
	p.MarkPrice = MustParseFloat(pr.MarkPrice)
	p.UnrealizedProfit = MustParseFloat(pr.UnRealizedProfit)
	p.LiquidationPrice = MustParseFloat(pr.LiquidationPrice)
	p.Notional = MustParseFloat(pr.Notional)
	
	leverage, _ := strconv.ParseInt(pr.Leverage, 10, 64)
	p.Leverage = int(leverage)
	
	// Determine position direction
	p.IsLong = p.PositionAmt > 0
	p.IsShort = p.PositionAmt < 0
	p.IsOpen = p.PositionAmt != 0
	
	// Calculate PnL percentage
	if p.EntryPrice > 0 && p.IsOpen {
		if p.IsLong {
			p.PnLPercent = (p.MarkPrice - p.EntryPrice) / p.EntryPrice * 100
		} else {
			p.PnLPercent = (p.EntryPrice - p.MarkPrice) / p.EntryPrice * 100
		}
	}
	
	return p
}
