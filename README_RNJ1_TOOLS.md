# RNJ-1 Model with Tools Integration

Your `rnj-1:8b` model is now fully equipped with the tools mentioned in your system prompt!

## ✅ What's Been Set Up

### Model Verification
- ✅ **Ollama 0.13.3-rc0** - Latest version with tool support
- ✅ **rnj-1:8b model** - Confirmed tool capability enabled
- ✅ **Architecture**: gemma3, 8.3B parameters, 32K context

### Implemented Tools
1. **web_search(query: str) → str** - Search web and get concise results
2. **store_memory(text: str, tags: list[str]) → str** - Store facts with tags
3. **search_memory(query: str, k: int = 5) → list[dict]** - Search stored memories
4. **write_note(text: str) → str** - Save longer documents
5. **read_notes() → list[dict]** - Retrieve saved notes

## 🚀 Quick Start

### Interactive Mode (Recommended)
```bash
python3 rnj1_with_tools.py
```

### Batch Mode
```bash
python3 rnj1_with_tools.py "Search for latest AI news and summarize it"
```

### Direct Tool Usage
```bash
source ollama_tools_env/bin/activate

# Store information
python3 ollama_tools_integration.py store_memory --text "User likes Python" --tags preference coding

# Search memory
python3 ollama_tools_integration.py search_memory --query "python" --k 3

# Web search
python3 ollama_tools_integration.py web_search --query "latest AI developments"

# Write note
python3 ollama_tools_integration.py write_note --text "Project plan: build AI assistant"

# Read notes
python3 ollama_tools_integration.py read_notes
```

## 📁 Files Created

- `ollama_tools_integration.py` - Core tool implementation
- `rnj1_with_tools.py` - Main integration script with interactive mode
- `demo_rnj1_tools.py` - Demo and verification script
- `rnj1_tools_config.json` - Tool configuration schema
- `ollama_tools_env/` - Virtual environment with dependencies
- `ollama_tools.db` - SQLite database for memory storage
- `ollama_notes/` - Directory for saved notes

## 🧠 How It Works

### Tool Calling Format
When using the interactive mode, the model calls tools using this format:
```
TOOL_CALL: {"tool": "web_search", "args": {"query": "latest AI news"}}
```

### Storage
- **Memory**: SQLite database (`ollama_tools.db`) for quick facts
- **Notes**: Markdown files (`ollama_notes/`) for longer content

### Web Search
Uses DuckDuckGo API for web searches with:
- Abstract extraction
- Related topics
- Error handling and timeouts

## 🔧 Customization

### Modify Tools
Edit `ollama_tools_integration.py` to:
- Change web search provider
- Add memory categories
- Customize note formatting
- Add new tools

### Database Location
Change the database path in the `OllamaToolManager` constructor:
```python
tool_manager = OllamaToolManager(db_path="custom_path.db", notes_dir="custom_notes")
```

## 📋 Usage Examples

### Research Assistant
```
You: Research the latest developments in quantum computing
Assistant: TOOL_CALL: {"tool": "web_search", "args": {"query": "quantum computing latest developments 2025"}}
[Returns search results, summarizes key findings]
```

### Personal Memory
```
You: Remember that I prefer vim over emacs
Assistant: TOOL_CALL: {"tool": "store_memory", "args": {"text": "User prefers vim over emacs for text editing", "tags": ["preference", "editor"]}}}
```

### Project Planning
```
You: Create a plan for building a web app
Assistant: TOOL_CALL: {"tool": "write_note", "args": {"text": "# Web App Development Plan\n\n## Phase 1: Setup\n- Choose framework\n- Setup repository\n..."}}}
```

## 🔍 Verification

Run the demo to verify everything works:
```bash
python3 demo_rnj1_tools.py
```

## 🐛 Troubleshooting

### Common Issues

1. **Ollama not running**
   ```bash
   ollama serve
   ```

2. **Python dependencies missing**
   ```bash
   source ollama_tools_env/bin/activate
   pip install -r requirements.txt
   ```

3. **Model not found**
   ```bash
   ollama pull rnj-1:8b
   ```

4. **Permission errors**
   ```bash
   chmod +x *.py
   ```

### Debug Mode
Add debug prints by setting environment variable:
```bash
export DEBUG=1
python3 rnj1_with_tools.py
```

## 🚀 Next Steps

1. **Start using it**: Run `python3 rnj1_with_tools.py`
2. **Store preferences**: Use `store_memory` for your preferences
3. **Build knowledge base**: Create notes for projects
4. **Customize**: Modify tools to fit your workflow

## 📚 System Prompt

Your rnj-1 model now follows this behavior pattern:
- ✅ Goal-directed problem solver
- ✅ Step-by-step thinking
- ✅ Tool usage for external information
- ✅ Memory for long-term knowledge
- ✅ Notes for longer artifacts
- ✅ Concise, relevant responses

Your rnj-1 model is ready to be the disciplined, tool-using AI agent you described in your system prompt! 🎉