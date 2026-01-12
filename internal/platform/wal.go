package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type LogEntry struct {
	ID        string    `json:"id"`        // Unique UUID for the intent
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`      // BUY/SELL
	Qty       float64   `json:"qty"`
	Price     float64   `json:"price,omitempty"`
	Status    string    `json:"status"`    // INTENT, COMMITTED, FAILED, MARKET_DATA
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

type WAL struct {
	file         *os.File
	mu           sync.Mutex
	buffer       chan *LogEntry  // Buffered channel for batch writes
	flushTicker  *time.Ticker
	flushSize    int             // Flush after N entries
	marketDataWAL *WAL           // Separate WAL for high-frequency market data
}

func NewWAL(path string) (*WAL, error) {
	// Open in Append mode, Create if not exists
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	w := &WAL{
		file:        f,
		buffer:      make(chan *LogEntry, 1000), // Buffer up to 1000 entries
		flushTicker: time.NewTicker(100 * time.Millisecond),
		flushSize:   50,
	}

	// Start background flusher
	go w.backgroundFlusher()

	return w, nil
}

// NewMarketDataWAL creates a separate WAL for high-frequency market data
func NewMarketDataWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	return &WAL{
		file:   f,
		buffer: make(chan *LogEntry, 5000), // Larger buffer for market data
		flushTicker: time.NewTicker(50 * time.Millisecond), // More frequent flushes
		flushSize:   100,
	}, nil
}

// LogIntent records the plan to trade. Synchronous fsync for high safety.
func (w *WAL) LogIntent(entry LogEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	entry.Status = "INTENT"
	entry.Timestamp = time.Now()
	
	data, _ := json.Marshal(entry)
	if _, err := w.file.Write(append(data, '\n')); err != nil {
		return err
	}
	
	// fsync ensures the intent is physically on the disk before we hit the API
	return w.file.Sync() 
}

// LogMarketData records market data with buffered writes for performance
func (w *WAL) LogMarketData(symbol string, price float64, message string) {
	entry := &LogEntry{
		ID:        fmt.Sprintf("md_%d", time.Now().UnixNano()),
		Symbol:    symbol,
		Price:     price,
		Status:    "MARKET_DATA",
		Timestamp: time.Now(),
		Message:   message,
	}

	select {
	case w.buffer <- entry:
		// Successfully queued
	default:
		// Buffer full, drop the entry but log warning
		logrus.Warn("WAL buffer full, dropping market data entry")
	}
}

// CommitUpdate marks the trade as successful.
func (w *WAL) CommitUpdate(id string, status string) {
	// For performance, we append a new line with the same ID but status=COMMITTED
	// The Reconciler will always take the LATEST status for any ID.
	entry := LogEntry{ID: id, Status: status, Timestamp: time.Now()}
	data, _ := json.Marshal(entry)
	
	w.mu.Lock()
	w.file.Write(append(data, '\n'))
	w.mu.Unlock()
}

// backgroundFlusher handles batch writes for buffered entries
func (w *WAL) backgroundFlusher() {
	var batch []*LogEntry

	for {
		select {
		case entry := <-w.buffer:
			batch = append(batch, entry)
			
			// Flush when batch is full
			if len(batch) >= w.flushSize {
				w.flushBatch(batch)
				batch = batch[:0] // Clear batch
			}
			
			case <-w.flushTicker.C:
			// Flush on timer tick if we have entries
			if len(batch) > 0 {
				w.flushBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// flushBatch writes a batch of entries
func (w *WAL) flushBatch(batch []*LogEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check rotation before writing (per reply_unknown.md: 50MB limit)
	if err := w.checkRotation(); err != nil {
		logrus.WithError(err).Error("WAL rotation check failed")
	}

	for _, entry := range batch {
		data, _ := json.Marshal(entry)
		w.file.Write(append(data, '\n'))
	}
	
	// Sync after batch for durability
	w.file.Sync()
}

// checkRotation implements size-based log rotation per reply_unknown.md
func (w *WAL) checkRotation() error {
	const maxLogSize = 50 * 1024 * 1024 // 50MB
	
	stat, err := w.file.Stat()
	if err != nil {
		return err
	}

	if stat.Size() >= maxLogSize {
		// Rotate log file
		w.file.Close()
		
		// Rename: trade.wal → trade.{timestamp}.wal
		oldPath := w.file.Name()
		newPath := fmt.Sprintf("%s.%d.wal", oldPath, time.Now().Unix())
		
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to rotate wal: %w", err)
		}
		
		// Create new log file
		f, err := os.OpenFile(oldPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("failed to create new wal: %w", err)
		}
		
		w.file = f
		logrus.WithField("rotated_to", newPath).Info("WAL size limit reached, rotated log")
	}
	
	return nil
}

// Close shuts down the WAL and flushes remaining entries
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.flushTicker.Stop()
	close(w.buffer)

	// Flush any remaining entries
	for entry := range w.buffer {
		data, _ := json.Marshal(entry)
		w.file.Write(append(data, '\n'))
	}
	
	return w.file.Close()
}