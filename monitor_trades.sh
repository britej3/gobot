#!/bin/bash

LOG_FILE="${1:-gobot_gemini.log}"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 GOBOT TRADE MONITOR"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Monitoring: $LOG_FILE"
echo "Press Ctrl+C to stop"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

tail -f "$LOG_FILE" 2>/dev/null | while read line; do
    # Extract timestamp
    timestamp=$(echo "$line" | jq -r '.time' 2>/dev/null)
    msg=$(echo "$line" | jq -r '.msg' 2>/dev/null)
    
    # Color coding
    if echo "$line" | grep -q "executed successfully"; then
        echo "✅ $(date +%H:%M:%S) | $msg"
    elif echo "$line" | grep -q "No actionable targets"; then
        echo "⏸️  $(date +%H:%M:%S) | $msg"
    elif echo "$line" | grep -q "executing trade"; then
        echo "🎯 $(date +%H:%M:%S) | $msg"
    elif echo "$line" | grep -q "analyzing\|Processing"; then
        echo "🔍 $(date +%H:%M:%S) | $msg"
    elif echo "$line" | grep -q "decision.*BUY\|decision.*SELL"; then
        echo "💡 $(date +%H:%M:%S) | $msg"
    elif echo "$line" | grep -q "error\|Error\|ERROR"; then
        echo "❌ $(date +%H:%M:%S) | $msg"
    fi
done
