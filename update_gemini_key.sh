#!/bin/bash

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔑 GEMINI API KEY UPDATE HELPER"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Your current GEMINI_API_KEY is:"
grep "GEMINI_API_KEY=" /Users/britebrt/GOBOT/.env | grep -v "^#"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📝 TO UPDATE YOUR GEMINI API KEY:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "STEP 1: Get your free API key"
echo "   https://makersuite.google.com/app/apikey"
echo ""
echo "STEP 2: Open .env file in your editor"
echo "   nano /Users/britebrt/GOBOT/.env"
echo "   OR"
echo "   vim /Users/britebrt/GOBOT/.env"
echo ""
echo "STEP 3: Find this line:"
echo "   GEMINI_API_KEY=YOUR_GEMINI_API_KEY_HERE"
echo ""
echo "STEP 4: Replace with your actual key:"
echo "   GEMINI_API_KEY=AIzaSyB... (your real key)"
echo ""
echo "STEP 5: Save and close the file"
echo ""
echo "STEP 6: Restart your bot:"
echo "   ./gobot-gemini"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ ALTERNATIVE: Update key interactively"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
read -p "Enter your Gemini API key (or press Enter to cancel): " NEW_KEY

if [ ! -z "$NEW_KEY" ]; then
    # Backup current .env
    cp /Users/britebrt/GOBOT/.env /Users/britebrt/GOBOT/.env.backup.$(date +%Y%m%d_%H%M%S)
    
    # Update the key
    sed -i '' "s/^GEMINI_API_KEY=.*/GEMINI_API_KEY=$NEW_KEY/" /Users/britebrt/GOBOT/.env
    
    echo ""
    echo "✅ API key updated successfully!"
    echo ""
    echo "Restart the bot to use Gemini:"
    echo "   ./gobot-gemini"
else
    echo ""
    echo "No changes made. Current system will continue using local model."
fi
