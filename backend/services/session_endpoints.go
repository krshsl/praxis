package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/krshsl/praxis/backend/models"
	"github.com/krshsl/praxis/backend/repository"
)

type SessionEndpoints struct {
	repo          *repository.GORMRepository
	geminiService *GeminiService
}

// Global mutex for summary generation to prevent race conditions across services
var summaryGenerationMutex sync.Mutex

func NewSessionEndpoints(repo *repository.GORMRepository, geminiService *GeminiService) *SessionEndpoints {
	return &SessionEndpoints{
		repo:          repo,
		geminiService: geminiService,
	}
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

	// If no summary exists, trigger summary generation
	if summary == nil {
		// Use global mutex to prevent concurrent summary generation across services
		summaryGenerationMutex.Lock()
		defer summaryGenerationMutex.Unlock()

		// Double-check if summary was created by another goroutine
		summary, err = e.repo.GetInterviewSummary(r.Context(), sessionID)
		if err != nil {
			slog.Error("Failed to re-check for summary", "error", err, "session_id", sessionID)
			http.Error(w, "Failed to check summary status", http.StatusInternalServerError)
			return
		}

		if summary != nil {
			// Summary was created by another goroutine, return it
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"summary": summary,
				"status":  "ready",
			})
			return
		}

		slog.Info("No summary found, triggering automatic generation", "session_id", sessionID, "user_id", user.ID)

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

		// Trigger summary generation in a goroutine
		go func() {
			ctx := context.Background()
			slog.Info("Starting automatic summary generation", "session_id", sessionID, "transcript_count", len(transcripts), "user_id", user.ID)

			// Get agent information for personality-based summary
			agent, err := e.repo.GetAgent(ctx, session.AgentID)
			if err != nil {
				slog.Error("Failed to load agent for summary generation", "session_id", sessionID, "error", err)
				return
			}

			// Prepare conversation history for AI analysis
			conversationHistory := make([]string, 0, len(transcripts))
			for _, transcript := range transcripts {
				conversationHistory = append(conversationHistory,
					transcript.Speaker+": "+transcript.Content)
			}

			// Generate personality-based summary using Gemini
			summaryPrompt := e.buildPersonalityBasedSummaryPrompt(*agent, conversationHistory)

			slog.Info("Generating AI summary with Gemini", "session_id", sessionID, "agent_name", agent.Name, "conversation_length", len(conversationHistory))
			geminiService := e.getGeminiService() // You'll need to implement this method
			if geminiService == nil {
				slog.Error("Gemini service not available for summary generation", "session_id", sessionID)
				return
			}

			summary, err := geminiService.GenerateSummary(ctx, summaryPrompt)
			if err != nil {
				slog.Error("Failed to generate summary", "session_id", sessionID, "error", err, "user_id", user.ID)
				return
			}
			slog.Info("AI summary generated successfully", "session_id", sessionID, "summary_length", len(summary), "user_id", user.ID)

			// Parse the AI response to extract structured data
			parsedSummary := e.parseAISummary(summary)

			// Create summary record
			interviewSummary := models.InterviewSummary{
				SessionID:       session.ID,
				Summary:         parsedSummary.Summary,
				Strengths:       parsedSummary.Strengths,
				Weaknesses:      parsedSummary.Weaknesses,
				Recommendations: parsedSummary.Recommendations,
				OverallScore:    float64(parsedSummary.OverallScore),
			}

			if err := e.repo.CreateInterviewSummary(ctx, &interviewSummary); err != nil {
				slog.Error("Failed to save generated summary", "session_id", sessionID, "error", err)
				return
			}
			slog.Info("Summary saved to database", "session_id", sessionID, "summary_id", interviewSummary.ID)

			// Generate performance scores
			e.generatePerformanceScores(ctx, session.ID, parsedSummary)

			slog.Info("Automatic summary generation completed successfully", "session_id", sessionID, "overall_score", parsedSummary.OverallScore)
		}()

		// Return immediate response indicating generation has started
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // 202 Accepted - processing
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "generating",
			"message":    "Summary generation has been triggered. Please check back in a few minutes.",
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

// getGeminiService returns the Gemini service instance
func (e *SessionEndpoints) getGeminiService() *GeminiService {
	return e.geminiService
}

// buildPersonalityBasedSummaryPrompt creates a summary prompt tailored to the agent's personality
func (e *SessionEndpoints) buildPersonalityBasedSummaryPrompt(agent models.Agent, conversationHistory []string) string {
	// Determine scoring strictness based on agent personality
	scoringGuidance := e.getScoringGuidance(agent.Personality)

	// Build industry-specific context
	industryContext := e.getIndustryContext(agent.Industry, agent.Level)

	// Create personality-specific tone and expectations
	personalityTone := e.getPersonalityTone(agent.Personality)

	prompt := fmt.Sprintf(`You are %s, a %s interviewer in the %s industry. 
Your personality: %s

%s

Based on this interview conversation, provide a comprehensive analysis that reflects your interviewing style and personality:

1. A narrative summary of the interview (written in your voice and style)
2. Key strengths demonstrated by the candidate
3. Areas for improvement (be specific and constructive)
4. Specific recommendations for the candidate's growth
5. An overall score (0-100) using this scoring guidance: %s

%s

Conversation:
%s

Please structure your response as:
SUMMARY: [Your narrative summary]
STRENGTHS: [Key strengths]
WEAKNESSES: [Areas for improvement]
RECOMMENDATIONS: [Specific recommendations]
SCORE: [Numerical score 0-100]`,
		agent.Name,
		agent.Level,
		agent.Industry,
		agent.Personality,
		industryContext,
		scoringGuidance,
		personalityTone,
		joinStrings(conversationHistory, "\n"))

	return prompt
}

// parseAISummary parses the structured JSON response from Gemini
func (e *SessionEndpoints) parseAISummary(response string) *ParsedSummary {
	// Parse structured JSON response from Gemini
	var jsonResponse struct {
		Summary         string  `json:"summary"`
		Strengths       string  `json:"strengths"`
		Weaknesses      string  `json:"weaknesses"`
		Recommendations string  `json:"recommendations"`
		OverallScore    float64 `json:"overallScore"`
		TechnicalSkills []struct {
			Skill  string  `json:"skill"`
			Rating float64 `json:"rating"`
		} `json:"technicalSkills"`
		CommunicationSkills []struct {
			Skill  string  `json:"skill"`
			Rating float64 `json:"rating"`
		} `json:"communicationSkills"`
	}

	// Parse the JSON response
	if err := json.Unmarshal([]byte(response), &jsonResponse); err != nil {
		slog.Error("Failed to parse AI summary JSON", "error", err, "response", response)
		// Fallback to basic parsing if JSON parsing fails
		return &ParsedSummary{
			Summary:         response,
			Strengths:       "Unable to parse structured response",
			Weaknesses:      "Unable to parse structured response",
			Recommendations: "Unable to parse structured response",
			OverallScore:    50.0, // Default score
		}
	}

	// Validate and sanitize the response
	if jsonResponse.OverallScore < 0 {
		jsonResponse.OverallScore = 0
	}
	if jsonResponse.OverallScore > 100 {
		jsonResponse.OverallScore = 100
	}

	// Ensure we have valid strings
	if jsonResponse.Summary == "" {
		jsonResponse.Summary = "No summary provided"
	}
	if jsonResponse.Strengths == "" {
		jsonResponse.Strengths = "No strengths identified"
	}
	if jsonResponse.Weaknesses == "" {
		jsonResponse.Weaknesses = "No weaknesses identified"
	}
	if jsonResponse.Recommendations == "" {
		jsonResponse.Recommendations = "No recommendations provided"
	}

	slog.Info("Successfully parsed structured AI summary",
		"overall_score", jsonResponse.OverallScore,
		"technical_skills_count", len(jsonResponse.TechnicalSkills),
		"communication_skills_count", len(jsonResponse.CommunicationSkills))

	return &ParsedSummary{
		Summary:         jsonResponse.Summary,
		Strengths:       jsonResponse.Strengths,
		Weaknesses:      jsonResponse.Weaknesses,
		Recommendations: jsonResponse.Recommendations,
		OverallScore:    jsonResponse.OverallScore,
	}
}

// generatePerformanceScores creates detailed performance scores
func (e *SessionEndpoints) generatePerformanceScores(ctx context.Context, sessionID string, parsedSummary *ParsedSummary) {
	// Calculate performance scores based on the overall score and session characteristics
	baseScore := parsedSummary.OverallScore

	// Create performance scores that are related to the overall score
	scores := []models.PerformanceScore{
		{
			SessionID: sessionID,
			Metric:    "Communication",
			Score:     e.calculateMetricScore(baseScore, 0.1), // Slightly higher than base
			MaxScore:  100.0,
		},
		{
			SessionID: sessionID,
			Metric:    "Technical Knowledge",
			Score:     e.calculateMetricScore(baseScore, -0.05), // Slightly lower than base
			MaxScore:  100.0,
		},
		{
			SessionID: sessionID,
			Metric:    "Problem Solving",
			Score:     e.calculateMetricScore(baseScore, 0.0), // Same as base
			MaxScore:  100.0,
		},
		{
			SessionID: sessionID,
			Metric:    "Professionalism",
			Score:     e.calculateMetricScore(baseScore, 0.05), // Slightly higher than base
			MaxScore:  100.0,
		},
	}

	// Save performance scores to database
	for _, score := range scores {
		if err := e.repo.CreatePerformanceScore(ctx, &score); err != nil {
			slog.Error("Failed to create performance score", "session_id", sessionID, "metric", score.Metric, "error", err)
		}
	}

	slog.Info("Performance scores generation completed", "session_id", sessionID, "scores_count", len(scores))
}

// calculateMetricScore calculates a metric score based on the base score and adjustment
func (e *SessionEndpoints) calculateMetricScore(baseScore float64, adjustment float64) float64 {
	adjustedScore := baseScore + (baseScore * adjustment)
	if adjustedScore < 0 {
		return 0
	}
	if adjustedScore > 100 {
		return 100
	}
	return adjustedScore
}

// Helper methods for summary generation
func (e *SessionEndpoints) getScoringGuidance(personality string) string {
	switch strings.ToLower(personality) {
	case "strict", "tough", "demanding":
		return "Be very strict and demanding. Only give high scores (80+) for exceptional performance. Average performance should score 50-70."
	case "encouraging", "supportive", "friendly":
		return "Be encouraging and supportive. Focus on potential and growth. Give higher scores (70+) for good effort and communication."
	case "technical", "analytical":
		return "Focus heavily on technical accuracy and problem-solving skills. Be precise in evaluation."
	default:
		return "Be balanced and fair in your evaluation. Consider both technical skills and communication."
	}
}

func (e *SessionEndpoints) getIndustryContext(industry, level string) string {
	return fmt.Sprintf("This is a %s level interview in the %s industry. Focus on relevant skills and knowledge for this domain.", level, industry)
}

func (e *SessionEndpoints) getPersonalityTone(personality string) string {
	switch strings.ToLower(personality) {
	case "strict", "tough":
		return "Be direct and honest in your feedback. Don't sugarcoat areas for improvement."
	case "encouraging", "supportive":
		return "Be positive and constructive. Focus on growth opportunities and potential."
	case "technical", "analytical":
		return "Be precise and detailed in your analysis. Focus on technical accuracy and methodology."
	default:
		return "Be professional and balanced in your tone."
	}
}
