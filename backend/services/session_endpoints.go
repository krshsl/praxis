package services

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/krshsl/praxis/backend/models"
	"github.com/krshsl/praxis/backend/repository"
)

type SessionEndpoints struct {
	repo *repository.GORMRepository
}

type CreateSessionRequest struct {
	AgentID string `json:"agent_id" validate:"required"`
}

type CreateSessionResponse struct {
	Session models.InterviewSession `json:"session"`
	Message string                  `json:"message"`
}

type GetSessionsResponse struct {
	Sessions []models.InterviewSession `json:"sessions"`
	Count    int                       `json:"count"`
}

func NewSessionEndpoints(repo *repository.GORMRepository) *SessionEndpoints {
	return &SessionEndpoints{
		repo: repo,
	}
}

func (e *SessionEndpoints) RegisterRoutes(r chi.Router) {
	r.Route("/sessions", func(r chi.Router) {
		r.Post("/", e.CreateSessionHandler)
		r.Get("/", e.GetSessionsHandler)
		r.Get("/{id}", e.GetSessionHandler)
	})
}

func (e *SessionEndpoints) CreateSessionHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate agent exists
	agent, err := e.repo.GetAgentByID(r.Context(), req.AgentID, user.ID)
	if err != nil {
		slog.Error("Failed to get agent", "error", err, "agent_id", req.AgentID)
		http.Error(w, "Failed to validate agent", http.StatusInternalServerError)
		return
	}
	if agent == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Create new interview session
	now := time.Now()
	session := models.InterviewSession{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		AgentID:   req.AgentID,
		Status:    "active",
		StartedAt: now,
	}

	if err := e.repo.CreateInterviewSession(r.Context(), &session); err != nil {
		slog.Error("Failed to create interview session", "error", err, "user_id", user.ID)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	response := CreateSessionResponse{
		Session: session,
		Message: "Session created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	slog.Info("Interview session created", "session_id", session.ID, "user_id", user.ID, "agent_id", req.AgentID)
}

func (e *SessionEndpoints) GetSessionsHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	sessions, err := e.repo.GetInterviewSessions(r.Context(), user.ID)
	if err != nil {
		slog.Error("Failed to get interview sessions", "error", err, "user_id", user.ID)
		http.Error(w, "Failed to get sessions", http.StatusInternalServerError)
		return
	}

	response := GetSessionsResponse{
		Sessions: sessions,
		Count:    len(sessions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	slog.Info("Interview sessions retrieved", "user_id", user.ID, "count", len(sessions))
}

func (e *SessionEndpoints) GetSessionHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Get session with transcripts and summary
	session, err := e.repo.GetInterviewSessionWithDetails(r.Context(), sessionID, user.ID)
	if err != nil {
		slog.Error("Failed to get interview session", "error", err, "session_id", sessionID, "user_id", user.ID)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session": session,
	})

	slog.Info("Interview session retrieved", "session_id", sessionID, "user_id", user.ID)
}
