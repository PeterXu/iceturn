#!/bin/bash

# Quick Docker deployment script for IceTurn

set -e

echo "=== IceTurn Docker Deployment ==="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Detect Docker Compose command
if command -v docker-compose &> /dev/null; then
    DC="docker-compose"
else
    DC="docker compose"
fi

echo "Using: $DC"

# Ask for PUBLIC_IP
echo ""
echo "Please enter your PUBLIC_IP (reachable from browser):"
echo "Examples:"
echo "  - 192.168.1.100 (LAN IP)"
echo "  - your-domain.com (domain)"
echo ""
read -p "PUBLIC_IP: " PUBLIC_IP

if [ -z "$PUBLIC_IP" ]; then
    echo "❌ PUBLIC_IP is required"
    exit 1
fi

# Create .env file
echo ""
echo "Creating .env.docker file..."
cat > .env.docker << EOF
REALM=magpie.local
PUBLIC_IP=${PUBLIC_IP}
STUN_PENDING_TIMEOUT=30s
TURN_CREDENTIALS=magpie:magpie
EOF

echo "✅ Configuration saved to .env.docker"

# Build and start
echo ""
echo "Building Docker image..."
$DC build

echo ""
echo "Starting IceTurn server..."
$DC up -d

# Wait for startup
sleep 3

# Check if running
if $DC ps | grep -q "Up"; then
    echo ""
    echo "=========================================="
    echo "✅ IceTurn is running!"
    echo "=========================================="
    echo ""
    echo "Server details:"
    echo "  - Address: ${PUBLIC_IP}:3478"
    echo "  - Realm: magpie.local"
    echo "  - Username: magpie"
    echo "  - Password: magpie"
    echo ""
    echo "Useful commands:"
    echo "  - View logs: $DC logs -f"
    echo "  - Stop: $DC down"
    echo "  - Restart: $DC restart"
    echo ""
    echo "Integration with Magpie:"
    echo "  Set these environment variables in Magpie:"
    echo "    MAGPIE_HOST_IP=${PUBLIC_IP}"
    echo "    MAGPIE_TURN_USER=magpie"
    echo "    MAGPIE_TURN_PASS=magpie"
    echo ""
else
    echo ""
    echo "❌ IceTurn failed to start. Check logs:"
    echo "  $DC logs"
    exit 1
fi