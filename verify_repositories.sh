#!/bin/bash
# Repository Verification Script
# Tests both SimpleMem and Ralph implementations

set -e

echo "╔═══════════════════════════════════════════════════════╗"
echo "║      GOBOT Repository Implementation Verification     ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

FAILED=0

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

check_pass() {
    echo -e "${GREEN}✓${NC} $1"
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    FAILED=1
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Test 1: SimpleMem Directory Structure
echo ""
echo "Testing SimpleMem Implementation..."
echo "-----------------------------------"

if [ -d "memory" ]; then
    check_pass "Memory directory exists"
else
    check_fail "Memory directory not found"
fi

if [ -f "memory/main.py" ]; then
    check_pass "SimpleMem main.py exists"
else
    check_fail "SimpleMem main.py not found"
fi

if [ -f "memory/trading_memory.py" ]; then
    check_pass "Trading memory wrapper exists"
else
    check_fail "Trading memory wrapper not found"
fi

if [ -f "memory/config.py" ]; then
    check_pass "Configuration file exists"
else
    check_fail "Configuration file not found"
fi

if [ -f "memory/setup.sh" ]; then
    check_pass "Setup script exists"
else
    check_fail "Setup script not found"
fi

# Test 2: Python Environment
echo ""
echo "Testing Python Environment..."
echo "------------------------------"

if command -v python3 &> /dev/null; then
    PYTHON=python3
    check_pass "Python 3 available: $(python3 --version)"
elif command -v python &> /dev/null; then
    PYTHON=python
    check_pass "Python available: $(python --version)"
else
    check_fail "Python not found"
    PYTHON=""
fi

if [ -n "$PYTHON" ] && [ -d "memory/venv" ]; then
    check_pass "Virtual environment exists"
elif [ -n "$PYTHON" ]; then
    check_warn "Virtual environment not found - run memory/setup.sh"
fi

# Test 3: Ollama
echo ""
echo "Testing Ollama..."
echo "-----------------"

if command -v ollama &> /dev/null; then
    check_pass "Ollama installed: $(ollama --version)"
    
    if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        check_pass "Ollama is running"
        
        if ollama list 2>/dev/null | grep -q "nomic-embed-text"; then
            check_pass "Embedding model available"
        else
            check_warn "nomic-embed-text model not found - will install on setup"
        fi
    else
        check_warn "Ollama not running - start with: ollama serve"
    fi
else
    check_warn "Ollama not installed - install from https://ollama.ai"
fi

# Test 4: Configuration
echo ""
echo "Testing Configuration..."
echo "------------------------"

if [ -f "memory/.env" ]; then
    check_pass ".env file exists"
    
    if grep -q "OPENROUTER_API_KEY" memory/.env; then
        API_KEY=$(grep "OPENROUTER_API_KEY" memory/.env | cut -d '=' -f2)
        if [ "$API_KEY" != "your-openrouter-api-key-here" ] && [ -n "$API_KEY" ]; then
            check_pass "OpenRouter API key configured"
        else
            check_warn "OpenRouter API key not configured (using example)"
        fi
    fi
else
    check_warn ".env file not found - copy from .env.example"
fi

# Test 5: Core Files
echo ""
echo "Testing Core Implementation Files..."
echo "------------------------------------"

CORE_FILES=(
    "memory/core/memory_builder.py"
    "memory/core/hybrid_retriever.py"
    "memory/core/answer_generator.py"
    "memory/database/vector_store.py"
    "memory/models/memory_entry.py"
    "memory/utils/llm_client.py"
    "memory/utils/embedding.py"
)

for file in "${CORE_FILES[@]}"; do
    if [ -f "$file" ]; then
        check_pass "$(basename $file) exists"
    else
        check_fail "$(basename $file) not found"
    fi
done

# Test 6: Go Integration
echo ""
echo "Testing Go Integration..."
echo "-------------------------"

if [ -f "internal/memory/memory.go" ]; then
    check_pass "Go memory integration exists"
else
    check_fail "Go memory integration not found"
fi

# Test 7: Ralph Implementation
echo ""
echo "Testing Ralph Implementation..."
echo "-------------------------------"

if [ -d "scripts/ralph" ]; then
    check_pass "Ralph directory exists"
else
    check_fail "Ralph directory not found"
fi

if [ -f "scripts/ralph/ralph.sh" ]; then
    check_pass "Ralph.sh script exists"
else
    check_fail "Ralph.sh script not found"
fi

if [ -x "scripts/ralph/ralph.sh" ]; then
    check_pass "Ralph.sh is executable"
else
    check_warn "Ralph.sh is not executable (fix with: chmod +x)"
fi

if [ -f "scripts/ralph/prompt.md" ]; then
    check_pass "Ralph prompt/instructions exists"
else
    check_fail "Ralph prompt not found"
fi

# Test 8: Dependencies
echo ""
echo "Testing Dependencies..."
echo "-----------------------"

if [ -f "memory/requirements.txt" ]; then
    check_pass "requirements.txt exists ($(wc -l < memory/requirements.txt) packages)"
else
    check_fail "requirements.txt not found"
fi

# Test 9: Quick Import Test
echo ""
echo "Testing Python Imports..."
echo "-------------------------"

if [ -d "memory/venv" ]; then
    cd memory
    source venv/bin/activate
    
    if $PYTHON -c "import config" 2>/dev/null; then
        check_pass "Config module imports successfully"
    else
        check_fail "Config module import failed"
    fi
    
    if $PYTHON -c "from main import SimpleMemSystem" 2>/dev/null; then
        check_pass "SimpleMemSystem imports successfully"
    else
        check_fail "SimpleMemSystem import failed"
    fi
    
    cd ..
else
    check_warn "Virtual environment not set up - skipping import tests"
fi

# Test 10: Database Directory
echo ""
echo "Testing Database Setup..."
echo "-------------------------"

if [ -d "memory/lancedb_data" ]; then
    check_pass "Database directory exists"
    
    DB_SIZE=$(du -sh memory/lancedb_data 2>/dev/null | cut -f1)
    check_pass "Database size: $DB_SIZE"
else
    check_warn "Database directory not found - will create on first use"
fi

# Summary
echo ""
echo "╔═══════════════════════════════════════════════════════╗"
echo "║                  Verification Summary                   ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All critical checks passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Run: cd memory && ./setup.sh (if not done)"
    echo "2. Edit: memory/.env with your OpenRouter API key"
    echo "3. Start Ollama: ollama serve"
    echo "4. Test SimpleMem: python main.py"
    echo "5. Test Ralph: cd scripts/ralph && ./ralph.sh 1"
    echo ""
    echo "For detailed usage guide:"
    echo "cat REPOSITORY_USAGE_GUIDE.md"
    exit 0
else
    echo -e "${RED}✗ Some checks failed${NC}"
    echo ""
    echo "Please review the errors above and:"
    echo "1. Ensure all files are present"
    echo "2. Install missing dependencies"
    echo "3. Run setup scripts as needed"
    echo ""
    echo "For help, see: REPOSITORY_USAGE_GUIDE.md"
    exit 1
fi
