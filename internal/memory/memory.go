// Package memory provides integration with SimpleMem for long-term memory
// Uses Python subprocess to communicate with SimpleMem
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// MemoryEntry represents a stored memory
type MemoryEntry struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Content     string    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TradeMemory represents a trade execution memory
type TradeMemory struct {
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	EntryPrice   float64 `json:"entry_price"`
	ExitPrice    float64 `json:"exit_price"`
	PnL          float64 `json:"pnl"`
	PnLPercent   float64 `json:"pnl_percent"`
	Leverage     int     `json:"leverage"`
	Confidence   float64 `json:"confidence"`
	Reason       string  `json:"reason"`
	Outcome      string  `json:"outcome"` // win, loss, breakeven
	LessonsLearned string `json:"lessons_learned,omitempty"`
}

// MarketInsight represents a market observation memory
type MarketInsight struct {
	Symbol      string   `json:"symbol"`
	Observation string   `json:"observation"`
	Indicators  map[string]float64 `json:"indicators,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	Timeframe   string   `json:"timeframe"`
}

// MemoryStore manages the SimpleMem integration
type MemoryStore struct {
	pythonPath string
	memoryDir  string
	mu         sync.RWMutex
}

// NewMemoryStore creates a new memory store
func NewMemoryStore(projectRoot string) (*MemoryStore, error) {
	memoryDir := filepath.Join(projectRoot, "memory")
	
	// Check if Python is available
	pythonPath, err := exec.LookPath("python3")
	if err != nil {
		pythonPath, err = exec.LookPath("python")
		if err != nil {
			return nil, fmt.Errorf("python not found: %w", err)
		}
	}
	
	return &MemoryStore{
		pythonPath: pythonPath,
		memoryDir:  memoryDir,
	}, nil
}

// AddTradeMemory stores a trade execution memory
func (m *MemoryStore) AddTradeMemory(ctx context.Context, trade TradeMemory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	content := fmt.Sprintf(
		"Trade %s %s: Entry $%.4f, Exit $%.4f, PnL %.2f%% (%s). Leverage %dx, Confidence %.0f%%. Reason: %s",
		trade.Side, trade.Symbol,
		trade.EntryPrice, trade.ExitPrice,
		trade.PnLPercent, trade.Outcome,
		trade.Leverage, trade.Confidence*100,
		trade.Reason,
	)
	
	if trade.LessonsLearned != "" {
		content += fmt.Sprintf(" Lesson: %s", trade.LessonsLearned)
	}
	
	return m.addDialogue(ctx, "TradeExecutor", content, "trade_execution")
}

// AddMarketInsight stores a market observation
func (m *MemoryStore) AddMarketInsight(ctx context.Context, insight MarketInsight) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	content := fmt.Sprintf(
		"Market insight for %s (%s): %s",
		insight.Symbol, insight.Timeframe,
		insight.Observation,
	)
	
	if insight.Pattern != "" {
		content += fmt.Sprintf(" Pattern detected: %s", insight.Pattern)
	}
	
	return m.addDialogue(ctx, "MarketAnalyzer", content, "market_insight")
}

// AddStrategyLearning stores a strategy performance learning
func (m *MemoryStore) AddStrategyLearning(ctx context.Context, learning string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.addDialogue(ctx, "StrategyOptimizer", learning, "strategy_learning")
}

// AddRiskEvent stores a risk event
func (m *MemoryStore) AddRiskEvent(ctx context.Context, event string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.addDialogue(ctx, "RiskManager", event, "risk_event")
}

// Query retrieves relevant memories for a question
func (m *MemoryStore) Query(ctx context.Context, question string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	script := fmt.Sprintf(`
import sys
sys.path.insert(0, '%s')
from main import SimpleMemSystem

system = SimpleMemSystem(clear_db=False)
answer = system.ask(%q)
print(answer)
`, m.memoryDir, question)
	
	return m.runPython(ctx, script)
}

// QuerySimilarTrades finds similar past trades
func (m *MemoryStore) QuerySimilarTrades(ctx context.Context, symbol string, side string) (string, error) {
	question := fmt.Sprintf(
		"What were the outcomes of previous %s trades on %s? What patterns led to wins vs losses?",
		side, symbol,
	)
	return m.Query(ctx, question)
}

// QueryMarketPatterns finds relevant market patterns
func (m *MemoryStore) QueryMarketPatterns(ctx context.Context, symbol string) (string, error) {
	question := fmt.Sprintf(
		"What market patterns have been observed for %s? What conditions led to successful trades?",
		symbol,
	)
	return m.Query(ctx, question)
}

// addDialogue adds a dialogue to SimpleMem
func (m *MemoryStore) addDialogue(ctx context.Context, speaker, content, category string) error {
	timestamp := time.Now().Format(time.RFC3339)
	
	script := fmt.Sprintf(`
import sys
sys.path.insert(0, '%s')
from main import SimpleMemSystem

system = SimpleMemSystem(clear_db=False)
system.add_dialogue(%q, %q, %q)
system.finalize()
print("OK")
`, m.memoryDir, speaker, content, timestamp)
	
	_, err := m.runPython(ctx, script)
	return err
}

// runPython executes a Python script
func (m *MemoryStore) runPython(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, m.pythonPath, "-c", script)
	cmd.Dir = m.memoryDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("python error: %w - stderr: %s", err, stderr.String())
	}
	
	return stdout.String(), nil
}

// Initialize sets up the memory system
func (m *MemoryStore) Initialize(ctx context.Context) error {
	script := fmt.Sprintf(`
import sys
sys.path.insert(0, '%s')
from main import SimpleMemSystem

# Initialize with fresh database if needed
system = SimpleMemSystem(clear_db=False)
print("SimpleMem initialized")
`, m.memoryDir)
	
	_, err := m.runPython(ctx, script)
	return err
}

// GetRecentMemories retrieves recent memories for context
func (m *MemoryStore) GetRecentMemories(ctx context.Context, limit int) ([]MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	script := fmt.Sprintf(`
import sys
import json
sys.path.insert(0, '%s')
from main import SimpleMemSystem

system = SimpleMemSystem(clear_db=False)
memories = system.get_all_memories()

# Return last N memories as JSON
result = []
for mem in memories[-%d:]:
    result.append({
        "id": mem.entry_id,
        "content": mem.lossless_restatement,
        "timestamp": mem.timestamp or "",
    })
print(json.dumps(result))
`, m.memoryDir, limit)
	
	output, err := m.runPython(ctx, script)
	if err != nil {
		return nil, err
	}
	
	var entries []MemoryEntry
	if err := json.Unmarshal([]byte(output), &entries); err != nil {
		return nil, fmt.Errorf("failed to parse memories: %w", err)
	}
	
	return entries, nil
}
