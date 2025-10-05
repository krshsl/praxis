package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/krshsl/praxis/backend/models"

	"google.golang.org/genai"
)

const (
	ModelName                    = "gemini-2.5-flash"
	MaxConversationTurns         = 20    // Maximum turns before summarization
	MaxTokensBeforeSummarization = 30000 // Approximate token limit
)

// GeminiService handles all Gemini AI operations with caching and session management
type GeminiService struct {
	genaiClient *genai.Client

	// Per-session cache management
	sessionCaches map[string]*SessionCache
	cacheMutex    sync.RWMutex
}

// SessionCache holds the cache and chat session for an interview
type SessionCache struct {
	CacheName           string
	ConversationSummary string
	TurnCount           int
	LastActivity        time.Time
	Agent               *models.Agent
}

func NewGeminiService(apiKey string) *GeminiService {
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		slog.Error("Failed to create genai client", "error", err)
		return nil
	}

	service := &GeminiService{
		genaiClient:   genaiClient,
		sessionCaches: make(map[string]*SessionCache),
	}

	// Start background cleanup of stale caches
	go service.cleanupStaleCaches()

	return service
}

// GetOrCreateSessionCache gets or creates a cached session for an interview
func (g *GeminiService) GetOrCreateSessionCache(ctx context.Context, sessionID string, agent *models.Agent) (*SessionCache, error) {
	g.cacheMutex.Lock()
	defer g.cacheMutex.Unlock()

	// Check if cache already exists
	if cache, exists := g.sessionCaches[sessionID]; exists {
		cache.LastActivity = time.Now()
		return cache, nil
	}

	// Create system instruction based on agent personality
	systemInstruction := g.buildSystemInstruction(agent)

	// Create initial cached content with system instruction
	cacheConfig := &genai.CreateCachedContentConfig{
		Contents: []*genai.Content{
			genai.NewContentFromText(systemInstruction, genai.RoleUser),
		},
		SystemInstruction: genai.NewContentFromText(
			fmt.Sprintf("You are %s. %s", agent.Name, agent.Personality),
			genai.RoleUser,
		),
	}

	cache, err := g.genaiClient.Caches.Create(ctx, ModelName, cacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	sessionCache := &SessionCache{
		CacheName:    cache.Name,
		TurnCount:    0,
		LastActivity: time.Now(),
		Agent:        agent,
	}

	g.sessionCaches[sessionID] = sessionCache
	slog.Info("Created new session cache", "session_id", sessionID, "agent", agent.Name, "cache_name", cache.Name)

	return sessionCache, nil
}

// GenerateInterviewResponse generates AI response using the session cache
func (g *GeminiService) GenerateInterviewResponse(ctx context.Context, sessionID string, agent *models.Agent, userMessage string, conversationHistory []models.InterviewTranscript) (string, error) {
	if g.genaiClient == nil {
		return "", fmt.Errorf("genai client not initialized")
	}

	// Get or create session cache
	sessionCache, err := g.GetOrCreateSessionCache(ctx, sessionID, agent)
	if err != nil {
		return "", fmt.Errorf("failed to get session cache: %w", err)
	}

	// Check if we need to summarize and recreate cache
	if sessionCache.TurnCount >= MaxConversationTurns {
		slog.Info("Conversation too long, creating summary", "session_id", sessionID, "turns", sessionCache.TurnCount)
		if err := g.summarizeAndRecreateCache(ctx, sessionID, agent, conversationHistory); err != nil {
			slog.Error("Failed to summarize conversation", "error", err, "session_id", sessionID)
			// Continue anyway with existing cache
		}
	}

	// Build conversation history for context
	historyContents := g.buildConversationContents(conversationHistory, sessionCache.ConversationSummary)

	// Add current user message
	historyContents = append(historyContents, genai.NewContentFromText(userMessage, genai.RoleUser))

	// Generate response using cached content
	thinkingBudget := int32(-1)
	config := &genai.GenerateContentConfig{
		CachedContent: sessionCache.CacheName,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: &thinkingBudget, // Dynamic thinking
		},
	}

	result, err := g.genaiClient.Models.GenerateContent(
		ctx,
		ModelName,
		historyContents,
		config,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	response := result.Text()

	// Update session cache
	g.cacheMutex.Lock()
	sessionCache.TurnCount++
	sessionCache.LastActivity = time.Now()
	g.cacheMutex.Unlock()

	slog.Info("Generated interview response",
		"session_id", sessionID,
		"turns", sessionCache.TurnCount,
		"response_length", len(response))

	return response, nil
}

// TranscribeAudio transcribes audio using Gemini
func (g *GeminiService) TranscribeAudio(ctx context.Context, audioData []byte) (string, error) {
	slog.Info("Transcribing audio with Gemini", "size", len(audioData))

	if g.genaiClient == nil {
		return "", fmt.Errorf("genai client not initialized")
	}

	// Save audio to temp file
	tmpFile, err := os.CreateTemp("", "audio-*.webm")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(audioData); err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}
	tmpFile.Close()

	// Upload audio file to Gemini
	uploadedFile, err := g.genaiClient.Files.UploadFromPath(
		ctx,
		tmpFile.Name(),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload audio: %w", err)
	}

	// Create transcription request
	parts := []*genai.Part{
		genai.NewPartFromText("Transcribe this audio to text. Provide only the transcript, no additional commentary."),
		genai.NewPartFromURI(uploadedFile.URI, uploadedFile.MIMEType),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	// Generate transcript
	result, err := g.genaiClient.Models.GenerateContent(
		ctx,
		ModelName,
		contents,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate transcript: %w", err)
	}

	transcript := result.Text()
	slog.Info("Audio transcribed successfully", "transcript_length", len(transcript))

	return transcript, nil
}

// AnalyzeCode analyzes code with Gemini
func (g *GeminiService) AnalyzeCode(ctx context.Context, code string, language string) (string, error) {
	if g.genaiClient == nil {
		return "", fmt.Errorf("genai client not initialized")
	}

	prompt := fmt.Sprintf(`You are an expert code reviewer and technical interviewer. Analyze the following %s code and provide constructive feedback:

Code:
%s

Please provide:
1. Code quality assessment (readability, efficiency, best practices)
2. Potential bugs or issues
3. Suggestions for improvement
4. Overall technical skill evaluation

Be specific and actionable in your feedback.`, language, code)

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(
			"You are an expert technical interviewer and code reviewer.",
			genai.RoleUser,
		),
	}

	result, err := g.genaiClient.Models.GenerateContent(
		ctx,
		ModelName,
		genai.Text(prompt),
		config,
	)
	if err != nil {
		return "", fmt.Errorf("failed to analyze code: %w", err)
	}

	return result.Text(), nil
}

// Helper functions

func (g *GeminiService) buildSystemInstruction(agent *models.Agent) string {
	return fmt.Sprintf(`You are %s, a professional %s interviewer for %s positions.

Your personality: %s

Your role:
- Conduct technical interviews with professionalism and empathy
- Ask relevant questions based on the candidate's level (%s)
- Provide constructive feedback
- Evaluate technical skills, communication, and problem-solving abilities
- Keep responses concise and engaging
- Ask follow-up questions to dive deeper into topics

Remember to adapt your questions and evaluation criteria to the %s level.`,
		agent.Name,
		agent.Industry,
		agent.Level,
		agent.Personality,
		agent.Level,
		agent.Level,
	)
}

func (g *GeminiService) buildConversationContents(transcripts []models.InterviewTranscript, summary string) []*genai.Content {
	var contents []*genai.Content

	// Add summary if exists
	if summary != "" {
		contents = append(contents, genai.NewContentFromText(
			fmt.Sprintf("Previous conversation summary: %s", summary),
			genai.RoleModel,
		))
	}

	// Add recent conversation history (last 10 turns to avoid context bloat)
	startIdx := 0
	if len(transcripts) > 10 {
		startIdx = len(transcripts) - 10
	}

	for _, transcript := range transcripts[startIdx:] {
		if transcript.Speaker == "agent" {
			contents = append(contents, genai.NewContentFromText(transcript.Content, genai.RoleModel))
		} else {
			contents = append(contents, genai.NewContentFromText(transcript.Content, genai.RoleUser))
		}
	}

	return contents
}

func (g *GeminiService) summarizeAndRecreateCache(ctx context.Context, sessionID string, agent *models.Agent, transcripts []models.InterviewTranscript) error {
	g.cacheMutex.Lock()
	defer g.cacheMutex.Unlock()

	// Build conversation text for summarization
	var conversationText strings.Builder
	for _, transcript := range transcripts {
		conversationText.WriteString(fmt.Sprintf("%s: %s\n", transcript.Speaker, transcript.Content))
	}

	// Generate summary
	summaryPrompt := fmt.Sprintf(`Summarize the following interview conversation concisely, focusing on:
- Key topics discussed
- Candidate's responses and insights
- Technical assessments made
- Any areas that need follow-up

Conversation:
%s

Provide a clear, concise summary (max 500 words).`, conversationText.String())

	result, err := g.genaiClient.Models.GenerateContent(
		ctx,
		ModelName,
		genai.Text(summaryPrompt),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	summary := result.Text()

	// Create new cache with summary
	systemInstruction := g.buildSystemInstruction(agent)
	cacheConfig := &genai.CreateCachedContentConfig{
		Contents: []*genai.Content{
			genai.NewContentFromText(systemInstruction, genai.RoleUser),
			genai.NewContentFromText(fmt.Sprintf("Conversation summary so far: %s", summary), genai.RoleModel),
		},
		SystemInstruction: genai.NewContentFromText(
			fmt.Sprintf("You are %s. %s", agent.Name, agent.Personality),
			genai.RoleUser,
		),
	}

	newCache, err := g.genaiClient.Caches.Create(ctx, ModelName, cacheConfig)
	if err != nil {
		return fmt.Errorf("failed to create new cache: %w", err)
	}

	// Update session cache
	if sessionCache, exists := g.sessionCaches[sessionID]; exists {
		sessionCache.CacheName = newCache.Name
		sessionCache.ConversationSummary = summary
		sessionCache.TurnCount = 0
		slog.Info("Recreated session cache with summary", "session_id", sessionID, "summary_length", len(summary))
	}

	return nil
}

func (g *GeminiService) cleanupStaleCaches() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		g.cacheMutex.Lock()
		now := time.Now()
		for sessionID, cache := range g.sessionCaches {
			// Remove caches inactive for more than 2 hours
			if now.Sub(cache.LastActivity) > 2*time.Hour {
				delete(g.sessionCaches, sessionID)
				slog.Info("Cleaned up stale session cache", "session_id", sessionID)
			}
		}
		g.cacheMutex.Unlock()
	}
}

// ClearSessionCache removes a session cache (called when interview ends)
func (g *GeminiService) ClearSessionCache(sessionID string) {
	g.cacheMutex.Lock()
	defer g.cacheMutex.Unlock()

	delete(g.sessionCaches, sessionID)
	slog.Info("Cleared session cache", "session_id", sessionID)
}

// GenerateSummary generates a simple text summary without caching (used for timeout summaries)
func (g *GeminiService) GenerateSummary(ctx context.Context, prompt string) (string, error) {
	if g.genaiClient == nil {
		return "", fmt.Errorf("genai client not initialized")
	}

	result, err := g.genaiClient.Models.GenerateContent(
		ctx,
		ModelName,
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return result.Text(), nil
}
