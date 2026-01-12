"""
SimpleMem Configuration for GOBOT
Configured with:
- OpenRouter API (OpenAI-compatible endpoint) for LLM - FREE TIER by default
- Ollama for local embeddings
- OpenInference-compatible tracing support
"""

import os

# ============================================================================
# LLM Configuration - OpenRouter (OpenAI-compatible) - FREE TIER DEFAULT
# ============================================================================

# OpenRouter API Keys (Primary + Backup for rate limit resilience)
# Get your keys from: https://openrouter.ai/keys (free tier available)
OPENAI_API_KEY = os.getenv("OPENROUTER_API_KEY", "your-openrouter-api-key-here")
OPENAI_API_KEY_BACKUP = os.getenv("OPENROUTER_API_KEY_BACKUP", None)

# List of all available API keys (for rotation on rate limit)
def get_api_keys() -> list:
    """Get list of available API keys for rotation."""
    keys = [OPENAI_API_KEY]
    if OPENAI_API_KEY_BACKUP and OPENAI_API_KEY_BACKUP != "your-backup-api-key-here":
        keys.append(OPENAI_API_KEY_BACKUP)
    return keys

# OpenRouter Base URL (OpenAI-compatible endpoint)
OPENAI_BASE_URL = os.getenv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1")

# ============================================================================
# FREE TIER MODELS - No cost, no credit card required
# ============================================================================
# These models are free on OpenRouter. Rotate if rate limited.

FREE_MODELS = [
    "meta-llama/llama-3.2-3b-instruct:free",      # Fast, good for simple tasks
    "meta-llama/llama-3.1-8b-instruct:free",      # Better reasoning
    "google/gemma-2-9b-it:free",                   # Google's free model
    "mistralai/mistral-7b-instruct:free",          # Mistral free tier
    "qwen/qwen-2-7b-instruct:free",                # Qwen free tier
    "microsoft/phi-3-mini-128k-instruct:free",     # Microsoft Phi-3
    "huggingfaceh4/zephyr-7b-beta:free",           # HuggingFace Zephyr
]

# Default to the best free model for trading analysis
LLM_MODEL = os.getenv("LLM_MODEL", "meta-llama/llama-3.1-8b-instruct:free")

# Fallback models (tried in order if primary fails)
LLM_FALLBACK_MODELS = [
    "google/gemma-2-9b-it:free",
    "qwen/qwen-2-7b-instruct:free",
    "mistralai/mistral-7b-instruct:free",
]


# ============================================================================
# Embedding Configuration - Ollama (Local)
# ============================================================================

# Ollama Embedding Model
# Run: ollama pull nomic-embed-text (or mxbai-embed-large, all-minilm, etc.)
EMBEDDING_MODEL = os.getenv("EMBEDDING_MODEL", "ollama:nomic-embed-text")
EMBEDDING_DIMENSION = int(os.getenv("EMBEDDING_DIMENSION", "768"))  # nomic-embed-text = 768, mxbai-embed-large = 1024
EMBEDDING_CONTEXT_LENGTH = 8192

# Ollama endpoint for embeddings
OLLAMA_BASE_URL = os.getenv("OLLAMA_BASE_URL", "http://localhost:11434")


# ============================================================================
# Advanced LLM Features
# ============================================================================

# Enable deep thinking mode (for compatible models like Qwen)
ENABLE_THINKING = False

# Enable streaming responses
USE_STREAMING = True

# Enable JSON format mode
USE_JSON_FORMAT = False


# ============================================================================
# Memory Building Parameters
# ============================================================================

# Number of dialogues per window
WINDOW_SIZE = 40

# Window overlap size (for context continuity)
OVERLAP_SIZE = 2


# ============================================================================
# Retrieval Parameters
# ============================================================================

# Max entries returned by semantic search (vector similarity)
SEMANTIC_TOP_K = 25

# Max entries returned by keyword search (BM25 matching)
KEYWORD_TOP_K = 5

# Max entries returned by structured search (metadata filtering)
STRUCTURED_TOP_K = 5


# ============================================================================
# Database Configuration
# ============================================================================

# Path to LanceDB storage (relative to GOBOT root)
LANCEDB_PATH = os.getenv("LANCEDB_PATH", "./memory/lancedb_data")

# Memory table name
MEMORY_TABLE_NAME = "gobot_memory"


# ============================================================================
# Parallel Processing Configuration
# ============================================================================

# Memory Building Parallel Processing
ENABLE_PARALLEL_PROCESSING = True
MAX_PARALLEL_WORKERS = 8

# Retrieval Parallel Processing  
ENABLE_PARALLEL_RETRIEVAL = True
MAX_RETRIEVAL_WORKERS = 4

# Planning and Reflection Configuration
ENABLE_PLANNING = True
ENABLE_REFLECTION = True
MAX_REFLECTION_ROUNDS = 2


# ============================================================================
# GOBOT-Specific Memory Settings
# ============================================================================

# Trade memory retention (days)
TRADE_MEMORY_RETENTION_DAYS = int(os.getenv("TRADE_MEMORY_RETENTION_DAYS", "90"))

# Market insight memory retention (days)
MARKET_MEMORY_RETENTION_DAYS = int(os.getenv("MARKET_MEMORY_RETENTION_DAYS", "30"))

# Memory categories for trading bot
MEMORY_CATEGORIES = [
    "trade_execution",      # Executed trades and outcomes
    "market_insight",       # Market observations and patterns
    "strategy_learning",    # Strategy performance learnings
    "risk_event",           # Risk events and responses
    "system_event",         # System events and errors
]


# ============================================================================
# Judge Configuration (for evaluation - optional)
# ============================================================================

JUDGE_API_KEY = os.getenv("JUDGE_API_KEY", OPENAI_API_KEY)
JUDGE_BASE_URL = os.getenv("JUDGE_BASE_URL", OPENAI_BASE_URL)
JUDGE_MODEL = os.getenv("JUDGE_MODEL", LLM_MODEL)
JUDGE_ENABLE_THINKING = False
JUDGE_USE_STREAMING = False
JUDGE_TEMPERATURE = 0.3


# ============================================================================
# OpenInference Tracing Configuration
# ============================================================================
# OpenInference is an open standard for AI observability
# Compatible with: Arize Phoenix, LangSmith, etc.

ENABLE_OPENINFERENCE = os.getenv("ENABLE_OPENINFERENCE", "false").lower() == "true"
OPENINFERENCE_ENDPOINT = os.getenv("OPENINFERENCE_ENDPOINT", "http://localhost:6006/v1/traces")

# Trace settings
TRACE_LLM_CALLS = True
TRACE_EMBEDDINGS = True
TRACE_MEMORY_OPERATIONS = True


# ============================================================================
# OpenRouter Request Headers (for free tier optimization)
# ============================================================================
# These headers help OpenRouter route requests properly

OPENROUTER_HEADERS = {
    "HTTP-Referer": os.getenv("OPENROUTER_REFERER", "https://github.com/gobot"),
    "X-Title": os.getenv("OPENROUTER_TITLE", "GOBOT Trading Bot"),
}


# ============================================================================
# Model Selection Helper
# ============================================================================

def get_model_with_fallback(preferred_model: str = None) -> str:
    """
    Get model name, defaulting to free tier.
    Returns the preferred model or the default free model.
    """
    if preferred_model:
        return preferred_model
    return LLM_MODEL


def get_fallback_models() -> list:
    """Get list of fallback models to try if primary fails."""
    return LLM_FALLBACK_MODELS.copy()
