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
		r.Delete("/{id}", e.DeleteSessionHandler)
		r.Delete("/bulk", e.BulkDeleteSessionsHandler)
	})

	// Summary routes
	r.Route("/summaries", func(r chi.Router) {
		r.Get("/session/{id}", e.GetSummaryBySessionHandler)
		r.Post("/session/{id}/generate", e.GenerateSummaryHandler)
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

func (e *SessionEndpoints) GetSummaryBySessionHandler(w http.ResponseWriter, r *http.Request) {
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

	// First verify the session belongs to the user
	session, err := e.repo.GetInterviewSessionWithDetails(r.Context(), sessionID, user.ID)
	if err != nil {
		slog.Error("Failed to get interview session", "error", err, "session_id", sessionID, "user_id", user.ID)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get the summary for this session
	summary, err := e.repo.GetInterviewSummary(r.Context(), sessionID)
	if err != nil {
		slog.Error("Failed to get interview summary", "error", err, "session_id", sessionID, "user_id", user.ID)
		http.Error(w, "Failed to get summary", http.StatusInternalServerError)
		return
	}

	// If no summary exists, return a specific status indicating summary is being generated
	if summary == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // 202 Accepted - processing
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "generating",
			"message":    "Summary is being generated. Please check back in a few minutes.",
			"session_id": sessionID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"summary": summary,
		"status":  "ready",
	})

	slog.Info("Interview summary retrieved", "session_id", sessionID, "user_id", user.ID)
}

func (e *SessionEndpoints) GenerateSummaryHandler(w http.ResponseWriter, r *http.Request) {
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

	// First verify the session belongs to the user and is completed
	session, err := e.repo.GetInterviewSessionWithDetails(r.Context(), sessionID, user.ID)
	if err != nil {
		slog.Error("Failed to get interview session", "error", err, "session_id", sessionID, "user_id", user.ID)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	if session.Status != "completed" {
		http.Error(w, "Session must be completed to generate summary", http.StatusBadRequest)
		return
	}

	// Check if summary already exists
	existingSummary, err := e.repo.GetInterviewSummary(r.Context(), sessionID)
	if err != nil {
		slog.Error("Failed to check existing summary", "error", err, "session_id", sessionID)
		http.Error(w, "Failed to check existing summary", http.StatusInternalServerError)
		return
	}

	if existingSummary != nil {
		http.Error(w, "Summary already exists", http.StatusConflict)
		return
	}

	// Get transcripts for the session
	transcripts, err := e.repo.GetInterviewTranscripts(r.Context(), sessionID)
	if err != nil {
		slog.Error("Failed to get transcripts for summary generation", "error", err, "session_id", sessionID)
		http.Error(w, "Failed to get session transcripts", http.StatusInternalServerError)
		return
	}

	if len(transcripts) == 0 {
		http.Error(w, "No transcripts available for summary generation", http.StatusBadRequest)
		return
	}

	// Trigger summary generation (this would need to be implemented with proper timeout service access)
	// For now, return a message that manual generation is not fully implemented
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Manual summary generation is not yet implemented. Summaries are generated automatically when sessions end.",
		"status":  "not_implemented",
	})

	slog.Info("Manual summary generation requested", "session_id", sessionID, "user_id", user.ID, "transcript_count", len(transcripts))
}

func (e *SessionEndpoints) DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
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

	// Verify session belongs to user before deleting
	_, err := e.repo.GetInterviewSessionWithDetails(r.Context(), sessionID, user.ID)
	if err != nil {
		slog.Error("Failed to get interview session for deletion", "error", err, "session_id", sessionID, "user_id", user.ID)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Delete the session (this will cascade delete transcripts, summaries, and scores due to foreign key constraints)
	if err := e.repo.DeleteInterviewSession(r.Context(), sessionID); err != nil {
		slog.Error("Failed to delete interview session", "error", err, "session_id", sessionID, "user_id", user.ID)
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	slog.Info("Interview session deleted", "session_id", sessionID, "user_id", user.ID)
}

type BulkDeleteRequest struct {
	SessionIDs []string `json:"session_ids" validate:"required,min=1"`
}

func (e *SessionEndpoints) BulkDeleteSessionsHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	var req BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.SessionIDs) == 0 {
		http.Error(w, "At least one session ID is required", http.StatusBadRequest)
		return
	}

	// Verify all sessions belong to user before deleting
	sessions, err := e.repo.GetInterviewSessions(r.Context(), user.ID)
	if err != nil {
		slog.Error("Failed to get user sessions for bulk deletion", "error", err, "user_id", user.ID)
		http.Error(w, "Failed to verify sessions", http.StatusInternalServerError)
		return
	}

	// Create a map of user's session IDs for quick lookup
	userSessionIDs := make(map[string]bool)
	for _, session := range sessions {
		userSessionIDs[session.ID] = true
	}

	// Verify all requested sessions belong to the user
	for _, sessionID := range req.SessionIDs {
		if !userSessionIDs[sessionID] {
			http.Error(w, "One or more sessions do not belong to the user", http.StatusForbidden)
			return
		}
	}

	// Delete all sessions
	deletedCount, err := e.repo.BulkDeleteInterviewSessions(r.Context(), req.SessionIDs)
	if err != nil {
		slog.Error("Failed to bulk delete interview sessions", "error", err, "session_ids", req.SessionIDs, "user_id", user.ID)
		http.Error(w, "Failed to delete sessions", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":       "Sessions deleted successfully",
		"deleted_count": deletedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	slog.Info("Bulk interview sessions deleted", "deleted_count", deletedCount, "user_id", user.ID)
}
