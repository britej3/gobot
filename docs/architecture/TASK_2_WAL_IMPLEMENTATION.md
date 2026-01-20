# Task 2: Write-Ahead Logging (WAL) for Trade Durability
## Status: RESEARCH IN PROGRESS (Unknown: Optimal Checkpoint Strategy)

**Implementation Started:** Partial design  
**Fully Complete:** No (blocker: checkpoint timing unknown)  
**Priority:** P0 - Critical for production reliability

### What Is Known:
1. ✅ WAL Pattern: Append trade intent to log → fsync() → Execute trade
2. ✅ Log format: JSONL with timestamp, symbol, side, price, quantity
3. ✅ Recovery: On startup, scan log for unexecuted trades
4. ✅ Location: Log file per session (e.g., `wal_20260109_104800.log`)
5. ⚠️ **Unknown:** Optimal checkpoint interval for HFT (per-trade vs batched)
6. ⚠️ **Unknown:** Performance impact of fsync() on execution latency
7. ⚠️ **Unknown:** How to detect and handle partial writes during crash

### Design Sketch:
```go
type WALEntry struct {
    Timestamp   int64   `json:"ts"`
    Symbol      string  `json:"sym"`
    Side        string  `json:"side"`
    EntryPrice  float64 `json:"price"`
    Quantity    float64 `json:"qty"`
    StopLoss    float64 `json:"sl"`
    TakeProfit  float64 `json:"tp"`
    Executed    bool    `json:"exec"`
}

// Before trade execution:
wal.Append(entry)
wal.Fsync()
// Then execute trade and mark as executed
```

### Unknown Implementation Details:
1. **Checkpoint timing**: Do we fsync() after every trade (latency hit) or batch (risk)?
2. **Performance impact**: Benchmark required - fsync() adds ~1-5ms latency
3. **Partial write detection**: Need checksums or atomic writes
4. **Log rotation**: When to rotate logs to prevent unbounded growth?

### Recommendation:
**DO NOT IMPLEMENT** native WAL until benchmarking complete. Consider using **SQLite WAL mode** as interim solution (already have SQLite) for immediate durability with known performance characteristics.

**Alternative Quick Win:**
```go
// In existing feedback system:
db.Exec("PRAGMA journal_mode=WAL;")  // Already have SQLite
```

**Priority:** P0 - Blocking for any real money deployment
**Risk Level:** HIGH - Without WAL, system is not crash-resilient