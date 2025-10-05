#!/bin/bash

# Development script for Praxis
# This script starts the development environment with live reloading

echo "🚀 Starting Praxis Development Environment..."

# Check if .env file exists
if [ ! -f .env ]; then
    echo "⚠️  No .env file found. Creating from examples..."
    if [ -f backend/env.example ]; then
        cp backend/env.example .env
        echo "✅ Created .env from backend/env.example"
    fi
    if [ -f frontend/env.example ]; then
        echo "✅ Frontend env.example available"
    fi
    echo "📝 Please update .env with your actual values"
fi

# Start development environment
echo "🐳 Starting Docker containers with live reloading..."
docker-compose -f docker-compose.dev.yml up --build

echo "✅ Development environment started!"
echo "🌐 Frontend: http://localhost:5173"
echo "🔧 Backend: http://localhost:8080"
echo "🌍 Proxy: http://localhost"
echo ""
echo "📝 Changes to source code will be automatically reloaded!"
echo "🛑 Press Ctrl+C to stop"
