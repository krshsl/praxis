package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

var (
	db       *pgxpool.Pool
	upgrader = websocket.Upgrader{
		CheckOrigin: checkOrigin,
	}
)

func main() {
	// Setup structured logging with JSON format
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration
	loadConfig()

	// Initialize database connection
	var err error
	dbURL := viper.GetString("database.url")
	if dbURL != "" {
		db, err = pgxpool.New(context.Background(), dbURL)
		if err != nil {
			slog.Error("Failed to connect to database", "error", err)
		} else {
			defer db.Close()
			slog.Info("Connected to database")
		}
	}

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health endpoint
	r.Get("/health", healthHandler)

	// API v1 route group
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/", apiV1Handler)
		r.Get("/ws", websocketHandler)
	})

	// Start server
	port := viper.GetString("server.port")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		slog.Info("Starting server", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}

func loadConfig() {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("database.url", "")
	viper.SetDefault("websocket.allowed_origins", "")

	// Map environment variables to config keys
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("database.url", "DATABASE_URL")
	viper.BindEnv("websocket.allowed_origins", "WEBSOCKET_ALLOWED_ORIGINS")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Warn("Config file not found, using defaults and environment variables")
		} else {
			slog.Error("Error reading config file", "error", err)
		}
	}
}

// checkOrigin validates the origin of WebSocket connections to prevent CSRF attacks
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	
	// Get allowed origins from config
	allowedOriginsStr := viper.GetString("websocket.allowed_origins")
	
	// If no allowed origins are configured, deny all requests for security
	if allowedOriginsStr == "" {
		slog.Warn("WebSocket connection rejected: no allowed origins configured", "origin", origin)
		return false
	}
	
	// Parse allowed origins (comma-separated list)
	allowedOrigins := strings.Split(allowedOriginsStr, ",")
	
	// Trim whitespace from origins
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
	
	// Check if origin is in allowed list
	for _, allowed := range allowedOrigins {
		if allowed == origin {
			slog.Info("WebSocket connection accepted", "origin", origin)
			return true
		}
	}
	
	slog.Warn("WebSocket connection rejected: origin not allowed", "origin", origin, "allowed_origins", allowedOriginsStr)
	return false
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := "not configured"

	if db != nil {
		if err := db.Ping(r.Context()); err != nil {
			dbStatus = "down"
			status = "degraded"
		} else {
			dbStatus = "up"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"` + status + `","database":"` + dbStatus + `"}`))

	slog.Info("Health check", "status", status, "database", dbStatus)
}

func apiV1Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"API v1","version":"1.0.0"}`))

	slog.Info("API v1 accessed")
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	slog.Info("WebSocket connection established")

	// Simple echo server
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			slog.Error("WebSocket read error", "error", err)
			break
		}

		slog.Info("WebSocket message received", "message", string(message))

		if err := conn.WriteMessage(messageType, message); err != nil {
			slog.Error("WebSocket write error", "error", err)
			break
		}
	}
}
