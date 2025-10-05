# Environment Setup Guide

This guide will help you set up your environment variables for the Praxis application.

## Quick Setup

Run the setup script to create your `.env` files from the examples:

```bash
./setup-env.sh
```

This will create:
- `backend/.env` from `backend/env.example`
- `frontend/.env` from `frontend/env.example`

## Manual Setup

If you prefer to set up manually or the script doesn't work:

### Backend Environment Variables

Copy `backend/env.example` to `backend/.env` and update the values:

```bash
cp backend/env.example backend/.env
```

Required variables:
- `SUPABASE_URL`: Your Supabase project URL
- `SUPABASE_ANON_KEY`: Your Supabase anonymous key  
- `SUPABASE_SERVICE_ROLE_KEY`: Your Supabase service role key
- `GEMINI_API_KEY`: Your Google Gemini API key
- `ELEVENLABS_API_KEY`: Your ElevenLabs API key
- `JWT_SECRET`: A secure random string for JWT signing

### Frontend Environment Variables

Copy `frontend/env.example` to `frontend/.env` and update the values:

```bash
cp frontend/env.example frontend/.env
```

Required variables:
- `VITE_SUPABASE_URL`: Your Supabase project URL
- `VITE_SUPABASE_ANON_KEY`: Your Supabase anonymous key
- `VITE_API_URL`: Backend API URL (default: http://localhost:8080/api/v1)
- `VITE_WEBSOCKET_URL`: WebSocket URL (default: ws://localhost:8080/api/v1/ws)

## Getting Supabase Configuration

1. Go to your [Supabase Dashboard](https://supabase.com/dashboard)
2. Select your project
3. Go to Settings > API
4. Copy the following values:
   - **Project URL** → `SUPABASE_URL` / `VITE_SUPABASE_URL`
   - **anon public** → `SUPABASE_ANON_KEY` / `VITE_SUPABASE_ANON_KEY`
   - **service_role secret** → `SUPABASE_SERVICE_ROLE_KEY`

## Getting AI Service API Keys

### Google Gemini API Key
1. Go to [Google AI Studio](https://aistudio.google.com/)
2. Create a new API key
3. Copy the key to `GEMINI_API_KEY`

### ElevenLabs API Key
1. Go to [ElevenLabs](https://elevenlabs.io/)
2. Sign up/login and go to your profile
3. Copy your API key to `ELEVENLABS_API_KEY`

## JWT Secret

Generate a secure JWT secret:

```bash
# Option 1: Using openssl
openssl rand -base64 32

# Option 2: Using node
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"

# Option 3: Using Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

## Environment Variable Loading

The application loads environment variables in the following order:

1. **Backend**: Uses Viper to load from `.env` file and environment variables
2. **Frontend**: Uses Vite's built-in environment variable loading

### Backend Loading Order:
1. `.env` file in the backend directory
2. System environment variables
3. Default values (if configured)

### Frontend Loading Order:
1. `.env` file in the frontend directory  
2. System environment variables
3. Default values (if configured)

## Docker Environment Variables

When running with Docker, you can set environment variables in several ways:

### Option 1: Using .env file in project root
Create a `.env` file in the project root with your variables:

```bash
# .env (project root)
GEMINI_API_KEY=your_key_here
ELEVENLABS_API_KEY=your_key_here
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your_anon_key_here
SUPABASE_SERVICE_ROLE_KEY=your_service_role_key_here
JWT_SECRET=your_jwt_secret_here
```

### Option 2: Using docker-compose.override.yml
Create `docker-compose.override.yml`:

```yaml
services:
  backend:
    environment:
      GEMINI_API_KEY: your_key_here
      ELEVENLABS_API_KEY: your_key_here
      SUPABASE_URL: https://your-project.supabase.co
      SUPABASE_ANON_KEY: your_anon_key_here
      SUPABASE_SERVICE_ROLE_KEY: your_service_role_key_here
      JWT_SECRET: your_jwt_secret_here
```

### Option 3: Export environment variables
```bash
export GEMINI_API_KEY=your_key_here
export ELEVENLABS_API_KEY=your_key_here
export SUPABASE_URL=https://your-project.supabase.co
export SUPABASE_ANON_KEY=your_anon_key_here
export SUPABASE_SERVICE_ROLE_KEY=your_service_role_key_here
export JWT_SECRET=your_jwt_secret_here
```

## Verification

After setting up your environment variables, you can verify the setup:

### Backend
```bash
cd backend
go run main.go
```

Look for these log messages:
- "Connected to Supabase database with GORM"
- "Supabase authentication service initialized"
- "Gemini service initialized"
- "ElevenLabs service initialized"

### Frontend
```bash
cd frontend
npm run dev
```

Check the browser console for any environment variable errors.

## Troubleshooting

### Common Issues

1. **"Missing Supabase environment variables"**
   - Check that `VITE_SUPABASE_URL` and `VITE_SUPABASE_ANON_KEY` are set in `frontend/.env`

2. **"Supabase authentication not configured"**
   - Check that `SUPABASE_URL` and `SUPABASE_ANON_KEY` are set in `backend/.env`

3. **"Failed to connect to Supabase database"**
   - Verify your Supabase URL and service role key are correct
   - Check that your Supabase project is active

4. **"Config file not found"**
   - Make sure you've created `.env` files from the examples
   - Check that the files are in the correct directories

### Environment Variable Debugging

#### Backend
Add this to your backend code to debug:
```go
slog.Info("Environment check", 
    "supabase_url", viper.GetString("supabase.url"),
    "supabase_anon_key", viper.GetString("supabase.anon_key")[:10]+"...",
    "gemini_key", viper.GetString("gemini.api_key")[:10]+"...",
)
```

#### Frontend
Add this to your frontend code to debug:
```javascript
console.log('Environment check:', {
  supabaseUrl: import.meta.env.VITE_SUPABASE_URL,
  supabaseAnonKey: import.meta.env.VITE_SUPABASE_ANON_KEY?.substring(0, 10) + '...',
  apiUrl: import.meta.env.VITE_API_URL,
});
```

## Security Notes

- Never commit `.env` files to version control
- Use strong, unique JWT secrets
- Keep your API keys secure
- Use environment-specific configurations for different deployments
- Consider using a secrets management service for production
