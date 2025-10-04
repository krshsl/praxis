# praxis

A full-stack, Dockerized application with Go backend, React frontend, and NGINX reverse proxy.

## Architecture

- **Backend** (`/backend`): Go application using Chi router, slog for JSON logging, Viper for configuration, pgx/v5 for PostgreSQL, and gorilla/websocket for WebSocket support
- **Frontend** (`/frontend`): React application with Vite and TypeScript
- **Proxy** (`/proxy`): NGINX reverse proxy that routes `/api/*` to backend and `/` to frontend
- **Database**: PostgreSQL 16

## Quick Start

### Prerequisites

- Docker
- Docker Compose

### Running the Application

1. Clone the repository:
```bash
git clone https://github.com/krshsl/praxis.git
cd praxis
```

2. Start all services:
```bash
docker-compose up --build
```

3. Access the application:
   - Frontend: http://localhost
   - Backend API: http://localhost/api/v1
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

## API Endpoints

- `GET /health` - Health check endpoint
- `GET /api/v1` - API v1 base endpoint
- `GET /api/v1/ws` - WebSocket endpoint (echo server)

## Environment Variables

Backend configuration is managed through Viper and can be set via environment variables or `.env` file:

- `SERVER_PORT` - Server port (default: 8080)
- `DATABASE_URL` - PostgreSQL connection string
- `WEBSOCKET_ALLOWED_ORIGINS` - Comma-separated list of allowed WebSocket origins for CSRF protection (e.g., `http://localhost,http://localhost:80,http://localhost:5173`)

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

## License

MIT License - see LICENSE file for details