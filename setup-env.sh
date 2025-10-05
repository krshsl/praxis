#!/bin/bash

# Setup environment files from examples
# This script copies env.example files to .env files for both backend and frontend

echo "Setting up environment files..."

# Check if backend .env already exists
if [ -f "backend/.env" ]; then
    echo "Backend .env already exists. Skipping..."
else
    if [ -f "backend/env.example" ]; then
        cp backend/env.example backend/.env
        echo "Created backend/.env from env.example"
    else
        echo "Warning: backend/env.example not found"
    fi
fi

# Check if frontend .env already exists
if [ -f "frontend/.env" ]; then
    echo "Frontend .env already exists. Skipping..."
else
    if [ -f "frontend/env.example" ]; then
        cp frontend/env.example frontend/.env
        echo "Created frontend/.env from env.example"
    else
        echo "Warning: frontend/env.example not found"
    fi
fi

echo ""
echo "Environment files setup complete!"
echo ""
echo "Next steps:"
echo "1. Edit backend/.env and frontend/.env with your actual configuration values"
echo "2. Make sure to set your Supabase URL and keys"
echo "3. Set your AI service API keys (Gemini, ElevenLabs)"
echo "4. Set a secure JWT secret"
echo ""
echo "Required Supabase configuration:"
echo "- SUPABASE_URL: Your Supabase project URL"
echo "- SUPABASE_ANON_KEY: Your Supabase anonymous key"
echo "- SUPABASE_SERVICE_ROLE_KEY: Your Supabase service role key"
echo ""
echo "You can find these values in your Supabase project dashboard under Settings > API"
