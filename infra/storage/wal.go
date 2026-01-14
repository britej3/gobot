package wal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WAL struct {
	mu       sync.RWMutex
	file     *os.File
	writer   *bufio.Writer
	dir      string
	filename string
	sequence int64
}

type Entry struct {
	Sequence  int64           `json:"seq"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"ts"`
	Payload   json.RawMessage `json:"data"`
}

type EventHandler func(entry Entry) error

func New(dir string) (*WAL, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	filename := filepath.Join(dir, fmt.Sprintf("wal_%d.log", time.Now().UnixNano()))

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}

	wal := &WAL{
		file:     file,
		writer:   bufio.NewWriter(file),
		dir:      dir,
		filename: filename,
		sequence: 0,
	}

	return wal, nil
}

func (w *WAL) Write(ctx context.Context, entryType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.sequence++

	entry := Entry{
		Sequence:  w.sequence,
		Type:      entryType,
		Timestamp: time.Now(),
		Payload:   data,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := w.writer.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush WAL: %w", err)
	}

	return nil
}

func (w *WAL) Read(ctx context.Context, handler EventHandler) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	file, err := os.Open(w.filename)
	if err != nil {
		return fmt.Errorf("failed to open WAL file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		if err := handler(entry); err != nil {
			return fmt.Errorf("handler error at seq %d: %w", entry.Sequence, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

func (w *WAL) ReadFrom(ctx context.Context, sequence int64, handler EventHandler) error {
	w.mu.RLock()
	file, err := os.Open(w.filename)
	w.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to open WAL file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		if entry.Sequence <= sequence {
			continue
		}

		if err := handler(entry); err != nil {
			return fmt.Errorf("handler error at seq %d: %w", entry.Sequence, err)
		}
	}

	return nil
}

func (w *WAL) Truncate(ctx context.Context, sequence int64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.sequence = sequence

	filename := filepath.Join(w.dir, fmt.Sprintf("wal_%d.log", time.Now().UnixNano()))

	newFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new WAL file: %w", err)
	}

	oldFile := w.file
	w.file = newFile
	w.writer = bufio.NewWriter(newFile)
	w.filename = filename

	oldFile.Close()

	return nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush WAL: %w", err)
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close WAL file: %w", err)
	}

	return nil
}

func (w *WAL) Stats() WALStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	info, _ := os.Stat(w.filename)

	return WALStats{
		Filename: w.filename,
		Size:     info.Size(),
		Sequence: w.sequence,
	}
}

type WALStats struct {
	Filename string
	Size     int64
	Sequence int64
}
