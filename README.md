# Praxis - AI Conversational Agent

A real-time AI conversational agent built for hackathon demonstration, featuring voice conversation, live coding assessment, and intelligent code analysis.

## Architecture

- **Backend** (`/backend`): Go application with AI services integration
  - Chi router for HTTP routing
  - WebSocket hub for real-time communication
  - Google Gemini for conversational AI
  - ElevenLabs for text-to-speech synthesis
  - Supabase Auth for JWT-based authentication
  - Supabase PostgreSQL for conversation persistence
- **Frontend** (`/frontend`): React application with Vite and TypeScript
  - Supabase client for authentication
  - Real-time WebSocket communication
- **Proxy** (`/proxy`): NGINX reverse proxy that routes `/api/*` to backend and `/` to frontend
- **Database**: Supabase PostgreSQL with conversation history storage

## Quick Start

### 1. Environment Setup

First, set up your environment variables:

```bash
# Run the setup script to create .env files
./setup-env.sh

# Or manually copy the example files
cp backend/env.example backend/.env
cp frontend/env.example frontend/.env
```

Then edit the `.env` files with your actual configuration values. See [ENVIRONMENT_SETUP.md](./ENVIRONMENT_SETUP.md) for detailed instructions.

### Prerequisites

- Docker
- Docker Compose
- API Keys for AI services:
  - Google Gemini API key
  - ElevenLabs API key
  - Supabase project URL (for authentication)

### Running the Application

1. Clone the repository:
```bash
git clone https://github.com/krshsl/praxis.git
cd praxis
```

2. Set up environment variables:
```bash
# Copy the example environment file
cp backend/env.example .env

# Edit .env with your API keys
GEMINI_API_KEY=your_gemini_api_key_here
ELEVENLABS_API_KEY=your_elevenlabs_api_key_here
SUPABASE_URL=https://your-project-ref.supabase.co
```

3. Start all services:
```bash
docker-compose up --build
```

4. Access the application:
   - Frontend: http://localhost
   - Backend API: http://localhost/api/v1
   - WebSocket: ws://localhost/api/v1/ws
   - Health Check: http://localhost/health
   - Direct Backend: http://localhost:8080
   - Direct Frontend: http://localhost:5173

### Development

#### Backend

```bash
cd backend
go run main.go
```

#### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Features

### AI Conversational Agent
- **Real-time Voice Conversation**: WebSocket-based communication with AI
- **Text-to-Speech**: ElevenLabs integration for natural voice responses
- **Conversation Memory**: Maintains context throughout the session
- **Live Coding Assessment**: Submit code for AI analysis and feedback

### WebSocket Communication
- **Message Types**: `text`, `code`, `audio`
- **Real-time Audio Streaming**: Direct audio data transmission
- **Session Management**: Per-user conversation history
- **Authentication**: Clerk-based user authentication

## API Endpoints

- `GET /health` - Health check endpoint
- `GET /api/v1` - API v1 base endpoint
- `GET /api/v1/ws` - WebSocket endpoint for AI conversation
- `GET /api/v1/secure` - Protected endpoint (requires authentication)

### WebSocket Message Format

#### Text Message
```json
{
  "type": "text",
  "content": "Hello, how are you?",
  "session_id": "optional_session_id"
}
```

#### Code Submission
```json
{
  "type": "code",
  "content": "def fibonacci(n):\n    if n <= 1:\n        return n\n    return fibonacci(n-1) + fibonacci(n-2)",
  "language": "python",
  "session_id": "optional_session_id"
}
```

#### Audio Response
```json
{
  "type": "audio",
  "audio_data": "base64_encoded_audio_data",
  "session_id": "session_id"
}
```

## Environment Variables

Backend configuration is managed through Viper and can be set via environment variables or `.env` file:

- `SERVER_PORT` - Server port (default: 8080)
- `DATABASE_URL` - PostgreSQL connection string
- `WEBSOCKET_ALLOWED_ORIGINS` - Comma-separated list of allowed WebSocket origins for CSRF protection
- `GEMINI_API_KEY` - Google Gemini API key for AI conversation
- `ELEVENLABS_API_KEY` - ElevenLabs API key for text-to-speech
- `SUPABASE_URL` - Supabase project URL for JWT validation

### Security Notes

The WebSocket endpoint implements origin validation to prevent Cross-Site Request Forgery (CSRF) attacks. You must configure `WEBSOCKET_ALLOWED_ORIGINS` with the appropriate origins for your deployment:

- **Development**: Include `http://localhost`, `http://localhost:80`, and `http://localhost:5173`
- **Production**: Set to your actual domain(s), e.g., `https://yourdomain.com`
- **Important**: Leaving this empty will reject all WebSocket connections for security reasons

## Project Structure

```
.
├── backend/              # Go backend application
│   ├── main.go          # Main application file
│   ├── go.mod           # Go module definition
│   ├── go.sum           # Go dependencies
│   ├── .env             # Environment configuration
│   └── Dockerfile       # Multi-stage Docker build
├── frontend/            # React frontend application
│   ├── src/             # Source files
│   ├── package.json     # NPM dependencies
│   └── Dockerfile       # Multi-stage Docker build
├── proxy/               # NGINX reverse proxy
│   ├── nginx.conf       # NGINX configuration
│   └── Dockerfile       # NGINX Docker image
└── docker-compose.yml   # Docker Compose configuration
```

## Hackathon Demo Features

### Live Coding Assessment
1. **Connect via WebSocket**: `ws://localhost/api/v1/ws`
2. **Submit Code**: Send JSON message with type "code" and your code
3. **Get AI Feedback**: Receive voice analysis of your code quality
4. **Real-time Conversation**: Chat with AI about your code

### Demo Script
```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost/api/v1/ws');

// Send text message
ws.send(JSON.stringify({
  type: 'text',
  content: 'Hello, I want to practice coding'
}));

// Submit code for analysis
ws.send(JSON.stringify({
  type: 'code',
  content: 'def fibonacci(n):\n    return n if n <= 1 else fibonacci(n-1) + fibonacci(n-2)',
  language: 'python'
}));
```

### Key Talking Points
1. **Real-time AI Conversation**: Low-latency voice interaction
2. **Live Code Analysis**: Instant feedback on code quality
3. **Voice Synthesis**: Natural-sounding AI responses
4. **Session Persistence**: Conversation history stored in database
5. **Scalable Architecture**: WebSocket hub supports multiple users

## License

MIT License - see LICENSE file for details