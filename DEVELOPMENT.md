# Development Setup

This guide explains how to run Praxis in development mode with live reloading.

## Quick Start

1. **Start development environment:**
   ```bash
   ./dev.sh
   ```

2. **Or manually:**
   ```bash
   docker-compose -f docker-compose.dev.yml up --build
   ```

## What's Different in Development Mode

### Backend (Go)
- **Live Reloading**: Uses [Air](https://github.com/cosmtrek/air) for automatic rebuilds
- **Volume Mount**: `./backend` is mounted to `/app` in container
- **Port**: 8080 (same as production)
- **Auto-restart**: Changes to `.go` files trigger rebuild

### Frontend (React + Vite)
- **Live Reloading**: Vite dev server with HMR (Hot Module Replacement)
- **Volume Mount**: `./frontend` is mounted to `/app` in container
- **Port**: 5173 (Vite default)
- **Auto-restart**: Changes to `.tsx`, `.ts`, `.css` files trigger reload

### Proxy (Nginx)
- **Same as production**: Routes traffic between frontend and backend
- **Port**: 80 (main entry point)

## Development URLs

- **Main App**: http://localhost (via proxy)
- **Frontend Direct**: http://localhost:5173 (Vite dev server)
- **Backend Direct**: http://localhost:8080 (Go server)
- **Backend Health**: http://localhost:8080/health

## File Watching & Auto-reload

### Backend (Go)
- **Watched**: `.go` files, templates, HTML
- **Excluded**: `node_modules`, `dist`, `tmp`, `vendor`
- **Trigger**: File save → Air rebuilds → Container restarts

### Frontend (React)
- **Watched**: `.tsx`, `.ts`, `.css`, `.html` files
- **Excluded**: `node_modules`, `dist`
- **Trigger**: File save → Vite HMR → Browser updates

## Environment Variables

Create a `.env` file in the project root:

```bash
# Backend
SERVER_PORT=8080
GEMINI_API_KEY=your_gemini_key
ELEVENLABS_API_KEY=your_elevenlabs_key
SUPABASE_URL=your_supabase_url
SUPABASE_ANON_KEY=your_supabase_anon_key
SUPABASE_SERVICE_ROLE_KEY=your_supabase_service_key
JWT_SECRET=your_jwt_secret

# Frontend
VITE_API_URL=http://localhost:8080/api/v1
VITE_WS_URL=localhost:8080
```

## Troubleshooting

### Backend not reloading
- Check if Air is running: `docker logs praxis-backend-dev`
- Verify `.air.toml` configuration
- Check file permissions

### Frontend not reloading
- Check if Vite is running: `docker logs praxis-frontend-dev`
- Verify port 5173 is accessible
- Check browser console for errors

### Port conflicts
- Stop other services using ports 80, 5173, 8080
- Use `docker-compose -f docker-compose.dev.yml down` to stop

### Database issues
- Ensure your database is accessible from Docker
- Check connection strings in environment variables

## Production vs Development

| Feature | Development | Production |
|---------|-------------|------------|
| Backend | Air live reload | Static binary |
| Frontend | Vite dev server | Nginx static files |
| Volumes | Source mounted | Built assets |
| Ports | 5173, 8080, 80 | 80 only |
| Rebuild | Automatic | Manual |

## Stopping Development

```bash
# Stop all containers
docker-compose -f docker-compose.dev.yml down

# Stop and remove volumes
docker-compose -f docker-compose.dev.yml down -v
```
