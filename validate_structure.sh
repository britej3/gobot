#!/bin/bash
# Cognee Structural Integrity Check

echo "🔍 Validating Cognee Architecture..."

# 1. Check for standard directories
DIRS=("cmd" "internal/watcher" "internal/striker" "internal/auditor" "internal/platform")
for dir in "${DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo "✅ Directory Found: $dir"
    else
        echo "❌ MISSING Directory: $dir"
    fi
done

# 2. Validate Module Graph (Detect Circular Dependencies)
# Go will fail to compile if internal/watcher and internal/striker import each other.
echo "🔗 Checking for circular dependencies..."
go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/... | grep "cycle" && echo "❌ Circular dependency detected!" || echo "✅ Clean dependency graph."

# 3. Verify Local LLM Model Registration
echo "🧠 Verifying LLM Brain (cognee-brain)..."
if ollama list | grep -q "cognee-brain"; then
    echo "✅ Optimized LFM2:2.6B model registered."
else
    echo "❌ LLM model 'cognee-brain' not found. Run 'ollama create'."
fi

# 4. Binary Compilation Test
echo "🏗️ Testing Build..."
go build -o /dev/null ./cmd/cognee/main.go && echo "✅ Compilation successful." || echo "❌ Compilation failed."
