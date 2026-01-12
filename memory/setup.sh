#!/bin/bash
# SimpleMem Setup Script for GOBOT

set -e

echo "=========================================="
echo "GOBOT Memory System Setup"
echo "=========================================="

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Check Python
echo ""
echo "1. Checking Python..."
if command -v python3 &> /dev/null; then
    PYTHON=python3
elif command -v python &> /dev/null; then
    PYTHON=python
else
    echo "❌ Python not found. Please install Python 3.10+"
    exit 1
fi
echo "✅ Python: $($PYTHON --version)"

# Create virtual environment
echo ""
echo "2. Setting up virtual environment..."
if [ ! -d "venv" ]; then
    $PYTHON -m venv venv
    echo "✅ Created virtual environment"
else
    echo "✅ Virtual environment exists"
fi

# Activate venv
source venv/bin/activate

# Install dependencies
echo ""
echo "3. Installing dependencies..."
pip install --upgrade pip -q
pip install -r requirements.txt -q
pip install requests -q  # For Ollama client
echo "✅ Dependencies installed"

# Check Ollama
echo ""
echo "4. Checking Ollama..."
if command -v ollama &> /dev/null; then
    echo "✅ Ollama installed"
    
    # Check if Ollama is running
    if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        echo "✅ Ollama is running"
        
        # Check for embedding model
        if ollama list | grep -q "nomic-embed-text"; then
            echo "✅ nomic-embed-text model available"
        else
            echo "⚠️  nomic-embed-text not found. Installing..."
            ollama pull nomic-embed-text
        fi
    else
        echo "⚠️  Ollama not running. Start with: ollama serve"
    fi
else
    echo "⚠️  Ollama not installed."
    echo "   Install from: https://ollama.ai"
    echo "   Then run: ollama pull nomic-embed-text"
fi

# Setup environment file
echo ""
echo "5. Checking environment..."
if [ ! -f ".env" ]; then
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo "⚠️  Created .env from example. Please edit with your API keys."
    else
        echo "⚠️  No .env file. Create one with OPENROUTER_API_KEY."
    fi
else
    echo "✅ .env file exists"
fi

# Create database directory
echo ""
echo "6. Setting up database directory..."
mkdir -p lancedb_data
echo "✅ Database directory ready"

# Test the system
echo ""
echo "7. Testing SimpleMem..."
$PYTHON -c "
import sys
sys.path.insert(0, '.')
try:
    import config
    print('✅ Config loaded')
    print(f'   LLM Model: {config.LLM_MODEL}')
    print(f'   Embedding: {config.EMBEDDING_MODEL}')
    print(f'   OpenRouter: {config.OPENAI_BASE_URL}')
except Exception as e:
    print(f'❌ Config error: {e}')
"

echo ""
echo "=========================================="
echo "Setup Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Edit memory/.env with your OpenRouter API key"
echo "2. Start Ollama: ollama serve"
echo "3. Test: cd memory && source venv/bin/activate && python trading_memory.py ask --question 'Hello'"
echo ""
