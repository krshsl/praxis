package services

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/krshsl/praxis/backend/models"
	"gorm.io/gorm"
)

type SessionTimeoutService struct {
	db             *gorm.DB
	geminiService  *GeminiService
	activeSessions map[string]*ActiveSession
	mutex          sync.RWMutex
}

type ActiveSession struct {
	SessionID    string
	UserID       string
	AgentID      string
	LastActivity time.Time
	Transcripts  []models.InterviewTranscript
	CancelFunc   context.CancelFunc
}

func NewSessionTimeoutService(db *gorm.DB, geminiService *GeminiService) *SessionTimeoutService {
	service := &SessionTimeoutService{
		db:             db,
		geminiService:  geminiService,
		activeSessions: make(map[string]*ActiveSession),
	}

	// Start the timeout checker
	go service.startTimeoutChecker()

	return service
}

func (s *SessionTimeoutService) RegisterSession(sessionID, userID, agentID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	_ = ctx // Will be used for future context operations

	s.activeSessions[sessionID] = &ActiveSession{
		SessionID:    sessionID,
		UserID:       userID,
		AgentID:      agentID,
		LastActivity: time.Now(),
		Transcripts:  make([]models.InterviewTranscript, 0),
		CancelFunc:   cancel,
	}

	slog.Info("Session registered for timeout tracking", "session_id", sessionID, "user_id", userID)
}

func (s *SessionTimeoutService) UpdateActivity(sessionID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, exists := s.activeSessions[sessionID]; exists {
		session.LastActivity = time.Now()
		slog.Debug("Session activity updated", "session_id", sessionID)
	}
}

func (s *SessionTimeoutService) AddTranscript(sessionID string, transcript models.InterviewTranscript) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, exists := s.activeSessions[sessionID]; exists {
		session.Transcripts = append(session.Transcripts, transcript)
		session.LastActivity = time.Now()
		slog.Debug("Transcript added to session", "session_id", sessionID, "turn_order", transcript.TurnOrder)
	}
}

func (s *SessionTimeoutService) EndSession(sessionID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, exists := s.activeSessions[sessionID]; exists {
		session.CancelFunc()
		delete(s.activeSessions, sessionID)
		slog.Info("Session ended and removed from timeout tracking", "session_id", sessionID)
	}
}

func (s *SessionTimeoutService) startTimeoutChecker() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkTimeouts()
		}
	}
}

func (s *SessionTimeoutService) checkTimeouts() {
	s.mutex.RLock()
	now := time.Now()
	timeoutDuration := 5 * time.Minute

	var timedOutSessions []*ActiveSession

	for _, session := range s.activeSessions {
		if now.Sub(session.LastActivity) > timeoutDuration {
			timedOutSessions = append(timedOutSessions, session)
		}
	}
	s.mutex.RUnlock()

	// Process timed out sessions
	for _, session := range timedOutSessions {
		slog.Info("Session timed out, generating summary",
			"session_id", session.SessionID,
			"inactive_duration", now.Sub(session.LastActivity))

		s.handleTimedOutSession(session)
	}
}

func (s *SessionTimeoutService) handleTimedOutSession(session *ActiveSession) {
	ctx := context.Background()

	// Update session status in database
	var dbSession models.InterviewSession
	err := s.db.Where("id = ?", session.SessionID).First(&dbSession).Error
	if err != nil {
		slog.Error("Failed to find session in database", "session_id", session.SessionID, "error", err)
		return
	}

	// Mark session as completed
	now := time.Now()
	dbSession.Status = "completed"
	dbSession.EndedAt = &now
	dbSession.Duration = int(now.Sub(dbSession.StartedAt).Seconds())

	if err := s.db.Save(&dbSession).Error; err != nil {
		slog.Error("Failed to update session status", "session_id", session.SessionID, "error", err)
		return
	}

	// Generate summary if we have transcripts
	if len(session.Transcripts) > 0 {
		s.generateAutoSummary(ctx, &dbSession, session.Transcripts)
	}

	// Remove from active sessions
	s.EndSession(session.SessionID)
}

func (s *SessionTimeoutService) generateAutoSummary(ctx context.Context, session *models.InterviewSession, transcripts []models.InterviewTranscript) {
	if s.geminiService == nil {
		slog.Warn("Gemini service not available, skipping auto summary generation")
		return
	}

	// Prepare conversation history for AI analysis
	conversationHistory := make([]string, 0, len(transcripts))
	for _, transcript := range transcripts {
		conversationHistory = append(conversationHistory,
			transcript.Speaker+": "+transcript.Content)
	}

	// Generate summary using Gemini
	summaryPrompt := `Based on this interview conversation, provide a comprehensive analysis including:
1. A narrative summary of the interview
2. Key strengths demonstrated by the candidate
3. Areas for improvement
4. Specific recommendations for the candidate
5. An overall score (0-100)

Conversation:
` + joinStrings(conversationHistory, "\n")

	summary, err := s.geminiService.GenerateSummary(ctx, summaryPrompt)
	if err != nil {
		slog.Error("Failed to generate auto summary", "session_id", session.ID, "error", err)
		return
	}

	// Parse the AI response to extract structured data
	parsedSummary := s.parseAISummary(summary)

	// Create summary record
	interviewSummary := models.InterviewSummary{
		SessionID:       session.ID,
		Summary:         parsedSummary.Summary,
		Strengths:       parsedSummary.Strengths,
		Weaknesses:      parsedSummary.Weaknesses,
		Recommendations: parsedSummary.Recommendations,
		OverallScore:    parsedSummary.OverallScore,
	}

	if err := s.db.Create(&interviewSummary).Error; err != nil {
		slog.Error("Failed to save auto-generated summary", "session_id", session.ID, "error", err)
		return
	}

	// Generate performance scores
	s.generatePerformanceScores(ctx, session.ID, parsedSummary)

	slog.Info("Auto summary generated for timed out session", "session_id", session.ID)
}

type ParsedSummary struct {
	Summary         string
	Strengths       string
	Weaknesses      string
	Recommendations string
	OverallScore    float64
}

func (s *SessionTimeoutService) parseAISummary(aiResponse string) ParsedSummary {
	// Parse AI response to extract structured data
	// This is a simplified parser - in production, you'd want more sophisticated parsing
	// that can extract scores, strengths, weaknesses, etc. from the AI response

	// For now, we'll analyze the response length and content to determine a score
	score := s.calculateScoreFromResponse(aiResponse)

	return ParsedSummary{
		Summary:         aiResponse,
		Strengths:       "Demonstrated technical knowledge and communication skills",
		Weaknesses:      "Session ended due to timeout - limited interaction data",
		Recommendations: "Consider completing the full interview for more comprehensive feedback",
		OverallScore:    score,
	}
}

func (s *SessionTimeoutService) calculateScoreFromResponse(response string) float64 {
	// Simple scoring based on response characteristics
	// In production, this would be much more sophisticated

	score := 60.0 // Base score

	// Positive indicators
	if len(response) > 200 {
		score += 10.0 // Longer responses indicate more engagement
	}

	// Check for positive keywords
	positiveKeywords := []string{"good", "excellent", "strong", "clear", "well", "impressive", "solid"}
	for _, keyword := range positiveKeywords {
		if strings.Contains(strings.ToLower(response), keyword) {
			score += 2.0
		}
	}

	// Check for technical indicators
	technicalKeywords := []string{"algorithm", "data structure", "code", "programming", "technical", "implementation"}
	for _, keyword := range technicalKeywords {
		if strings.Contains(strings.ToLower(response), keyword) {
			score += 3.0
		}
	}

	// Cap the score at 95
	if score > 95.0 {
		score = 95.0
	}

	return score
}

func (s *SessionTimeoutService) calculateMetricScore(baseScore float64, adjustment float64) float64 {
	// Calculate a metric score based on the base score with an adjustment
	adjustedScore := baseScore + (baseScore * adjustment)

	// Ensure score is within bounds
	if adjustedScore < 0 {
		adjustedScore = 0
	}
	if adjustedScore > 100 {
		adjustedScore = 100
	}

	return adjustedScore
}

func (s *SessionTimeoutService) generatePerformanceScores(ctx context.Context, sessionID string, summary ParsedSummary) {
	// Calculate performance scores based on the overall score and session characteristics
	baseScore := summary.OverallScore

	// Create performance scores that are related to the overall score
	scores := []models.PerformanceScore{
		{
			SessionID: sessionID,
			Metric:    "Communication",
			Score:     s.calculateMetricScore(baseScore, 0.1), // Slightly higher than base
			MaxScore:  100.0,
			Weight:    0.25,
		},
		{
			SessionID: sessionID,
			Metric:    "Technical Knowledge",
			Score:     s.calculateMetricScore(baseScore, 0.05), // Close to base score
			MaxScore:  100.0,
			Weight:    0.3,
		},
		{
			SessionID: sessionID,
			Metric:    "Engagement",
			Score:     s.calculateMetricScore(baseScore, -0.1), // Slightly lower than base
			MaxScore:  100.0,
			Weight:    0.2,
		},
		{
			SessionID: sessionID,
			Metric:    "Session Completion",
			Score:     s.calculateMetricScore(baseScore, -0.15), // Lower due to timeout
			MaxScore:  100.0,
			Weight:    0.25,
		},
	}

	for _, score := range scores {
		if err := s.db.Create(&score).Error; err != nil {
			slog.Error("Failed to create performance score", "session_id", sessionID, "metric", score.Metric, "error", err)
		}
	}
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
