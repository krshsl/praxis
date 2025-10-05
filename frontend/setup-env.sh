#!/bin/bash

# Create .env file for frontend if it doesn't exist
if [ ! -f .env ]; then
    echo "Creating .env file..."
    cat > .env << EOF
# Backend API Configuration
VITE_API_URL=http://localhost:8080/api/v1

# WebSocket Configuration
VITE_WS_URL=localhost:8080
EOF
    echo ".env file created successfully!"
else
    echo ".env file already exists"
fi
