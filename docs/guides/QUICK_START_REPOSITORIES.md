# Quick Start Guide - SimpleMem + Ralph

## TL;DR - Get Started in 5 Minutes

```bash
# 1. Set up SimpleMem
cd /Users/britebrt/GOBOT/memory
./setup.sh

# 2. Configure API key
cp .env.example .env
# Edit .env and add your OpenRouter API key

# 3. Start Ollama (in another terminal)
ollama serve

# 4. Test SimpleMem
source venv/bin/activate
python main.py

# 5. Test Ralph
cd /Users/britebrt/GOBOT/scripts/ralph
cp ../../prd.json ./prd.json  # Copy PRD if needed
./ralph.sh 3

# Success! Both systems are working.
```

---

## Detailed Setup

### Step 1: SimpleMem (Memory System)

**What it does:** Stores and retrieves trading experiences using semantic search

**Setup:**
```bash
cd /Users/britebrt/GOBOT/memory
./setup.sh
```

**What this does:**
- ✅ Creates Python virtual environment
- ✅ Installs 145+ dependencies
- ✅ Checks Ollama and pulls embedding model
- ✅ Creates `.env` from template
- ✅ Sets up LanceDB database directory
- ✅ Runs quick import test

**Configuration:**
```bash
# Edit memory/.env
OPENROUTER_API_KEY=your-key-here  # Get from: https://openrouter.ai/keys
OPENROUTER_API_KEY_BACKUP=your-backup-key  # Optional, for rate limits
```

**Test it:**
```bash
source venv/bin/activate
python main.py
```

Expected output:
```
============================================================
Initializing SimpleMem System
============================================================

System initialization complete!
============================================================

🚀 Running SimpleMem Quick Test with Qwen3...
📌 Using embedding model: ollama:nomic-embed-text
✅ Quick test completed!
```

---

### Step 2: Ralph (Autonomous Agent)

**What it does:** Executes tasks from PRD automatically, commits code, tracks progress

**Requirements:**
- `amp` CLI tool (for AI agent execution)
- `jq` (JSON processor)
- `git` (configured)
- PRD file (`prd.json`)

**Setup:**
```bash
cd /Users/britebrt/GOBOT/scripts/ralph

# Copy PRD if you have one
cp ../../prd.json ./prd.json  # Optional

# Make sure it's executable (should already be)
chmod +x ralph.sh
```

**Test it:**
```bash
# Run with 1 iteration to test
./ralph.sh 1
```

Expected behavior:
- Reads `prd.json` for tasks
- Executes highest priority incomplete task
- Commits changes if successful
- Updates `progress.txt`
- Exits after 1 iteration

---

## Usage Examples

### Example 1: Store a Trade in Memory

```bash
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
    --reason "Bullish divergence on 1m" \
    --outcome win \
    --lesson "Wait for confirmation candle"
```

### Example 2: Query Trading History

```bash
# Query similar trades
python trading_memory.py query_trades \
    --symbol BTCUSDT \
    --side long

# Get trading context before a trade
python trading_memory.py context \
    --symbol ETHUSDT \
    --side short
```

### Example 3: Use in Python Code

```python
from trading_memory import TradingMemory

# Initialize
memory = TradingMemory()

# Store a trade
memory.add_trade(
    symbol="BTCUSDT",
    side="long",
    entry_price=43500.50,
    exit_price=43800.00,
    pnl=150.0,
    pnl_percent=0.68,
    leverage=25,
    confidence=0.85,
    reason="Bullish divergence",
    outcome="win",
    lessons_learned="Wait for confirmation"
)

# Query before next trade
context = memory.get_trading_context("BTCUSDT", "long")
print(context)
```

### Example 4: Run Ralph for Automation

```bash
cd /Users/britebrt/GOBOT/scripts/ralph

# Create a PRD
cat > prd.json << 'EOF'
{
  "branchName": "ralph/example-feature",
  "userStories": [
    {
      "id": "EX-001",
      "title": "Add logging to trade execution",
      "description": "Log entry and exit points for debugging",
      "priority": 1,
      "passes": false
    }
  ]
}
EOF

# Run for 5 iterations
./ralph.sh 5

# Watch progress
tail -f progress.txt
```

---

## Integration: Use Them Together

### Why use both?

- **Ralph** implements features automatically
- **SimpleMem** learns from what Ralph implements
- Together: Self-improving trading system

### Workflow

```
┌────────────────────────────────┐
│  User: Add feature request     │
└─────────┬──────────────────────┘
          │
          ▼
┌────────────────────────────────┐
│  Ralph: Reads PRD              │
│  → Implements feature          │
│  → Commits code                │
└─────────┬──────────────────────┘
          │
          ▼
┌────────────────────────────────┐
│  SimpleMem: Stores result      │
│  → "Feature X works/broken"    │
│  → "Pattern: use Y for Z"      │
└─────────┬──────────────────────┘
          │
          ▼
┌────────────────────────────────┐
│  Next iteration: Query memory  │
│  → "How did we solve X?"       │
│  → Apply learned patterns      │
└─────────┬──────────────────────┘
          │
          ▼
┌────────────────────────────────┐
│  Improve automatically         │
└────────────────────────────────┘
```

### Example: Self-Improving Trading Strategy

```python
# In your trading bot:
from trading_memory import TradingMemory
from internal.memory import MemoryStore
import context

# 1. Before trading: Query past performance
mem = TradingMemory()
past_performance = mem.query_similar_trades("BTCUSDT", "long")

# 2. LLM decides based on memory + market data
decision = llm.analyze(f"""
Market data: {current_market}
Past similar trades: {past_performance}
Make trading decision.
""")

# 3. Execute trade
if decision.should_trade:
    result = execute_trade(decision)
    
    # 4. Store outcome
    mem.add_trade(
        symbol="BTCUSDT",
        side=decision.side,
        entry_price=result.entry,
        exit_price=result.exit,
        pnl=result.pnl,
        pnl_percent=result.pnl_percent,
        leverage=decision.leverage,
        confidence=decision.confidence,
        reason=decision.reasoning,
        outcome=result.outcome,
        lessons_learned=result.lessons
    )
    
    # 5. Ralph periodically reviews and optimizes
    #    the trading logic based on memory patterns
```

---

## Monitoring

### Check System Health

```bash
cd /Users/britebrt/GOBOT

# Health check script
./verify_repositories.sh

# Manual checks:

# SimpleMem status
cd memory
source venv/bin/activate
python -c "
from trading_memory import TradingMemory
m = TradingMemory()
memories = m.get_all_memories()
print(f'Memories stored: {len(memories)}')
"

# Ralph status
cd scripts/ralph
tail -n 20 progress.txt

# Ollama status
ollama list
```

### View Stored Memories

```bash
cd /Users/britebrt/GOBOT/memory
source venv/bin/activate

# List all memories
python trading_memory.py list

# Query specific patterns
python trading_memory.py ask \
    --question "What patterns lead to winning trades?"

# Get strategy insights
python trading_memory.py query_learnings
```

### Archive Ralph Runs

```bash
cd /Users/britebrt/GOBOT/scripts/ralph

# List archived runs
ls -la archive/

# View specific archive
cat archive/2025-01-12-*/progress.txt

# Clean old archives (30+ days)
find archive/ -type d -mtime +30 -exec rm -rf {} +
```

---

## Troubleshooting

### Issue: SimpleMem won't start

```bash
# Check Ollama
curl http://localhost:11434/api/tags
# Should show: nomic-embed-text

# If not:
ollama pull nomic-embed-text

# Check Python env
cd memory
source venv/bin/activate
python -c "import config"
```

### Issue: Ralph won't run

```bash
# Check dependencies
which amp jq git

# Check PRD format
cd scripts/ralph
jq . prd.json

# Check permissions
ls -la ralph.sh
chmod +x ralph.sh
```

### Issue: Memory queries are slow

```bash
# Check embedding model is local (should be ollama:*)
grep EMBEDDING_MODEL memory/config.py

# Should be: ollama:nomic-embed-text
# Not: openai or openrouter

# Check database size
du -sh memory/lancedb_data/
# If > 1GB, consider archiving old data
```

### Issue: Out of memory

```bash
# Reduce parallel workers in memory/config.py
ENABLE_PARALLEL_PROCESSING = True
MAX_PARALLEL_WORKERS = 4  # Reduce from 8
MAX_RETRIEVAL_WORKERS = 2  # Reduce from 4

# Restart with lower limits
```

---

## Next Steps

1. **Read the full guide:**
   ```bash
   cat REPOSITORY_USAGE_GUIDE.md
   ```

2. **Set up your trading bot to use memory:**
   - See `internal/memory/memory.go` for Go integration
   - See `trading_memory.py` for Python wrapper

3. **Create your first PRD:**
   - Use `prd.json` as template
   - Add user stories with clear acceptance criteria
   - Run Ralph to implement automatically

4. **Start building memory:**
   - Every trade: store to SimpleMem
   - Every pattern: store insight
   - Every lesson: add to memory
   - Query before decisions

5. **Monitor and improve:**
   - Check memory quality weekly
   - Review Ralph progress logs
   - Update AGENTS.md with patterns
   - Refine strategies based on memory

---

## Summary

**Both repositories are fully implemented and ready to use:**

- ✅ SimpleMem: Complete memory system with semantic search
- ✅ Ralph: Autonomous agent for task execution
- ✅ Integration: Go bridge in `internal/memory/`
- ✅ Configuration: Free tier optimized
- ✅ Documentation: Comprehensive guides
- ✅ Testing: Verification script included

**Status:** Operational and ready for production use

**For detailed information:** See `REPOSITORY_USAGE_GUIDE.md`
