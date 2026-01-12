// Package market - Data validation utilities
package market

import (
	"fmt"
	"math"
)

// ============================================================================
// Validation Rules
// ============================================================================

// ValidationRule defines a validation check
type ValidationRule struct {
	Name        string
	Description string
	Check       func(interface{}) error
}

// ValidationResult holds the result of validation
type ValidationResult struct {
	IsValid     bool
	Errors      []string
	Warnings    []string
	FieldsValid map[string]bool
}

// ============================================================================
// Ticker Validation
// ============================================================================

// TickerValidationConfig defines thresholds for ticker validation
type TickerValidationConfig struct {
	MinQuoteVolume      float64 // Minimum 24h quote volume (USDT)
	MaxPriceChange      float64 // Maximum allowed price change %
	MinPriceChange      float64 // Minimum price change % for top movers
	MaxSpread           float64 // Maximum spread %
	MinTradeCount       int64   // Minimum trade count
}

// DefaultTickerValidation returns default validation config
func DefaultTickerValidation() TickerValidationConfig {
	return TickerValidationConfig{
		MinQuoteVolume: 10_000_000, // $10M
		MaxPriceChange: 50,         // 50% max
		MinPriceChange: 3,          // 3% min for top movers
		MaxSpread:      0.1,        // 0.1%
		MinTradeCount:  1000,       // Min 1000 trades
	}
}

// ValidateTicker validates a parsed ticker against rules
func ValidateTicker(t *ParsedTicker24hr, cfg TickerValidationConfig) *ValidationResult {
	result := &ValidationResult{
		IsValid:     true,
		FieldsValid: make(map[string]bool),
	}

	// Check quote volume
	if t.QuoteVolume < cfg.MinQuoteVolume {
		result.Errors = append(result.Errors, 
			fmt.Sprintf("quoteVolume %.2f < min %.2f", t.QuoteVolume, cfg.MinQuoteVolume))
		result.FieldsValid["quoteVolume"] = false
	} else {
		result.FieldsValid["quoteVolume"] = true
	}

	// Check price change bounds
	absChange := math.Abs(t.PriceChangePercent)
	if absChange > cfg.MaxPriceChange {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("priceChangePercent %.2f%% exceeds max %.2f%%", absChange, cfg.MaxPriceChange))
		result.FieldsValid["priceChangePercent"] = false
	} else if absChange < cfg.MinPriceChange {
		result.Errors = append(result.Errors,
			fmt.Sprintf("priceChangePercent %.2f%% below min %.2f%%", absChange, cfg.MinPriceChange))
		result.FieldsValid["priceChangePercent"] = false
	} else {
		result.FieldsValid["priceChangePercent"] = true
	}

	// Check trade count
	if t.TradeCount < cfg.MinTradeCount {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("tradeCount %d < min %d", t.TradeCount, cfg.MinTradeCount))
		result.FieldsValid["tradeCount"] = false
	} else {
		result.FieldsValid["tradeCount"] = true
	}

	// Check for valid prices
	if t.LastPrice <= 0 || t.HighPrice <= 0 || t.LowPrice <= 0 {
		result.Errors = append(result.Errors, "invalid price data (<=0)")
		result.FieldsValid["prices"] = false
	} else {
		result.FieldsValid["prices"] = true
	}

	// Check price consistency
	if t.HighPrice < t.LowPrice {
		result.Errors = append(result.Errors, "highPrice < lowPrice")
		result.FieldsValid["priceConsistency"] = false
	} else {
		result.FieldsValid["priceConsistency"] = true
	}

	// Set overall validity
	result.IsValid = len(result.Errors) == 0

	return result
}

// ============================================================================
// Order Book Validation
// ============================================================================

// OrderBookValidationConfig defines thresholds for order book validation
type OrderBookValidationConfig struct {
	MaxSpread      float64 // Maximum spread %
	MinBidDepth    float64 // Minimum bid depth (USDT)
	MinAskDepth    float64 // Minimum ask depth (USDT)
	MinLevels      int     // Minimum order book levels
}

// DefaultOrderBookValidation returns default config
func DefaultOrderBookValidation() OrderBookValidationConfig {
	return OrderBookValidationConfig{
		MaxSpread:   0.1,       // 0.1%
		MinBidDepth: 100_000,   // $100K
		MinAskDepth: 100_000,   // $100K
		MinLevels:   10,
	}
}

// ValidateOrderBook validates a parsed order book
func ValidateOrderBook(ob *ParsedOrderBook, cfg OrderBookValidationConfig) *ValidationResult {
	result := &ValidationResult{
		IsValid:     true,
		FieldsValid: make(map[string]bool),
	}

	// Check spread
	if ob.Spread > cfg.MaxSpread {
		result.Errors = append(result.Errors,
			fmt.Sprintf("spread %.4f%% > max %.4f%%", ob.Spread, cfg.MaxSpread))
		result.FieldsValid["spread"] = false
	} else {
		result.FieldsValid["spread"] = true
	}

	// Check bid depth
	if ob.BidDepth < cfg.MinBidDepth {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("bidDepth $%.2f < min $%.2f", ob.BidDepth, cfg.MinBidDepth))
		result.FieldsValid["bidDepth"] = false
	} else {
		result.FieldsValid["bidDepth"] = true
	}

	// Check ask depth
	if ob.AskDepth < cfg.MinAskDepth {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("askDepth $%.2f < min $%.2f", ob.AskDepth, cfg.MinAskDepth))
		result.FieldsValid["askDepth"] = false
	} else {
		result.FieldsValid["askDepth"] = true
	}

	// Check levels
	if len(ob.Bids) < cfg.MinLevels || len(ob.Asks) < cfg.MinLevels {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("insufficient levels: bids=%d, asks=%d (min=%d)", 
				len(ob.Bids), len(ob.Asks), cfg.MinLevels))
		result.FieldsValid["levels"] = false
	} else {
		result.FieldsValid["levels"] = true
	}

	// Check for crossed book
	if ob.BestBid >= ob.BestAsk {
		result.Errors = append(result.Errors, "crossed order book: bid >= ask")
		result.FieldsValid["notCrossed"] = false
	} else {
		result.FieldsValid["notCrossed"] = true
	}

	result.IsValid = len(result.Errors) == 0

	return result
}

// ============================================================================
// Position Validation
// ============================================================================

// PositionValidationConfig defines thresholds for position validation
type PositionValidationConfig struct {
	MaxLeverage       int     // Maximum allowed leverage
	MaxPositionSize   float64 // Maximum position size (USDT notional)
	MinLiquidationDist float64 // Minimum distance to liquidation %
}

// DefaultPositionValidation returns default config
func DefaultPositionValidation() PositionValidationConfig {
	return PositionValidationConfig{
		MaxLeverage:       50,
		MaxPositionSize:   100_000, // $100K
		MinLiquidationDist: 5,      // 5%
	}
}

// ValidatePosition validates a parsed position
func ValidatePosition(pos *ParsedPosition, cfg PositionValidationConfig) *ValidationResult {
	result := &ValidationResult{
		IsValid:     true,
		FieldsValid: make(map[string]bool),
	}

	if !pos.IsOpen {
		return result // No validation needed for closed positions
	}

	// Check leverage
	if pos.Leverage > cfg.MaxLeverage {
		result.Errors = append(result.Errors,
			fmt.Sprintf("leverage %d > max %d", pos.Leverage, cfg.MaxLeverage))
		result.FieldsValid["leverage"] = false
	} else {
		result.FieldsValid["leverage"] = true
	}

	// Check position size
	absNotional := math.Abs(pos.Notional)
	if absNotional > cfg.MaxPositionSize {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("notional $%.2f > max $%.2f", absNotional, cfg.MaxPositionSize))
		result.FieldsValid["positionSize"] = false
	} else {
		result.FieldsValid["positionSize"] = true
	}

	// Check liquidation distance
	if pos.LiquidationPrice > 0 && pos.MarkPrice > 0 {
		var liqDist float64
		if pos.IsLong {
			liqDist = (pos.MarkPrice - pos.LiquidationPrice) / pos.MarkPrice * 100
		} else {
			liqDist = (pos.LiquidationPrice - pos.MarkPrice) / pos.MarkPrice * 100
		}

		if liqDist < cfg.MinLiquidationDist {
			result.Errors = append(result.Errors,
				fmt.Sprintf("liquidation distance %.2f%% < min %.2f%%", liqDist, cfg.MinLiquidationDist))
			result.FieldsValid["liquidationDist"] = false
		} else {
			result.FieldsValid["liquidationDist"] = true
		}
	}

	result.IsValid = len(result.Errors) == 0

	return result
}

// ============================================================================
// Account Validation
// ============================================================================

// AccountValidationConfig defines thresholds for account validation
type AccountValidationConfig struct {
	MinBalance      float64 // Minimum balance to trade
	MaxMarginRatio  float64 // Maximum margin usage ratio
	MinAvailable    float64 // Minimum available balance
}

// DefaultAccountValidation returns default config
func DefaultAccountValidation() AccountValidationConfig {
	return AccountValidationConfig{
		MinBalance:     100,  // $100 minimum
		MaxMarginRatio: 0.8,  // 80% max margin usage
		MinAvailable:   50,   // $50 minimum available
	}
}

// ValidateAccount validates account info
func ValidateAccount(acc *ParsedAccountInfo, cfg AccountValidationConfig) *ValidationResult {
	result := &ValidationResult{
		IsValid:     true,
		FieldsValid: make(map[string]bool),
	}

	// Check minimum balance
	if acc.TotalWalletBalance < cfg.MinBalance {
		result.Errors = append(result.Errors,
			fmt.Sprintf("balance $%.2f < min $%.2f", acc.TotalWalletBalance, cfg.MinBalance))
		result.FieldsValid["minBalance"] = false
	} else {
		result.FieldsValid["minBalance"] = true
	}

	// Check margin ratio
	if acc.MarginRatio > cfg.MaxMarginRatio {
		result.Errors = append(result.Errors,
			fmt.Sprintf("marginRatio %.2f%% > max %.2f%%", acc.MarginRatio*100, cfg.MaxMarginRatio*100))
		result.FieldsValid["marginRatio"] = false
	} else {
		result.FieldsValid["marginRatio"] = true
	}

	// Check available balance
	if acc.AvailableBalance < cfg.MinAvailable {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("availableBalance $%.2f < min $%.2f", acc.AvailableBalance, cfg.MinAvailable))
		result.FieldsValid["availableBalance"] = false
	} else {
		result.FieldsValid["availableBalance"] = true
	}

	result.IsValid = len(result.Errors) == 0

	return result
}
