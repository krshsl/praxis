package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/krshsl/praxis/backend/models"
	"gorm.io/gorm"
)

const (
	DefaultTimeout = 30 * time.Minute
	InterviewLimit = 5 * time.Minute
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
	// Audio chunking support
	AudioChunks map[int][]byte // chunkIndex -> chunk data
	TotalChunks int
	ChunksMutex sync.RWMutex
	// Penalty tracking
	EmptyResponseCount int
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
		AudioChunks:  make(map[int][]byte),
		TotalChunks:  0,
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

func (s *SessionTimeoutService) IsInterviewExpired(sessionID string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if session, exists := s.activeSessions[sessionID]; exists {
		elapsed := time.Since(session.LastActivity)
		return elapsed > InterviewLimit
	}
	return false
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

// ConcludeSession finalizes a session immediately: updates DB, generates summary, and removes from active tracking
func (s *SessionTimeoutService) ConcludeSession(sessionID string, reason string) {
	s.mutex.RLock()
	session, exists := s.activeSessions[sessionID]
	s.mutex.RUnlock()
	if !exists {
		slog.Warn("ConcludeSession called for non-active session", "session_id", sessionID)
		return
	}

	// Optionally add a final agent transcript noting the reason
	if strings.TrimSpace(reason) != "" {
		s.AddTranscript(sessionID, models.InterviewTranscript{
			SessionID: sessionID,
			Speaker:   "agent",
			Content:   fmt.Sprintf("Session concluded: %s", reason),
			Timestamp: time.Now(),
		})
	}

	// Reuse the timed-out finalization flow
	s.handleTimedOutSession(session)
}

// IncrementEmptyResponse increments the empty/unintelligible response counter and returns the updated count
func (s *SessionTimeoutService) IncrementEmptyResponse(sessionID string) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, exists := s.activeSessions[sessionID]; exists {
		session.EmptyResponseCount++
		slog.Info("Empty response recorded", "session_id", sessionID, "count", session.EmptyResponseCount)
		return session.EmptyResponseCount
	}
	return 0
}

// ResetEmptyResponse resets the empty response counter for a session
func (s *SessionTimeoutService) ResetEmptyResponse(sessionID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if session, exists := s.activeSessions[sessionID]; exists {
		if session.EmptyResponseCount != 0 {
			session.EmptyResponseCount = 0
			slog.Debug("Empty response counter reset", "session_id", sessionID)
		}
	}
}

func (s *SessionTimeoutService) startTimeoutChecker() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for range ticker.C {
		s.checkTimeouts()
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
		slog.Info("Starting automatic summary generation", "session_id", session.SessionID, "transcript_count", len(session.Transcripts))
		s.generateAutoSummary(ctx, &dbSession, session.Transcripts)
		slog.Info("Automatic summary generation completed", "session_id", session.SessionID)
	} else {
		slog.Warn("No transcripts available for summary generation", "session_id", session.SessionID)
	}

	// Remove from active sessions
	s.EndSession(session.SessionID)
}

func (s *SessionTimeoutService) generateAutoSummary(ctx context.Context, session *models.InterviewSession, transcripts []models.InterviewTranscript) {
	if s.geminiService == nil {
		slog.Warn("Gemini service not available, skipping auto summary generation")
		return
	}

	// Check if summary already exists to prevent duplicates
	var existingSummary models.InterviewSummary
	err := s.db.Where("session_id = ?", session.ID).First(&existingSummary).Error
	if err == nil {
		slog.Info("Summary already exists for session, skipping generation", "session_id", session.ID)
		return
	}
	if err != gorm.ErrRecordNotFound {
		slog.Error("Failed to check for existing summary", "session_id", session.ID, "error", err)
		return
	}

	// Get agent information for personality-based summary
	var agent models.Agent
	if err := s.db.Preload("User").First(&agent, session.AgentID).Error; err != nil {
		slog.Error("Failed to load agent for summary generation", "session_id", session.ID, "error", err)
		return
	}

	// Prepare conversation history for AI analysis
	conversationHistory := make([]string, 0, len(transcripts))
	for _, transcript := range transcripts {
		conversationHistory = append(conversationHistory,
			transcript.Speaker+": "+transcript.Content)
	}

	// Generate personality-based summary using Gemini
	summaryPrompt := s.buildPersonalityBasedSummaryPrompt(agent, conversationHistory)

	slog.Info("Generating AI summary with Gemini", "session_id", session.ID, "agent_name", agent.Name, "conversation_length", len(conversationHistory))
	summary, err := s.geminiService.GenerateSummary(ctx, summaryPrompt)
	if err != nil {
		slog.Error("Failed to generate auto summary", "session_id", session.ID, "error", err)
		return
	}
	slog.Info("AI summary generated successfully", "session_id", session.ID, "summary_length", len(summary))

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
	slog.Info("Summary saved to database", "session_id", session.ID, "summary_id", interviewSummary.ID)

	// Generate performance scores
	s.generatePerformanceScores(ctx, session.ID, parsedSummary)

	slog.Info("Auto summary generation completed successfully", "session_id", session.ID, "overall_score", parsedSummary.OverallScore)
}

// buildPersonalityBasedSummaryPrompt creates a summary prompt tailored to the agent's personality
func (s *SessionTimeoutService) buildPersonalityBasedSummaryPrompt(agent models.Agent, conversationHistory []string) string {
	// Determine scoring strictness based on agent personality
	scoringGuidance := s.getScoringGuidance(agent.Personality)

	// Build industry-specific context
	industryContext := s.getIndustryContext(agent.Industry, agent.Level)

	// Create personality-specific tone and expectations
	personalityTone := s.getPersonalityTone(agent.Personality)

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

// getScoringGuidance returns scoring criteria based on agent personality
func (s *SessionTimeoutService) getScoringGuidance(personality string) string {
	personalityLower := strings.ToLower(personality)

	if strings.Contains(personalityLower, "strict") || strings.Contains(personalityLower, "rigorous") || strings.Contains(personalityLower, "demanding") {
		return "Be very strict and demanding. Only give high scores (80+) for exceptional performance. Average performance should score 50-70. Poor performance should score below 50. Focus heavily on technical accuracy and depth."
	} else if strings.Contains(personalityLower, "encouraging") || strings.Contains(personalityLower, "supportive") || strings.Contains(personalityLower, "mentor") {
		return "Be encouraging and supportive. Give credit for effort and potential. High scores (80+) for good performance with growth potential. Average performance should score 60-80. Focus on potential and learning attitude."
	} else if strings.Contains(personalityLower, "grilling") || strings.Contains(personalityLower, "intense") || strings.Contains(personalityLower, "challenging") {
		return "Be very challenging and thorough. Only give high scores (85+) for outstanding performance under pressure. Average performance should score 40-70. Poor performance should score below 40. Focus on handling pressure and technical depth."
	} else if strings.Contains(personalityLower, "friendly") || strings.Contains(personalityLower, "approachable") || strings.Contains(personalityLower, "collaborative") {
		return "Be fair and balanced. High scores (80+) for strong performance. Average performance should score 60-80. Focus on communication and collaboration skills."
	}

	// Default balanced approach
	return "Be fair and balanced. High scores (80+) for strong performance. Average performance should score 60-80. Focus on both technical skills and soft skills."
}

// getIndustryContext returns industry-specific evaluation criteria
func (s *SessionTimeoutService) getIndustryContext(industry, level string) string {
	switch strings.ToLower(industry) {
	case "software engineering", "technology":
		return "Focus on technical problem-solving, code quality, system design thinking, and ability to learn new technologies. Consider algorithmic thinking, debugging skills, and understanding of software development practices."
	case "finance", "banking":
		return "Focus on analytical thinking, attention to detail, risk assessment, and understanding of financial concepts. Consider quantitative skills, regulatory knowledge, and market awareness."
	case "consulting":
		return "Focus on problem-solving frameworks, client communication, business acumen, and structured thinking. Consider case study performance, presentation skills, and strategic thinking."
	case "marketing", "sales":
		return "Focus on creativity, communication skills, market understanding, and customer orientation. Consider campaign thinking, brand awareness, and persuasive abilities."
	case "healthcare", "medical":
		return "Focus on attention to detail, patient care orientation, medical knowledge, and ethical considerations. Consider clinical thinking, empathy, and professional standards."
	default:
		return "Focus on relevant technical skills, problem-solving abilities, communication, and cultural fit for the role."
	}
}

// getPersonalityTone returns tone guidance based on agent personality
func (s *SessionTimeoutService) getPersonalityTone(personality string) string {
	personalityLower := strings.ToLower(personality)

	if strings.Contains(personalityLower, "strict") || strings.Contains(personalityLower, "rigorous") {
		return "Write your feedback in a direct, professional tone. Be specific about shortcomings and don't sugarcoat issues. Use precise technical language."
	} else if strings.Contains(personalityLower, "encouraging") || strings.Contains(personalityLower, "supportive") {
		return "Write your feedback in an encouraging, constructive tone. Focus on potential and growth opportunities. Be supportive while being honest about areas for improvement."
	} else if strings.Contains(personalityLower, "grilling") || strings.Contains(personalityLower, "intense") {
		return "Write your feedback in a direct, challenging tone. Be thorough in your analysis and don't hold back on criticism. Focus on performance under pressure."
	} else if strings.Contains(personalityLower, "friendly") || strings.Contains(personalityLower, "approachable") {
		return "Write your feedback in a warm, professional tone. Balance constructive criticism with positive reinforcement. Be encouraging while maintaining professionalism."
	}

	// Default professional tone
	return "Write your feedback in a professional, balanced tone. Be constructive and specific in your recommendations."
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

// AddAudioChunk stores an audio chunk for a session
func (s *SessionTimeoutService) AddAudioChunk(sessionID string, chunkData []byte, chunkIndex int, totalChunks int, isLastChunk bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if session, exists := s.activeSessions[sessionID]; exists {
		session.ChunksMutex.Lock()
		defer session.ChunksMutex.Unlock()

		// Store the chunk
		session.AudioChunks[chunkIndex] = make([]byte, len(chunkData))
		copy(session.AudioChunks[chunkIndex], chunkData)
		session.TotalChunks = totalChunks

		slog.Info("Audio chunk stored", "session_id", sessionID, "chunk_index", chunkIndex, "total_chunks", totalChunks)
	}
}

// ReconstructAudio reconstructs the complete audio from stored chunks
func (s *SessionTimeoutService) ReconstructAudio(sessionID string) ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	session, exists := s.activeSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session.ChunksMutex.RLock()
	defer session.ChunksMutex.RUnlock()

	// Check if we have all chunks
	if len(session.AudioChunks) != session.TotalChunks {
		return nil, fmt.Errorf("incomplete chunks: have %d, expected %d", len(session.AudioChunks), session.TotalChunks)
	}

	// Calculate total size
	totalSize := 0
	for i := 0; i < session.TotalChunks; i++ {
		if chunk, exists := session.AudioChunks[i]; exists {
			totalSize += len(chunk)
		} else {
			return nil, fmt.Errorf("missing chunk %d", i)
		}
	}

	// Reconstruct the complete audio
	completeAudio := make([]byte, 0, totalSize)
	for i := 0; i < session.TotalChunks; i++ {
		chunk := session.AudioChunks[i]
		completeAudio = append(completeAudio, chunk...)
	}

	slog.Info("Audio reconstructed from chunks", "session_id", sessionID, "total_chunks", session.TotalChunks)

	// Clear chunks after reconstruction
	session.AudioChunks = make(map[int][]byte)
	session.TotalChunks = 0

	return completeAudio, nil
}
