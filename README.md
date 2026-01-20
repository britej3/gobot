# Ralph-Inspired Claude Orchestrator

A Python implementation inspired by Ralph for Claude Code, featuring enhanced control patterns: exit detection, circuit breakers, rate limits, and a complete data scan → idea → code edit → backtest → report cycle.

## 🎯 Features

### Ralph's Control Patterns (Ported from `/Users/britebrt/GOBOT/scripts/ralph/ralph.sh`)

1. **Exit Detection**: Detects `<promise>COMPLETE</promise>` signal for graceful completion
2. **Iteration Control**: Max iterations with configurable sleep delays
3. **Branch Detection**: Archives runs when branch changes (preserves state)
4. **State Persistence**: PRD tracking, progress logs, pattern consolidation
5. **Progress Tracking**: Detailed progress logs with learnings

### Enhanced Features

1. **Circuit Breaker Pattern**: Prevents cascading failures with:
   - Configurable failure thresholds
   - Automatic recovery after timeout
   - Half-open state for safe recovery

2. **Rate Limiter**: Token bucket algorithm with:
   - Requests per minute/hour limits
   - Adaptive backoff on high frequency
   - Automatic bucket refill

3. **Complete Cycle Orchestration**:
   - Data Scan → Analyze codebase
   - Idea → Generate actionable improvements
   - Code Edit → Implement changes
   - Backtest → Run tests and validate
   - Report → Generate comprehensive report

4. **State Management**:
   - Automatic archival on branch changes
   - Pattern discovery and consolidation
   - Detailed progress tracking

## 📁 Files

- `orchestrator.py` - Main orchestrator implementation
- `test_orchestrator.py` - Comprehensive test suite
- `example_basic.py` - Basic usage example
- `example_advanced.py` - Advanced example with Claude Code integration
- `README.md` - This file

## 🚀 Quick Start

### Basic Example

```bash
python example_basic.py
```

### Run Tests

```bash
pip install pytest pytest-asyncio
pytest test_orchestrator.py -v
```

### Advanced Example (with Claude Code)

```bash
python example_advanced.py
```

## 📊 Architecture

### Core Components

```
┌─────────────────────────────────────┐
│    ClaudeOrchestrator               │
│  - Manages overall execution        │
│  - Implements Ralph patterns        │
└─────────────────────────────────────┘
              │
              ├────────────────┬────────────────┐
              │                │                │
    ┌─────────▼─────┐  ┌──────▼──────┐  ┌────▼──────────┐
    │StateManager   │  │CircuitBreaker│  │RateLimiter    │
    │- Branch tracking│  │- Failure protection│  │- Request throttling│
    │- Archival     │  │- Recovery   │  │- Backoff      │
    │- Progress     │  │             │  │               │
    └───────────────┘  └─────────────┘  └───────────────┘
              │
              └────────────────┬────────────────┐
                                │                │
                      ┌─────────▼───────────────▼────┐
                      │  Five-Phase Cycle           │
                      │  ┌─────┐ ┌────┐ ┌──────┐ │
                      │  │Scan │ │Idea│ │Edit  │ │
                      │  └─────┘ └────┘ └──────┘ │
                      │  ┌────────┐ ┌──────┐   │
                      │  │Backtest│ │Report│   │
                      │  └────────┘ └──────┘   │
                      └────────────────────────┘
```

### Phase Flow

```
┌──────────┐     ┌─────────┐     ┌──────────┐     ┌──────────┐     ┌─────────┐
│   Data   │────▶│  Idea   │────▶│   Code   │────▶│ Backtest │────▶│ Report  │
│  Scan    │     │Generation│     │  Edit    │     │          │     │         │
└──────────┘     └─────────┘     └──────────┘     └──────────┘     └─────────┘
     │               │               │               │               │
     ▼               ▼               ▼               ▼               ▼
  Analyze        Generate       Implement       Validate        Document
  Codebase       Ideas          Changes         Quality         Learnings
```

## 🔧 Configuration

```python
config = OrchestratorConfig(
    # Iteration control
    max_iterations=10,
    sleep_between_iterations=2.0,

    # Circuit breaker
    circuit_breaker_failure_threshold=5,
    circuit_breaker_recovery_timeout=60,

    # Rate limiting
    requests_per_minute=60,
    requests_per_hour=1000,

    # State management
    state_dir="./orchestrator_state",
    archive_dir="./orchestrator_archive",
)
```

## 🧪 Testing

### Run All Tests

```bash
pytest test_orchestrator.py -v
```

### Test Coverage

- **Circuit Breaker**: Closed/open/half-open states, failure threshold, recovery
- **Rate Limiter**: Token bucket, refills, backoff, enforcement
- **State Management**: Branch tracking, archival, pattern consolidation
- **Orchestrator**: Full cycle execution, completion detection, error handling

### Example Test Output

```bash
$ pytest test_orchestrator.py -v

test_orchestrator.py::TestCircuitBreaker::test_circuit_breaker_opens_after_threshold PASSED
test_orchestrator.py::TestRateLimiter::test_rate_limiter_enforces_limits PASSED
test_orchestrator.py::TestStateManager::test_archive_on_branch_change PASSED
test_orchestrator.py::TestClaudeOrchestrator::test_single_cycle_execution PASSED

========= 20 passed in 2.34s =========
```

## 📈 Ralph Pattern Comparison

| Ralph (Bash) | Python Orchestrator | Enhancement |
|-------------|---------------------|-------------|
| `seq 1 $MAX_ITERATIONS` | `max_iterations` config | Configurable per run |
| `grep -q "<promise>COMPLETE</promise>"` | `_check_completion_signal()` | Multiple detection methods |
| Branch archival logic | `StateManager.archive_current_state()` | State persistence |
| Progress appending | Pattern consolidation | Learning extraction |
| - | Circuit breaker | NEW: Failure protection |
| - | Rate limiter | NEW: Throttling |

## 🎓 Key Learnings

### Ralph Patterns Implemented

1. **Exit Detection**: `<promise>COMPLETE</promise>` signal recognition
2. **Branch Tracking**: Archive on branch change to preserve context
3. **Pattern Consolidation**: Collect learnings in `progress.json`
4. **State Persistence**: Save/restore state across runs

### Circuit Breaker Usage

```python
# Circuit breaker protects against cascading failures
circuit_breaker = CircuitBreaker(
    failure_threshold=5,      # Open after 5 failures
    recovery_timeout=60       # Try recovery after 60s
)

# Wrap calls with circuit breaker
result = circuit_breaker.call(api_call, *args, **kwargs)
```

### Rate Limiter Usage

```python
# Rate limiter prevents API overload
rate_limiter = RateLimiter(
    requests_per_minute=60,   # 60 requests per minute
    requests_per_hour=1000    # 1000 requests per hour
)

# Acquire permission before making request
await rate_limiter.acquire()
result = await make_api_call()
```

## 🔍 Integration with Claude Code

The `example_advanced.py` demonstrates integration with Claude Code CLI:

```python
class ClaudeCodeOrchestrator(ClaudeOrchestrator):
    async def _run_claude_command(self, command, prompt):
        # Execute: claude -p <phase> --max-iterations 1
        # Send prompt to Claude
        # Parse response
        return result
```

### Command Flow

```
┌──────────┐
│   User   │  Runs orchestrator
└────┬─────┘
     │
     ▼
┌─────────────────┐
│ Orchestrator    │  Creates cycle
│ 5 phases        │  State: RUNNING
└──────┬──────────┘
       │
       │ Phase 1: Data Scan
       ├────────────────────────┐
       │                        │
       ▼                        ▼
   ┌─────────┐            ┌─────────────┐
   │ Claude  │  Scans    │ StateManager│
   │ Command │  Codebase │ Saves Scan  │
   └─────────┘            └─────────────┘
       │
       │ Phase 2: Idea
       ├────────────────────────┐
       │                        │
       ▼                        ▼
   ┌─────────┐            ┌─────────────┐
   │ Claude  │  Generates│ StateManager│
   │ Command │  Ideas    │ Saves Ideas │
   └─────────┘            └─────────────┘
       │
       │ ... (continue for all 5 phases)
       │
       ▼
┌──────────┐
│ Report   │  Cycle Complete
│ Saved    │
└──────────┘
```

## 📝 State Files

### Progress Structure

```json
{
  "start_time": "2026-01-20T10:00:00",
  "cycles": [
    {
      "cycle_id": "cycle_1_1640000000",
      "duration_seconds": 45.2,
      "phases_completed": [
        "data_scan",
        "idea",
        "code_edit",
        "backtest",
        "report"
      ],
      "success": true,
      "output": "Cycle completed successfully",
      "learnings": [
        "Phase data_scan executed successfully",
        "Found 10 files with async patterns"
      ]
    }
  ],
  "patterns": [
    "Use async/await for I/O operations",
    "Always validate input parameters",
    "Cache expensive computations"
  ]
}
```

### Branch Archive

```
orchestrator_archive/
├── 2026-01-20-feature-old/
│   ├── progress.json
│   └── prd.json
└── 2026-01-20-main/
    ├── progress.json
    └── prd.json
```

## 🎯 Use Cases

1. **Automated Code Review**: Scan → Analyze → Suggest → Validate
2. **Refactoring Projects**: Scan → Plan → Refactor → Test → Document
3. **Performance Optimization**: Scan → Identify → Optimize → Benchmark → Report
4. **Security Auditing**: Scan → Detect → Fix → Verify → Document
5. **Test Coverage**: Scan → Identify → Add → Run → Report

## 🔐 Error Handling

The orchestrator handles errors at multiple levels:

1. **Phase Level**: Individual phase failures don't stop the cycle
2. **Cycle Level**: Failed cycles are logged, archived, and reported
3. **Circuit Breaker**: API failures trigger protection mode
4. **Rate Limiter**: Prevents overload with backoff

### Error Recovery

```python
try:
    result = await phase_handler(phase)
except CircuitBreakerOpen:
    # Switch to fallback mode
    result = await fallback_handler(phase)
except RateLimitExceeded:
    # Wait and retry
    await asyncio.sleep(wait_time)
    result = await phase_handler(phase)
```

## 📦 Dependencies

- Python 3.8+
- asyncio (built-in)
- dataclasses (built-in)
- pathlib (built-in)
- pytest (for testing)
- pytest-asyncio (for async tests)

## 🚦 Status

- ✅ Core orchestrator implemented
- ✅ Circuit breaker pattern
- ✅ Rate limiter pattern
- ✅ State management with archival
- ✅ Five-phase cycle
- ✅ Comprehensive tests
- ✅ Example usage
- ✅ Claude Code integration example

## 🔮 Future Enhancements

- [ ] Plugin system for custom phase handlers
- [ ] Distributed execution across multiple nodes
- [ ] Real-time progress streaming
- [ ] Integration with popular CI/CD platforms
- [ ] Web dashboard for monitoring
- [ ] Machine learning for optimization

## 📚 References

- Ralph for Claude Code: `/Users/britebrt/GOBOT/scripts/ralph/`
- Circuit Breaker Pattern: Martin Fowler
- Token Bucket Algorithm: Rate Limiting
- Ralph Progress Tracking: Automated agent iteration

## 🤝 Contributing

This is an educational/reference implementation. Feel free to adapt and extend for your needs.

## 📄 License

MIT License - Use freely for learning and development.

---

**Built with ❤️ inspired by Ralph for Claude Code**
