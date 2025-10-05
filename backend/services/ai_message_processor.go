package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
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
	ctx := context.Background()

	// Update session activity
	if p.timeoutService != nil && client.SessionID != "" {
		p.timeoutService.UpdateActivity(client.SessionID)
	}

	// Decode base64 audio data
	decodedAudioData, err := p.decodeBase64Audio(audioData)
	if err != nil {
		slog.Error("Failed to decode audio data", "error", err, "session_id", client.SessionID)
		p.sendErrorMessage(client, "Failed to decode audio data")
		return
	}

	// Transcribe audio using Gemini
	if p.geminiService != nil {
		transcription, err := p.geminiService.TranscribeAudio(ctx, decodedAudioData)
		if err != nil {
			slog.Error("Failed to transcribe audio", "error", err, "session_id", client.SessionID)
			p.sendErrorMessage(client, "Failed to transcribe audio")
			return
		}

		// Add user transcript
		if p.timeoutService != nil && client.SessionID != "" {
			userTranscript := models.InterviewTranscript{
				SessionID: client.SessionID,
				Speaker:   "user",
				Content:   transcription,
				TurnOrder: len(client.GetConversationHistory()) + 1,
				Timestamp: time.Now(),
			}
			p.timeoutService.AddTranscript(client.SessionID, userTranscript)
		}

		// Save user transcript to database
		if p.repo != nil {
			userTranscript := &models.InterviewTranscript{
				SessionID: client.SessionID,
				Speaker:   "user",
				Content:   transcription,
				TurnOrder: len(client.GetConversationHistory()) + 1,
				Timestamp: time.Now(),
			}

			if err := p.repo.CreateInterviewTranscript(ctx, userTranscript); err != nil {
				slog.Error("Failed to save user transcript", "error", err, "session_id", client.SessionID)
			}
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
		response, err := p.geminiService.GenerateInterviewResponse(ctx, client.SessionID, agent, transcription, transcripts)
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
		slog.Warn("Gemini service not available for audio processing", "session_id", client.SessionID)
		p.sendErrorMessage(client, "AI service not available")
	}
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
