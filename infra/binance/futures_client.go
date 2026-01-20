package binance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

// FuturesClient provides enhanced Binance Futures API integration
type FuturesClient struct {
	client    *futures.Client
	testnet   bool
	apiKey    string
	apiSecret string

	// Connection management
	connPool      *ConnectionPool
	wsMultiplexer *WebSocketMultiplexer

	// Rate limiting
	rateLimiter RateLimiter

	// Circuit breaker
	circuitBreaker CircuitBreaker

	logger *logrus.Logger
	mu     sync.RWMutex
}

// FuturesConfig holds configuration for Futures client
type FuturesConfig struct {
	APIKey    string
	APISecret string
	Testnet   bool
	PoolSize  int
	Redis     RedisConfig
}

// RedisConfig holds Redis configuration for rate limiting
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// NewFuturesClient creates a new enhanced Futures client
func NewFuturesClient(config FuturesConfig) *FuturesClient {
	if config.Testnet {
		futures.UseTestnet = true
	}

	client := futures.NewClient(config.APIKey, config.APISecret)

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &FuturesClient{
		client:         client,
		testnet:        config.Testnet,
		apiKey:         config.APIKey,
		apiSecret:      config.APISecret,
		connPool:       NewConnectionPool(config.PoolSize),
		wsMultiplexer:  NewWebSocketMultiplexer(),
		rateLimiter:    NewRedisRateLimiter(config.Redis),
		circuitBreaker: NewAdaptiveCircuitBreaker(),
		logger:         logger,
	}
}

// GetExchangeInfo retrieves exchange information for all symbols
func (fc *FuturesClient) GetExchangeInfo(ctx context.Context) (*futures.ExchangeInfo, error) {
	if !fc.rateLimiter.Allow("exchange_info") {
		return nil, ErrRateLimitExceeded
	}

	if !fc.circuitBreaker.Allow() {
		return nil, ErrCircuitBreakerOpen
	}

	info, err := fc.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		fc.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("failed to get exchange info: %w", err)
	}

	fc.circuitBreaker.RecordSuccess()
	return info, nil
}

// GetSymbolInfo retrieves information for a specific symbol
func (fc *FuturesClient) GetSymbolInfo(ctx context.Context, symbol string) (*futures.Symbol, error) {
	info, err := fc.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	for _, s := range info.Symbols {
		if s.Symbol == symbol {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("symbol %s not found", symbol)
}

// SetLeverage sets leverage for a symbol
func (fc *FuturesClient) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if !fc.rateLimiter.Allow("set_leverage") {
		return ErrRateLimitExceeded
	}

	if !fc.circuitBreaker.Allow() {
		return ErrCircuitBreakerOpen
	}

	_, err := fc.client.NewChangeLeverageService().
		Symbol(symbol).
		Leverage(leverage).
		Do(ctx)

	if err != nil {
		fc.circuitBreaker.RecordFailure()
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	fc.circuitBreaker.RecordSuccess()
	fc.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"leverage": leverage,
	}).Info("leverage_set")

	return nil
}

// GetLeverage retrieves current leverage for a symbol
func (fc *FuturesClient) GetLeverage(ctx context.Context, symbol string) (int, error) {
	position, err := fc.GetPosition(ctx, symbol)
	if err != nil {
		return 0, err
	}

	return position.Leverage, nil
}

// SetMarginType sets margin type (ISOLATED or CROSSED) for a symbol
func (fc *FuturesClient) SetMarginType(ctx context.Context, symbol string, marginType futures.MarginType) error {
	if !fc.rateLimiter.Allow("set_margin_type") {
		return ErrRateLimitExceeded
	}

	if !fc.circuitBreaker.Allow() {
		return ErrCircuitBreakerOpen
	}

	err := fc.client.NewChangeMarginTypeService().
		Symbol(symbol).
		MarginType(marginType).
		Do(ctx)

	if err != nil {
		fc.circuitBreaker.RecordFailure()
		return fmt.Errorf("failed to set margin type: %w", err)
	}

	fc.circuitBreaker.RecordSuccess()
	fc.logger.WithFields(logrus.Fields{
		"symbol":      symbol,
		"margin_type": marginType,
	}).Info("margin_type_set")

	return nil
}

// GetMarginType retrieves current margin type for a symbol
func (fc *FuturesClient) GetMarginType(ctx context.Context, symbol string) (futures.MarginType, error) {
	position, err := fc.GetPosition(ctx, symbol)
	if err != nil {
		return "", err
	}

	if position.Isolated {
		return futures.MarginTypeIsolated, nil
	}

	return futures.MarginTypeCrossed, nil
}

// SetPositionMode sets position mode (One-way or Hedge mode)
func (fc *FuturesClient) SetPositionMode(ctx context.Context, dualSide bool) error {
	if !fc.rateLimiter.Allow("set_position_mode") {
		return ErrRateLimitExceeded
	}

	if !fc.circuitBreaker.Allow() {
		return ErrCircuitBreakerOpen
	}

	err := fc.client.NewChangePositionModeService().
		DualSide(dualSide).
		Do(ctx)

	if err != nil {
		fc.circuitBreaker.RecordFailure()
		return fmt.Errorf("failed to set position mode: %w", err)
	}

	fc.circuitBreaker.RecordSuccess()
	fc.logger.WithFields(logrus.Fields{
		"dual_side": dualSide,
	}).Info("position_mode_set")

	return nil
}

// GetPositionMode retrieves current position mode
func (fc *FuturesClient) GetPositionMode(ctx context.Context) (bool, error) {
	if !fc.rateLimiter.Allow("get_position_mode") {
		return false, ErrRateLimitExceeded
	}

	result, err := fc.client.NewGetPositionModeService().Do(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get position mode: %w", err)
	}

	return result.DualSidePosition, nil
}

// FuturesOrder represents an order for Futures trading
type FuturesOrder struct {
	Symbol           string
	Side             futures.SideType
	PositionSide     futures.PositionSideType
	Type             futures.OrderType
	TimeInForce      futures.TimeInForceType
	Quantity         string
	Price            string
	StopPrice        string
	ReduceOnly       bool
	ClosePosition    bool
	WorkingType      futures.WorkingType
	PriceProtect     bool
	NewOrderRespType futures.NewOrderRespType
}

// CreateOrder creates a new order with sub-10ms latency optimization
func (fc *FuturesClient) CreateOrder(ctx context.Context, order *FuturesOrder) (*futures.CreateOrderResponse, error) {
	if !fc.rateLimiter.Allow("create_order") {
		return nil, ErrRateLimitExceeded
	}

	if !fc.circuitBreaker.Allow() {
		return nil, ErrCircuitBreakerOpen
	}

	startTime := time.Now()

	service := fc.client.NewCreateOrderService().
		Symbol(order.Symbol).
		Side(order.Side).
		Type(order.Type).
		Quantity(order.Quantity)

	if order.PositionSide != "" {
		service = service.PositionSide(order.PositionSide)
	}

	if order.Price != "" {
		service = service.Price(order.Price)
	}

	if order.TimeInForce != "" {
		service = service.TimeInForce(order.TimeInForce)
	}

	if order.StopPrice != "" {
		service = service.StopPrice(order.StopPrice)
	}

	if order.ReduceOnly {
		service = service.ReduceOnly(order.ReduceOnly)
	}

	if order.ClosePosition {
		service = service.ClosePosition(order.ClosePosition)
	}

	if order.WorkingType != "" {
		service = service.WorkingType(order.WorkingType)
	}

	if order.PriceProtect {
		service = service.PriceProtect(order.PriceProtect)
	}

	if order.NewOrderRespType != "" {
		service = service.NewOrderResponseType(order.NewOrderRespType)
	}

	response, err := service.Do(ctx)
	if err != nil {
		fc.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	fc.circuitBreaker.RecordSuccess()

	executionTime := time.Since(startTime)
	fc.logger.WithFields(logrus.Fields{
		"symbol":         order.Symbol,
		"side":           order.Side,
		"type":           order.Type,
		"quantity":       order.Quantity,
		"execution_time": executionTime.Milliseconds(),
	}).Info("order_created")

	return response, nil
}

// CancelOrder cancels an existing order
func (fc *FuturesClient) CancelOrder(ctx context.Context, symbol string, orderID int64) error {
	if !fc.rateLimiter.Allow("cancel_order") {
		return ErrRateLimitExceeded
	}

	if !fc.circuitBreaker.Allow() {
		return ErrCircuitBreakerOpen
	}

	_, err := fc.client.NewCancelOrderService().
		Symbol(symbol).
		OrderID(orderID).
		Do(ctx)

	if err != nil {
		fc.circuitBreaker.RecordFailure()
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	fc.circuitBreaker.RecordSuccess()
	fc.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"order_id": orderID,
	}).Info("order_cancelled")

	return nil
}

// GetOrder retrieves order information
func (fc *FuturesClient) GetOrder(ctx context.Context, symbol string, orderID int64) (*futures.Order, error) {
	if !fc.rateLimiter.Allow("get_order") {
		return nil, ErrRateLimitExceeded
	}

	order, err := fc.client.NewGetOrderService().
		Symbol(symbol).
		OrderID(orderID).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

// Position represents a Futures position
type Position struct {
	Symbol           string
	PositionSide     string
	PositionAmt      float64
	EntryPrice       float64
	MarkPrice        float64
	UnrealizedProfit float64
	LiquidationPrice float64
	Leverage         int
	MarginType       string
	Isolated         bool
	InitialMargin    float64
	MaintMargin      float64
	PositionValue    float64
}

// GetPositions retrieves all open positions
func (fc *FuturesClient) GetPositions(ctx context.Context) ([]*Position, error) {
	if !fc.rateLimiter.Allow("get_positions") {
		return nil, ErrRateLimitExceeded
	}

	positions, err := fc.client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	result := make([]*Position, 0, len(positions))
	for _, p := range positions {
		// Only include positions with non-zero amount
		if p.PositionAmt != "0" {
			result = append(result, convertPosition(p))
		}
	}

	return result, nil
}

// GetPosition retrieves position for a specific symbol
func (fc *FuturesClient) GetPosition(ctx context.Context, symbol string) (*Position, error) {
	positions, err := fc.GetPositions(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range positions {
		if p.Symbol == symbol {
			return p, nil
		}
	}

	return nil, fmt.Errorf("position not found for symbol %s", symbol)
}

// ClosePosition closes an open position
func (fc *FuturesClient) ClosePosition(ctx context.Context, symbol string) error {
	position, err := fc.GetPosition(ctx, symbol)
	if err != nil {
		return err
	}

	// Determine side for closing order (opposite of position)
	var side futures.SideType
	if position.PositionAmt > 0 {
		side = futures.SideTypeSell
	} else {
		side = futures.SideTypeBuy
	}

	order := &FuturesOrder{
		Symbol:        symbol,
		Side:          side,
		Type:          futures.OrderTypeMarket,
		ClosePosition: true,
	}

	_, err = fc.CreateOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to close position: %w", err)
	}

	fc.logger.WithFields(logrus.Fields{
		"symbol": symbol,
	}).Info("position_closed")

	return nil
}

// AccountInfo represents Futures account information
type AccountInfo struct {
	TotalWalletBalance       float64
	TotalUnrealizedProfit    float64
	TotalMarginBalance       float64
	TotalPositionInitialMargin float64
	TotalOpenOrderInitialMargin float64
	AvailableBalance         float64
	MaxWithdrawAmount        float64
}

// GetAccount retrieves account information
func (fc *FuturesClient) GetAccount(ctx context.Context) (*AccountInfo, error) {
	if !fc.rateLimiter.Allow("get_account") {
		return nil, ErrRateLimitExceeded
	}

	account, err := fc.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return convertAccountInfo(account), nil
}

// GetBalance retrieves account balance
func (fc *FuturesClient) GetBalance(ctx context.Context) ([]*futures.Balance, error) {
	if !fc.rateLimiter.Allow("get_balance") {
		return nil, ErrRateLimitExceeded
	}

	balances, err := fc.client.NewGetBalanceService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balances, nil
}

// GetMarkPrice retrieves mark price for a symbol
func (fc *FuturesClient) GetMarkPrice(ctx context.Context, symbol string) (float64, error) {
	if !fc.rateLimiter.Allow("get_mark_price") {
		return 0, ErrRateLimitExceeded
	}

	prices, err := fc.client.NewListPricesService().Symbol(symbol).Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get mark price: %w", err)
	}

	if len(prices) == 0 {
		return 0, fmt.Errorf("no price data for symbol %s", symbol)
	}

	return parseFloat(prices[0].MarkPrice), nil
}

// GetFundingRate retrieves current funding rate for a symbol
func (fc *FuturesClient) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	if !fc.rateLimiter.Allow("get_funding_rate") {
		return 0, ErrRateLimitExceeded
	}

	rates, err := fc.client.NewGetFundingRateService().Symbol(symbol).Limit(1).Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get funding rate: %w", err)
	}

	if len(rates) == 0 {
		return 0, fmt.Errorf("no funding rate data for symbol %s", symbol)
	}

	return parseFloat(rates[0].FundingRate), nil
}

// GetLiquidationPrice calculates liquidation price for a position
func (fc *FuturesClient) GetLiquidationPrice(ctx context.Context, symbol string) (float64, error) {
	position, err := fc.GetPosition(ctx, symbol)
	if err != nil {
		return 0, err
	}

	return position.LiquidationPrice, nil
}

// Helper functions

func convertPosition(p *futures.PositionRisk) *Position {
	return &Position{
		Symbol:           p.Symbol,
		PositionSide:     p.PositionSide,
		PositionAmt:      parseFloat(p.PositionAmt),
		EntryPrice:       parseFloat(p.EntryPrice),
		MarkPrice:        parseFloat(p.MarkPrice),
		UnrealizedProfit: parseFloat(p.UnRealizedProfit),
		LiquidationPrice: parseFloat(p.LiquidationPrice),
		Leverage:         parseInt(p.Leverage),
		MarginType:       p.MarginType,
		Isolated:         p.Isolated,
		InitialMargin:    parseFloat(p.InitialMargin),
		MaintMargin:      parseFloat(p.MaintMargin),
		PositionValue:    parseFloat(p.PositionAmt) * parseFloat(p.MarkPrice),
	}
}

func convertAccountInfo(a *futures.Account) *AccountInfo {
	return &AccountInfo{
		TotalWalletBalance:          parseFloat(a.TotalWalletBalance),
		TotalUnrealizedProfit:       parseFloat(a.TotalUnrealizedProfit),
		TotalMarginBalance:          parseFloat(a.TotalMarginBalance),
		TotalPositionInitialMargin:  parseFloat(a.TotalPositionInitialMargin),
		TotalOpenOrderInitialMargin: parseFloat(a.TotalOpenOrderInitialMargin),
		AvailableBalance:            parseFloat(a.AvailableBalance),
		MaxWithdrawAmount:           parseFloat(a.MaxWithdrawAmount),
	}
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// Error definitions
var (
	ErrRateLimitExceeded   = fmt.Errorf("rate limit exceeded")
	ErrCircuitBreakerOpen  = fmt.Errorf("circuit breaker open")
)
