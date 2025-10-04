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