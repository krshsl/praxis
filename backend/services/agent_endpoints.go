package services

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/krshsl/praxis/backend/models"
	"github.com/krshsl/praxis/backend/repository"
)

type AgentEndpoints struct {
	repo *repository.GORMRepository
}

type CreateAgentRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Personality string `json:"personality" validate:"required"`
	Industry    string `json:"industry"`
	Level       string `json:"level"`
	IsPublic    bool   `json:"is_public"`
}

type CreateAgentResponse struct {
	Agent   models.Agent `json:"agent"`
	Message string       `json:"message"`
}

type GetAgentsResponse struct {
	Agents []models.Agent `json:"agents"`
	Count  int            `json:"count"`
}

func NewAgentEndpoints(repo *repository.GORMRepository) *AgentEndpoints {
	return &AgentEndpoints{
		repo: repo,
	}
}

func (e *AgentEndpoints) RegisterRoutes(r chi.Router) {
	r.Route("/agents", func(r chi.Router) {
		r.Post("/", e.CreateAgentHandler)
		r.Get("/", e.GetAgentsHandler)
		r.Get("/{id}", e.GetAgentHandler)
		r.Put("/{id}", e.UpdateAgentHandler)
		r.Delete("/{id}", e.DeleteAgentHandler)
	})
}

func (e *AgentEndpoints) CreateAgentHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create new agent
	agent := models.Agent{
		ID:          uuid.New().String(),
		UserID:      &user.ID,
		Name:        req.Name,
		Description: req.Description,
		Personality: req.Personality,
		Industry:    req.Industry,
		Level:       req.Level,
		IsPublic:    req.IsPublic,
		IsActive:    true,
	}

	if err := e.repo.CreateAgent(r.Context(), &agent); err != nil {
		slog.Error("Failed to create agent", "error", err, "user_id", user.ID)
		http.Error(w, "Failed to create agent", http.StatusInternalServerError)
		return
	}

	response := CreateAgentResponse{
		Agent:   agent,
		Message: "Agent created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	slog.Info("Agent created", "agent_id", agent.ID, "user_id", user.ID, "name", agent.Name)
}

func (e *AgentEndpoints) GetAgentsHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Get both public agents and user's private agents
	agents, err := e.repo.GetAgents(r.Context(), user.ID, true)
	if err != nil {
		slog.Error("Failed to get agents", "error", err, "user_id", user.ID)
		http.Error(w, "Failed to get agents", http.StatusInternalServerError)
		return
	}

	response := GetAgentsResponse{
		Agents: agents,
		Count:  len(agents),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	slog.Info("Agents retrieved", "user_id", user.ID, "count", len(agents))
}

func (e *AgentEndpoints) GetAgentHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	agentID := chi.URLParam(r, "id")
	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Get agent (check if it's public or belongs to user)
	agent, err := e.repo.GetAgentByID(r.Context(), agentID, user.ID)
	if err != nil {
		slog.Error("Failed to get agent", "error", err, "agent_id", agentID, "user_id", user.ID)
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agent": agent,
	})

	slog.Info("Agent retrieved", "agent_id", agentID, "user_id", user.ID)
}

func (e *AgentEndpoints) UpdateAgentHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	agentID := chi.URLParam(r, "id")
	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Get existing agent
	agent, err := e.repo.GetAgentByID(r.Context(), agentID, user.ID)
	if err != nil {
		slog.Error("Failed to get agent for update", "error", err, "agent_id", agentID, "user_id", user.ID)
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Check if user owns this agent
	if agent.UserID == nil || *agent.UserID != user.ID {
		http.Error(w, "Not authorized to update this agent", http.StatusForbidden)
		return
	}

	var req CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update agent fields
	agent.Name = req.Name
	agent.Description = req.Description
	agent.Personality = req.Personality
	agent.Industry = req.Industry
	agent.Level = req.Level
	agent.IsPublic = req.IsPublic

	if err := e.repo.UpdateAgent(r.Context(), agent); err != nil {
		slog.Error("Failed to update agent", "error", err, "agent_id", agentID, "user_id", user.ID)
		http.Error(w, "Failed to update agent", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agent":   agent,
		"message": "Agent updated successfully",
	})

	slog.Info("Agent updated", "agent_id", agentID, "user_id", user.ID)
}

func (e *AgentEndpoints) DeleteAgentHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	agentID := chi.URLParam(r, "id")
	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Get existing agent
	agent, err := e.repo.GetAgentByID(r.Context(), agentID, user.ID)
	if err != nil {
		slog.Error("Failed to get agent for deletion", "error", err, "agent_id", agentID, "user_id", user.ID)
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Check if user owns this agent
	if agent.UserID == nil || *agent.UserID != user.ID {
		http.Error(w, "Not authorized to delete this agent", http.StatusForbidden)
		return
	}

	if err := e.repo.DeleteAgent(r.Context(), agentID); err != nil {
		slog.Error("Failed to delete agent", "error", err, "agent_id", agentID, "user_id", user.ID)
		http.Error(w, "Failed to delete agent", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Agent deleted successfully",
	})

	slog.Info("Agent deleted", "agent_id", agentID, "user_id", user.ID)
}
