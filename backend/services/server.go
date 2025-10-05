package services

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
	"github.com/krshsl/praxis/backend/models"
	"github.com/krshsl/praxis/backend/repository"
	ws "github.com/krshsl/praxis/backend/websocket"
	"gorm.io/gorm"
)

// Server holds all server dependencies
type Server struct {
	config             *Config
	gormDB             *repository.GORMRepository
	rawDB              interface{} // Store the raw GORM DB for services that need it
	geminiService      *GeminiService
	elevenLabsService  *ElevenLabsService
	timeoutService     *SessionTimeoutService
	aiMessageProcessor *AIMessageProcessor
	websocketHandler   *WebSocketHandler
	authService        *AuthService
	authEndpoints      *AuthEndpoints
	sessionEndpoints   *SessionEndpoints
	agentEndpoints     *AgentEndpoints
	wsHub              *ws.Hub
	upgrader           websocket.Upgrader
}

// NewServer creates a new server instance
func NewServer(config *Config) *Server {
	return &Server{
		config: config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return checkOrigin(r, config.WebSocket.AllowedOrigins)
			},
		},
	}
}

// InitializeServices initializes all server services
func (s *Server) InitializeServices() error {
	// Initialize database connection
	if s.config.Database.URL != "" {
		// Database initialization is handled in main.go
		slog.Info("Database connection will be initialized in main.go")
	} else {
		slog.Warn("Database URL not configured, running without database")
	}

	// Initialize AI services
	if s.config.AI.GeminiAPIKey != "" {
		s.geminiService = NewGeminiService(s.config.AI.GeminiAPIKey)
		slog.Info("Gemini service initialized")
	}

	if s.config.AI.ElevenLabsKey != "" {
		s.elevenLabsService = NewElevenLabsService(s.config.AI.ElevenLabsKey)
		slog.Info("ElevenLabs service initialized")
	}

	// Initialize session timeout service
	if s.rawDB != nil && s.geminiService != nil {
		if gormDB, ok := s.rawDB.(*gorm.DB); ok {
			s.timeoutService = NewSessionTimeoutService(gormDB, s.geminiService)
			slog.Info("Session timeout service initialized")
		}
	}

	// Initialize AI message processor
	if s.geminiService != nil && s.elevenLabsService != nil && s.timeoutService != nil && s.gormDB != nil {
		s.aiMessageProcessor = NewAIMessageProcessor(s.geminiService, s.elevenLabsService, s.timeoutService, s.gormDB)
		slog.Info("AI message processor initialized")
	}

	// Initialize authentication services
	if s.config.JWT.Secret != "" && s.gormDB != nil {
		s.authService = NewAuthService(s.gormDB, s.config.JWT.Secret)
		s.authEndpoints = NewAuthEndpoints(s.authService)
		s.sessionEndpoints = NewSessionEndpoints(s.gormDB)
		s.agentEndpoints = NewAgentEndpoints(s.gormDB)
		slog.Info("Authentication service initialized")
	}

	// Initialize WebSocket handler
	if s.aiMessageProcessor != nil {
		s.websocketHandler = NewWebSocketHandler(s.aiMessageProcessor)
		slog.Info("WebSocket handler initialized")
	}

	// Initialize WebSocket hub
	s.wsHub = ws.NewHub()
	go s.wsHub.Run()

	return nil
}

// SetDatabase sets the database connection
func (s *Server) SetDatabase(db *repository.GORMRepository, rawDB interface{}) {
	s.gormDB = db
	s.rawDB = rawDB
}

// SetupRoutes configures all HTTP routes
func (s *Server) SetupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health endpoint
	r.Get("/health", s.healthHandler)

	// API v1 route group
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/", s.apiV1Handler)
		// WebSocket route (protected)
		if s.authService != nil {
			r.Group(func(r chi.Router) {
				r.Use(s.authService.Middleware)
				r.Get("/ws", s.websocketHandlerFunc)
			})
		} else {
			r.Get("/ws", s.websocketHandlerFunc)
		}

		// Authentication routes
		if s.authEndpoints != nil {
			r.Route("/auth", func(r chi.Router) {
				// Public auth routes (no middleware)
				r.Post("/login", s.authEndpoints.LoginHandler)
				r.Post("/signup", s.authEndpoints.SignupHandler)
				r.Post("/refresh", s.authEndpoints.RefreshHandler)
				r.Post("/logout", s.authEndpoints.LogoutHandler)

				// Protected auth routes (with middleware)
				r.Group(func(r chi.Router) {
					r.Use(s.authService.Middleware)
					r.Get("/me", s.authEndpoints.MeHandler)
				})
			})
		}

		// Session routes (protected)
		if s.sessionEndpoints != nil && s.authService != nil {
			r.Group(func(r chi.Router) {
				r.Use(s.authService.Middleware)
				s.sessionEndpoints.RegisterRoutes(r)
			})
		}

		// Agent routes (protected)
		if s.agentEndpoints != nil && s.authService != nil {
			r.Group(func(r chi.Router) {
				r.Use(s.authService.Middleware)
				s.agentEndpoints.RegisterRoutes(r)
			})
		}
	})

	return r
}

// Start starts the HTTP server
func (s *Server) Start() {
	port := s.config.Server.Port
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: s.SetupRoutes(),
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

// checkOrigin validates the origin of WebSocket connections to prevent CSRF attacks
func checkOrigin(r *http.Request, allowedOriginsStr string) bool {
	origin := r.Header.Get("Origin")

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

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := "not configured"

	if s.rawDB != nil {
		// We need to cast the rawDB to the actual GORM DB type
		// This is a bit of a hack, but it works for now
		if gormDB, ok := s.rawDB.(*gorm.DB); ok {
			if sqlDB, err := gormDB.DB(); err == nil {
				if err := sqlDB.Ping(); err != nil {
					dbStatus = "down"
					status = "degraded"
				} else {
					dbStatus = "up"
				}
			} else {
				dbStatus = "down"
				status = "degraded"
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"` + status + `","database":"` + dbStatus + `"}`))

	slog.Info("Health check", "status", status, "database", dbStatus)
}

func (s *Server) apiV1Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"API v1","version":"1.0.0"}`))

	slog.Info("API v1 accessed")
}

func (s *Server) websocketHandlerFunc(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		slog.Error("WebSocket connection failed - user not found in context")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	slog.Info("WebSocket connection established", "user_id", user.ID, "email", user.Email)

	// Register client with hub
	client := s.wsHub.RegisterClient(conn, user.ID)

	// Set up message handler for AI processing
	if s.websocketHandler != nil {
		client.MessageHandler = func(c *ws.Client, messageBytes []byte) {
			s.websocketHandler.HandleWebSocketMessage(c, messageBytes)
		}
	}

	// Register session with timeout service if available
	if s.timeoutService != nil {
		// Extract session ID from query parameters - this should be an existing InterviewSession ID
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			slog.Error("WebSocket connection requires session_id parameter")
			http.Error(w, "Session ID is required", http.StatusBadRequest)
			return
		}

		// Extract agent ID from query parameters
		agentID := r.URL.Query().Get("agent_id")
		if agentID == "" {
			agentID = "default_agent"
		}

		// Update the client's session ID to use the provided one
		client.SessionID = sessionID
		s.timeoutService.RegisterSession(sessionID, user.ID, agentID)
	}

	// Start goroutines for reading and writing
	go client.ReadPump()
	go client.WritePump()

	// Handle AI conversation flow
	go s.handleAIConversation(client)

	// Keep connection alive
	select {}
}

func (s *Server) handleAIConversation(client *ws.Client) {
	// This function is now handled by the AI message processor
	// The actual message processing happens in the WebSocket client handlers
	// which are connected to the AI message processor
	slog.Info("AI conversation handler started", "session_id", client.SessionID, "user_id", client.UserID)
}
