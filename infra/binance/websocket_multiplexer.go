package binance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

// WebSocketMultiplexer manages multiple WebSocket connections for real-time data
type WebSocketMultiplexer struct {
	connections map[string]*WebSocketConnection
	subscribers map[string][]chan interface{}
	mu          sync.RWMutex
	logger      *logrus.Logger
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	streamName string
	doneC      chan struct{}
	stopC      chan struct{}
	handler    futures.WsHandler
	errHandler futures.ErrHandler
}

// OrderBookUpdate represents an order book update
type OrderBookUpdate struct {
	Symbol       string
	Bids         [][2]string
	Asks         [][2]string
	LastUpdateID int64
	Timestamp    time.Time
}

// TradeUpdate represents a trade update
type TradeUpdate struct {
	Symbol    string
	Price     float64
	Quantity  float64
	Side      string
	Timestamp time.Time
}

// AccountUpdate represents an account update
type AccountUpdate struct {
	Balances  map[string]float64
	Positions map[string]*PositionUpdate
	Timestamp time.Time
}

// PositionUpdate represents a position update
type PositionUpdate struct {
	Symbol           string
	PositionAmt      float64
	EntryPrice       float64
	UnrealizedProfit float64
	MarginType       string
	Timestamp        time.Time
}

// MarkPriceUpdate represents a mark price update
type MarkPriceUpdate struct {
	Symbol      string
	MarkPrice   float64
	FundingRate float64
	Timestamp   time.Time
}

// NewWebSocketMultiplexer creates a new WebSocket multiplexer
func NewWebSocketMultiplexer() *WebSocketMultiplexer {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &WebSocketMultiplexer{
		connections: make(map[string]*WebSocketConnection),
		subscribers: make(map[string][]chan interface{}),
		logger:      logger,
		stopChan:    make(chan struct{}),
	}
}

// SubscribeOrderBook subscribes to order book updates for a symbol
func (wsm *WebSocketMultiplexer) SubscribeOrderBook(symbol string) (<-chan *OrderBookUpdate, error) {
	streamName := fmt.Sprintf("%s@depth", symbol)

	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	// Create channel for this subscriber
	updateChan := make(chan *OrderBookUpdate, 100)

	// Check if connection already exists
	if _, exists := wsm.connections[streamName]; !exists {
		// Create new WebSocket connection
		doneC, stopC, err := futures.WsDepthServe(symbol, wsm.createDepthHandler(streamName), wsm.createErrorHandler(streamName))
		if err != nil {
			return nil, fmt.Errorf("failed to create depth websocket: %w", err)
		}

		wsm.connections[streamName] = &WebSocketConnection{
			streamName: streamName,
			doneC:      doneC,
			stopC:      stopC,
		}

		wsm.logger.WithFields(logrus.Fields{
			"stream": streamName,
		}).Info("websocket_connected")
	}

	// Add subscriber
	if wsm.subscribers[streamName] == nil {
		wsm.subscribers[streamName] = make([]chan interface{}, 0)
	}
	wsm.subscribers[streamName] = append(wsm.subscribers[streamName], updateChan)

	return updateChan, nil
}

// SubscribeTrades subscribes to trade updates for a symbol
func (wsm *WebSocketMultiplexer) SubscribeTrades(symbol string) (<-chan *TradeUpdate, error) {
	streamName := fmt.Sprintf("%s@aggTrade", symbol)

	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	updateChan := make(chan *TradeUpdate, 100)

	if _, exists := wsm.connections[streamName]; !exists {
		doneC, stopC, err := futures.WsAggTradeServe(symbol, wsm.createAggTradeHandler(streamName), wsm.createErrorHandler(streamName))
		if err != nil {
			return nil, fmt.Errorf("failed to create trade websocket: %w", err)
		}

		wsm.connections[streamName] = &WebSocketConnection{
			streamName: streamName,
			doneC:      doneC,
			stopC:      stopC,
		}

		wsm.logger.WithFields(logrus.Fields{
			"stream": streamName,
		}).Info("websocket_connected")
	}

	if wsm.subscribers[streamName] == nil {
		wsm.subscribers[streamName] = make([]chan interface{}, 0)
	}
	wsm.subscribers[streamName] = append(wsm.subscribers[streamName], updateChan)

	return updateChan, nil
}

// SubscribeAccount subscribes to account updates
func (wsm *WebSocketMultiplexer) SubscribeAccount(listenKey string) (<-chan *AccountUpdate, error) {
	streamName := "account"

	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	updateChan := make(chan *AccountUpdate, 100)

	if _, exists := wsm.connections[streamName]; !exists {
		doneC, stopC, err := futures.WsUserDataServe(listenKey, wsm.createAccountHandler(streamName), wsm.createErrorHandler(streamName))
		if err != nil {
			return nil, fmt.Errorf("failed to create account websocket: %w", err)
		}

		wsm.connections[streamName] = &WebSocketConnection{
			streamName: streamName,
			doneC:      doneC,
			stopC:      stopC,
		}

		wsm.logger.WithFields(logrus.Fields{
			"stream": streamName,
		}).Info("websocket_connected")

		// Start listen key refresh goroutine
		wsm.wg.Add(1)
		go wsm.refreshListenKey(listenKey)
	}

	if wsm.subscribers[streamName] == nil {
		wsm.subscribers[streamName] = make([]chan interface{}, 0)
	}
	wsm.subscribers[streamName] = append(wsm.subscribers[streamName], updateChan)

	return updateChan, nil
}

// SubscribePositions subscribes to position updates
func (wsm *WebSocketMultiplexer) SubscribePositions() (<-chan *PositionUpdate, error) {
	// Positions are part of account updates
	// This is a convenience method
	return nil, fmt.Errorf("use SubscribeAccount for position updates")
}

// SubscribeMarkPrice subscribes to mark price updates for a symbol
func (wsm *WebSocketMultiplexer) SubscribeMarkPrice(symbol string) (<-chan *MarkPriceUpdate, error) {
	streamName := fmt.Sprintf("%s@markPrice", symbol)

	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	updateChan := make(chan *MarkPriceUpdate, 100)

	if _, exists := wsm.connections[streamName]; !exists {
		doneC, stopC, err := futures.WsMarkPriceServe(symbol, wsm.createMarkPriceHandler(streamName), wsm.createErrorHandler(streamName))
		if err != nil {
			return nil, fmt.Errorf("failed to create mark price websocket: %w", err)
		}

		wsm.connections[streamName] = &WebSocketConnection{
			streamName: streamName,
			doneC:      doneC,
			stopC:      stopC,
		}

		wsm.logger.WithFields(logrus.Fields{
			"stream": streamName,
		}).Info("websocket_connected")
	}

	if wsm.subscribers[streamName] == nil {
		wsm.subscribers[streamName] = make([]chan interface{}, 0)
	}
	wsm.subscribers[streamName] = append(wsm.subscribers[streamName], updateChan)

	return updateChan, nil
}

// Unsubscribe removes a subscriber from a stream
func (wsm *WebSocketMultiplexer) Unsubscribe(streamName string, ch chan interface{}) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	if subscribers, exists := wsm.subscribers[streamName]; exists {
		for i, subscriber := range subscribers {
			if subscriber == ch {
				wsm.subscribers[streamName] = append(subscribers[:i], subscribers[i+1:]...)
				close(ch)
				break
			}
		}

		// If no more subscribers, close the connection
		if len(wsm.subscribers[streamName]) == 0 {
			if conn, exists := wsm.connections[streamName]; exists {
				close(conn.stopC)
				delete(wsm.connections, streamName)
				delete(wsm.subscribers, streamName)

				wsm.logger.WithFields(logrus.Fields{
					"stream": streamName,
				}).Info("websocket_disconnected")
			}
		}
	}
}

// Close closes all WebSocket connections
func (wsm *WebSocketMultiplexer) Close() {
	close(wsm.stopChan)
	wsm.wg.Wait()

	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	for streamName, conn := range wsm.connections {
		close(conn.stopC)
		wsm.logger.WithFields(logrus.Fields{
			"stream": streamName,
		}).Info("websocket_closed")
	}

	for streamName, subscribers := range wsm.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
		delete(wsm.subscribers, streamName)
	}

	wsm.connections = make(map[string]*WebSocketConnection)
}

// Handler creators

func (wsm *WebSocketMultiplexer) createDepthHandler(streamName string) futures.WsDepthHandler {
	return func(event *futures.WsDepthEvent) {
		update := &OrderBookUpdate{
			Symbol:       event.Symbol,
			Bids:         event.Bids,
			Asks:         event.Asks,
			LastUpdateID: event.LastUpdateID,
			Timestamp:    time.Now(),
		}

		wsm.broadcast(streamName, update)
	}
}

func (wsm *WebSocketMultiplexer) createAggTradeHandler(streamName string) futures.WsAggTradeHandler {
	return func(event *futures.WsAggTradeEvent) {
		update := &TradeUpdate{
			Symbol:    event.Symbol,
			Price:     parseFloat(event.Price),
			Quantity:  parseFloat(event.Quantity),
			Side:      determineSide(event.Maker),
			Timestamp: time.Unix(0, event.TradeTime*int64(time.Millisecond)),
		}

		wsm.broadcast(streamName, update)
	}
}

func (wsm *WebSocketMultiplexer) createAccountHandler(streamName string) futures.WsUserDataHandler {
	return func(event *futures.WsUserDataEvent) {
		if event.Event == "ACCOUNT_UPDATE" {
			balances := make(map[string]float64)
			for _, balance := range event.AccountUpdate.Balances {
				balances[balance.Asset] = parseFloat(balance.WalletBalance)
			}

			positions := make(map[string]*PositionUpdate)
			for _, position := range event.AccountUpdate.Positions {
				positions[position.Symbol] = &PositionUpdate{
					Symbol:           position.Symbol,
					PositionAmt:      parseFloat(position.Amount),
					EntryPrice:       parseFloat(position.EntryPrice),
					UnrealizedProfit: parseFloat(position.UnrealizedPnL),
					MarginType:       position.MarginType,
					Timestamp:        time.Now(),
				}
			}

			update := &AccountUpdate{
				Balances:  balances,
				Positions: positions,
				Timestamp: time.Now(),
			}

			wsm.broadcast(streamName, update)
		}
	}
}

func (wsm *WebSocketMultiplexer) createMarkPriceHandler(streamName string) futures.WsMarkPriceHandler {
	return func(event *futures.WsMarkPriceEvent) {
		update := &MarkPriceUpdate{
			Symbol:      event.Symbol,
			MarkPrice:   parseFloat(event.MarkPrice),
			FundingRate: parseFloat(event.FundingRate),
			Timestamp:   time.Unix(0, event.Time*int64(time.Millisecond)),
		}

		wsm.broadcast(streamName, update)
	}
}

func (wsm *WebSocketMultiplexer) createErrorHandler(streamName string) futures.ErrHandler {
	return func(err error) {
		wsm.logger.WithFields(logrus.Fields{
			"stream": streamName,
			"error":  err.Error(),
		}).Error("websocket_error")

		// Attempt to reconnect
		wsm.reconnect(streamName)
	}
}

// broadcast sends an update to all subscribers of a stream
func (wsm *WebSocketMultiplexer) broadcast(streamName string, update interface{}) {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	if subscribers, exists := wsm.subscribers[streamName]; exists {
		for _, ch := range subscribers {
			select {
			case ch <- update:
			default:
				// Channel full, skip this update
				wsm.logger.WithFields(logrus.Fields{
					"stream": streamName,
				}).Warn("subscriber_channel_full")
			}
		}
	}
}

// reconnect attempts to reconnect a WebSocket connection
func (wsm *WebSocketMultiplexer) reconnect(streamName string) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	wsm.logger.WithFields(logrus.Fields{
		"stream": streamName,
	}).Info("attempting_reconnect")

	// Close existing connection
	if conn, exists := wsm.connections[streamName]; exists {
		close(conn.stopC)
		delete(wsm.connections, streamName)
	}

	// Wait before reconnecting
	time.Sleep(5 * time.Second)

	// Reconnect logic would go here
	// For now, just log the attempt
	wsm.logger.WithFields(logrus.Fields{
		"stream": streamName,
	}).Info("reconnect_attempted")
}

// refreshListenKey refreshes the listen key every 30 minutes
func (wsm *WebSocketMultiplexer) refreshListenKey(listenKey string) {
	defer wsm.wg.Done()

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-wsm.stopChan:
			return
		case <-ticker.C:
			// Refresh listen key via API
			// This would require access to the Futures client
			wsm.logger.Info("listen_key_refresh_needed")
		}
	}
}

// Helper functions

func determineSide(isMaker bool) string {
	if isMaker {
		return "SELL"
	}
	return "BUY"
}
