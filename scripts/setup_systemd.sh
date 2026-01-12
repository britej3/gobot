#!/bin/bash
# Systemd Service Setup Script for Cognee
# This script sets up Cognee as a system daemon

set -e

echo "🔧 Setting up Cognee systemd service..."

# Check if running as root (required for systemctl)
if [ "$EUID" -ne 0 ]; then 
   echo "⚠️  Please run as root or with sudo"
   echo "Usage: sudo ./scripts/setup_systemd.sh"
   exit 1
fi

# Configuration
SERVICE_NAME="cognee"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CURRENT_USER=${SUDO_USER:-$USER}
PROJECT_DIR="/home/${CURRENT_USER}/GOBOT"

# Check if service already exists
if [ -f "$SERVICE_FILE" ]; then
    echo "⚠️  Service file already exists at $SERVICE_FILE"
    read -p "Do you want to overwrite it? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "❌ Setup cancelled"
        exit 1
    fi
fi

# Create logs directory
echo "📁 Creating logs directory..."
sudo -u $CURRENT_USER mkdir -p "${PROJECT_DIR}/logs"

# Set proper permissions on .env
echo "🔐 Setting permissions on .env file..."
sudo -u $CURRENT_USER chmod 600 "${PROJECT_DIR}/.env"

# Copy service file
echo "📄 Copying service file..."
cp "${PROJECT_DIR}/cognee.service" "$SERVICE_FILE"

# Replace placeholder with actual user
sed -i "s/YOUR_USER/${CURRENT_USER}/g" "$SERVICE_FILE"

# Verify gobot binary exists
if [ ! -f "${PROJECT_DIR}/gobot" ]; then
    echo "⚠️  gobot binary not found at ${PROJECT_DIR}/gobot"
    echo "Run 'go build -o gobot ./cmd/cognee' first"
    exit 1
fi

# Set binary permissions
sudo -u $CURRENT_USER chmod +x "${PROJECT_DIR}/gobot"

# Reload systemd
echo "🔄 Reloading systemd..."
systemctl daemon-reload

# Enable service to start on boot
echo "✅ Enabling service to start on boot..."
systemctl enable $SERVICE_NAME

echo ""
echo "✅ Systemd service setup completed!"
echo ""
echo "🚀 To start Cognee:"
echo "   sudo systemctl start $SERVICE_NAME"
echo ""
echo "📊 To check status:"
echo "   sudo systemctl status $SERVICE_NAME"
echo ""
echo "📝 To view logs:"
echo "   journalctl -u $SERVICE_NAME -f"
echo ""
echo "🛑 To stop:"
echo "   sudo systemctl stop $SERVICE_NAME"
echo ""
echo "🔄 To restart:"
echo "   sudo systemctl restart $SERVICE_NAME"
echo ""
echo "💡 Note: The service will automatically restart on crashes (max 5 times in 10 minutes)"
echo "   This prevents IP bans from Binance during error loops."
