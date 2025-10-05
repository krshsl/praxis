package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/krshsl/praxis/backend/models"
	"github.com/krshsl/praxis/backend/repository"
	ws "github.com/krshsl/praxis/backend/websocket"
)

type AIMessageProcessor struct {
	geminiService     *GeminiService
	elevenLabsService *ElevenLabsService
	timeoutService    *SessionTimeoutService
	repo              *repository.GORMRepository
}

type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeCode  MessageType = "code"
	MessageTypeAudio MessageType = "audio"
)

type ProcessedMessage struct {
	Type      MessageType `json:"type"`
	Content   string      `json:"content"`
	Language  string      `json:"language,omitempty"`
	SessionID string      `json:"session_id"`
	UserID    string      `json:"user_id"`
}

func NewAIMessageProcessor(
	geminiService *GeminiService,
	elevenLabsService *ElevenLabsService,
	timeoutService *SessionTimeoutService,
	repo *repository.GORMRepository,
) *AIMessageProcessor {
	return &AIMessageProcessor{
		geminiService:     geminiService,
		elevenLabsService: elevenLabsService,
		timeoutService:    timeoutService,
		repo:              repo,
	}
}

// sendMessage sends a message to the WebSocket client
func (p *AIMessageProcessor) sendMessage(client *ws.Client, content string, messageType string, language string) {
	message := ws.Message{
		Type:     messageType,
		Content:  content,
		Language: language,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		slog.Error("Failed to marshal message", "error", err, "session_id", client.SessionID)
		return
	}

	select {
	case client.Send <- messageBytes:
		slog.Info("Message sent to client", "session_id", client.SessionID, "type", messageType, "content_length", len(content))
	default:
		slog.Warn("Failed to send message - client channel full", "session_id", client.SessionID)
	}
}

func (p *AIMessageProcessor) sendUserMessage(client *ws.Client, content string) {
	message := ws.Message{
		Type:     "user_message",
		Content:  content,
		Language: "",
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		slog.Error("Failed to marshal user message", "error", err, "session_id", client.SessionID)
		return
	}

	select {
	case client.Send <- messageBytes:
		slog.Info("User message sent to client", "session_id", client.SessionID, "content_length", len(content))
	default:
		slog.Warn("Failed to send user message - client channel full", "session_id", client.SessionID)
	}
}

func (p *AIMessageProcessor) sendAudioMessage(client *ws.Client, audioData []byte) {
	// Convert audio data to base64
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)

	message := ws.Message{
		Type:            "audio",
		AudioDataBase64: audioBase64,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		slog.Error("Failed to marshal audio message", "error", err, "session_id", client.SessionID)
		return
	}

	select {
	case client.Send <- messageBytes:
		slog.Info("Audio message sent to client", "session_id", client.SessionID, "audio_size", len(audioData))
	default:
		slog.Warn("Failed to send audio message - client channel full", "session_id", client.SessionID)
	}
}

// AutoStartInterview automatically starts the interview when a client connects
func (p *AIMessageProcessor) AutoStartInterview(client *ws.Client) {
	ctx := context.Background()

	slog.Info("Auto-start check", "session_id", client.SessionID)

	// Check if interview has already started by looking for existing transcripts
	existingTranscripts, err := p.repo.GetInterviewTranscripts(ctx, client.SessionID)
	if err != nil {
		slog.Error("Failed to check existing transcripts", "error", err, "session_id", client.SessionID)
		return
	}

	// If there are already transcripts, don't auto-start again
	if len(existingTranscripts) > 0 {
		slog.Info("Interview already started", "session_id", client.SessionID, "existing_transcripts", len(existingTranscripts))
		return
	}

	slog.Info("Starting new interview", "session_id", client.SessionID)

	// Get session and agent from database
	session, err := p.repo.GetInterviewSession(ctx, client.SessionID)
	if err != nil {
		slog.Error("Failed to get interview session for auto-start", "error", err, "session_id", client.SessionID)
		return
	}

	// Get agent details
	agent, err := p.repo.GetAgent(ctx, session.AgentID)
	if err != nil {
		slog.Error("Failed to get agent for auto-start", "error", err, "agent_id", session.AgentID)
		return
	}

	// Generate welcome message using Gemini
	if p.geminiService != nil {
		welcomeMessage := fmt.Sprintf("Hello! I'm %s, and I'll be conducting your %s interview today. I'm excited to learn about your experience and skills. Let's start with a brief introduction - could you tell me about yourself and what brings you to this interview?",
			agent.Name, agent.Industry)

		// Save AI welcome message to database
		if p.repo != nil {
			aiTranscript := &models.InterviewTranscript{
				SessionID: client.SessionID,
				Speaker:   "agent",
				Content:   welcomeMessage,
				TurnOrder: 1,
				Timestamp: time.Now(),
			}

			if err := p.repo.CreateInterviewTranscript(ctx, aiTranscript); err != nil {
				slog.Error("Failed to save AI welcome transcript", "error", err, "session_id", client.SessionID)
			}
		}

		// Send welcome message to client
		p.sendMessage(client, welcomeMessage, "text", "")

		slog.Info("Auto-started interview", "session_id", client.SessionID, "agent", agent.Name)
	}
}

// ProcessAudioChunk handles chunked audio messages from users
func (p *AIMessageProcessor) ProcessAudioChunk(client *ws.Client, audioData []byte, chunkIndex int, totalChunks int, isLastChunk bool) {
	slog.Info("Audio chunk received", "session_id", client.SessionID, "chunk_index", chunkIndex, "total_chunks", totalChunks)

	// Update session activity
	if p.timeoutService != nil && client.SessionID != "" {
		p.timeoutService.UpdateActivity(client.SessionID)
	}

	// Store chunk in session storage
	if p.timeoutService != nil {
		// Add chunk to session storage
		p.timeoutService.AddAudioChunk(client.SessionID, audioData, chunkIndex, totalChunks, isLastChunk)
	}

	// If this is the last chunk, reconstruct and process the complete audio
	if isLastChunk {
		slog.Info("Reconstructing complete audio", "session_id", client.SessionID, "total_chunks", totalChunks)

		// Get all chunks and reconstruct the complete audio
		completeAudio, err := p.timeoutService.ReconstructAudio(client.SessionID)
		if err != nil {
			slog.Error("Failed to reconstruct audio from chunks", "error", err, "session_id", client.SessionID)
			p.sendErrorMessage(client, "Failed to reconstruct audio from chunks")
			return
		}

		slog.Info("Audio reconstructed", "session_id", client.SessionID, "complete_size", len(completeAudio))

		// Process the complete reconstructed audio
		p.processAudioData(client, completeAudio)
	}
}

// processAudioData processes the actual audio data (extracted from ProcessAudioMessage)
func (p *AIMessageProcessor) processAudioData(client *ws.Client, audioData []byte) {
	ctx := context.Background()

	// If audio chunk is too small (<50KB), treat as silence/unintelligible and do not process
	const minAudioSize = 51200 // 50 KB
	if len(audioData) < minAudioSize {
		slog.Info("Audio chunk below 50KB, treating as silence/unintelligible", "session_id", client.SessionID, "audio_size", len(audioData))
		// Instead of sending a user message, send only a hardcoded AI message
		if p.timeoutService != nil && client.SessionID != "" {
			count := p.timeoutService.IncrementEmptyResponse(client.SessionID)
			if count >= 3 {
				finalMsg := "It seems we've had several attempts without a valid response. We'll end the session here and prepare your summary."
				p.sendMessage(client, finalMsg, "text", "")
				// Send end_session message to trigger frontend session end
				p.sendMessage(client, "Session ended", "end_session", "")
				p.timeoutService.ConcludeSession(client.SessionID, "Empty response limit reached")
				return
			}
		}
		// Always send the interviewer warning as an AI message
		p.sendMessage(client, "I couldn't hear a clear response. Please try again.", "text", "")
		return
	}

	// Transcribe audio using Gemini
	if p.geminiService != nil {
		// Add a prompt to Gemini to ignore silence and only transcribe clear speech
		transcriptionPrompt := "Transcribe only clear, intelligible speech. If the audio is silent, empty, or unintelligible, return an empty string."
		transcription, err := p.geminiService.TranscribeAudioWithPrompt(ctx, audioData, transcriptionPrompt)
		if err != nil {
			slog.Error("Failed to transcribe audio", "error", err, "session_id", client.SessionID)
			p.sendErrorMessage(client, "Failed to transcribe audio")
			return
		}

		// Log successful transcription
		slog.Info("Audio transcribed", "session_id", client.SessionID, "transcription_length", len(transcription), "transcription", transcription)

		// Empty/unintelligible response penalty handling (3 strikes)
		trimmed := strings.TrimSpace(transcription)
		lower := strings.ToLower(trimmed)

		// Patterns to treat as empty/unintelligible
		isEmpty := false
		if lower == "" || lower == "[inaudible]" || lower == "[vocalization]" || len([]rune(trimmed)) < 2 {
			isEmpty = true
		}
		// Repeated word patterns (e.g., 'audio audio audio', 'humming humming')
		words := strings.Fields(lower)
		if len(words) > 0 {
			allSame := true
			for _, w := range words {
				if w != words[0] {
					allSame = false
					break
				}
			}
			if allSame && len(words) > 1 {
				isEmpty = true
			}
		}
		// Known non-speech/filler patterns
		badPatterns := []string{"vocalization", "humming", "mumbling", "audio", "noise", "unintelligible"}
		for _, pat := range badPatterns {
			if strings.Contains(lower, pat) && len(words) <= 5 {
				isEmpty = true
				break
			}
		}

		if isEmpty {
			// Instead of sending a user message, send only a hardcoded AI message
			if p.timeoutService != nil && client.SessionID != "" {
				count := p.timeoutService.IncrementEmptyResponse(client.SessionID)
				if count >= 3 {
					finalMsg := "It seems we've had several attempts without a valid response. We'll end the session here and prepare your summary."
					p.sendMessage(client, finalMsg, "text", "")
					p.timeoutService.ConcludeSession(client.SessionID, "Empty response limit reached")
					return
				}
			}
			// Always send the interviewer warning as an AI message
			p.sendMessage(client, "I couldn't hear a clear response. Please try again.", "text", "")
			// Do not proceed further on empty input
			return
		}

		// Reset empty-response counter on valid content
		if p.timeoutService != nil && client.SessionID != "" {
			p.timeoutService.ResetEmptyResponse(client.SessionID)
		}

		// Send user message to frontend
		p.sendUserMessage(client, transcription)

		// Add user transcript
		if p.timeoutService != nil && client.SessionID != "" {
			userTranscript := models.InterviewTranscript{
				SessionID: client.SessionID,
				Speaker:   "user",
				Content:   transcription,
				Timestamp: time.Now(),
			}

			p.timeoutService.AddTranscript(client.SessionID, userTranscript)
		}

		// Generate AI response
		if p.repo != nil {
			// Get conversation history
			conversationHistory, err := p.repo.GetInterviewTranscripts(ctx, client.SessionID)
			if err != nil {
				slog.Error("Failed to get conversation history", "error", err, "session_id", client.SessionID)
				return
			}

			// Get session and agent
			session, err := p.repo.GetInterviewSession(ctx, client.SessionID)
			if err != nil {
				slog.Error("Failed to get interview session", "error", err, "session_id", client.SessionID)
				return
			}

			agent, err := p.repo.GetAgent(ctx, session.AgentID)
			if err != nil {
				slog.Error("Failed to get agent", "error", err, "agent_id", session.AgentID)
				return
			}

			// Check if interview has exceeded 5-minute limit
			if p.timeoutService != nil && p.timeoutService.IsInterviewExpired(client.SessionID) {
				slog.Info("Interview time limit exceeded (5 minutes)", "session_id", client.SessionID)
				endingMessage := "Thank you for your time! We've reached the 5-minute interview limit. This concludes our interview session. We'll review your responses and get back to you soon."
				p.sendMessage(client, endingMessage, "text", "")

				// End the session
				if p.timeoutService != nil {
					p.timeoutService.EndSession(client.SessionID)
				}
				return
			}

			// Generate AI response
			slog.Info("Generating AI response", "session_id", client.SessionID, "transcription", transcription, "history_length", len(conversationHistory))
			aiResponse, err := p.geminiService.GenerateInterviewResponse(ctx, client.SessionID, agent, transcription, conversationHistory)
			if err != nil {
				slog.Error("Failed to generate AI response", "error", err, "session_id", client.SessionID)
				p.sendErrorMessage(client, "Failed to generate AI response")
				return
			}
			slog.Info("AI response generated", "session_id", client.SessionID, "response", aiResponse)

			// Save AI response to database
			if p.timeoutService != nil && client.SessionID != "" {
				aiTranscript := models.InterviewTranscript{
					SessionID: client.SessionID,
					Speaker:   "agent",
					Content:   aiResponse,
					Timestamp: time.Now(),
				}

				p.timeoutService.AddTranscript(client.SessionID, aiTranscript)
			}

			// Send AI response as text to client
			slog.Info("Sending AI response to client", "session_id", client.SessionID, "response_length", len(aiResponse))
			p.sendMessage(client, aiResponse, "text", "")

			// TODO: Re-enable audio generation later
			// Generate and send AI response as audio
			// if p.elevenLabsService != nil {
			// 	audioStream, err := p.elevenLabsService.TextToSpeech(ctx, aiResponse)
			// 	if err != nil {
			// 		slog.Error("Failed to generate AI audio", "error", err, "session_id", client.SessionID)
			// 	} else {
			// 		// Read audio data
			// 		audioData, err := io.ReadAll(audioStream)
			// 		audioStream.Close()
			// 		if err != nil {
			// 			slog.Error("Failed to read AI audio data", "error", err, "session_id", client.SessionID)
			// 		} else {
			// 			// Send audio to client
			// 			p.sendAudioMessage(client, audioData)
			// 		}
			// 	}
			// }
		} // close: if p.repo != nil
	} else {
		slog.Warn("Gemini service not available for audio transcription", "session_id", client.SessionID)
		p.sendErrorMessage(client, "AI service not available")
	}
}

// ProcessTextMessage handles text messages from users
func (p *AIMessageProcessor) ProcessTextMessage(client *ws.Client, content string) {
	ctx := context.Background()

	// Update session activity
	if p.timeoutService != nil && client.SessionID != "" {
		p.timeoutService.UpdateActivity(client.SessionID)

		// Add user transcript
		userTranscript := models.InterviewTranscript{
			SessionID: client.SessionID,
			Speaker:   "user",
			Content:   content,
			TurnOrder: len(client.GetConversationHistory()) + 1,
			Timestamp: time.Now(),
		}
		p.timeoutService.AddTranscript(client.SessionID, userTranscript)
	}

	// Save user message to database
	if p.repo != nil {
		userTranscript := &models.InterviewTranscript{
			SessionID: client.SessionID,
			Speaker:   "user",
			Content:   content,
			TurnOrder: len(client.GetConversationHistory()) + 1,
			Timestamp: time.Now(),
		}

		if err := p.repo.CreateInterviewTranscript(ctx, userTranscript); err != nil {
			slog.Error("Failed to save user transcript", "error", err, "session_id", client.SessionID)
		}
	}

	// Handle empty text content with penalty (3 strikes)
	if strings.TrimSpace(content) == "" {
		if p.timeoutService != nil && client.SessionID != "" {
			count := p.timeoutService.IncrementEmptyResponse(client.SessionID)
			if count >= 3 {
				finalMsg := "It seems we've had several attempts without a valid response. We'll end the session here and prepare your summary."
				p.sendMessage(client, finalMsg, "text", "")
				// Send end_session message to trigger frontend session end
				p.sendMessage(client, "Session ended", "end_session", "")
				p.timeoutService.ConcludeSession(client.SessionID, "Empty response limit reached")
				return
			}
			warning := fmt.Sprintf("I couldn't read a valid response. Please try again. (Warning %d/3)", count)
			p.sendMessage(client, warning, "text", "")
			return
		}
	}

	// Reset empty-response counter on valid content
	if p.timeoutService != nil && client.SessionID != "" {
		p.timeoutService.ResetEmptyResponse(client.SessionID)
	}

	// Get session and agent from database
	session, err := p.repo.GetInterviewSession(ctx, client.SessionID)
	if err != nil {
		slog.Error("Failed to get interview session", "error", err, "session_id", client.SessionID)
		p.sendErrorMessage(client, "Failed to retrieve interview session")
		return
	}

	// Get agent details
	agent, err := p.repo.GetAgent(ctx, session.AgentID)
	if err != nil {
		slog.Error("Failed to get agent", "error", err, "agent_id", session.AgentID)
		p.sendErrorMessage(client, "Failed to retrieve interviewer details")
		return
	}

	// Get conversation history from database
	transcripts, err := p.repo.GetInterviewTranscripts(ctx, client.SessionID)
	if err != nil {
		slog.Error("Failed to get conversation history", "error", err, "session_id", client.SessionID)
		transcripts = []models.InterviewTranscript{} // Continue with empty history
	}

	// Generate AI response using Gemini with session cache
	if p.geminiService != nil {
		response, err := p.geminiService.GenerateInterviewResponse(ctx, client.SessionID, agent, content, transcripts)
		if err != nil {
			slog.Error("Failed to generate AI response", "error", err, "session_id", client.SessionID)
			p.sendErrorMessage(client, "Failed to generate AI response")
			return
		}

		// Update session activity for AI response
		if p.timeoutService != nil && client.SessionID != "" {
			p.timeoutService.UpdateActivity(client.SessionID)

			// Add agent transcript
			agentTranscript := models.InterviewTranscript{
				SessionID: client.SessionID,
				Speaker:   "agent",
				Content:   response,
				TurnOrder: len(client.GetConversationHistory()) + 2,
				Timestamp: time.Now(),
			}
			p.timeoutService.AddTranscript(client.SessionID, agentTranscript)
		}

		// Save agent response to database
		if p.repo != nil {
			agentTranscript := &models.InterviewTranscript{
				SessionID: client.SessionID,
				Speaker:   "agent",
				Content:   response,
				TurnOrder: len(client.GetConversationHistory()) + 1,
				Timestamp: time.Now(),
			}

			if err := p.repo.CreateInterviewTranscript(ctx, agentTranscript); err != nil {
				slog.Error("Failed to save agent transcript", "error", err, "session_id", client.SessionID)
			}
		}

		// Convert to speech using ElevenLabs
		if p.elevenLabsService != nil {
			audioStream, err := p.elevenLabsService.TextToSpeech(ctx, response)
			if err != nil {
				slog.Error("Failed to generate speech", "error", err, "session_id", client.SessionID)
				// Send text response as fallback
				p.sendTextResponse(client, response)
				return
			}
			defer audioStream.Close()

			// Read audio data and send to client
			audioData, err := p.readAudioData(audioStream)
			if err != nil {
				slog.Error("Failed to read audio data", "error", err, "session_id", client.SessionID)
				// Send text response as fallback
				p.sendTextResponse(client, response)
				return
			}

			// Send audio to client
			client.SendAudio(audioData)
		} else {
			// Send text response if no audio service
			p.sendTextResponse(client, response)
		}
	} else {
		slog.Warn("Gemini service not available", "session_id", client.SessionID)
		p.sendErrorMessage(client, "AI service not available")
	}
}

// ProcessCodeMessage handles code submission messages
func (p *AIMessageProcessor) ProcessCodeMessage(client *ws.Client, content, language string) {
	ctx := context.Background()

	// Update session activity
	if p.timeoutService != nil && client.SessionID != "" {
		p.timeoutService.UpdateActivity(client.SessionID)
	}

	// Analyze code using Gemini
	if p.geminiService != nil {
		analysis, err := p.geminiService.AnalyzeCode(ctx, content, language)
		if err != nil {
			slog.Error("Failed to analyze code", "error", err, "session_id", client.SessionID)
			p.sendErrorMessage(client, "Failed to analyze code")
			return
		}

		// Update session activity for AI response
		if p.timeoutService != nil && client.SessionID != "" {
			p.timeoutService.UpdateActivity(client.SessionID)

			// Add agent transcript
			agentTranscript := models.InterviewTranscript{
				SessionID: client.SessionID,
				Speaker:   "agent",
				Content:   analysis,
				TurnOrder: len(client.GetConversationHistory()) + 1,
				Timestamp: time.Now(),
			}
			p.timeoutService.AddTranscript(client.SessionID, agentTranscript)
		}

		// Save code analysis to database
		if p.repo != nil {
			agentTranscript := &models.InterviewTranscript{
				SessionID: client.SessionID,
				Speaker:   "agent",
				Content:   analysis,
				TurnOrder: len(client.GetConversationHistory()) + 1,
				Timestamp: time.Now(),
			}

			if err := p.repo.CreateInterviewTranscript(ctx, agentTranscript); err != nil {
				slog.Error("Failed to save code analysis transcript", "error", err, "session_id", client.SessionID)
			}
		}

		// Convert analysis to speech
		if p.elevenLabsService != nil {
			audioStream, err := p.elevenLabsService.TextToSpeech(ctx, analysis)
			if err != nil {
				slog.Error("Failed to generate speech for code analysis", "error", err, "session_id", client.SessionID)
				// Send text response as fallback
				p.sendTextResponse(client, analysis)
				return
			}
			defer audioStream.Close()

			// Read audio data and send to client
			audioData, err := p.readAudioData(audioStream)
			if err != nil {
				slog.Error("Failed to read audio data", "error", err, "session_id", client.SessionID)
				// Send text response as fallback
				p.sendTextResponse(client, analysis)
				return
			}

			// Send audio to client
			client.SendAudio(audioData)
		} else {
			// Send text response if no audio service
			p.sendTextResponse(client, analysis)
		}
	} else {
		slog.Warn("Gemini service not available for code analysis", "session_id", client.SessionID)
		p.sendErrorMessage(client, "AI service not available")
	}
}

// ProcessAudioMessage handles audio messages from users
func (p *AIMessageProcessor) ProcessAudioMessage(client *ws.Client, audioData []byte) {
	slog.Info("Audio received", "session_id", client.SessionID, "audio_size", len(audioData))
	if p.timeoutService != nil && client.SessionID != "" {
		p.timeoutService.UpdateActivity(client.SessionID)
	}
	// Delegate to shared processing
	p.processAudioData(client, audioData)
}

// Helper methods

func (p *AIMessageProcessor) readAudioData(audioStream interface{}) ([]byte, error) {
	// This is a placeholder - you'll need to implement based on your audio stream type
	// For now, return empty bytes
	return []byte{}, nil
}

func (p *AIMessageProcessor) sendTextResponse(client *ws.Client, content string) {
	response := map[string]interface{}{
		"type":    "text",
		"content": content,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		slog.Error("Failed to marshal text response", "error", err)
		return
	}

	client.Send <- responseBytes
}

func (p *AIMessageProcessor) sendErrorMessage(client *ws.Client, message string) {
	errorResponse := map[string]interface{}{
		"type":    "error",
		"content": message,
	}

	errorBytes, err := json.Marshal(errorResponse)
	if err != nil {
		slog.Error("Failed to marshal error response", "error", err)
		return
	}

	client.Send <- errorBytes
}

func (p *AIMessageProcessor) decodeBase64Audio(audioData []byte) ([]byte, error) {
	// Decode base64 audio data
	decoded, err := base64.StdEncoding.DecodeString(string(audioData))
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
