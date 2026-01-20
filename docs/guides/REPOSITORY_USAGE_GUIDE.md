# GOBOT Repository Usage Guide

## Overview

This guide explains how to run and utilize two integrated repositories:

1. **SimpleMem** - Efficient Lifelong Memory for LLM Agents (Python-based)
2. **Ralph** - Long-running AI agent loop for automated task completion (Bash-based)

Both repositories are already implemented and integrated into GOBOT.

---

## 1. SimpleMem - Memory System

### What is SimpleMem?

SimpleMem is a semantic memory system for LLM agents that implements **Semantic Lossless Compression** - efficiently storing and retrieving conversational experiences without losing important details.

**Repository:** https://github.com/aiming-lab/SimpleMem.git

### Architecture Overview

SimpleMem uses a three-stage pipeline:

1. **Semantic Structured Compression**: Dialogue → MemoryBuilder → VectorStore
2. **Structured Indexing and Recursive Consolidation**: Background processing (future enhancement)
3. **Adaptive Query-Aware Retrieval**: Query → HybridRetriever → AnswerGenerator

### Directory Structure

```
memory/
├── main.py                 # Main SimpleMem system class
├── trading_memory.py       # GOBOT-specific trading memory wrapper
├── config.py              # Configuration (OpenRouter + Ollama)
├── setup.sh               # Setup script
├── requirements.txt       # Python dependencies
├── .env.example          # Environment template
├── core/
│   ├── memory_builder.py    # Builds semantic memories from dialogues
│   ├── hybrid_retriever.py  # Multi-strategy retrieval (semantic + keyword + structured)
│   └── answer_generator.py  # Generates answers from retrieved context
├── database/
│   └── vector_store.py      # LanceDB vector storage
├── models/
│   └── memory_entry.py      # Data models
└── utils/
    ├── llm_client.py        # OpenRouter/OpenAI client
    ├── embedding.py         # Ollama embedding client
    └── ollama_embedding.py  # Original embedding implementation
```

### Setup Instructions

#### Prerequisites

1. **Python 3.10+** installed
2. **Ollama** installed and running (for embeddings)
3. **OpenRouter API key** (free tier available)

#### Step-by-Step Setup

1. **Navigate to the memory directory:**
```bash
cd /Users/britebrt/GOBOT/memory
```

2. **Run the setup script:**
```bash
./setup.sh
```

This script will:
- Check Python installation
- Create a virtual environment (`venv/`)
- Install all dependencies from `requirements.txt`
- Check Ollama installation and status
- Install the `nomic-embed-text` embedding model (if not present)
- Create `.env` from `.env.example` template
- Create the database directory (`lancedb_data/`)
- Run a quick test

3. **Configure your API keys:**
```bash
cp .env.example .env
# Edit .env with your OpenRouter API key
nano .env
```

**Required configuration:**
```bash
OPENROUTER_API_KEY=your-openrouter-api-key-here
# Optional: Add a backup key for rate limit resilience
OPENROUTER_API_KEY_BACKUP=your-backup-key-here
```

4. **Start Ollama (if not running):**
```bash
ollama serve
```

5. **Pull the embedding model (if not done by setup):**
```bash
ollama pull nomic-embed-text
```

### Configuration

The system is pre-configured with **free tier models** by default:

**LLM Models** (OpenRouter - Free Tier):
- Primary: `meta-llama/llama-3.1-8b-instruct:free`
- Fallbacks: gemma-2, qwen-2, mistral-7b, etc.

**Embedding Model** (Local):
- `ollama:nomic-embed-text` (768 dimensions)

Key settings in `config.py`:
```python
# Parallel processing
ENABLE_PARALLEL_PROCESSING = True
MAX_PARALLEL_WORKERS = 8

# Retrieval features
ENABLE_PLANNING = True          # Multi-query decomposition
ENABLE_REFLECTION = True        # Self-correction for adversarial queries
MAX_REFLECTION_ROUNDS = 2

# Database
LANCEDB_PATH = "./memory/lancedb_data"
MEMORY_TABLE_NAME = "gobot_memory"
```

### Usage Examples

#### Basic Python API

```python
from trading_memory import TradingMemory

# Initialize memory system
memory = TradingMemory(clear_db=False)

# Add a completed trade
memory.add_trade(
    symbol="BTCUSDT",
    side="long",
    entry_price=43500.50,
    exit_price=43800.00,
    pnl=150.0,
    pnl_percent=0.68,
    leverage=25,
    confidence=0.85,
    reason="Bullish divergence on 1m with volume spike",
    outcome="win",
    lessons_learned="Wait for confirmation candle before entry"
)

# Add a market insight
memory.add_market_insight(
    symbol="ETHUSDT",
    timeframe="5m",
    observation="RSI overbought, volume declining",
    pattern="Potential head and shoulders forming"
)

# Query similar past trades
response = memory.query_similar_trades("BTCUSDT", "long")
print(response)

# Get trading context before making a decision
context = memory.get_trading_context("BTCUSDT", "long")
print(context)
```

#### CLI Interface

```bash
# Add a trade
cd /Users/britebrt/GOBOT/memory
source venv/bin/activate

python trading_memory.py add_trade \
    --symbol BTCUSDT \
    --side long \
    --entry 43500.50 \
    --exit 43800.00 \
    --pnl 150.0 \
    --pnl-pct 0.68 \
    --leverage 25 \
    --confidence 0.85 \
    --reason "Bullish divergence" \
    --outcome win \
    --lesson "Wait for confirmation"

# Query memory
python trading_memory.py query_trades --symbol BTCUSDT --side long

# Add market insight
python trading_memory.py add_insight \
    --symbol ETHUSDT \
    --timeframe 5m \
    --observation "RSI overbought" \
    --pattern "Head and shoulders"

# Ask a custom question
python trading_memory.py ask \
    --question "What's the best performing strategy for BTC?"

# List all memories
python trading_memory.py list
```

#### Direct SimpleMem API

```python
from main import SimpleMemSystem

system = SimpleMemSystem(clear_db=True)

# Add dialogues
system.add_dialogue("User", "What is the price of Bitcoin?", "2025-01-12T10:00:00")
system.add_dialogue("Assistant", "Bitcoin is currently at $43,500", "2025-01-12T10:00:01")

# Finalize and store
system.finalize()

# Query
response = system.ask("What is the Bitcoin price?")
print(response)

# View all stored memories
system.print_memories()
```

### Advanced Features

#### 1. Parallel Processing
- **Memory Building**: Multi-worker parallel processing for large dialogue batches
- **Retrieval**: Parallel query execution for faster responses

#### 2. Hybrid Retrieval Strategy
The system combines three retrieval methods:
- **Semantic Search**: Vector similarity (top-k: 25)
- **Keyword Search**: BM25 algorithm (top-k: 5)
- **Structured Search**: Metadata filtering (top-k: 5)

#### 3. Multi-Query Planning
When `enable_planning=True`, the system:
- Decomposes complex queries into sub-queries
- Executes each sub-query independently
- Aggregates results for comprehensive answers

#### 4. Reflection Mechanism
When `enable_reflection=True`:
- Detects adversarial or unclear questions
- Performs additional retrieval rounds (max: 2)
- Self-corrects to provide more accurate answers

#### 5. OpenInference Tracing
Optional integration with observability tools:
- Compatible with Arize Phoenix, LangSmith
- Traces LLM calls, embeddings, and memory operations
- Enable with `ENABLE_OPENINFERENCE=true` in `.env`

### Integration with GOBOT

The Go integration is in `internal/memory/memory.go`:

```go
import "gobot/internal/memory"

// Initialize memory store
store, err := memory.NewMemoryStore("/Users/britebrt/GOBOT")

// Add trade memory
trade := memory.TradeMemory{
    Symbol:     "BTCUSDT",
    Side:       "long",
    EntryPrice: 43500.50,
    ExitPrice:  43800.00,
    PnL:        150.0,
    PnLPercent: 0.68,
    Leverage:   25,
    Confidence: 0.85,
    Reason:     "Bullish divergence",
    Outcome:    "win",
}

err = store.AddTradeMemory(context.Background(), trade)

// Query memory
response, err := store.QueryMemory(context.Background(), 
    "What were the outcomes of BTC long trades?")
```

### Troubleshooting

**Issue: Ollama not responding**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# If not, start it
ollama serve

# Pull embedding model
ollama pull nomic-embed-text
```

**Issue: OpenRouter rate limits**
- The system automatically rotates between available API keys
- Add a backup key in `.env` under `OPENROUTER_API_KEY_BACKUP`
- System will fallback to alternative free models

**Issue: Database locked**
```bash
# Remove lock files
rm -rf memory/lancedb_data/*.lock
```

**Issue: Python dependencies**
```bash
cd memory
source venv/bin/activate
pip install --upgrade pip
pip install -r requirements.txt
```

---

## 2. Ralph - Autonomous AI Agent

### What is Ralph?

Ralph is a long-running AI agent loop that:
- Reads tasks from a PRD (Product Requirements Document)
- Executes tasks autonomously
- Tracks progress in a progress log
- Commits code when tasks complete
- Archives previous runs when switching branches

**Repository:** https://github.com/snarktank/ralph.git

### Directory Structure

```
scripts/ralph/
├── ralph.sh      # Main Ralph execution script
└── prompt.md     # Agent instructions and SOP

# Generated during execution:
├── prd.json      # Product Requirements Document
├── progress.txt  # Progress tracking log
├── .last-branch  # Last executed branch tracker
└── archive/      # Archived runs by date and branch
```

### How Ralph Works

```
┌─────────────────────────────────────────────────────────────┐
│                        RALPH WORKFLOW                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. READ PRD (prd.json)                                    │
│     └─ Get current branch and task list                    │
│                                                             │
│  2. READ PROGRESS (progress.txt)                           │
│     └─ Check Codebase Patterns section                     │
│                                                             │
│  3. CHECKOUT BRANCH                                        │
│     └─ Create from main if doesn't exist                   │
│                                                             │
│  4. SELECT HIGHEST PRIORITY TASK                           │
│     └─ Find story where passes: false                      │
│                                                             │
│  5. IMPLEMENT TASK                                         │
│     └─ Code changes + quality checks                       │
│                                                             │
│  6. UPDATE AGENTS.md                                       │
│     └─ Document reusable patterns                          │
│                                                             │
│  7. COMMIT CHANGES                                         │
│     └─ Message: "feat: [Story ID] - [Title]"               │
│                                                             │
│  8. UPDATE PRD                                             │
│     └─ Set passes: true for completed story                │
│                                                             │
│  9. LOG PROGRESS                                           │
│     └─ Append to progress.txt                              │
│                                                             │
│  10. REPEAT UNTIL ALL TASKS COMPLETE                       │
│      └─ Or max iterations reached                          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Setup Requirements

Ralph requires:
1. **amp CLI tool** - for AI agent execution
2. **jq** - for JSON processing
3. **Git** - for version control
4. **PRD file** - product requirements (prd.json)

### Configuration

#### PRD Format (prd.json)

```json
{
  "projectName": "GOBOT Trading System",
  "branchName": "ralph/trading-enhancements",
  "userStories": [
    {
      "id": "US-001",
      "title": "Add trailing stop loss functionality",
      "description": "Implement trailing stop loss that follows price at 0.15% distance",
      "priority": 1,
      "passes": false,
      "acceptanceCriteria": [
        "Trailing stop activates at 0.3% profit",
        "Maintains 0.15% trail distance",
        "Works for both long and short positions"
      ]
    },
    {
      "id": "US-002",
      "title": "Integrate SimpleMem for trade history",
      "description": "Store and retrieve past trade data for decision making",
      "priority": 2,
      "passes": false,
      "dependencies": ["US-001"]
    }
  ]
}
```

#### Progress Log Format (progress.txt)

```markdown
## Codebase Patterns
- Use `config.toml` for all configuration
- Always validate inputs with `validate/` package
- Use interface types for testability
- Log with structured JSON format

---

## [2025-01-12 10:30] - US-001
Thread: https://amp.api.com/threads/abc123

**What was implemented:**
- Added trailing stop loss calculator in `pkg/risk/`
- Integrated with position manager
- Added tests with 95% coverage

**Files changed:**
- pkg/risk/trailing_stop.go (new)
- pkg/risk/trailing_stop_test.go (new)
- internal/position/manager.go (modified)

**Learnings for future iterations:**
- Pattern: Use `decimal` package for all price calculations
- Gotcha: Remember to update position snapshot after stop adjustment
- Context: Risk engine processes stops every 100ms
```

### Running Ralph

#### Basic Usage

```bash
# Navigate to ralph directory
cd /Users/britebrt/GOBOT/scripts/ralph

# Run with default 10 iterations
./ralph.sh

# Run with custom max iterations
./ralph.sh 5

# Run with unlimited iterations (until complete)
./ralph.sh 999
```

#### Execution Flow

1. **Archive Previous Run** (if branch changed)
   - Copies prd.json and progress.txt to `archive/YYYY-MM-DD-branch-name/`
   - Resets progress.txt for new branch

2. **Initialize Progress Tracking**
   - Creates progress.txt if it doesn't exist
   - Reads Codebase Patterns section

3. **Loop Execution**
   ```bash
   For each iteration:
     - Display iteration number
     - Execute amp with prompt.md
     - Check for <promise>COMPLETE</promise>
     - If found: exit success
     - If not: continue to next iteration
   ```

4. **Completion**
   - Success: All tasks complete
   - Partial: Max iterations reached, review progress.txt

### Ralph Agent SOP (prompt.md)

The agent follows this standard operating procedure:

#### 1. Task Selection
- Read PRD to find current branch
- Read progress.txt for codebase patterns
- Select highest priority story with `passes: false`
- Verify branch, checkout or create if needed

#### 2. Implementation
- Implement ONE story per iteration
- Run quality checks (typecheck, lint, test)
- Update AGENTS.md files if discovering reusable patterns
- Keep changes focused and minimal

#### 3. Quality Requirements
- ALL commits must pass project quality checks
- Do NOT commit broken code
- Follow existing code patterns
- Keep CI green

#### 4. Frontend Stories
For UI changes, MUST verify in browser:
```
1. Load dev-browser skill
2. Navigate to relevant page
3. Verify UI changes work
4. Take screenshot for progress log
```

#### 5. Progress Logging
Always append to progress.txt in this format:
```markdown
## [Date/Time] - [Story ID]
Thread: https://amp.api.com/threads/$THREAD_ID
- What was implemented
- Files changed
- **Learnings for future iterations:**
  - Patterns discovered
  - Gotchas encountered
  - Useful context
---
```

#### 6. AGENTS.md Updates
Update AGENTS.md files when discovering:
- API patterns or conventions
- Gotchas or non-obvious requirements
- Dependencies between files
- Testing approaches
- Configuration requirements

**Good additions:**
- "When modifying X, also update Y"
- "This module uses pattern Z for all API calls"
- "Tests require dev server on PORT 3000"

**Don't add:**
- Story-specific details
- Temporary debugging notes
- Information already in progress.txt

#### 7. Completion Criteria
- After each story, check if ALL stories have `passes: true`
- If complete: respond with `<promise>COMPLETE</promise>`
- If incomplete: continue normally

### Usage Examples

#### Example 1: Adding a New Feature

1. **Create PRD** (prd.json):
```json
{
  "branchName": "ralph/websocket-reconnect",
  "userStories": [
    {
      "id": "WS-001",
      "title": "Add automatic WebSocket reconnection",
      "priority": 1,
      "passes": false,
      "description": "Reconnect with exponential backoff when connection drops"
    },
    {
      "id": "WS-002",
      "title": "Add connection health checks",
      "priority": 2,
      "passes": false,
      "description": "Ping/pong every 30 seconds to verify connection"
    }
  ]
}
```

2. **Run Ralph**:
```bash
cd /Users/britebrt/GOBOT/scripts/ralph
./ralph.sh 10
```

3. **Monitor Progress**:
```bash
tail -f progress.txt
```

4. **Check Results**:
```bash
git log --oneline -n 5
git diff main..ralph/websocket-reconnect
```

#### Example 2: Bug Fixes

```json
{
  "branchName": "ralph/fix-memory-leak",
  "userStories": [
    {
      "id": "BUG-001",
      "title": "Fix goroutine leak in WebSocket handler",
      "priority": 1,
      "passes": false,
      "description": "Context cancellation not properly handled"
    }
  ]
}
```

#### Example 3: Documentation Updates

```json
{
  "branchName": "ralph/update-docs",
  "userStories": [
    {
      "id": "DOC-001",
      "title": "Document API authentication flow",
      "priority": 1,
      "passes": false
    },
    {
      "id": "DOC-002",
      "title": "Add examples for all endpoints",
      "priority": 2,
      "passes": false
    }
  ]
}
```

### Branch Management

#### Automatic Archiving

When Ralph detects a branch change:
1. Creates archive folder: `archive/2025-01-12-branch-name/`
2. Copies: `prd.json`, `progress.txt`
3. Resets progress.txt with Codebase Patterns preserved
4. Logs archive location

#### Manual Archive Review

```bash
# List all archives
ls -la scripts/ralph/archive/

# View archived PRD
cat scripts/ralph/archive/2025-01-12-websocket-reconnect/prd.json

# View archived progress
cat scripts/ralph/archive/2025-01-12-websocket-reconnect/progress.txt

# Compare archives
diff scripts/ralph/archive/2025-01-10-*/prd.json
```

### Integration with SimpleMem

Ralph can use SimpleMem to:
1. **Store implementation patterns**: Save successful patterns to memory
2. **Query past solutions**: "How did we solve X before?"
3. **Learn from mistakes**: Store and avoid repeating errors

**Example integration in Ralph workflow:**
```python
# After completing a story
if story_id.startswith("BUG-"):
    memory.add_strategy_learning(
        learning=f"Fixed {story_title}",
        context=f"Root cause: {root_cause}. Solution: {solution}"
    )
```

### Troubleshooting

**Issue: amp command not found**
```bash
# Install amp (if available via package manager)
# Or specify full path in ralph.sh
AMP_PATH="/path/to/amp"
```

**Issue: jq not found**
```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# Or update ralph.sh to skip JSON processing
```

**Issue: No prd.json found**
```bash
# Create example PRD
cat > scripts/ralph/prd.json << 'EOF'
{
  "branchName": "ralph/example",
  "userStories": [
    {
      "id": "EX-001",
      "title": "Example task",
      "priority": 1,
      "passes": false,
      "description": "This is an example task"
    }
  ]
}
EOF
```

**Issue: Git authentication failed**
```bash
# Configure git credentials
git config --global credential.helper store
# Or use SSH keys
```

**Issue: Progress.txt not updating**
```bash
# Check file permissions
ls -la scripts/ralph/progress.txt

# Fix permissions
chmod 644 scripts/ralph/progress.txt
```

### Best Practices

1. **Start Small**: Begin with 2-3 simple stories to test workflow
2. **Clear Acceptance Criteria**: Define what "done" means for each story
3. **Commit Granularity**: One commit per story, clear commit messages
4. **Document Patterns**: Update Codebase Patterns section religiously
5. **Review Progress**: Check progress.txt after each run
6. **Test Thoroughly**: Validate changes before committing
7. **Use Threads**: Include thread URLs for future reference
8. **Archive Regularly**: Clean up old runs to save space

---

## Combined Usage: Ralph + SimpleMem

### Workflow for Autonomous Development with Memory

```
┌─────────────────────────────────────────────────────────────┐
│                    INTEGRATED WORKFLOW                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. RALPH PLANNING PHASE                                   │
│     └─ Read PRD, select highest priority story            │
│                                                             │
│  2. MEMORY QUERY                                            │
│     └─ Query SimpleMem: "How did we solve similar issues?" │
│        └─ Retrieve past implementations and patterns       │
│                                                             │
│  3. RALPH IMPLEMENTATION PHASE                             │
│     └─ Implement story with learned patterns               │
│                                                             │
│  4. QUALITY CHECKS                                          │
│     └─ Tests, linting, type checking                       │
│                                                             │
│  5. MEMORY STORAGE                                          │
│     └─ Store solution:                                     │
│        - What worked                                         │
│        - What didn't work                                    │
│        - Key learnings                                       │
│                                                             │
│  6. DOCUMENTATION                                           │
│     └─ Update AGENTS.md and progress.txt                   │
│                                                             │
│  7. GIT OPERATIONS                                          │
│     └─ Commit and push changes                             │
│                                                             │
│  8. UPDATE PRD                                              │
│     └─ Mark story as passes: true                          │
│                                                             │
│  9. REPEAT FOR NEXT STORY                                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Setup for Combined Usage

1. **Verify both systems work independently:**
```bash
# Test SimpleMem
cd /Users/britebrt/GOBOT/memory
source venv/bin/activate
python main.py

# Test Ralph
cd /Users/britebrt/GOBOT/scripts/ralph
./ralph.sh 1
```

2. **Create integrated PRD:**
```json
{
  "branchName": "ralph/memory-integration",
  "userStories": [
    {
      "id": "MEM-001",
      "title": "Store successful trade patterns in memory",
      "priority": 1,
      "passes": false,
      "description": "After each winning trade, store pattern to SimpleMem"
    },
    {
      "id": "MEM-002",
      "title": "Query memory before trade execution",
      "priority": 2,
      "passes": false,
      "description": "Get similar past trades to inform decision"
    }
  ]
}
```

### Example: Trade Bot with Memory

```go
package main

import (
    "context"
    "log"
    "gobot/internal/memory"
)

func main() {
    // Initialize memory store
    store, err := memory.NewMemoryStore("/Users/britebrt/GOBOT")
    if err != nil {
        log.Fatal(err)
    }
    
    // Before trading: Query past performance
    ctx := context.Background()
    similarTrades, _ := store.QueryMemory(ctx, 
        "What were outcomes of BTC long trades in last 24h?")
    
    log.Printf("Past patterns: %s", similarTrades)
    
    // Execute trade based on analysis...
    
    // After trading: Store result
    trade := memory.TradeMemory{
        Symbol:     "BTCUSDT",
        Side:       "long",
        EntryPrice: 43500.50,
        ExitPrice:  43800.00,
        PnL:        150.0,
        PnLPercent: 0.68,
        Leverage:   25,
        Confidence: 0.85,
        Reason:     "Bullish divergence + volume spike",
        Outcome:    "win",
        LessonsLearned: "Enter on confirmation candle close",
    }
    
    if err := store.AddTradeMemory(ctx, trade); err != nil {
        log.Printf("Failed to store memory: %v", err)
    }
}
```

### Monitoring and Maintenance

#### Daily Checks
```bash
# Check Ralph status
cd /Users/britebrt/GOBOT/scripts/ralph
tail -n 20 progress.txt

# Check SimpleMem health
cd /Users/britebrt/GOBOT/memory
source venv/bin/activate
python -c "from trading_memory import TradingMemory; m = TradingMemory(); print('OK')"

# Check database size
du -sh memory/lancedb_data/

# Check Ollama status
ollama list
```

#### Weekly Maintenance
```bash
# Archive old Ralph runs (older than 30 days)
find scripts/ralph/archive/ -type d -mtime +30 -exec rm -rf {} +

# Vacuum SimpleMem database
cd memory
source venv/bin/activate
python -c "
from main import create_system
s = create_system()
s.vector_store.db.cleanup_old_versions()
"

# Review and cleanup logs
ls -lah scripts/ralph/*.log 2>/dev/null
```

#### Monthly Review
```bash
# Analyze memory patterns
cd /Users/britebrt/GOBOT/memory
source venv/bin/activate
python trading_memory.py query_learnings

# Review Ralph efficiency
grep -r "Iteration" scripts/ralph/archive/*/progress.txt | wc -l
grep -r "Completed at iteration" scripts/ralph/archive/*/progress.txt

# Update AGENTS.md files
git diff --name-only $(git log --since="1 month ago" --pretty=format:"%h" | tail -1)
```

---

## Quick Reference Commands

### SimpleMem

```bash
# Setup
cd /Users/britebrt/GOBOT/memory
./setup.sh

# Activate environment
source venv/bin/activate

# Quick test
python main.py

# CLI usage
python trading_memory.py add_trade --help
python trading_memory.py query_trades --symbol BTCUSDT --side long
python trading_memory.py ask --question "Best performing strategy?"

# View all memories
python trading_memory.py list

# Clear database
python trading_memory.py add_trade --clear [other params...]

# Check database
du -sh lancedb_data/
ls -lah lancedb_data/
```

### Ralph

```bash
# Run Ralph
cd /Users/britebrt/GOBOT/scripts/ralph
./ralph.sh 10              # 10 iterations
./ralph.sh 999            # Until complete

# Monitor
tail -f progress.txt

# View history
cat progress.txt
git log --oneline -n 10

# Archive management
ls -la archive/
ls -la archive/2025-01-12-*

# Reset (careful!)
rm -f progress.txt .last-branch
```

### Combined

```bash
# Full system health check
echo "=== Ralph Status ==="
tail -n 5 scripts/ralph/progress.txt 2>/dev/null || echo "No progress file"

echo "=== SimpleMem Status ==="
cd memory && source venv/bin/activate && python -c "
from trading_memory import TradingMemory
try:
    m = TradingMemory()
    memories = m.get_all_memories()
    print(f'Database: OK ({len(memories)} memories)')
except Exception as e:
    print(f'Error: {e}')
" 2>/dev/null

echo "=== Ollama Status ==="
curl -s http://localhost:11434/api/tags | jq -r '.models[].name' 2>/dev/null | grep embed || echo "No embedding models"

echo "=== System Ready ==="
```

---

## Support and Troubleshooting

### Getting Help

1. **Check logs:**
   - Ralph: `scripts/ralph/progress.txt`
   - SimpleMem: `memory/lancedb_data/`
   - System: `startup.log`, `aggressive_test.log`

2. **Common issues:**
   - **SimpleMem won't start**: Check Ollama, verify `.env` exists
   - **Ralph won't run**: Check `prd.json` format, verify `amp` available
   - **Memory queries slow**: Check embedding model is local, not API
   - **Database errors**: Remove `.lock` files, restart Ollama

3. **Reset procedures:**
   ```bash
   # Reset SimpleMem (clears all memories)
   cd memory
   source venv/bin/activate
   python -c "from main import create_system; s = create_system(clear_db=True); print('Reset complete')"
   
   # Reset Ralph (keeps archives)
   cd scripts/ralph
   mv progress.txt progress.txt.backup
   cp prompt.md progress.txt  # Start fresh
   ```

### Performance Optimization

**SimpleMem:**
- Enable parallel processing in `config.py`
- Use local Ollama (not API) for embeddings
- Increase `WINDOW_SIZE` for batching
- Vacuum database regularly

**Ralph:**
- Limit iterations for faster feedback loops
- Use smaller, focused stories
- Cache amp responses if possible
- Run during off-peak hours

---

## Documentation Files

- **SimpleMem**: `memory/main.py` (docstrings)
- **Ralph**: `scripts/ralph/prompt.md` (SOP)
- **Integration**: `internal/memory/memory.go` (Go bridge)
- **This guide**: `REPOSITORY_USAGE_GUIDE.md`
- **Config**: `memory/config.py`, `memory/.env.example`
- **Examples**: `memory/trading_memory.py` (CLI examples at bottom)

---

**Last Updated: 2025-01-12**
**Status: Both repositories fully implemented and operational**
