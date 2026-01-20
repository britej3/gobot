# Repository Overview - Quick Reference

## What's Implemented?

Both repositories (SimpleMem and Ralph) are **fully implemented** in GOBOT and ready to use.

---

## Repository 1: SimpleMem

**What it does:** Stores and retrieves trading experiences using AI-powered semantic search

**Location:** `/Users/britebrt/GOBOT/memory/`

### Key Files
- `main.py` - Core memory system
- `trading_memory.py` - Trading-specific wrapper (what you'll use most)
- `config.py` - Configuration (free tier optimized)
- `setup.sh` - Automated setup script

### How to Use

**Quick test:**
```bash
cd /Users/britebrt/GOBOT/memory
./setup.sh              # One-time setup
source venv/bin/activate # Activate environment
python main.py          # Run test
```

**In your Python code:**
```python
from trading_memory import TradingMemory

memory = TradingMemory()

# Store a trade
memory.add_trade(
    symbol="BTCUSDT",
    side="long",
    entry_price=43500.50,
    exit_price=43800.00,
    pnl_percent=0.68,
    outcome="win",
    lessons_learned="Wait for confirmation"
)

# Query before trading
context = memory.get_trading_context("BTCUSDT", "long")
print(context)
```

**From command line:**
```bash
# Add a trade
python trading_memory.py add_trade \
    --symbol BTCUSDT \
    --side long \
    --entry 43500.50 \
    --exit 43800.00 \
    --pnl-pct 0.68 \
    --outcome win \
    --lesson "Wait for confirmation"

# Query
python trading_memory.py query_trades --symbol BTCUSDT --side long
```

### Setup Required

1. **Run setup:** `cd memory && ./setup.sh`
2. **Add API key:** Edit `memory/.env` with OpenRouter API key (free)
3. **Start Ollama:** `ollama serve` (in another terminal)

---

## Repository 2: Ralph

**What it does:** Automatically executes tasks from a PRD (Product Requirements Document)

**Location:** `/Users/britebrt/GOBOT/scripts/ralph/`

### Key Files
- `ralph.sh` - Main agent loop
- `prompt.md` - Agent instructions
- `prd.json` - Product requirements (you create this)
- `progress.txt` - Progress log (created automatically)

### How to Use

**Create a PRD (prd.json):**
```json
{
  "branchName": "ralph/my-feature",
  "userStories": [
    {
      "id": "US-001",
      "title": "Add logging to trades",
      "priority": 1,
      "passes": false,
      "description": "Log entry and exit prices"
    }
  ]
}
```

**Run Ralph:**
```bash
cd /Users/britebrt/GOBOT/scripts/ralph
./ralph.sh 10  # Run for 10 iterations
```

**What happens:**
1. Reads your PRD
2. Checks out/creates branch
3. Implements highest priority task
4. Commits code
5. Updates PRD (sets passes: true)
6. Logs progress
7. Repeats until done or max iterations

**Watch progress:**
```bash
tail -f progress.txt
```

### Setup Required

1. **Install dependencies:** `amp` and `jq` tools
2. **Create PRD:** `prd.json` in `scripts/ralph/`
3. **Make executable:** `chmod +x ralph.sh` (should already be)

---

## Integration: Use Them Together

### Workflow
```
You → Ralph → Implements Feature
            ↓
            SimpleMem ← Stores what worked/failed
            ↓
            Next iteration ← Uses learned patterns
```

### Example: Trading Bot
```python
from trading_memory import TradingMemory

# Initialize
memory = TradingMemory()

# Before trading
context = memory.get_trading_context("BTCUSDT", "long")

# LLM decides using context + market data
decision = llm.analyze(context, market_data)

if decision.should_trade:
    # Execute trade
    result = execute(decision)
    
    # Store outcome
    memory.add_trade(
        symbol="BTCUSDT",
        pnl_percent=result.pnl,
        outcome=result.outcome,
        lessons_learned=result.lessons
    )

# Ralph periodically reviews code
# and optimizes based on memory patterns
```

---

## Documentation Files

### Quick Start
**File:** `QUICK_START_REPOSITORIES.md` (10KB)
**Use when:** You want to get started in 5 minutes

**Run:**
```bash
cat QUICK_START_REPOSITORIES.md | less
```

### Detailed Guide
**File:** `REPOSITORY_USAGE_GUIDE.md` (29KB)
**Use when:** You need detailed technical information

**Run:**
```bash
cat REPOSITORY_USAGE_GUIDE.md | less
```

### Verification
**File:** `verify_repositories.sh`
**Use when:** You want to verify everything works

**Run:**
```bash
./verify_repositories.sh
```

### Summary
**File:** `REPOSITORY_SUMMARY.md` (9.8KB)
**Use when:** You want a visual overview

**Run:**
```bash
cat REPOSITORY_SUMMARY.md
```

---

## Configuration Files

### SimpleMem Config
**File:** `memory/config.py`

**Key settings:**
- Free tier LLM models (OpenRouter)
- Ollama embeddings (local)
- Parallel processing enabled
- 8 workers for memory building
- 4 workers for retrieval

### Environment
**File:** `memory/.env` (you create this)

**Required:**
```bash
OPENROUTER_API_KEY=your-key-here
```

**Get key:** https://openrouter.ai/keys (free tier)

---

## Testing & Verification

### Quick Test (30 seconds)
```bash
# Test SimpleMem
cd /Users/britebrt/GOBOT/memory
./setup.sh
source venv/bin/activate
python main.py

# Test Ralph
cd /Users/britebrt/GOBOT/scripts/ralph
./ralph.sh 3  # 3 iterations
```

### Full Verification (15 seconds)
```bash
cd /Users/britebrt/GOBOT
./verify_repositories.sh
```

---

## Common Issues

### SimpleMem won't start
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

### Ralph won't run
```bash
# Check PRD format
cd scripts/ralph
jq . prd.json

# Check dependencies
which amp jq git
```

### Memory queries are slow
```bash
# Check embedding is local
grep EMBEDDING_MODEL memory/config.py
# Should be: ollama:nomic-embed-text
```

---

## Next Steps

### Today
1. Run `./verify_repositories.sh`
2. Read `QUICK_START_REPOSITORIES.md`
3. Set up SimpleMem: `cd memory && ./setup.sh`
4. Test with one trade

### This Week
1. Integrate into your trading bot
2. Store every trade to memory
3. Query memory before trading
4. Try Ralph with simple task

### This Month
1. Build memory database (100+ trades)
2. Automate feature dev with Ralph
3. Document patterns in AGENTS.md
4. Refine strategies based on memory

---

## Quick Command Reference

```bash
# SimpleMem
cd memory
./setup.sh                          # Setup
source venv/bin/activate            # Activate
python main.py                      # Test
python trading_memory.py --help     # CLI help

# Ralph
cd scripts/ralph
./ralph.sh 10                       # Run 10 iterations
tail -f progress.txt                # Watch progress
ls -la archive/                     # View archives

# Verification
cd /Users/britebrt/GOBOT
./verify_repositories.sh            # Health check
```

---

## Summary

✅ **SimpleMem** - Fully operational memory system
✅ **Ralph** - Fully operational autonomous agent
✅ **Integration** - Go bridge implemented
✅ **Documentation** - 48KB comprehensive guides
✅ **Verification** - Automated testing script
✅ **Status** - Ready for production use

**To get started:**
```bash
cd /Users/britebrt/GOBOT
./verify_repositories.sh
```

Then read `QUICK_START_REPOSITORIES.md` for your first 5 minutes!
